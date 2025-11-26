package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AsyncEvaluationService handles asynchronous guardrail evaluations with webhook callbacks
type AsyncEvaluationService struct {
	guardrailService *GuardrailService
	httpClient       *http.Client
	logger           *zap.Logger
	jobQueue         chan *AsyncEvaluationJob
	workers          int
}

// AsyncEvaluationJob represents an async evaluation job
type AsyncEvaluationJob struct {
	ID           uuid.UUID
	Input        *EvaluationInput
	WebhookURL   string
	WebhookAuth  string // Bearer token for webhook authentication
	Status       string // pending, processing, completed, failed
	Result       *EvaluationResult
	Error        string
	CreatedAt    time.Time
	CompletedAt  *time.Time
	RetryCount   int
	MaxRetries   int
}

// AsyncEvaluationResponse is sent to the webhook
type AsyncEvaluationResponse struct {
	JobID       string             `json:"job_id"`
	Status      string             `json:"status"`
	Result      *EvaluationResult  `json:"result,omitempty"`
	Error       string             `json:"error,omitempty"`
	CompletedAt string             `json:"completed_at,omitempty"`
}

// NewAsyncEvaluationService creates a new async evaluation service
func NewAsyncEvaluationService(
	guardrailService *GuardrailService,
	logger *zap.Logger,
	workers int,
) *AsyncEvaluationService {
	if workers == 0 {
		workers = 5 // Default to 5 workers
	}

	service := &AsyncEvaluationService{
		guardrailService: guardrailService,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:   logger,
		jobQueue: make(chan *AsyncEvaluationJob, 1000),
		workers:  workers,
	}

	// Start worker pool
	service.startWorkers()

	return service
}

// SubmitEvaluation submits an evaluation job for async processing
func (s *AsyncEvaluationService) SubmitEvaluation(
	input *EvaluationInput,
	webhookURL string,
	webhookAuth string,
	maxRetries int,
) (*AsyncEvaluationJob, error) {
	if maxRetries == 0 {
		maxRetries = 3 // Default retry count
	}

	job := &AsyncEvaluationJob{
		ID:          uuid.New(),
		Input:       input,
		WebhookURL:  webhookURL,
		WebhookAuth: webhookAuth,
		Status:      "pending",
		CreatedAt:   time.Now(),
		MaxRetries:  maxRetries,
	}

	// Submit to queue (non-blocking with timeout)
	select {
	case s.jobQueue <- job:
		s.logger.Info("async evaluation job submitted",
			zap.String("job_id", job.ID.String()),
			zap.String("project_id", input.ProjectID.String()),
		)
		return job, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("job queue is full, try again later")
	}
}

// startWorkers starts the worker pool for processing jobs
func (s *AsyncEvaluationService) startWorkers() {
	for i := 0; i < s.workers; i++ {
		go s.worker(i)
	}
	s.logger.Info("async evaluation workers started", zap.Int("count", s.workers))
}

// worker processes jobs from the queue
func (s *AsyncEvaluationService) worker(id int) {
	s.logger.Debug("worker started", zap.Int("worker_id", id))

	for job := range s.jobQueue {
		s.processJob(job)
	}
}

// processJob processes a single evaluation job
func (s *AsyncEvaluationService) processJob(job *AsyncEvaluationJob) {
	job.Status = "processing"

	s.logger.Info("processing async evaluation",
		zap.String("job_id", job.ID.String()),
		zap.String("project_id", job.Input.ProjectID.String()),
	)

	// Execute evaluation with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := s.guardrailService.Evaluate(ctx, job.Input)

	now := time.Now()
	job.CompletedAt = &now

	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		s.logger.Error("async evaluation failed",
			zap.Error(err),
			zap.String("job_id", job.ID.String()),
		)
	} else {
		job.Status = "completed"
		job.Result = result
		s.logger.Info("async evaluation completed",
			zap.String("job_id", job.ID.String()),
			zap.Bool("passed", result.Passed),
		)
	}

	// Send webhook notification
	if job.WebhookURL != "" {
		if err := s.sendWebhook(job); err != nil {
			s.logger.Error("webhook delivery failed",
				zap.Error(err),
				zap.String("job_id", job.ID.String()),
				zap.String("webhook_url", job.WebhookURL),
			)

			// Retry webhook if under max retries
			if job.RetryCount < job.MaxRetries {
				job.RetryCount++
				s.logger.Info("retrying webhook",
					zap.String("job_id", job.ID.String()),
					zap.Int("attempt", job.RetryCount),
				)
				time.Sleep(time.Duration(job.RetryCount) * time.Second)
				_ = s.sendWebhook(job)
			}
		}
	}
}

// sendWebhook sends the evaluation result to the webhook URL
func (s *AsyncEvaluationService) sendWebhook(job *AsyncEvaluationJob) error {
	response := &AsyncEvaluationResponse{
		JobID:  job.ID.String(),
		Status: job.Status,
		Result: job.Result,
		Error:  job.Error,
	}

	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt.Format(time.RFC3339)
	}

	payload, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, job.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OTelGuard-Job-ID", job.ID.String())
	req.Header.Set("X-OTelGuard-Status", job.Status)

	if job.WebhookAuth != "" {
		req.Header.Set("Authorization", "Bearer "+job.WebhookAuth)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	s.logger.Info("webhook delivered successfully",
		zap.String("job_id", job.ID.String()),
		zap.String("webhook_url", job.WebhookURL),
		zap.Int("status_code", resp.StatusCode),
	)

	return nil
}

// GetQueueSize returns the current size of the job queue
func (s *AsyncEvaluationService) GetQueueSize() int {
	return len(s.jobQueue)
}

// Shutdown gracefully shuts down the service
func (s *AsyncEvaluationService) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down async evaluation service")

	// Close the queue
	close(s.jobQueue)

	// Wait for workers to finish (with timeout)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		s.logger.Warn("async evaluation service shutdown timeout")
		return nil
	}
}

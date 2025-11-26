package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AsyncEvaluationJob represents an asynchronous evaluation job
type AsyncEvaluationJob struct {
	ID          uuid.UUID
	Input       *EvaluationInput
	WebhookURL  string
	Status      string // pending, running, completed, failed
	Result      *EvaluationResult
	Error       string
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// AsyncEvaluationService handles async evaluation jobs
type AsyncEvaluationService struct {
	guardrailService *GuardrailService
	jobs             map[uuid.UUID]*AsyncEvaluationJob
	jobsMu           sync.RWMutex
	logger           *zap.Logger
	httpClient       *http.Client
}

// NewAsyncEvaluationService creates a new async evaluation service
func NewAsyncEvaluationService(guardrailService *GuardrailService, logger *zap.Logger) *AsyncEvaluationService {
	return &AsyncEvaluationService{
		guardrailService: guardrailService,
		jobs:             make(map[uuid.UUID]*AsyncEvaluationJob),
		logger:           logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SubmitJob submits an async evaluation job
func (s *AsyncEvaluationService) SubmitJob(ctx context.Context, input *EvaluationInput, webhookURL string) (*AsyncEvaluationJob, error) {
	job := &AsyncEvaluationJob{
		ID:         uuid.New(),
		Input:      input,
		WebhookURL: webhookURL,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	s.jobsMu.Lock()
	s.jobs[job.ID] = job
	s.jobsMu.Unlock()

	// Start processing job in goroutine
	go s.processJob(job)

	s.logger.Info("submitted async evaluation job",
		zap.String("job_id", job.ID.String()),
		zap.String("webhook_url", webhookURL),
	)

	return job, nil
}

// GetJob retrieves a job by ID
func (s *AsyncEvaluationService) GetJob(ctx context.Context, jobID uuid.UUID) (*AsyncEvaluationJob, error) {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}

	return job, nil
}

// ListJobs lists all jobs for a project
func (s *AsyncEvaluationService) ListJobs(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*AsyncEvaluationJob, int, error) {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	// Filter jobs by project
	var filtered []*AsyncEvaluationJob
	for _, job := range s.jobs {
		if job.Input.ProjectID == projectID {
			filtered = append(filtered, job)
		}
	}

	// Sort by created time (newest first)
	// Simple implementation - in production, use proper sorting
	total := len(filtered)

	// Apply pagination
	start := offset
	end := offset + limit
	if start > total {
		return []*AsyncEvaluationJob{}, total, nil
	}
	if end > total {
		end = total
	}

	return filtered[start:end], total, nil
}

// processJob processes an evaluation job
func (s *AsyncEvaluationService) processJob(job *AsyncEvaluationJob) {
	ctx := context.Background()

	// Update status to running
	s.jobsMu.Lock()
	job.Status = "running"
	now := time.Now()
	job.StartedAt = &now
	s.jobsMu.Unlock()

	s.logger.Info("processing async evaluation job", zap.String("job_id", job.ID.String()))

	// Execute evaluation
	result, err := s.guardrailService.Evaluate(ctx, job.Input)
	if err != nil {
		s.jobsMu.Lock()
		job.Status = "failed"
		job.Error = err.Error()
		completedAt := time.Now()
		job.CompletedAt = &completedAt
		s.jobsMu.Unlock()

		s.logger.Error("async evaluation job failed",
			zap.String("job_id", job.ID.String()),
			zap.Error(err),
		)

		// Send failure webhook
		if job.WebhookURL != "" {
			s.sendWebhook(job, result, err)
		}

		return
	}

	// Update job with result
	s.jobsMu.Lock()
	job.Status = "completed"
	job.Result = result
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	s.jobsMu.Unlock()

	s.logger.Info("async evaluation job completed",
		zap.String("job_id", job.ID.String()),
		zap.Bool("passed", result.Passed),
	)

	// Send success webhook
	if job.WebhookURL != "" {
		s.sendWebhook(job, result, nil)
	}
}

// sendWebhook sends a webhook notification
func (s *AsyncEvaluationService) sendWebhook(job *AsyncEvaluationJob, result *EvaluationResult, evalErr error) {
	// Prepare webhook payload
	payload := map[string]interface{}{
		"job_id":     job.ID.String(),
		"status":     job.Status,
		"created_at": job.CreatedAt,
		"completed_at": job.CompletedAt,
	}

	if evalErr != nil {
		payload["error"] = evalErr.Error()
	} else if result != nil {
		payload["result"] = map[string]interface{}{
			"passed":       result.Passed,
			"violations":   result.Violations,
			"remediated":   result.Remediated,
			"output":       result.Output,
			"latency_ms":   result.LatencyMs,
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("failed to marshal webhook payload",
			zap.Error(err),
			zap.String("job_id", job.ID.String()),
		)
		return
	}

	// Send webhook request
	req, err := http.NewRequest("POST", job.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		s.logger.Error("failed to create webhook request",
			zap.Error(err),
			zap.String("job_id", job.ID.String()),
		)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OTelGuard-Event", "evaluation.completed")
	req.Header.Set("X-OTelGuard-Job-ID", job.ID.String())

	// Send with retry
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		resp, err := s.httpClient.Do(req)
		if err != nil {
			s.logger.Warn("webhook request failed",
				zap.Error(err),
				zap.String("job_id", job.ID.String()),
				zap.Int("attempt", i+1),
			)
			time.Sleep(time.Duration(i+1) * time.Second) // exponential backoff
			continue
		}

		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			s.logger.Info("webhook sent successfully",
				zap.String("job_id", job.ID.String()),
				zap.String("webhook_url", job.WebhookURL),
				zap.Int("status_code", resp.StatusCode),
			)
			return
		}

		s.logger.Warn("webhook returned non-success status",
			zap.String("job_id", job.ID.String()),
			zap.Int("status_code", resp.StatusCode),
			zap.Int("attempt", i+1),
		)

		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	s.logger.Error("webhook delivery failed after retries",
		zap.String("job_id", job.ID.String()),
		zap.String("webhook_url", job.WebhookURL),
	)
}

// CleanupOldJobs removes completed/failed jobs older than the retention period
func (s *AsyncEvaluationService) CleanupOldJobs(retentionDuration time.Duration) int {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	cutoff := time.Now().Add(-retentionDuration)
	removed := 0

	for id, job := range s.jobs {
		if job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
			delete(s.jobs, id)
			removed++
		}
	}

	if removed > 0 {
		s.logger.Info("cleaned up old async evaluation jobs", zap.Int("count", removed))
	}

	return removed
}

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"

	"github.com/otelguard/otelguard/internal/domain"
	chrepo "github.com/otelguard/otelguard/internal/repository/clickhouse"
	pgrepo "github.com/otelguard/otelguard/internal/repository/postgres"
)

// EvaluatorService handles LLM-as-a-Judge evaluation operations
type EvaluatorService struct {
	evaluatorRepo *pgrepo.EvaluatorRepository
	jobRepo       *pgrepo.EvaluationJobRepository
	resultRepo    *chrepo.EvaluationResultRepository
	traceRepo     *chrepo.TraceRepository
	llmService    LLMService
	pricing       *PricingService
	logger        *zap.Logger

	// Job queue
	jobQueue     chan uuid.UUID
	stopChan     chan struct{}
	wg           sync.WaitGroup
	workerCount  int
	isRunning    bool
	runningMutex sync.RWMutex
}

// NewEvaluatorService creates a new EvaluatorService
func NewEvaluatorService(
	evaluatorRepo *pgrepo.EvaluatorRepository,
	jobRepo *pgrepo.EvaluationJobRepository,
	resultRepo *chrepo.EvaluationResultRepository,
	traceRepo *chrepo.TraceRepository,
	llmService LLMService,
	pricing *PricingService,
	logger *zap.Logger,
) *EvaluatorService {
	return &EvaluatorService{
		evaluatorRepo: evaluatorRepo,
		jobRepo:       jobRepo,
		resultRepo:    resultRepo,
		traceRepo:     traceRepo,
		llmService:    llmService,
		pricing:       pricing,
		logger:        logger,
		jobQueue:      make(chan uuid.UUID, 1000),
		stopChan:      make(chan struct{}),
		workerCount:   3, // Number of concurrent evaluation workers
	}
}

// Start starts the background job processor
func (s *EvaluatorService) Start() {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if s.isRunning {
		return
	}

	s.isRunning = true
	s.stopChan = make(chan struct{})

	// Start worker goroutines
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// Start job poller
	s.wg.Add(1)
	go s.pollPendingJobs()

	s.logger.Info("evaluator service started",
		zap.Int("worker_count", s.workerCount),
	)
}

// Stop stops the background job processor
func (s *EvaluatorService) Stop() {
	s.runningMutex.Lock()
	if !s.isRunning {
		s.runningMutex.Unlock()
		return
	}
	s.isRunning = false
	s.runningMutex.Unlock()

	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info("evaluator service stopped")
}

// worker processes jobs from the queue
func (s *EvaluatorService) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		case jobID := <-s.jobQueue:
			s.processJob(context.Background(), jobID)
		}
	}
}

// pollPendingJobs periodically checks for pending jobs
func (s *EvaluatorService) pollPendingJobs() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			jobs, err := s.jobRepo.GetPendingJobs(context.Background(), 10)
			if err != nil {
				s.logger.Error("failed to get pending jobs", zap.Error(err))
				continue
			}

			for _, job := range jobs {
				select {
				case s.jobQueue <- job.ID:
				default:
					// Queue is full, skip for now
				}
			}
		}
	}
}

// CreateEvaluator creates a new evaluator configuration
func (s *EvaluatorService) CreateEvaluator(ctx context.Context, create *domain.EvaluatorCreate) (*domain.Evaluator, error) {
	evaluator := &domain.Evaluator{
		ID:          uuid.New(),
		ProjectID:   create.ProjectID,
		Name:        create.Name,
		Description: create.Description,
		Type:        create.Type,
		Provider:    create.Provider,
		Model:       create.Model,
		Template:    create.Template,
		OutputType:  create.OutputType,
		Categories:  create.Categories,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if create.Enabled != nil {
		evaluator.Enabled = *create.Enabled
	}

	evaluator.MinValue = create.MinValue
	evaluator.MaxValue = create.MaxValue

	if create.Config != nil {
		configJSON, err := json.Marshal(create.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		evaluator.Config = configJSON
	} else {
		evaluator.Config = []byte("{}")
	}

	if err := s.evaluatorRepo.Create(ctx, evaluator); err != nil {
		return nil, err
	}

	return evaluator, nil
}

// GetEvaluator retrieves an evaluator by ID
func (s *EvaluatorService) GetEvaluator(ctx context.Context, id uuid.UUID) (*domain.Evaluator, error) {
	return s.evaluatorRepo.GetByID(ctx, id)
}

// UpdateEvaluator updates an evaluator
func (s *EvaluatorService) UpdateEvaluator(ctx context.Context, id uuid.UUID, update *domain.EvaluatorUpdate) (*domain.Evaluator, error) {
	evaluator, err := s.evaluatorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if update.Name != nil {
		evaluator.Name = *update.Name
	}
	if update.Description != nil {
		evaluator.Description = *update.Description
	}
	if update.Provider != nil {
		evaluator.Provider = *update.Provider
	}
	if update.Model != nil {
		evaluator.Model = *update.Model
	}
	if update.Template != nil {
		evaluator.Template = *update.Template
	}
	if update.OutputType != nil {
		evaluator.OutputType = *update.OutputType
	}
	if update.MinValue != nil {
		evaluator.MinValue = update.MinValue
	}
	if update.MaxValue != nil {
		evaluator.MaxValue = update.MaxValue
	}
	if update.Categories != nil {
		evaluator.Categories = *update.Categories
	}
	if update.Enabled != nil {
		evaluator.Enabled = *update.Enabled
	}
	if update.Config != nil {
		configJSON, err := json.Marshal(update.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		evaluator.Config = configJSON
	}

	evaluator.UpdatedAt = time.Now()

	if err := s.evaluatorRepo.Update(ctx, evaluator); err != nil {
		return nil, err
	}

	return evaluator, nil
}

// DeleteEvaluator deletes an evaluator
func (s *EvaluatorService) DeleteEvaluator(ctx context.Context, id uuid.UUID) error {
	return s.evaluatorRepo.Delete(ctx, id)
}

// ListEvaluators lists evaluators based on filter criteria
func (s *EvaluatorService) ListEvaluators(ctx context.Context, filter *domain.EvaluatorFilter) ([]*domain.Evaluator, int, error) {
	return s.evaluatorRepo.List(ctx, filter)
}

// GetTemplates returns all built-in evaluation templates
func (s *EvaluatorService) GetTemplates() []domain.EvaluatorTemplate {
	return domain.GetBuiltInEvaluatorTemplates()
}

// GetTemplateByID returns a specific template
func (s *EvaluatorService) GetTemplateByID(id string) *domain.EvaluatorTemplate {
	return domain.GetEvaluatorTemplateByID(id)
}

// RunEvaluation runs a single evaluation synchronously
func (s *EvaluatorService) RunEvaluation(ctx context.Context, req *domain.RunEvaluationRequest) (*domain.EvaluationResult, error) {
	evaluator, err := s.evaluatorRepo.GetByID(ctx, req.EvaluatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get evaluator: %w", err)
	}

	if !evaluator.Enabled {
		return nil, fmt.Errorf("evaluator is disabled")
	}

	// Get the trace
	trace, err := s.traceRepo.GetByID(ctx, req.TraceID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}

	return s.executeEvaluation(ctx, evaluator, trace, req.SpanID, nil)
}

// CreateJob creates an async evaluation job
func (s *EvaluatorService) CreateJob(ctx context.Context, create *domain.EvaluationJobCreate) (*domain.EvaluationJob, error) {
	// Verify evaluator exists and is enabled
	evaluator, err := s.evaluatorRepo.GetByID(ctx, create.EvaluatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get evaluator: %w", err)
	}

	if !evaluator.Enabled {
		return nil, fmt.Errorf("evaluator is disabled")
	}

	job := &domain.EvaluationJob{
		ID:          uuid.New(),
		ProjectID:   create.ProjectID,
		EvaluatorID: create.EvaluatorID,
		Status:      domain.EvaluationJobStatusPending,
		TargetType:  create.TargetType,
		TargetIDs:   create.TargetIDs,
		TotalItems:  len(create.TargetIDs),
		Completed:   0,
		Failed:      0,
		TotalCost:   0,
		TotalTokens: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, err
	}

	// Queue the job
	select {
	case s.jobQueue <- job.ID:
	default:
		s.logger.Warn("job queue full, job will be processed by poller",
			zap.String("job_id", job.ID.String()),
		)
	}

	return job, nil
}

// GetJob retrieves an evaluation job
func (s *EvaluatorService) GetJob(ctx context.Context, id uuid.UUID) (*domain.EvaluationJob, error) {
	return s.jobRepo.GetByID(ctx, id)
}

// ListJobs lists evaluation jobs
func (s *EvaluatorService) ListJobs(ctx context.Context, filter *domain.EvaluationJobFilter) ([]*domain.EvaluationJob, int, error) {
	return s.jobRepo.List(ctx, filter)
}

// GetResults retrieves evaluation results
func (s *EvaluatorService) GetResults(ctx context.Context, filter *domain.EvaluationResultFilter) ([]*domain.EvaluationResult, int, error) {
	return s.resultRepo.List(ctx, filter)
}

// GetResultsByTrace retrieves all evaluation results for a trace
func (s *EvaluatorService) GetResultsByTrace(ctx context.Context, projectID, traceID uuid.UUID) ([]*domain.EvaluationResult, error) {
	return s.resultRepo.GetByTrace(ctx, projectID, traceID)
}

// GetStats retrieves evaluation statistics
func (s *EvaluatorService) GetStats(ctx context.Context, projectID uuid.UUID, evaluatorID *uuid.UUID, startDate, endDate time.Time) (*domain.EvaluationStats, error) {
	return s.resultRepo.GetStats(ctx, projectID, evaluatorID, startDate, endDate)
}

// GetCostSummary retrieves cost summary by evaluator
func (s *EvaluatorService) GetCostSummary(ctx context.Context, projectID uuid.UUID, startDate, endDate time.Time) ([]*domain.EvaluationCostSummary, error) {
	summaries, err := s.resultRepo.GetCostSummary(ctx, projectID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Enrich with evaluator names
	for _, summary := range summaries {
		evaluator, err := s.evaluatorRepo.GetByID(ctx, summary.EvaluatorID)
		if err == nil {
			summary.EvaluatorName = evaluator.Name
		}
	}

	return summaries, nil
}

// processJob processes an evaluation job
func (s *EvaluatorService) processJob(ctx context.Context, jobID uuid.UUID) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		s.logger.Error("failed to get job", zap.Error(err), zap.String("job_id", jobID.String()))
		return
	}

	// Check if job is still pending
	if job.Status != domain.EvaluationJobStatusPending {
		return
	}

	// Mark job as running
	job.Status = domain.EvaluationJobStatusRunning
	job.StartedAt = sql.NullTime{Time: time.Now(), Valid: true}
	if err := s.jobRepo.Update(ctx, job); err != nil {
		s.logger.Error("failed to update job status", zap.Error(err))
		return
	}

	// Get evaluator
	evaluator, err := s.evaluatorRepo.GetByID(ctx, job.EvaluatorID)
	if err != nil {
		s.failJob(ctx, job, fmt.Sprintf("failed to get evaluator: %v", err))
		return
	}

	// Process each target
	for _, targetID := range job.TargetIDs {
		// Get trace
		trace, err := s.traceRepo.GetByID(ctx, targetID.String())
		if err != nil {
			s.logger.Error("failed to get trace",
				zap.Error(err),
				zap.String("trace_id", targetID.String()),
			)
			job.Failed++
			continue
		}

		// Execute evaluation
		result, err := s.executeEvaluation(ctx, evaluator, trace, nil, &job.ID)
		if err != nil {
			s.logger.Error("evaluation failed",
				zap.Error(err),
				zap.String("trace_id", targetID.String()),
			)
			job.Failed++
		} else {
			job.Completed++
			job.TotalCost += result.Cost.InexactFloat64()
			job.TotalTokens += result.PromptTokens + result.CompletionTokens
		}

		// Update job progress
		if err := s.jobRepo.Update(ctx, job); err != nil {
			s.logger.Error("failed to update job progress", zap.Error(err))
		}
	}

	// Mark job as completed
	job.Status = domain.EvaluationJobStatusCompleted
	job.CompletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	if err := s.jobRepo.Update(ctx, job); err != nil {
		s.logger.Error("failed to update job status", zap.Error(err))
	}

	s.logger.Info("job completed",
		zap.String("job_id", job.ID.String()),
		zap.Int("completed", job.Completed),
		zap.Int("failed", job.Failed),
		zap.Float64("total_cost", job.TotalCost),
	)
}

func (s *EvaluatorService) failJob(ctx context.Context, job *domain.EvaluationJob, errorMsg string) {
	job.Status = domain.EvaluationJobStatusFailed
	job.ErrorMessage = &errorMsg
	job.CompletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	if err := s.jobRepo.Update(ctx, job); err != nil {
		s.logger.Error("failed to update job status", zap.Error(err))
	}
}

// executeEvaluation executes a single evaluation
func (s *EvaluatorService) executeEvaluation(
	ctx context.Context,
	evaluator *domain.Evaluator,
	trace *domain.Trace,
	spanID *uuid.UUID,
	jobID *uuid.UUID,
) (*domain.EvaluationResult, error) {
	startTime := time.Now()

	// Parse config
	var config domain.EvaluatorConfig
	if len(evaluator.Config) > 0 {
		if err := json.Unmarshal(evaluator.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to parse evaluator config: %w", err)
		}
	}

	// Build prompt from template
	prompt, err := s.buildPrompt(evaluator.Template, trace, spanID, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Execute LLM call
	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 500
	}

	temperature := config.Temperature
	if temperature == 0 {
		temperature = 0.0 // Use 0 for deterministic evaluations
	}

	llmReq := domain.LLMRequest{
		Provider:    evaluator.Provider,
		Model:       evaluator.Model,
		Prompt:      prompt,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	llmResp, err := s.llmService.ExecutePrompt(ctx, llmReq)
	if err != nil {
		// Create error result
		errMsg := err.Error()
		errorResult := &domain.EvaluationResult{
			ID:           uuid.New(),
			JobID:        jobID,
			EvaluatorID:  evaluator.ID,
			ProjectID:    evaluator.ProjectID,
			TraceID:      trace.ID,
			SpanID:       spanID,
			Status:       "error",
			ErrorMessage: &errMsg,
			LatencyMs:    int(time.Since(startTime).Milliseconds()),
			CreatedAt:    time.Now(),
		}
		if insertErr := s.resultRepo.Insert(ctx, errorResult); insertErr != nil {
			s.logger.Error("failed to insert error result", zap.Error(insertErr))
		}
		return nil, err
	}

	// Parse score from response
	score, stringValue, reasoning, err := s.parseEvaluationResponse(llmResp.Text, evaluator, &config)
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse response: %v", err)
		errorResult := &domain.EvaluationResult{
			ID:               uuid.New(),
			JobID:            jobID,
			EvaluatorID:      evaluator.ID,
			ProjectID:        evaluator.ProjectID,
			TraceID:          trace.ID,
			SpanID:           spanID,
			RawResponse:      llmResp.Text,
			PromptTokens:     llmResp.Usage.PromptTokens,
			CompletionTokens: llmResp.Usage.CompletionTokens,
			Status:           "error",
			ErrorMessage:     &errMsg,
			LatencyMs:        int(time.Since(startTime).Milliseconds()),
			CreatedAt:        time.Now(),
		}
		if insertErr := s.resultRepo.Insert(ctx, errorResult); insertErr != nil {
			s.logger.Error("failed to insert error result", zap.Error(insertErr))
		}
		return nil, err
	}

	// Calculate cost
	cost, _ := s.pricing.EstimateCost(evaluator.Provider, evaluator.Model,
		llmResp.Usage.PromptTokens, llmResp.Usage.CompletionTokens)

	// Create result
	result := &domain.EvaluationResult{
		ID:               uuid.New(),
		JobID:            jobID,
		EvaluatorID:      evaluator.ID,
		ProjectID:        evaluator.ProjectID,
		TraceID:          trace.ID,
		SpanID:           spanID,
		Score:            score,
		StringValue:      stringValue,
		Reasoning:        reasoning,
		RawResponse:      llmResp.Text,
		PromptTokens:     llmResp.Usage.PromptTokens,
		CompletionTokens: llmResp.Usage.CompletionTokens,
		Cost:             decimal.NewFromFloat(cost),
		LatencyMs:        int(time.Since(startTime).Milliseconds()),
		Status:           "success",
		CreatedAt:        time.Now(),
	}

	// Save result
	if err := s.resultRepo.Insert(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to save result: %w", err)
	}

	// Also create a score record for easy querying
	scoreRecord := &domain.Score{
		ID:        uuid.New(),
		ProjectID: evaluator.ProjectID,
		TraceID:   trace.ID,
		SpanID:    spanID,
		Name:      evaluator.Name,
		Value:     score,
		DataType:  evaluator.OutputType,
		Source:    "llm_judge",
		Comment:   reasoning,
		CreatedAt: time.Now(),
	}
	if stringValue != nil {
		scoreRecord.StringValue = stringValue
	}

	// We don't fail if score creation fails, just log it
	// The evaluation result is the primary record
	s.logger.Debug("evaluation completed",
		zap.String("evaluator_id", evaluator.ID.String()),
		zap.String("trace_id", trace.ID.String()),
		zap.Float64("score", score),
		zap.Float64("cost", cost),
	)

	return result, nil
}

// buildPrompt builds the evaluation prompt from template
func (s *EvaluatorService) buildPrompt(template string, trace *domain.Trace, spanID *uuid.UUID, config *domain.EvaluatorConfig) (string, error) {
	prompt := template

	// Default variable mappings
	variables := map[string]string{
		"input":  trace.Input,
		"output": trace.Output,
	}

	// Parse metadata for additional context
	if trace.Metadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(trace.Metadata), &metadata); err == nil {
			if ctx, ok := metadata["context"].(string); ok {
				variables["context"] = ctx
			}
			if expected, ok := metadata["expected_answer"].(string); ok {
				variables["expected_answer"] = expected
			}
		}
	}

	// Apply custom variable mappings from config
	if config.Variables != nil {
		for key, path := range config.Variables {
			value := s.extractValue(trace, path)
			if value != "" {
				variables[key] = value
			}
		}
	}

	// Replace variables in template
	for key, value := range variables {
		// Handle simple variables: {{variable}}
		prompt = strings.ReplaceAll(prompt, "{{"+key+"}}", value)

		// Handle conditional blocks: {{#if variable}}...{{/if}}
		ifPattern := regexp.MustCompile(`\{\{#if\s+` + key + `\}\}([\s\S]*?)\{\{/if\}\}`)
		if value != "" {
			prompt = ifPattern.ReplaceAllString(prompt, "$1")
		} else {
			prompt = ifPattern.ReplaceAllString(prompt, "")
		}
	}

	// Remove any remaining unset conditional blocks
	remainingIfPattern := regexp.MustCompile(`\{\{#if\s+\w+\}\}[\s\S]*?\{\{/if\}\}`)
	prompt = remainingIfPattern.ReplaceAllString(prompt, "")

	return strings.TrimSpace(prompt), nil
}

// extractValue extracts a value from trace based on a path
func (s *EvaluatorService) extractValue(trace *domain.Trace, path string) string {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return ""
	}

	switch parts[0] {
	case "trace":
		switch parts[1] {
		case "input":
			return trace.Input
		case "output":
			return trace.Output
		case "name":
			return trace.Name
		case "model":
			return trace.Model
		case "metadata":
			if len(parts) > 2 {
				// Extract from metadata JSON
				return gjson.Get(trace.Metadata, strings.Join(parts[2:], ".")).String()
			}
			return trace.Metadata
		}
	}
	return ""
}

// parseEvaluationResponse parses the LLM response to extract score
func (s *EvaluatorService) parseEvaluationResponse(
	response string,
	evaluator *domain.Evaluator,
	config *domain.EvaluatorConfig,
) (float64, *string, *string, error) {
	var score float64
	var stringValue *string
	var reasoning *string

	// Default to JSON parsing
	outputFormat := config.OutputFormat
	if outputFormat == "" {
		outputFormat = "json"
	}

	switch outputFormat {
	case "json":
		// Try to extract JSON from response
		jsonStr := extractJSON(response)
		if jsonStr == "" {
			return 0, nil, nil, fmt.Errorf("no JSON found in response")
		}

		// Extract score using gjson path or default
		scorePath := config.ScoreExtractor
		if scorePath == "" {
			scorePath = "score"
		}
		// Remove leading $. if present
		scorePath = strings.TrimPrefix(scorePath, "$.")

		scoreResult := gjson.Get(jsonStr, scorePath)
		if !scoreResult.Exists() {
			return 0, nil, nil, fmt.Errorf("score not found at path: %s", scorePath)
		}

		switch evaluator.OutputType {
		case "numeric":
			score = scoreResult.Float()
		case "boolean":
			if scoreResult.Bool() {
				score = 1.0
			} else {
				score = 0.0
			}
			sv := scoreResult.String()
			stringValue = &sv
		case "categorical":
			sv := scoreResult.String()
			stringValue = &sv
			// For categorical, we need to map to numeric if needed
			score = 1.0 // Default score for categorical
		}

		// Extract reasoning if present
		reasoningResult := gjson.Get(jsonStr, "reasoning")
		if reasoningResult.Exists() {
			r := reasoningResult.String()
			reasoning = &r
		}

	case "text":
		// Extract score from text using regex
		if config.ScoreExtractor != "" {
			re := regexp.MustCompile(config.ScoreExtractor)
			matches := re.FindStringSubmatch(response)
			if len(matches) > 1 {
				fmt.Sscanf(matches[1], "%f", &score)
			}
		}
		reasoning = &response

	default:
		return 0, nil, nil, fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	// Validate score is within range
	if evaluator.MinValue != nil && score < *evaluator.MinValue {
		score = *evaluator.MinValue
	}
	if evaluator.MaxValue != nil && score > *evaluator.MaxValue {
		score = *evaluator.MaxValue
	}

	return score, stringValue, reasoning, nil
}

// extractJSON tries to extract JSON from a response that might have markdown or other text
func extractJSON(text string) string {
	// First try to parse the whole text as JSON
	text = strings.TrimSpace(text)
	if gjson.Valid(text) {
		return text
	}

	// Try to find JSON in markdown code blocks
	codeBlockPattern := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")
	matches := codeBlockPattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		jsonStr := strings.TrimSpace(matches[1])
		if gjson.Valid(jsonStr) {
			return jsonStr
		}
	}

	// Try to find JSON object in text
	braceStart := strings.Index(text, "{")
	braceEnd := strings.LastIndex(text, "}")
	if braceStart != -1 && braceEnd > braceStart {
		jsonStr := text[braceStart : braceEnd+1]
		if gjson.Valid(jsonStr) {
			return jsonStr
		}
	}

	return ""
}

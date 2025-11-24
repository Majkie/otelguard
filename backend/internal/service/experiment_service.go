package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// ExperimentService handles experiment execution and management
type ExperimentService struct {
	experimentRepo *postgres.ExperimentRepository
	datasetRepo    *postgres.DatasetRepository
	promptRepo     *postgres.PromptRepository
	llmService     LLMService
	evaluatorSvc   *EvaluatorService
	logger         *zap.Logger

	// Execution queue
	executionQueue chan uuid.UUID
	stopChan       chan struct{}
	wg             sync.WaitGroup
	isRunning      bool
	runningMutex   sync.RWMutex
	workerCount    int
}

// NewExperimentService creates a new experiment service
func NewExperimentService(
	experimentRepo *postgres.ExperimentRepository,
	datasetRepo *postgres.DatasetRepository,
	promptRepo *postgres.PromptRepository,
	llmService LLMService,
	evaluatorSvc *EvaluatorService,
	logger *zap.Logger,
) *ExperimentService {
	return &ExperimentService{
		experimentRepo: experimentRepo,
		datasetRepo:    datasetRepo,
		promptRepo:     promptRepo,
		llmService:     llmService,
		evaluatorSvc:   evaluatorSvc,
		logger:         logger,
		executionQueue: make(chan uuid.UUID, 100),
		stopChan:       make(chan struct{}),
		workerCount:    2, // Number of concurrent experiment workers
	}
}

// Start starts the background experiment executor
func (s *ExperimentService) Start() {
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

	s.logger.Info("experiment service started",
		zap.Int("worker_count", s.workerCount),
	)
}

// Stop stops the background experiment executor
func (s *ExperimentService) Stop() {
	s.runningMutex.Lock()
	if !s.isRunning {
		s.runningMutex.Unlock()
		return
	}
	s.isRunning = false
	s.runningMutex.Unlock()

	close(s.stopChan)
	s.wg.Wait()

	s.logger.Info("experiment service stopped")
}

// worker processes experiment executions from the queue
func (s *ExperimentService) worker(id int) {
	defer s.wg.Done()

	s.logger.Info("experiment worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("experiment worker stopped", zap.Int("worker_id", id))
			return

		case runID := <-s.executionQueue:
			s.logger.Info("worker processing experiment run",
				zap.Int("worker_id", id),
				zap.String("run_id", runID.String()),
			)

			if err := s.executeRun(context.Background(), runID); err != nil {
				s.logger.Error("failed to execute experiment run",
					zap.Int("worker_id", id),
					zap.String("run_id", runID.String()),
					zap.Error(err),
				)
			}
		}
	}
}

// Create creates a new experiment
func (s *ExperimentService) Create(ctx context.Context, input *domain.ExperimentCreate) (*domain.Experiment, error) {
	// Verify dataset exists
	_, err := s.datasetRepo.GetByID(ctx, input.DatasetID.String())
	if err != nil {
		return nil, fmt.Errorf("dataset not found: %w", err)
	}

	// Marshal config to JSON
	configJSON, err := json.Marshal(input.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	experiment := &domain.Experiment{
		ID:          uuid.New(),
		ProjectID:   input.ProjectID,
		DatasetID:   input.DatasetID,
		Name:        input.Name,
		Description: input.Description,
		Config:      configJSON,
		Status:      "pending",
		CreatedBy:   input.CreatedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.experimentRepo.Create(ctx, experiment); err != nil {
		s.logger.Error("failed to create experiment", zap.Error(err))
		return nil, fmt.Errorf("failed to create experiment: %w", err)
	}

	return experiment, nil
}

// GetByID retrieves an experiment by ID
func (s *ExperimentService) GetByID(ctx context.Context, id string) (*domain.Experiment, error) {
	return s.experimentRepo.GetByID(ctx, id)
}

// List returns experiments for a project
func (s *ExperimentService) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.Experiment, int, error) {
	return s.experimentRepo.List(ctx, projectID, &postgres.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

// ListByDataset returns experiments for a specific dataset
func (s *ExperimentService) ListByDataset(ctx context.Context, datasetID string) ([]*domain.Experiment, error) {
	return s.experimentRepo.ListByDataset(ctx, datasetID)
}

// Execute executes an experiment
func (s *ExperimentService) Execute(ctx context.Context, input *domain.ExperimentExecute) (*domain.ExperimentRun, error) {
	// Get experiment
	experiment, err := s.experimentRepo.GetByID(ctx, input.ExperimentID.String())
	if err != nil {
		return nil, err
	}

	// Get dataset items count
	itemCount, err := s.datasetRepo.GetItemCount(ctx, experiment.DatasetID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset item count: %w", err)
	}

	// Get next run number
	runNumber, err := s.experimentRepo.GetNextRunNumber(ctx, experiment.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get next run number: %w", err)
	}

	// Create run
	run := &domain.ExperimentRun{
		ID:           uuid.New(),
		ExperimentID: experiment.ID,
		RunNumber:    runNumber,
		Status:       "pending",
		StartedAt:    time.Now(),
		TotalItems:   itemCount,
		CreatedAt:    time.Now(),
	}

	if err := s.experimentRepo.CreateRun(ctx, run); err != nil {
		s.logger.Error("failed to create experiment run", zap.Error(err))
		return nil, fmt.Errorf("failed to create experiment run: %w", err)
	}

	// Update experiment status
	if err := s.experimentRepo.UpdateStatus(ctx, experiment.ID.String(), "running"); err != nil {
		s.logger.Warn("failed to update experiment status", zap.Error(err))
	}

	// Execute async or sync
	if input.Async {
		// Queue for async execution
		s.executionQueue <- run.ID
		s.logger.Info("experiment run queued for async execution",
			zap.String("experiment_id", experiment.ID.String()),
			zap.String("run_id", run.ID.String()),
		)
	} else {
		// Execute synchronously
		if err := s.executeRun(ctx, run.ID); err != nil {
			s.logger.Error("failed to execute experiment run", zap.Error(err))
			return nil, fmt.Errorf("failed to execute experiment run: %w", err)
		}
	}

	return run, nil
}

// executeRun executes a specific experiment run
func (s *ExperimentService) executeRun(ctx context.Context, runID uuid.UUID) error {
	// Get run
	run, err := s.experimentRepo.GetRunByID(ctx, runID.String())
	if err != nil {
		return err
	}

	// Get experiment
	experiment, err := s.experimentRepo.GetByID(ctx, run.ExperimentID.String())
	if err != nil {
		return err
	}

	// Parse config
	var config domain.ExperimentConfig
	if err := json.Unmarshal(experiment.Config, &config); err != nil {
		return fmt.Errorf("failed to parse experiment config: %w", err)
	}

	// Update run status to running
	run.Status = "running"
	if err := s.experimentRepo.UpdateRun(ctx, run); err != nil {
		s.logger.Warn("failed to update run status", zap.Error(err))
	}

	s.logger.Info("starting experiment run execution",
		zap.String("experiment_id", experiment.ID.String()),
		zap.String("run_id", run.ID.String()),
		zap.Int("total_items", run.TotalItems),
	)

	// Get dataset items
	items, _, err := s.datasetRepo.ListItems(ctx, experiment.DatasetID.String(), &postgres.ListOptions{
		Limit:  10000, // Large limit to get all items
		Offset: 0,
	})
	if err != nil {
		return s.failRun(ctx, run, fmt.Sprintf("failed to get dataset items: %v", err))
	}

	// Get prompt if specified
	var promptContent string
	if config.PromptID != nil {
		var promptVersion *domain.PromptVersion
		if config.PromptVersion != nil {
			promptVersion, err = s.promptRepo.GetVersion(ctx, config.PromptID.String(), *config.PromptVersion)
		} else {
			latestVersion, err := s.promptRepo.GetLatestVersion(ctx, config.PromptID.String())
			if err == nil && latestVersion > 0 {
				promptVersion, err = s.promptRepo.GetVersion(ctx, config.PromptID.String(), latestVersion)
			}
		}

		if promptVersion != nil {
			promptContent = promptVersion.Content
		}
	}

	// Execute each dataset item
	for _, item := range items {
		if err := s.executeDatasetItem(ctx, run, item, promptContent, &config); err != nil {
			s.logger.Error("failed to execute dataset item",
				zap.String("item_id", item.ID.String()),
				zap.Error(err),
			)
			// Continue with other items even if one fails
		}
	}

	// Mark run as completed
	run.Status = "completed"
	run.CompletedAt = sql.NullTime{Time: time.Now(), Valid: true}

	if err := s.experimentRepo.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}

	// Update experiment status
	if err := s.experimentRepo.UpdateStatus(ctx, experiment.ID.String(), "completed"); err != nil {
		s.logger.Warn("failed to update experiment status", zap.Error(err))
	}

	s.logger.Info("experiment run completed",
		zap.String("experiment_id", experiment.ID.String()),
		zap.String("run_id", run.ID.String()),
		zap.Int("completed_items", run.CompletedItems),
		zap.Int("failed_items", run.FailedItems),
	)

	return nil
}

// executeDatasetItem executes a single dataset item in an experiment
func (s *ExperimentService) executeDatasetItem(
	ctx context.Context,
	run *domain.ExperimentRun,
	item *domain.DatasetItem,
	promptTemplate string,
	config *domain.ExperimentConfig,
) error {
	startTime := time.Now()

	// Parse input
	var input map[string]interface{}
	if err := json.Unmarshal(item.Input, &input); err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}

	// Build prompt
	prompt := promptTemplate
	if prompt == "" {
		// If no prompt template, use input as prompt
		if inputText, ok := input["prompt"].(string); ok {
			prompt = inputText
		} else {
			inputJSON, _ := json.Marshal(input)
			prompt = string(inputJSON)
		}
	}

	// Execute LLM request
	llmReq := domain.LLMRequest{
		Provider:    config.Provider,
		Model:       config.Model,
		Prompt:      prompt,
		Parameters:  config.Parameters,
	}

	if config.Parameters != nil {
		if maxTokens, ok := config.Parameters["maxTokens"].(float64); ok {
			llmReq.MaxTokens = int(maxTokens)
		}
		if temp, ok := config.Parameters["temperature"].(float64); ok {
			llmReq.Temperature = temp
		}
	}

	resp, err := s.llmService.ExecutePrompt(ctx, llmReq)

	latencyMs := time.Since(startTime).Milliseconds()

	result := &domain.ExperimentResult{
		ID:            uuid.New(),
		RunID:         run.ID,
		DatasetItemID: item.ID,
		LatencyMs:     latencyMs,
		CreatedAt:     time.Now(),
	}

	if err != nil {
		// Record failure
		result.Status = "error"
		result.Error = err.Error()

		if err := s.experimentRepo.CreateResult(ctx, result); err != nil {
			return fmt.Errorf("failed to create result: %w", err)
		}

		if err := s.experimentRepo.IncrementRunProgress(ctx, run.ID.String(), false, 0, 0); err != nil {
			s.logger.Warn("failed to increment run progress", zap.Error(err))
		}

		return nil
	}

	// Record success
	result.Status = "success"
	outputJSON, _ := json.Marshal(map[string]interface{}{
		"text": resp.Text,
	})
	result.Output = outputJSON
	result.TokensUsed = resp.Usage.TotalTokens

	// Calculate cost
	cost, _ := s.llmService.EstimateCost(llmReq, resp.Usage.CompletionTokens)
	result.Cost = cost

	// TODO: Run evaluators if configured
	scores := make(map[string]interface{})
	scoresJSON, _ := json.Marshal(scores)
	result.Scores = scoresJSON

	if err := s.experimentRepo.CreateResult(ctx, result); err != nil {
		return fmt.Errorf("failed to create result: %w", err)
	}

	if err := s.experimentRepo.IncrementRunProgress(ctx, run.ID.String(), true, cost, latencyMs); err != nil {
		s.logger.Warn("failed to increment run progress", zap.Error(err))
	}

	return nil
}

// failRun marks a run as failed
func (s *ExperimentService) failRun(ctx context.Context, run *domain.ExperimentRun, errorMsg string) error {
	run.Status = "failed"
	run.Error = errorMsg
	run.CompletedAt = sql.NullTime{Time: time.Now(), Valid: true}

	if err := s.experimentRepo.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}

	return fmt.Errorf(errorMsg)
}

// GetRun retrieves an experiment run by ID
func (s *ExperimentService) GetRun(ctx context.Context, runID string) (*domain.ExperimentRun, error) {
	return s.experimentRepo.GetRunByID(ctx, runID)
}

// ListRuns returns runs for an experiment
func (s *ExperimentService) ListRuns(ctx context.Context, experimentID string) ([]*domain.ExperimentRun, error) {
	return s.experimentRepo.ListRuns(ctx, experimentID)
}

// GetResults returns results for an experiment run
func (s *ExperimentService) GetResults(ctx context.Context, runID string) ([]*domain.ExperimentResult, error) {
	return s.experimentRepo.GetResultsByRunID(ctx, runID)
}

// CompareRuns compares multiple experiment runs
func (s *ExperimentService) CompareRuns(ctx context.Context, runIDs []uuid.UUID) (*domain.ExperimentComparison, error) {
	if len(runIDs) == 0 {
		return nil, fmt.Errorf("no run IDs provided")
	}

	comparison := &domain.ExperimentComparison{
		RunIDs:  runIDs,
		Runs:    make([]*domain.ExperimentRun, 0, len(runIDs)),
		Metrics: make(map[string]*domain.ComparisonMetrics),
	}

	// Collect runs and results
	allResults := make([][]*domain.ExperimentResult, 0, len(runIDs))

	for _, runID := range runIDs {
		run, err := s.experimentRepo.GetRunByID(ctx, runID.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get run %s: %w", runID, err)
		}
		comparison.Runs = append(comparison.Runs, run)

		results, err := s.experimentRepo.GetResultsByRunID(ctx, runID.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get results for run %s: %w", runID, err)
		}
		allResults = append(allResults, results)
	}

	// Calculate comparison metrics
	comparison.Metrics["latency"] = s.calculateMetrics(allResults, func(r *domain.ExperimentResult) float64 {
		return float64(r.LatencyMs)
	})

	comparison.Metrics["cost"] = s.calculateMetrics(allResults, func(r *domain.ExperimentResult) float64 {
		return r.Cost
	})

	comparison.Metrics["tokens"] = s.calculateMetrics(allResults, func(r *domain.ExperimentResult) float64 {
		return float64(r.TokensUsed)
	})

	return comparison, nil
}

// calculateMetrics calculates statistical metrics for a given metric extractor
func (s *ExperimentService) calculateMetrics(
	allResults [][]*domain.ExperimentResult,
	extractor func(*domain.ExperimentResult) float64,
) *domain.ComparisonMetrics {
	metrics := &domain.ComparisonMetrics{}

	// Collect all values
	values := make([]float64, 0)
	for _, results := range allResults {
		for _, result := range results {
			if result.Status == "success" {
				values = append(values, extractor(result))
			}
		}
	}

	if len(values) == 0 {
		return metrics
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	metrics.Mean = sum / float64(len(values))

	// Calculate min/max
	metrics.Min = values[0]
	metrics.Max = values[0]
	for _, v := range values {
		if v < metrics.Min {
			metrics.Min = v
		}
		if v > metrics.Max {
			metrics.Max = v
		}
	}

	// Calculate median
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	if len(sorted)%2 == 0 {
		metrics.Median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	} else {
		metrics.Median = sorted[len(sorted)/2]
	}

	// Calculate standard deviation
	variance := 0.0
	for _, v := range values {
		diff := v - metrics.Mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	metrics.StdDev = math.Sqrt(variance)

	return metrics
}

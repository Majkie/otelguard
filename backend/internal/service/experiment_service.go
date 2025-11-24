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

	metrics.N = len(values)

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

// PerformStatisticalComparison performs statistical significance testing between experiment runs
func (s *ExperimentService) PerformStatisticalComparison(ctx context.Context, runIDs []uuid.UUID) (*domain.StatisticalComparison, error) {
	if len(runIDs) < 2 {
		return nil, fmt.Errorf("at least 2 runs required for statistical comparison")
	}

	// First get basic comparison
	baseComparison, err := s.CompareRuns(ctx, runIDs)
	if err != nil {
		return nil, err
	}

	comparison := &domain.StatisticalComparison{
		ExperimentComparison: *baseComparison,
		PairwiseTests:        make(map[string][]*domain.PairwiseComparison),
	}

	// Get results for each run
	runResults := make([][]*domain.ExperimentResult, len(runIDs))
	for i, runID := range runIDs {
		results, err := s.experimentRepo.GetResultsByRunID(ctx, runID.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get results for run %s: %w", runID, err)
		}
		runResults[i] = results
	}

	// Metrics to test
	metrics := map[string]func(*domain.ExperimentResult) float64{
		"latency": func(r *domain.ExperimentResult) float64 { return float64(r.LatencyMs) },
		"cost":    func(r *domain.ExperimentResult) float64 { return r.Cost },
		"tokens":  func(r *domain.ExperimentResult) float64 { return float64(r.TokensUsed) },
	}

	// Perform pairwise t-tests for each metric
	for metricName, extractor := range metrics {
		comparisons := make([]*domain.PairwiseComparison, 0)

		// Compare each pair of runs
		for i := 0; i < len(runIDs); i++ {
			for j := i + 1; j < len(runIDs); j++ {
				comp := s.performTTest(
					comparison.Runs[i],
					comparison.Runs[j],
					runResults[i],
					runResults[j],
					metricName,
					extractor,
				)
				comparisons = append(comparisons, comp)
			}
		}

		comparison.PairwiseTests[metricName] = comparisons
	}

	return comparison, nil
}

// performTTest performs a two-sample t-test between two experiment runs
func (s *ExperimentService) performTTest(
	run1, run2 *domain.ExperimentRun,
	results1, results2 []*domain.ExperimentResult,
	metricName string,
	extractor func(*domain.ExperimentResult) float64,
) *domain.PairwiseComparison {
	// Extract values for each run
	values1 := make([]float64, 0)
	for _, r := range results1 {
		if r.Status == "success" {
			values1 = append(values1, extractor(r))
		}
	}

	values2 := make([]float64, 0)
	for _, r := range results2 {
		if r.Status == "success" {
			values2 = append(values2, extractor(r))
		}
	}

	comparison := &domain.PairwiseComparison{
		Run1ID:     run1.ID,
		Run2ID:     run2.ID,
		Run1Name:   fmt.Sprintf("Run #%d", run1.RunNumber),
		Run2Name:   fmt.Sprintf("Run #%d", run2.RunNumber),
		MetricName: metricName,
	}

	// Need at least 2 samples in each group
	if len(values1) < 2 || len(values2) < 2 {
		comparison.PValue = 1.0
		return comparison
	}

	// Calculate statistics for each group
	mean1, stdDev1 := s.calculateMeanAndStdDev(values1)
	mean2, stdDev2 := s.calculateMeanAndStdDev(values2)

	n1 := float64(len(values1))
	n2 := float64(len(values2))

	comparison.MeanDifference = mean1 - mean2

	// Calculate pooled standard deviation for effect size
	pooledStdDev := math.Sqrt(((n1-1)*stdDev1*stdDev1 + (n2-1)*stdDev2*stdDev2) / (n1 + n2 - 2))
	if pooledStdDev > 0 {
		comparison.EffectSize = comparison.MeanDifference / pooledStdDev
	}

	// Calculate t-statistic using Welch's t-test (unequal variances)
	se1 := stdDev1 * stdDev1 / n1
	se2 := stdDev2 * stdDev2 / n2
	standardError := math.Sqrt(se1 + se2)

	if standardError == 0 {
		comparison.PValue = 1.0
		return comparison
	}

	comparison.TStatistic = comparison.MeanDifference / standardError

	// Calculate degrees of freedom using Welch-Satterthwaite equation
	numerator := (se1 + se2) * (se1 + se2)
	denominator := (se1*se1)/(n1-1) + (se2*se2)/(n2-1)
	if denominator > 0 {
		comparison.DegreesOfFreedom = int(numerator / denominator)
	} else {
		comparison.DegreesOfFreedom = int(n1 + n2 - 2)
	}

	// Calculate two-tailed p-value
	comparison.PValue = s.calculatePValue(comparison.TStatistic, comparison.DegreesOfFreedom)

	// Determine significance levels
	comparison.SignificantAt05 = comparison.PValue < 0.05
	comparison.SignificantAt01 = comparison.PValue < 0.01

	return comparison
}

// calculateMeanAndStdDev calculates mean and standard deviation of a sample
func (s *ExperimentService) calculateMeanAndStdDev(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	// Calculate standard deviation
	if len(values) == 1 {
		return mean, 0
	}

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values) - 1) // Sample standard deviation (n-1)
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

// calculatePValue calculates the two-tailed p-value for a t-statistic
// Using approximation based on the t-distribution
func (s *ExperimentService) calculatePValue(tStat float64, df int) float64 {
	// Use absolute value for two-tailed test
	t := math.Abs(tStat)

	// For large degrees of freedom (df > 30), t-distribution approximates normal distribution
	if df > 30 {
		// Use normal approximation
		return 2.0 * (1.0 - s.normalCDF(t))
	}

	// For small df, use more accurate t-distribution approximation
	// This is a numerical approximation of the incomplete beta function
	x := float64(df) / (float64(df) + t*t)
	pValue := s.incompleteBeta(x, float64(df)/2.0, 0.5)

	return pValue
}

// normalCDF calculates the cumulative distribution function for standard normal distribution
func (s *ExperimentService) normalCDF(x float64) float64 {
	// Using error function approximation
	return 0.5 * (1.0 + math.Erf(x/math.Sqrt(2.0)))
}

// incompleteBeta approximates the regularized incomplete beta function
// This is used for calculating p-values from the t-distribution
func (s *ExperimentService) incompleteBeta(x, a, b float64) float64 {
	// For the t-distribution, we can use a continued fraction approximation
	// This is a simplified implementation suitable for our use case

	if x <= 0.0 {
		return 0.0
	}
	if x >= 1.0 {
		return 1.0
	}

	// Use symmetry property if needed
	if x > (a+1.0)/(a+b+2.0) {
		return 1.0 - s.incompleteBeta(1.0-x, b, a)
	}

	// Continued fraction approximation (Lentz's algorithm)
	const maxIterations = 200
	const epsilon = 1e-10

	// Calculate beta function B(a,b)
	lnBeta := s.logGamma(a) + s.logGamma(b) - s.logGamma(a+b)

	// Initial values
	front := math.Exp(a*math.Log(x) + b*math.Log(1.0-x) - lnBeta) / a

	f := 1.0
	c := 1.0
	d := 0.0

	for i := 0; i <= maxIterations; i++ {
		m := float64(i)

		// Calculate numerator and denominator
		var numerator, denominator float64

		if i == 0 {
			numerator = 1.0
		} else {
			m2 := 2.0 * m
			numerator = m * (b - m) * x / ((a + m2 - 1.0) * (a + m2))
		}

		denominator = 1.0

		// Update d
		d = denominator + numerator*d
		if math.Abs(d) < epsilon {
			d = epsilon
		}
		d = 1.0 / d

		// Update c
		c = denominator + numerator/c
		if math.Abs(c) < epsilon {
			c = epsilon
		}

		// Update f
		cd := c * d
		f *= cd

		// Check convergence
		if math.Abs(cd-1.0) < epsilon {
			return front * f
		}

		// Second term in continued fraction
		m2 := 2.0 * m
		numerator = -(a + m) * (a + b + m) * x / ((a + m2) * (a + m2 + 1.0))
		denominator = 1.0

		d = denominator + numerator*d
		if math.Abs(d) < epsilon {
			d = epsilon
		}
		d = 1.0 / d

		c = denominator + numerator/c
		if math.Abs(c) < epsilon {
			c = epsilon
		}

		cd = c * d
		f *= cd

		if math.Abs(cd-1.0) < epsilon {
			return front * f
		}
	}

	return front * f
}

// logGamma calculates the natural logarithm of the gamma function
// Using Stirling's approximation for large values
func (s *ExperimentService) logGamma(x float64) float64 {
	if x <= 0 {
		return math.Inf(1)
	}

	// Coefficients for Lanczos approximation
	coef := []float64{
		76.18009172947146,
		-86.50532032941677,
		24.01409824083091,
		-1.231739572450155,
		0.1208650973866179e-2,
		-0.5395239384953e-5,
	}

	y := x
	tmp := x + 5.5
	tmp -= (x + 0.5) * math.Log(tmp)
	ser := 1.000000000190015

	for i := 0; i < 6; i++ {
		y++
		ser += coef[i] / y
	}

	return -tmp + math.Log(2.5066282746310005*ser/x)
}

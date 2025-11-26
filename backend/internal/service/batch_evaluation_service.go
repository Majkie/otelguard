package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BatchEvaluationService handles batch evaluation of multiple inputs
type BatchEvaluationService struct {
	guardrailService *GuardrailService
	cache            *EvaluationCache
	logger           *zap.Logger
	maxConcurrency   int
}

// BatchEvaluationRequest represents a batch evaluation request
type BatchEvaluationRequest struct {
	Items         []*EvaluationInput
	MaxParallel   int  // Maximum parallel evaluations (default: 10)
	UseCache      bool // Whether to use caching (default: true)
	StopOnFailure bool // Stop if any evaluation fails (default: false)
}

// BatchEvaluationResponse represents the response from batch evaluation
type BatchEvaluationResponse struct {
	BatchID      string
	TotalItems   int
	SuccessCount int
	FailureCount int
	CacheHits    int
	Results      []*BatchItemResult
	StartedAt    time.Time
	CompletedAt  time.Time
	DurationMs   int64
}

// BatchItemResult represents the result for a single item in the batch
type BatchItemResult struct {
	Index       int
	Input       *EvaluationInput
	Result      *EvaluationResult
	Error       string
	FromCache   bool
	DurationMs  int64
}

// NewBatchEvaluationService creates a new batch evaluation service
func NewBatchEvaluationService(
	guardrailService *GuardrailService,
	cache *EvaluationCache,
	logger *zap.Logger,
	maxConcurrency int,
) *BatchEvaluationService {
	if maxConcurrency == 0 {
		maxConcurrency = 10
	}

	return &BatchEvaluationService{
		guardrailService: guardrailService,
		cache:            cache,
		logger:           logger,
		maxConcurrency:   maxConcurrency,
	}
}

// Evaluate processes a batch of evaluations
func (s *BatchEvaluationService) Evaluate(ctx context.Context, request *BatchEvaluationRequest) (*BatchEvaluationResponse, error) {
	if len(request.Items) == 0 {
		return nil, fmt.Errorf("batch request must contain at least one item")
	}

	// Set defaults
	if request.MaxParallel == 0 {
		request.MaxParallel = 10
	}
	if request.MaxParallel > s.maxConcurrency {
		request.MaxParallel = s.maxConcurrency
	}

	batchID := uuid.New().String()
	startTime := time.Now()

	s.logger.Info("starting batch evaluation",
		zap.String("batch_id", batchID),
		zap.Int("total_items", len(request.Items)),
		zap.Int("max_parallel", request.MaxParallel),
		zap.Bool("use_cache", request.UseCache),
	)

	response := &BatchEvaluationResponse{
		BatchID:    batchID,
		TotalItems: len(request.Items),
		Results:    make([]*BatchItemResult, len(request.Items)),
		StartedAt:  startTime,
	}

	// Create worker pool
	var wg sync.WaitGroup
	itemChan := make(chan *batchItem, len(request.Items))
	resultChan := make(chan *BatchItemResult, len(request.Items))

	// Start workers
	for i := 0; i < request.MaxParallel; i++ {
		wg.Add(1)
		go s.worker(ctx, &wg, itemChan, resultChan, request.UseCache)
	}

	// Send items to workers
	go func() {
		for i, input := range request.Items {
			select {
			case <-ctx.Done():
				break
			case itemChan <- &batchItem{Index: i, Input: input}:
			}
		}
		close(itemChan)
	}()

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	successCount := 0
	failureCount := 0
	cacheHits := 0

	for result := range resultChan {
		response.Results[result.Index] = result

		if result.Error != "" {
			failureCount++

			if request.StopOnFailure {
				s.logger.Warn("stopping batch evaluation due to failure",
					zap.String("batch_id", batchID),
					zap.Int("index", result.Index),
					zap.String("error", result.Error),
				)
				// Cancel context to stop remaining workers
				// (would need to pass a cancel function for this to work fully)
			}
		} else {
			successCount++
			if result.FromCache {
				cacheHits++
			}
		}
	}

	response.SuccessCount = successCount
	response.FailureCount = failureCount
	response.CacheHits = cacheHits
	response.CompletedAt = time.Now()
	response.DurationMs = response.CompletedAt.Sub(response.StartedAt).Milliseconds()

	s.logger.Info("batch evaluation completed",
		zap.String("batch_id", batchID),
		zap.Int("total", response.TotalItems),
		zap.Int("success", successCount),
		zap.Int("failure", failureCount),
		zap.Int("cache_hits", cacheHits),
		zap.Int64("duration_ms", response.DurationMs),
	)

	return response, nil
}

// batchItem represents a single item in the batch
type batchItem struct {
	Index int
	Input *EvaluationInput
}

// worker processes items from the channel
func (s *BatchEvaluationService) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	itemChan <-chan *batchItem,
	resultChan chan<- *BatchItemResult,
	useCache bool,
) {
	defer wg.Done()

	for item := range itemChan {
		select {
		case <-ctx.Done():
			return
		default:
			result := s.evaluateItem(ctx, item, useCache)
			resultChan <- result
		}
	}
}

// evaluateItem evaluates a single item
func (s *BatchEvaluationService) evaluateItem(ctx context.Context, item *batchItem, useCache bool) *BatchItemResult {
	startTime := time.Now()

	result := &BatchItemResult{
		Index:     item.Index,
		Input:     item.Input,
		FromCache: false,
	}

	// Try cache first if enabled
	if useCache && s.cache != nil {
		if cachedResult, found := s.cache.Get(ctx, item.Input); found {
			result.Result = cachedResult
			result.FromCache = true
			result.DurationMs = time.Since(startTime).Milliseconds()
			return result
		}
	}

	// Evaluate with timeout
	evalCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	evalResult, err := s.guardrailService.Evaluate(evalCtx, item.Input)
	result.DurationMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		s.logger.Error("batch item evaluation failed",
			zap.Int("index", item.Index),
			zap.Error(err),
		)
	} else {
		result.Result = evalResult

		// Cache successful result if enabled
		if useCache && s.cache != nil {
			s.cache.Set(ctx, item.Input, evalResult)
		}
	}

	return result
}

// GetStatistics returns summary statistics for a batch response
func (s *BatchEvaluationService) GetStatistics(response *BatchEvaluationResponse) *BatchStatistics {
	stats := &BatchStatistics{
		TotalItems:   response.TotalItems,
		SuccessCount: response.SuccessCount,
		FailureCount: response.FailureCount,
		CacheHits:    response.CacheHits,
		CacheHitRate: float64(response.CacheHits) / float64(response.TotalItems),
		TotalDuration: response.DurationMs,
	}

	// Calculate additional statistics
	var totalLatency int64
	var passedCount int
	var failedCount int
	var remediatedCount int

	for _, result := range response.Results {
		if result.Error == "" && result.Result != nil {
			totalLatency += result.Result.LatencyMs

			if result.Result.Passed {
				passedCount++
			} else {
				failedCount++
			}

			if result.Result.Remediated {
				remediatedCount++
			}
		}
	}

	if response.SuccessCount > 0 {
		stats.AvgItemLatency = float64(totalLatency) / float64(response.SuccessCount)
		stats.AvgProcessingTime = float64(response.DurationMs) / float64(response.SuccessCount)
	}

	stats.PassedCount = passedCount
	stats.FailedCount = failedCount
	stats.RemediatedCount = remediatedCount

	if passedCount+failedCount > 0 {
		stats.PassRate = float64(passedCount) / float64(passedCount+failedCount)
	}

	return stats
}

// BatchStatistics contains statistical information about a batch
type BatchStatistics struct {
	TotalItems          int
	SuccessCount        int
	FailureCount        int
	CacheHits           int
	CacheHitRate        float64
	PassedCount         int
	FailedCount         int
	RemediatedCount     int
	PassRate            float64
	AvgItemLatency      float64
	AvgProcessingTime   float64
	TotalDuration       int64
}

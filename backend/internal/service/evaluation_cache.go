package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EvaluationCache provides caching for guardrail evaluation results
type EvaluationCache struct {
	cache      map[string]*CacheEntry
	mutex      sync.RWMutex
	ttl        time.Duration
	maxSize    int
	logger     *zap.Logger
	cleanupInterval time.Duration
	stopCleanup chan struct{}
}

// CacheEntry represents a cached evaluation result
type CacheEntry struct {
	Key        string
	Result     *EvaluationResult
	ExpiresAt  time.Time
	HitCount   int64
	CreatedAt  time.Time
	LastAccess time.Time
}

// CacheConfig configures the evaluation cache
type CacheConfig struct {
	TTL             time.Duration // Time-to-live for cache entries
	MaxSize         int           // Maximum number of entries
	CleanupInterval time.Duration // Interval for cleanup goroutine
}

// NewEvaluationCache creates a new evaluation cache
func NewEvaluationCache(config CacheConfig, logger *zap.Logger) *EvaluationCache {
	if config.TTL == 0 {
		config.TTL = 5 * time.Minute // Default 5 minutes
	}
	if config.MaxSize == 0 {
		config.MaxSize = 10000 // Default 10k entries
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Minute // Default 1 minute
	}

	cache := &EvaluationCache{
		cache:           make(map[string]*CacheEntry),
		ttl:             config.TTL,
		maxSize:         config.MaxSize,
		logger:          logger,
		cleanupInterval: config.CleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	logger.Info("evaluation cache initialized",
		zap.Duration("ttl", config.TTL),
		zap.Int("max_size", config.MaxSize),
	)

	return cache
}

// Get retrieves a cached evaluation result
func (c *EvaluationCache) Get(ctx context.Context, input *EvaluationInput) (*EvaluationResult, bool) {
	key := c.generateKey(input)

	c.mutex.RLock()
	entry, exists := c.cache[key]
	c.mutex.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.mutex.Lock()
		delete(c.cache, key)
		c.mutex.Unlock()
		return nil, false
	}

	// Update access stats
	c.mutex.Lock()
	entry.HitCount++
	entry.LastAccess = time.Now()
	c.mutex.Unlock()

	c.logger.Debug("cache hit",
		zap.String("key", key[:16]+"..."),
		zap.Int64("hit_count", entry.HitCount),
	)

	return entry.Result, true
}

// Set stores an evaluation result in the cache
func (c *EvaluationCache) Set(ctx context.Context, input *EvaluationInput, result *EvaluationResult) {
	key := c.generateKey(input)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if cache is full
	if len(c.cache) >= c.maxSize {
		// Evict least recently used entry
		c.evictLRU()
	}

	now := time.Now()
	entry := &CacheEntry{
		Key:        key,
		Result:     result,
		ExpiresAt:  now.Add(c.ttl),
		HitCount:   0,
		CreatedAt:  now,
		LastAccess: now,
	}

	c.cache[key] = entry

	c.logger.Debug("cache set",
		zap.String("key", key[:16]+"..."),
		zap.Int("cache_size", len(c.cache)),
	)
}

// generateKey generates a cache key from evaluation input
func (c *EvaluationCache) generateKey(input *EvaluationInput) string {
	// Create a deterministic key from the input
	keyData := struct {
		ProjectID   string
		PolicyID    string
		Input       string
		Output      string
		Model       string
		Environment string
		Tags        []string
	}{
		ProjectID:   input.ProjectID.String(),
		Input:       input.Input,
		Output:      input.Output,
		Model:       input.Model,
		Environment: input.Environment,
		Tags:        input.Tags,
	}

	if input.PolicyID != nil {
		keyData.PolicyID = input.PolicyID.String()
	}

	jsonData, _ := json.Marshal(keyData)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// evictLRU evicts the least recently used entry
func (c *EvaluationCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range c.cache {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
		c.logger.Debug("evicted LRU entry", zap.String("key", oldestKey[:16]+"..."))
	}
}

// cleanupLoop periodically removes expired entries
func (c *EvaluationCache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries
func (c *EvaluationCache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	expiredCount := 0

	for key, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			delete(c.cache, key)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		c.logger.Debug("cleaned up expired entries",
			zap.Int("expired_count", expiredCount),
			zap.Int("remaining", len(c.cache)),
		)
	}
}

// Invalidate removes entries matching the given project and policy
func (c *EvaluationCache) Invalidate(projectID string, policyID *string) int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	invalidatedCount := 0

	// Note: This is a simple implementation that clears all cache
	// In production, you might want to store metadata with entries for targeted invalidation
	if policyID != nil {
		// For now, clear entire cache when policy changes
		invalidatedCount = len(c.cache)
		c.cache = make(map[string]*CacheEntry)
		c.logger.Info("invalidated cache for policy change",
			zap.String("policy_id", *policyID),
			zap.Int("entries_cleared", invalidatedCount),
		)
	}

	return invalidatedCount
}

// InvalidateAll clears the entire cache
func (c *EvaluationCache) InvalidateAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	count := len(c.cache)
	c.cache = make(map[string]*CacheEntry)

	c.logger.Info("invalidated all cache entries", zap.Int("count", count))
}

// GetStats returns cache statistics
func (c *EvaluationCache) GetStats() *CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := &CacheStats{
		Size:    len(c.cache),
		MaxSize: c.maxSize,
		TTL:     c.ttl,
	}

	var totalHits int64
	var oldestEntry time.Time = time.Now()
	var newestEntry time.Time

	for _, entry := range c.cache {
		totalHits += entry.HitCount
		if entry.CreatedAt.Before(oldestEntry) {
			oldestEntry = entry.CreatedAt
		}
		if entry.CreatedAt.After(newestEntry) {
			newestEntry = entry.CreatedAt
		}
	}

	stats.TotalHits = totalHits
	if len(c.cache) > 0 {
		stats.AvgHitCount = float64(totalHits) / float64(len(c.cache))
		stats.OldestEntry = &oldestEntry
		stats.NewestEntry = &newestEntry
	}

	return stats
}

// CacheStats represents cache statistics
type CacheStats struct {
	Size         int
	MaxSize      int
	TTL          time.Duration
	TotalHits    int64
	AvgHitCount  float64
	OldestEntry  *time.Time
	NewestEntry  *time.Time
}

// Shutdown stops the cleanup goroutine
func (c *EvaluationCache) Shutdown() {
	close(c.stopCleanup)
	c.logger.Info("evaluation cache shutdown")
}

// CachedGuardrailService wraps GuardrailService with caching
type CachedGuardrailService struct {
	service *GuardrailService
	cache   *EvaluationCache
	logger  *zap.Logger
}

// NewCachedGuardrailService creates a cached guardrail service
func NewCachedGuardrailService(
	service *GuardrailService,
	cache *EvaluationCache,
	logger *zap.Logger,
) *CachedGuardrailService {
	return &CachedGuardrailService{
		service: service,
		cache:   cache,
		logger:  logger,
	}
}

// Evaluate evaluates with caching
func (s *CachedGuardrailService) Evaluate(ctx context.Context, input *EvaluationInput) (*EvaluationResult, error) {
	// Try to get from cache
	if result, found := s.cache.Get(ctx, input); found {
		s.logger.Debug("returning cached evaluation result")
		return result, nil
	}

	// Cache miss - evaluate
	result, err := s.service.Evaluate(ctx, input)
	if err != nil {
		return nil, err
	}

	// Store in cache (only cache successful evaluations)
	s.cache.Set(ctx, input, result)

	return result, nil
}

// InvalidateCache invalidates cache entries for a project/policy
func (s *CachedGuardrailService) InvalidateCache(projectID string, policyID *string) int {
	return s.cache.Invalidate(projectID, policyID)
}

// GetCacheStats returns cache statistics
func (s *CachedGuardrailService) GetCacheStats() *CacheStats {
	return s.cache.GetStats()
}

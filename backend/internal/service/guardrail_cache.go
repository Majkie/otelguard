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

// CacheEntry represents a cached evaluation result
type CacheEntry struct {
	Result    *EvaluationResult
	CachedAt  time.Time
	ExpiresAt time.Time
}

// GuardrailCache provides caching for guardrail evaluations
type GuardrailCache struct {
	cache    map[string]*CacheEntry
	mu       sync.RWMutex
	ttl      time.Duration
	maxSize  int
	logger   *zap.Logger
	hitCount int64
	missCount int64
}

// NewGuardrailCache creates a new guardrail cache
func NewGuardrailCache(ttl time.Duration, maxSize int, logger *zap.Logger) *GuardrailCache {
	cache := &GuardrailCache{
		cache:   make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
		logger:  logger,
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a cached evaluation result
func (c *GuardrailCache) Get(ctx context.Context, input *EvaluationInput) (*EvaluationResult, bool) {
	key := c.generateKey(input)

	c.mu.RLock()
	entry, exists := c.cache[key]
	c.mu.RUnlock()

	if !exists {
		c.missCount++
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
		c.missCount++
		return nil, false
	}

	c.hitCount++
	c.logger.Debug("cache hit",
		zap.String("key", key),
		zap.Time("cached_at", entry.CachedAt),
	)

	return entry.Result, true
}

// Set stores an evaluation result in cache
func (c *GuardrailCache) Set(ctx context.Context, input *EvaluationInput, result *EvaluationResult) {
	key := c.generateKey(input)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if cache is full
	if len(c.cache) >= c.maxSize {
		// Evict oldest entry
		c.evictOldest()
	}

	now := time.Now()
	c.cache[key] = &CacheEntry{
		Result:    result,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}

	c.logger.Debug("cached evaluation result",
		zap.String("key", key),
		zap.Duration("ttl", c.ttl),
	)
}

// Invalidate removes specific entries from cache
func (c *GuardrailCache) Invalidate(ctx context.Context, projectID string, policyID *string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key := range c.cache {
		// Simple invalidation: remove all if no specific policy, or matching policy
		// In production, you'd want more sophisticated key matching
		delete(c.cache, key)
		count++
	}

	c.logger.Info("invalidated cache entries",
		zap.Int("count", count),
		zap.String("project_id", projectID),
	)

	return count
}

// Clear removes all entries from cache
func (c *GuardrailCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CacheEntry)
	c.hitCount = 0
	c.missCount = 0

	c.logger.Info("cleared cache")
}

// GetStats returns cache statistics
func (c *GuardrailCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hitCount + c.missCount
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hitCount) / float64(total)
	}

	return map[string]interface{}{
		"size":       len(c.cache),
		"max_size":   c.maxSize,
		"hit_count":  c.hitCount,
		"miss_count": c.missCount,
		"hit_rate":   hitRate,
		"ttl_seconds": c.ttl.Seconds(),
	}
}

// generateKey creates a cache key from evaluation input
func (c *GuardrailCache) generateKey(input *EvaluationInput) string {
	// Create a deterministic key from the input
	data := map[string]interface{}{
		"project_id": input.ProjectID.String(),
		"input":      input.Input,
		"output":     input.Output,
		"model":      input.Model,
		"tags":       input.Tags,
	}

	if input.PolicyID != nil {
		data["policy_id"] = input.PolicyID.String()
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// evictOldest removes the oldest cache entry (simple LRU)
func (c *GuardrailCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestKey == "" || entry.CachedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CachedAt
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
		c.logger.Debug("evicted oldest cache entry", zap.String("key", oldestKey))
	}
}

// cleanupExpired runs periodically to remove expired entries
func (c *GuardrailCache) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		removed := 0

		for key, entry := range c.cache {
			if now.After(entry.ExpiresAt) {
				delete(c.cache, key)
				removed++
			}
		}

		c.mu.Unlock()

		if removed > 0 {
			c.logger.Debug("cleaned up expired cache entries", zap.Int("count", removed))
		}
	}
}

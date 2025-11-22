package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // requests per interval
	interval time.Duration // time interval
	cleanup  time.Duration // cleanup interval for stale buckets
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
		cleanup:  interval * 10,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request is allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]

	if !exists {
		rl.buckets[key] = &bucket{
			tokens:    rl.rate - 1,
			lastReset: now,
		}
		return true
	}

	// Reset tokens if interval has passed
	if now.Sub(b.lastReset) >= rl.interval {
		b.tokens = rl.rate - 1
		b.lastReset = now
		return true
	}

	// Check if tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanupLoop removes stale buckets
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			if now.Sub(b.lastReset) > rl.cleanup {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns a gin middleware for rate limiting
func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use IP as the rate limit key by default
		key := c.ClientIP()

		// If authenticated, use user/project ID instead
		if projectID := c.GetString(string(ContextProjectID)); projectID != "" {
			key = "project:" + projectID
		} else if userID := c.GetString(string(ContextUserID)); userID != "" {
			key = "user:" + userID
		}

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "rate_limited",
				Message: "Too many requests. Please try again later.",
			})
			return
		}

		c.Next()
	}
}

// RateLimitByIP returns a gin middleware for IP-based rate limiting
func RateLimitByIP(rate int, interval time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, interval)
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "rate_limited",
				Message: "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}

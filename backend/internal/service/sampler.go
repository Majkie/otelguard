package service

import (
	"context"
	"hash/fnv"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/otelguard/otelguard/internal/domain"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// SamplerType defines the type of sampling strategy
type SamplerType string

const (
	// SamplerTypeAlways always samples all traces
	SamplerTypeAlways SamplerType = "always"
	// SamplerTypeRandom samples traces randomly based on a rate
	SamplerTypeRandom SamplerType = "random"
	// SamplerTypeRateLimit samples up to N traces per second
	SamplerTypeRateLimit SamplerType = "rate_limit"
	// SamplerTypeConsistent samples consistently based on trace ID hash
	SamplerTypeConsistent SamplerType = "consistent"
	// SamplerTypePriority samples based on trace priority (errors always sampled)
	SamplerTypePriority SamplerType = "priority"
)

// SamplerConfig contains configuration for the sampler
type SamplerConfig struct {
	Type          SamplerType `envconfig:"SAMPLER_TYPE" default:"always"`
	Rate          float64     `envconfig:"SAMPLER_RATE" default:"1.0"`        // 0.0 to 1.0 for random/consistent sampling
	MaxPerSecond  int         `envconfig:"SAMPLER_MAX_PER_SEC" default:"100"` // For rate_limit sampling
	SampleErrors  bool        `envconfig:"SAMPLER_ERRORS" default:"true"`     // Always sample errors
	SampleSlow    bool        `envconfig:"SAMPLER_SLOW" default:"true"`       // Always sample slow traces
	SlowThreshold int         `envconfig:"SAMPLER_SLOW_MS" default:"5000"`    // Threshold for slow traces in ms
}

// DefaultSamplerConfig returns default sampling configuration
func DefaultSamplerConfig() *SamplerConfig {
	return &SamplerConfig{
		Type:          SamplerTypeAlways,
		Rate:          1.0,
		MaxPerSecond:  100,
		SampleErrors:  true,
		SampleSlow:    true,
		SlowThreshold: 5000,
	}
}

// Sampler decides whether to sample traces
type Sampler interface {
	ShouldSample(ctx context.Context, trace *domain.Trace) bool
	GetStats() *SamplerStats
}

// SamplerStats contains sampling statistics
type SamplerStats struct {
	TotalReceived int64   `json:"totalReceived"`
	TotalSampled  int64   `json:"totalSampled"`
	TotalDropped  int64   `json:"totalDropped"`
	SampleRate    float64 `json:"sampleRate"`
}

// TraceSampler implements sampling logic
type TraceSampler struct {
	config *SamplerConfig
	logger *zap.Logger
	rng    *rand.Rand
	rngMu  sync.Mutex

	// Stats
	received atomic.Int64
	sampled  atomic.Int64
	dropped  atomic.Int64

	// Rate limiter state
	rateLimiter *rateLimiter
}

// rateLimiter implements token bucket rate limiting
type rateLimiter struct {
	maxPerSecond int
	tokens       int
	lastRefill   time.Time
	mu           sync.Mutex
}

func newRateLimiter(maxPerSecond int) *rateLimiter {
	return &rateLimiter{
		maxPerSecond: maxPerSecond,
		tokens:       maxPerSecond,
		lastRefill:   time.Now(),
	}
}

func (r *rateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	tokensToAdd := int(elapsed.Seconds() * float64(r.maxPerSecond))

	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.maxPerSecond {
			r.tokens = r.maxPerSecond
		}
		r.lastRefill = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

// NewTraceSampler creates a new trace sampler
func NewTraceSampler(config *SamplerConfig, logger *zap.Logger) *TraceSampler {
	if config == nil {
		config = DefaultSamplerConfig()
	}

	s := &TraceSampler{
		config: config,
		logger: logger,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if config.Type == SamplerTypeRateLimit {
		s.rateLimiter = newRateLimiter(config.MaxPerSecond)
	}

	return s
}

// ShouldSample determines whether a trace should be sampled
func (s *TraceSampler) ShouldSample(ctx context.Context, trace *domain.Trace) bool {
	s.received.Add(1)

	// Always sample if configured for errors and this is an error
	if s.config.SampleErrors && trace.Status == "error" {
		s.sampled.Add(1)
		return true
	}

	// Always sample slow traces if configured
	if s.config.SampleSlow && int(trace.LatencyMs) >= s.config.SlowThreshold {
		s.sampled.Add(1)
		return true
	}

	var shouldSample bool

	switch s.config.Type {
	case SamplerTypeAlways:
		shouldSample = true

	case SamplerTypeRandom:
		shouldSample = s.sampleRandom()

	case SamplerTypeRateLimit:
		shouldSample = s.rateLimiter.Allow()

	case SamplerTypeConsistent:
		shouldSample = s.sampleConsistent(trace.ID.String())

	case SamplerTypePriority:
		shouldSample = s.samplePriority(trace)

	default:
		shouldSample = true
	}

	if shouldSample {
		s.sampled.Add(1)
	} else {
		s.dropped.Add(1)
	}

	return shouldSample
}

// sampleRandom performs random sampling based on rate
func (s *TraceSampler) sampleRandom() bool {
	s.rngMu.Lock()
	r := s.rng.Float64()
	s.rngMu.Unlock()
	return r < s.config.Rate
}

// sampleConsistent performs consistent sampling based on trace ID hash
// This ensures the same trace ID always gets the same sampling decision
func (s *TraceSampler) sampleConsistent(traceID string) bool {
	h := fnv.New64a()
	h.Write([]byte(traceID))
	hash := h.Sum64()

	// Normalize hash to 0.0-1.0 range
	normalizedHash := float64(hash) / float64(^uint64(0))
	return normalizedHash < s.config.Rate
}

// samplePriority performs priority-based sampling
// Higher token/cost traces have higher priority
func (s *TraceSampler) samplePriority(trace *domain.Trace) bool {
	// Calculate priority score (0.0 to 1.0)
	priority := 0.0

	// Higher token count increases priority
	if trace.TotalTokens > 1000 {
		priority += 0.3
	} else if trace.TotalTokens > 100 {
		priority += 0.1
	}

	// Higher cost increases priority
	costThreshold01 := decimal.NewFromFloat(0.01)
	costThreshold001 := decimal.NewFromFloat(0.001)

	if trace.Cost.GreaterThan(costThreshold01) {
		priority += 0.3
	} else if trace.Cost.GreaterThan(costThreshold001) {
		priority += 0.1
	}

	// Longer latency increases priority
	if trace.LatencyMs > 2000 {
		priority += 0.2
	} else if trace.LatencyMs > 500 {
		priority += 0.1
	}

	// Traces with metadata have higher priority
	if trace.Metadata != "" && trace.Metadata != "{}" {
		priority += 0.1
	}

	// Combine with base sample rate
	effectiveRate := s.config.Rate + (priority * (1 - s.config.Rate))
	if effectiveRate > 1.0 {
		effectiveRate = 1.0
	}

	s.rngMu.Lock()
	r := s.rng.Float64()
	s.rngMu.Unlock()

	return r < effectiveRate
}

// GetStats returns current sampling statistics
func (s *TraceSampler) GetStats() *SamplerStats {
	received := s.received.Load()
	sampled := s.sampled.Load()
	dropped := s.dropped.Load()

	var sampleRate float64
	if received > 0 {
		sampleRate = float64(sampled) / float64(received)
	}

	return &SamplerStats{
		TotalReceived: received,
		TotalSampled:  sampled,
		TotalDropped:  dropped,
		SampleRate:    sampleRate,
	}
}

// ProjectSampler manages per-project sampling configurations
type ProjectSampler struct {
	defaultSampler *TraceSampler
	projectConfigs map[string]*SamplerConfig
	projectMu      sync.RWMutex
	logger         *zap.Logger
}

// NewProjectSampler creates a new project-aware sampler
func NewProjectSampler(defaultConfig *SamplerConfig, logger *zap.Logger) *ProjectSampler {
	return &ProjectSampler{
		defaultSampler: NewTraceSampler(defaultConfig, logger),
		projectConfigs: make(map[string]*SamplerConfig),
		logger:         logger,
	}
}

// SetProjectConfig sets sampling configuration for a specific project
func (ps *ProjectSampler) SetProjectConfig(projectID string, config *SamplerConfig) {
	ps.projectMu.Lock()
	defer ps.projectMu.Unlock()
	ps.projectConfigs[projectID] = config
}

// RemoveProjectConfig removes sampling configuration for a project
func (ps *ProjectSampler) RemoveProjectConfig(projectID string) {
	ps.projectMu.Lock()
	defer ps.projectMu.Unlock()
	delete(ps.projectConfigs, projectID)
}

// GetProjectConfig returns the sampling configuration for a project
func (ps *ProjectSampler) GetProjectConfig(projectID string) *SamplerConfig {
	ps.projectMu.RLock()
	defer ps.projectMu.RUnlock()

	if config, ok := ps.projectConfigs[projectID]; ok {
		return config
	}
	return ps.defaultSampler.config
}

// ShouldSample determines whether to sample a trace for a given project
func (ps *ProjectSampler) ShouldSample(ctx context.Context, trace *domain.Trace) bool {
	ps.projectMu.RLock()
	projectConfig, hasProjectConfig := ps.projectConfigs[trace.ProjectID.String()]
	ps.projectMu.RUnlock()

	if hasProjectConfig {
		// Create a temporary sampler with project-specific config
		sampler := NewTraceSampler(projectConfig, ps.logger)
		return sampler.ShouldSample(ctx, trace)
	}

	return ps.defaultSampler.ShouldSample(ctx, trace)
}

// GetStats returns the default sampler stats
func (ps *ProjectSampler) GetStats() *SamplerStats {
	return ps.defaultSampler.GetStats()
}

package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	// ErrCircuitBreakerOpen is returned when the circuit breaker is open
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	// ErrCircuitBreakerTimeout is returned when an operation times out
	ErrCircuitBreakerTimeout = errors.New("circuit breaker timeout")
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name               string
	maxFailures        int           // Number of failures before opening
	timeout            time.Duration // Timeout for operations
	resetTimeout       time.Duration // Time to wait before attempting to close
	halfOpenMaxAttempts int           // Max attempts in half-open state

	mutex              sync.RWMutex
	state              CircuitBreakerState
	failures           int
	successes          int
	lastFailureTime    time.Time
	lastStateChange    time.Time
	consecutiveSuccesses int

	logger             *zap.Logger
}

// CircuitBreakerConfig configures a circuit breaker
type CircuitBreakerConfig struct {
	Name                string
	MaxFailures         int           // Default: 5
	Timeout             time.Duration // Default: 30s
	ResetTimeout        time.Duration // Default: 60s
	HalfOpenMaxAttempts int           // Default: 3
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig, logger *zap.Logger) *CircuitBreaker {
	if config.MaxFailures == 0 {
		config.MaxFailures = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.ResetTimeout == 0 {
		config.ResetTimeout = 60 * time.Second
	}
	if config.HalfOpenMaxAttempts == 0 {
		config.HalfOpenMaxAttempts = 3
	}

	cb := &CircuitBreaker{
		name:                config.Name,
		maxFailures:         config.MaxFailures,
		timeout:             config.Timeout,
		resetTimeout:        config.ResetTimeout,
		halfOpenMaxAttempts: config.HalfOpenMaxAttempts,
		state:               StateClosed,
		lastStateChange:     time.Now(),
		logger:              logger.With(zap.String("circuit_breaker", config.Name)),
	}

	cb.logger.Info("circuit breaker initialized",
		zap.Int("max_failures", config.MaxFailures),
		zap.Duration("timeout", config.Timeout),
		zap.Duration("reset_timeout", config.ResetTimeout),
	)

	return cb
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Check if circuit is open
	if !cb.canExecute() {
		cb.logger.Debug("circuit breaker is open, rejecting request")
		return ErrCircuitBreakerOpen
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, cb.timeout)
	defer cancel()

	// Execute function
	errChan := make(chan error, 1)
	go func() {
		errChan <- fn(execCtx)
	}()

	// Wait for result or timeout
	select {
	case err := <-errChan:
		if err != nil {
			cb.recordFailure(err)
			return err
		}
		cb.recordSuccess()
		return nil
	case <-execCtx.Done():
		cb.recordFailure(ErrCircuitBreakerTimeout)
		return ErrCircuitBreakerTimeout
	}
}

// ExecuteWithFallback executes a function with a fallback
func (cb *CircuitBreaker) ExecuteWithFallback(
	ctx context.Context,
	fn func(context.Context) error,
	fallback func(context.Context) error,
) error {
	err := cb.Execute(ctx, fn)
	if err == nil {
		return nil
	}

	// If circuit breaker is open or operation failed, use fallback
	cb.logger.Debug("using fallback function", zap.Error(err))
	return fallback(ctx)
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastStateChange) > cb.resetTimeout {
			cb.setState(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited attempts in half-open state
		return cb.consecutiveSuccesses < cb.halfOpenMaxAttempts
	default:
		return false
	}
}

// recordSuccess records a successful execution
func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.successes++
	cb.consecutiveSuccesses++

	switch cb.state {
	case StateHalfOpen:
		// If we've had enough successes, close the circuit
		if cb.consecutiveSuccesses >= cb.halfOpenMaxAttempts {
			cb.setState(StateClosed)
			cb.failures = 0
		}
	case StateClosed:
		// Reset failure count on success
		if cb.failures > 0 {
			cb.failures = 0
		}
	}

	cb.logger.Debug("operation succeeded",
		zap.String("state", cb.state.String()),
		zap.Int("consecutive_successes", cb.consecutiveSuccesses),
	)
}

// recordFailure records a failed execution
func (cb *CircuitBreaker) recordFailure(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures++
	cb.consecutiveSuccesses = 0
	cb.lastFailureTime = time.Now()

	cb.logger.Warn("operation failed",
		zap.Error(err),
		zap.String("state", cb.state.String()),
		zap.Int("failures", cb.failures),
	)

	switch cb.state {
	case StateClosed:
		// Open circuit if max failures reached
		if cb.failures >= cb.maxFailures {
			cb.setState(StateOpen)
		}
	case StateHalfOpen:
		// Go back to open on any failure in half-open state
		cb.setState(StateOpen)
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	cb.logger.Info("circuit breaker state changed",
		zap.String("old_state", oldState.String()),
		zap.String("new_state", newState.String()),
		zap.Int("failures", cb.failures),
		zap.Int("successes", cb.successes),
	)

	// Reset counters on state change
	if newState == StateClosed {
		cb.failures = 0
		cb.consecutiveSuccesses = 0
	} else if newState == StateHalfOpen {
		cb.consecutiveSuccesses = 0
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() *CircuitBreakerStats {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return &CircuitBreakerStats{
		Name:                 cb.name,
		State:                cb.state.String(),
		Failures:             cb.failures,
		Successes:            cb.successes,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		LastFailureTime:      cb.lastFailureTime,
		LastStateChange:      cb.lastStateChange,
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.setState(StateClosed)
	cb.failures = 0
	cb.successes = 0
	cb.consecutiveSuccesses = 0

	cb.logger.Info("circuit breaker manually reset")
}

// CircuitBreakerStats contains statistics about the circuit breaker
type CircuitBreakerStats struct {
	Name                 string
	State                string
	Failures             int
	Successes            int
	ConsecutiveSuccesses int
	LastFailureTime      time.Time
	LastStateChange      time.Time
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
	logger   *zap.Logger
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(logger *zap.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (m *CircuitBreakerManager) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	m.mutex.RLock()
	if breaker, exists := m.breakers[name]; exists {
		m.mutex.RUnlock()
		return breaker
	}
	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := m.breakers[name]; exists {
		return breaker
	}

	config.Name = name
	breaker := NewCircuitBreaker(config, m.logger)
	m.breakers[name] = breaker

	return breaker
}

// Get retrieves a circuit breaker by name
func (m *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	breaker, exists := m.breakers[name]
	return breaker, exists
}

// GetAll returns all circuit breakers
func (m *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*CircuitBreaker, len(m.breakers))
	for name, breaker := range m.breakers {
		result[name] = breaker
	}

	return result
}

// GetAllStats returns statistics for all circuit breakers
func (m *CircuitBreakerManager) GetAllStats() map[string]*CircuitBreakerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]*CircuitBreakerStats, len(m.breakers))
	for name, breaker := range m.breakers {
		stats[name] = breaker.GetStats()
	}

	return stats
}

// ResetAll resets all circuit breakers
func (m *CircuitBreakerManager) ResetAll() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, breaker := range m.breakers {
		breaker.Reset()
	}

	m.logger.Info("all circuit breakers reset", zap.Int("count", len(m.breakers)))
}

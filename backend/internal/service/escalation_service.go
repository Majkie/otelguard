package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// EscalationService handles alert escalation logic
type EscalationService struct {
	alertRepo *postgres.AlertRepository
	notifier  *NotificationService
	logger    *zap.Logger

	// Escalation tracking
	activeEscalations map[string]*EscalationState
	mu                sync.RWMutex
}

// EscalationState tracks the state of an active escalation
type EscalationState struct {
	AlertHistoryID uuid.UUID
	PolicyID       uuid.UUID
	CurrentStep    int
	StartedAt      time.Time
	NextEscalation time.Time
	CancelFunc     context.CancelFunc
}

// NewEscalationService creates a new escalation service
func NewEscalationService(
	alertRepo *postgres.AlertRepository,
	notifier *NotificationService,
	logger *zap.Logger,
) *EscalationService {
	return &EscalationService{
		alertRepo:         alertRepo,
		notifier:          notifier,
		logger:            logger,
		activeEscalations: make(map[string]*EscalationState),
	}
}

// CreateEscalationPolicy creates a new escalation policy
func (s *EscalationService) CreateEscalationPolicy(ctx context.Context, policy *domain.AlertEscalationPolicy) error {
	policy.ID = uuid.New()
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	// Validate steps
	if len(policy.Steps) == 0 {
		return fmt.Errorf("escalation policy must have at least one step")
	}

	for i, step := range policy.Steps {
		if step.Delay < 0 {
			return fmt.Errorf("step %d: delay must be non-negative", i)
		}
		if len(step.Channels) == 0 {
			return fmt.Errorf("step %d: must have at least one notification channel", i)
		}
	}

	return s.alertRepo.CreateEscalationPolicy(ctx, policy)
}

// GetEscalationPolicy retrieves an escalation policy by ID
func (s *EscalationService) GetEscalationPolicy(ctx context.Context, id uuid.UUID) (*domain.AlertEscalationPolicy, error) {
	return s.alertRepo.GetEscalationPolicyByID(ctx, id)
}

// ListEscalationPolicies lists escalation policies for a project
func (s *EscalationService) ListEscalationPolicies(ctx context.Context, projectID uuid.UUID, opts *postgres.ListOptions) ([]*domain.AlertEscalationPolicy, int, error) {
	return s.alertRepo.ListEscalationPolicies(ctx, projectID, opts)
}

// UpdateEscalationPolicy updates an escalation policy
func (s *EscalationService) UpdateEscalationPolicy(ctx context.Context, policy *domain.AlertEscalationPolicy) error {
	policy.UpdatedAt = time.Now()

	// Validate steps
	if len(policy.Steps) == 0 {
		return fmt.Errorf("escalation policy must have at least one step")
	}

	for i, step := range policy.Steps {
		if step.Delay < 0 {
			return fmt.Errorf("step %d: delay must be non-negative", i)
		}
		if len(step.Channels) == 0 {
			return fmt.Errorf("step %d: must have at least one notification channel", i)
		}
	}

	return s.alertRepo.UpdateEscalationPolicy(ctx, policy)
}

// DeleteEscalationPolicy deletes an escalation policy
func (s *EscalationService) DeleteEscalationPolicy(ctx context.Context, id uuid.UUID) error {
	return s.alertRepo.DeleteEscalationPolicy(ctx, id)
}

// StartEscalation starts an escalation workflow for an alert
func (s *EscalationService) StartEscalation(ctx context.Context, alertHistory *domain.AlertHistory, policy *domain.AlertEscalationPolicy, rule *domain.AlertRule) error {
	escalationKey := s.getEscalationKey(alertHistory.ID, policy.ID)

	// Check if escalation is already active
	s.mu.RLock()
	if _, exists := s.activeEscalations[escalationKey]; exists {
		s.mu.RUnlock()
		s.logger.Debug("escalation already active",
			zap.String("alert_id", alertHistory.ID.String()),
			zap.String("policy_id", policy.ID.String()),
		)
		return nil
	}
	s.mu.RUnlock()

	// Create escalation context
	escalationCtx, cancel := context.WithCancel(context.Background())

	state := &EscalationState{
		AlertHistoryID: alertHistory.ID,
		PolicyID:       policy.ID,
		CurrentStep:    0,
		StartedAt:      time.Now(),
		NextEscalation: time.Now(),
		CancelFunc:     cancel,
	}

	s.mu.Lock()
	s.activeEscalations[escalationKey] = state
	s.mu.Unlock()

	// Start escalation goroutine
	go s.runEscalation(escalationCtx, state, policy, rule, alertHistory)

	s.logger.Info("escalation started",
		zap.String("alert_id", alertHistory.ID.String()),
		zap.String("policy_id", policy.ID.String()),
		zap.Int("steps", len(policy.Steps)),
	)

	return nil
}

// StopEscalation stops an active escalation
func (s *EscalationService) StopEscalation(alertHistoryID uuid.UUID, policyID uuid.UUID) {
	escalationKey := s.getEscalationKey(alertHistoryID, policyID)

	s.mu.Lock()
	defer s.mu.Unlock()

	if state, exists := s.activeEscalations[escalationKey]; exists {
		state.CancelFunc()
		delete(s.activeEscalations, escalationKey)

		s.logger.Info("escalation stopped",
			zap.String("alert_id", alertHistoryID.String()),
			zap.String("policy_id", policyID.String()),
		)
	}
}

// runEscalation executes the escalation workflow
func (s *EscalationService) runEscalation(ctx context.Context, state *EscalationState, policy *domain.AlertEscalationPolicy, rule *domain.AlertRule, alertHistory *domain.AlertHistory) {
	defer func() {
		escalationKey := s.getEscalationKey(state.AlertHistoryID, state.PolicyID)
		s.mu.Lock()
		delete(s.activeEscalations, escalationKey)
		s.mu.Unlock()
	}()

	for stepIdx, step := range policy.Steps {
		// Wait for the delay before executing this step
		delay := time.Duration(step.Delay) * time.Second

		select {
		case <-ctx.Done():
			s.logger.Info("escalation cancelled",
				zap.String("alert_id", state.AlertHistoryID.String()),
				zap.Int("step", stepIdx),
			)
			return
		case <-time.After(delay):
			// Execute escalation step
			s.executeEscalationStep(ctx, stepIdx, step, rule, alertHistory)

			// Update state
			s.mu.Lock()
			if state, exists := s.activeEscalations[s.getEscalationKey(state.AlertHistoryID, state.PolicyID)]; exists {
				state.CurrentStep = stepIdx + 1
				if stepIdx < len(policy.Steps)-1 {
					state.NextEscalation = time.Now().Add(time.Duration(policy.Steps[stepIdx+1].Delay) * time.Second)
				}
			}
			s.mu.Unlock()
		}
	}

	s.logger.Info("escalation completed",
		zap.String("alert_id", state.AlertHistoryID.String()),
		zap.Int("total_steps", len(policy.Steps)),
	)
}

// executeEscalationStep executes a single escalation step
func (s *EscalationService) executeEscalationStep(ctx context.Context, stepIdx int, step domain.EscalationStep, rule *domain.AlertRule, alertHistory *domain.AlertHistory) {
	s.logger.Info("executing escalation step",
		zap.String("alert_id", alertHistory.ID.String()),
		zap.Int("step", stepIdx),
		zap.Int("delay", step.Delay),
		zap.Int("channels", len(step.Channels)),
	)

	// Send notifications through the specified channels
	for _, channel := range step.Channels {
		parts := splitChannel(channel)
		if len(parts) != 2 {
			s.logger.Warn("invalid channel format", zap.String("channel", channel))
			continue
		}

		channelType := parts[0]
		channelTarget := parts[1]

		var err error
		switch channelType {
		case "email":
			message := fmt.Sprintf("[ESCALATION STEP %d] %s", stepIdx+1, *alertHistory.Message)
			err = s.notifier.SendEmail(ctx, channelTarget, fmt.Sprintf("[Escalated] %s", rule.Name), message)
		case "slack":
			err = s.notifier.SendSlack(ctx, channelTarget, rule, alertHistory)
		case "webhook":
			err = s.notifier.SendWebhook(ctx, channelTarget, rule, alertHistory)
		default:
			s.logger.Warn("unsupported notification channel type", zap.String("type", channelType))
			continue
		}

		// Log notification attempt
		logEntry := &domain.AlertNotificationLog{
			ID:             uuid.New(),
			AlertHistoryID: alertHistory.ID,
			ChannelType:    channelType,
			ChannelTarget:  channelTarget,
			Status:         "sent",
			SentAt:         time.Now(),
		}

		if err != nil {
			logEntry.Status = "failed"
			errMsg := err.Error()
			logEntry.ErrorMessage = &errMsg
			s.logger.Error("failed to send escalation notification",
				zap.String("channel", channel),
				zap.Error(err),
			)
		}

		if err := s.alertRepo.CreateNotificationLog(ctx, logEntry); err != nil {
			s.logger.Error("failed to create notification log", zap.Error(err))
		}
	}
}

// getEscalationKey generates a unique key for an escalation
func (s *EscalationService) getEscalationKey(alertHistoryID uuid.UUID, policyID uuid.UUID) string {
	return fmt.Sprintf("%s:%s", alertHistoryID.String(), policyID.String())
}

// GetActiveEscalations returns all active escalations
func (s *EscalationService) GetActiveEscalations() map[string]*EscalationState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid concurrent modifications
	result := make(map[string]*EscalationState)
	for k, v := range s.activeEscalations {
		result[k] = v
	}
	return result
}

// splitChannel splits a channel string into type and target
func splitChannel(channel string) []string {
	for i := 0; i < len(channel); i++ {
		if channel[i] == ':' {
			return []string{channel[:i], channel[i+1:]}
		}
	}
	return []string{channel}
}

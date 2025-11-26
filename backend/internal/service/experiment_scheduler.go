package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// ScheduledExperiment represents a scheduled experiment execution
type ScheduledExperiment struct {
	ID           uuid.UUID
	ExperimentID uuid.UUID
	ProjectID    uuid.UUID
	ScheduleType string    // once, daily, weekly, monthly, cron
	ScheduleTime time.Time // for "once" type
	Cron         string    // for "cron" type
	Enabled      bool
	LastRunAt    *time.Time
	NextRunAt    *time.Time
	CreatedBy    uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ExperimentScheduler handles scheduled experiment execution
type ExperimentScheduler struct {
	experimentService *ExperimentService
	experimentRepo    *postgres.ExperimentRepository
	schedules         map[uuid.UUID]*ScheduledExperiment
	schedulesMu       sync.RWMutex
	ticker            *time.Ticker
	stopChan          chan struct{}
	isRunning         bool
	runningMu         sync.RWMutex
	logger            *zap.Logger
}

// NewExperimentScheduler creates a new experiment scheduler
func NewExperimentScheduler(
	experimentService *ExperimentService,
	experimentRepo *postgres.ExperimentRepository,
	logger *zap.Logger,
) *ExperimentScheduler {
	return &ExperimentScheduler{
		experimentService: experimentService,
		experimentRepo:    experimentRepo,
		schedules:         make(map[uuid.UUID]*ScheduledExperiment),
		ticker:            time.NewTicker(time.Minute), // Check every minute
		stopChan:          make(chan struct{}),
		logger:            logger,
	}
}

// Start starts the scheduler
func (s *ExperimentScheduler) Start() {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()

	if s.isRunning {
		return
	}

	s.isRunning = true
	go s.run()

	s.logger.Info("experiment scheduler started")
}

// Stop stops the scheduler
func (s *ExperimentScheduler) Stop() {
	s.runningMu.Lock()
	if !s.isRunning {
		s.runningMu.Unlock()
		return
	}
	s.isRunning = false
	s.runningMu.Unlock()

	close(s.stopChan)
	s.ticker.Stop()

	s.logger.Info("experiment scheduler stopped")
}

// run is the main scheduler loop
func (s *ExperimentScheduler) run() {
	for {
		select {
		case <-s.stopChan:
			return
		case <-s.ticker.C:
			s.checkAndRunSchedules()
		}
	}
}

// checkAndRunSchedules checks for schedules that need to run
func (s *ExperimentScheduler) checkAndRunSchedules() {
	now := time.Now()

	s.schedulesMu.RLock()
	schedules := make([]*ScheduledExperiment, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}
	s.schedulesMu.RUnlock()

	for _, schedule := range schedules {
		if !schedule.Enabled {
			continue
		}

		// Check if schedule should run
		if s.shouldRun(schedule, now) {
			s.logger.Info("triggering scheduled experiment",
				zap.String("schedule_id", schedule.ID.String()),
				zap.String("experiment_id", schedule.ExperimentID.String()),
			)

			go s.executeScheduledExperiment(schedule)
		}
	}
}

// shouldRun checks if a schedule should run now
func (s *ExperimentScheduler) shouldRun(schedule *ScheduledExperiment, now time.Time) bool {
	// If no next run time set, calculate it
	if schedule.NextRunAt == nil {
		nextRun := s.calculateNextRun(schedule, now)
		if nextRun == nil {
			return false
		}
		schedule.NextRunAt = nextRun
	}

	// Check if it's time to run
	return !now.Before(*schedule.NextRunAt)
}

// calculateNextRun calculates the next run time for a schedule
func (s *ExperimentScheduler) calculateNextRun(schedule *ScheduledExperiment, from time.Time) *time.Time {
	switch schedule.ScheduleType {
	case "once":
		if schedule.LastRunAt != nil {
			// Already ran, no next run
			return nil
		}
		return &schedule.ScheduleTime

	case "daily":
		// Run at same time every day
		next := from.Add(24 * time.Hour)
		if schedule.LastRunAt != nil {
			next = schedule.LastRunAt.Add(24 * time.Hour)
		}
		return &next

	case "weekly":
		// Run at same time every week
		next := from.Add(7 * 24 * time.Hour)
		if schedule.LastRunAt != nil {
			next = schedule.LastRunAt.Add(7 * 24 * time.Hour)
		}
		return &next

	case "monthly":
		// Run at same day of month
		next := from.AddDate(0, 1, 0)
		if schedule.LastRunAt != nil {
			next = schedule.LastRunAt.AddDate(0, 1, 0)
		}
		return &next

	case "cron":
		// TODO: Implement cron parsing
		// For now, default to daily
		next := from.Add(24 * time.Hour)
		return &next

	default:
		return nil
	}
}

// executeScheduledExperiment executes a scheduled experiment
func (s *ExperimentScheduler) executeScheduledExperiment(schedule *ScheduledExperiment) {
	ctx := context.Background()

	// Get experiment
	experiment, err := s.experimentRepo.GetByID(ctx, schedule.ExperimentID.String())
	if err != nil {
		s.logger.Error("failed to get experiment for schedule",
			zap.String("schedule_id", schedule.ID.String()),
			zap.Error(err),
		)
		return
	}

	// Execute experiment
	_, err = s.experimentService.Execute(ctx, schedule.ExperimentID)
	if err != nil {
		s.logger.Error("failed to execute scheduled experiment",
			zap.String("schedule_id", schedule.ID.String()),
			zap.String("experiment_id", schedule.ExperimentID.String()),
			zap.Error(err),
		)
		return
	}

	// Update schedule
	now := time.Now()
	s.schedulesMu.Lock()
	schedule.LastRunAt = &now

	// Calculate next run time
	nextRun := s.calculateNextRun(schedule, now)
	schedule.NextRunAt = nextRun

	// Disable if it's a one-time schedule that has run
	if schedule.ScheduleType == "once" {
		schedule.Enabled = false
	}
	s.schedulesMu.Unlock()

	s.logger.Info("executed scheduled experiment",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("experiment_id", experiment.ID.String()),
		zap.String("experiment_name", experiment.Name),
	)
}

// AddSchedule adds a new schedule
func (s *ExperimentScheduler) AddSchedule(schedule *ScheduledExperiment) {
	s.schedulesMu.Lock()
	defer s.schedulesMu.Unlock()

	// Calculate initial next run time
	if schedule.NextRunAt == nil {
		nextRun := s.calculateNextRun(schedule, time.Now())
		schedule.NextRunAt = nextRun
	}

	s.schedules[schedule.ID] = schedule

	s.logger.Info("added experiment schedule",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("experiment_id", schedule.ExperimentID.String()),
		zap.String("schedule_type", schedule.ScheduleType),
	)
}

// RemoveSchedule removes a schedule
func (s *ExperimentScheduler) RemoveSchedule(scheduleID uuid.UUID) {
	s.schedulesMu.Lock()
	defer s.schedulesMu.Unlock()

	delete(s.schedules, scheduleID)

	s.logger.Info("removed experiment schedule",
		zap.String("schedule_id", scheduleID.String()),
	)
}

// GetSchedule retrieves a schedule by ID
func (s *ExperimentScheduler) GetSchedule(scheduleID uuid.UUID) (*ScheduledExperiment, bool) {
	s.schedulesMu.RLock()
	defer s.schedulesMu.RUnlock()

	schedule, exists := s.schedules[scheduleID]
	return schedule, exists
}

// ListSchedules lists all schedules for a project
func (s *ExperimentScheduler) ListSchedules(projectID uuid.UUID) []*ScheduledExperiment {
	s.schedulesMu.RLock()
	defer s.schedulesMu.RUnlock()

	var schedules []*ScheduledExperiment
	for _, schedule := range s.schedules {
		if schedule.ProjectID == projectID {
			schedules = append(schedules, schedule)
		}
	}

	return schedules
}

// UpdateSchedule updates an existing schedule
func (s *ExperimentScheduler) UpdateSchedule(scheduleID uuid.UUID, updates func(*ScheduledExperiment)) error {
	s.schedulesMu.Lock()
	defer s.schedulesMu.Unlock()

	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return domain.ErrNotFound
	}

	updates(schedule)
	schedule.UpdatedAt = time.Now()

	// Recalculate next run time if schedule changed
	nextRun := s.calculateNextRun(schedule, time.Now())
	schedule.NextRunAt = nextRun

	return nil
}

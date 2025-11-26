package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// DashboardService handles dashboard business logic
type DashboardService struct {
	repo   *postgres.DashboardRepository
	logger *zap.Logger
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(repo *postgres.DashboardRepository, logger *zap.Logger) *DashboardService {
	return &DashboardService{
		repo:   repo,
		logger: logger,
	}
}

// CreateDashboard creates a new dashboard
func (s *DashboardService) CreateDashboard(ctx context.Context, projectID, createdBy uuid.UUID, name, description string, layout json.RawMessage) (*domain.Dashboard, error) {
	dashboard := &domain.Dashboard{
		ID:          uuid.New(),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		Layout:      layout,
		IsTemplate:  false,
		IsPublic:    false,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, dashboard); err != nil {
		s.logger.Error("failed to create dashboard", zap.Error(err))
		return nil, err
	}

	return dashboard, nil
}

// GetDashboard retrieves a dashboard by ID
func (s *DashboardService) GetDashboard(ctx context.Context, id uuid.UUID) (*domain.Dashboard, error) {
	return s.repo.GetByID(ctx, id)
}

// ListDashboards lists dashboards for a project
func (s *DashboardService) ListDashboards(ctx context.Context, projectID uuid.UUID, includeTemplates bool) ([]*domain.Dashboard, error) {
	return s.repo.List(ctx, projectID, includeTemplates)
}

// UpdateDashboard updates a dashboard
func (s *DashboardService) UpdateDashboard(ctx context.Context, id uuid.UUID, name, description string, layout json.RawMessage, isPublic bool) error {
	dashboard, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	dashboard.Name = name
	dashboard.Description = description
	dashboard.Layout = layout
	dashboard.IsPublic = isPublic
	dashboard.UpdatedAt = time.Now()

	return s.repo.Update(ctx, dashboard)
}

// DeleteDashboard deletes a dashboard
func (s *DashboardService) DeleteDashboard(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// AddWidget adds a widget to a dashboard
func (s *DashboardService) AddWidget(ctx context.Context, dashboardID uuid.UUID, widgetType, title string, config, position json.RawMessage) (*domain.DashboardWidget, error) {
	// Verify dashboard exists
	_, err := s.repo.GetByID(ctx, dashboardID)
	if err != nil {
		return nil, err
	}

	widget := &domain.DashboardWidget{
		ID:          uuid.New(),
		DashboardID: dashboardID,
		WidgetType:  widgetType,
		Title:       title,
		Config:      config,
		Position:    position,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateWidget(ctx, widget); err != nil {
		s.logger.Error("failed to create widget", zap.Error(err))
		return nil, err
	}

	return widget, nil
}

// GetWidgets retrieves all widgets for a dashboard
func (s *DashboardService) GetWidgets(ctx context.Context, dashboardID uuid.UUID) ([]*domain.DashboardWidget, error) {
	return s.repo.GetWidgets(ctx, dashboardID)
}

// UpdateWidget updates a widget
func (s *DashboardService) UpdateWidget(ctx context.Context, id uuid.UUID, widgetType, title string, config, position json.RawMessage) error {
	widget := &domain.DashboardWidget{
		ID:         id,
		WidgetType: widgetType,
		Title:      title,
		Config:     config,
		Position:   position,
		UpdatedAt:  time.Now(),
	}

	return s.repo.UpdateWidget(ctx, widget)
}

// DeleteWidget deletes a widget
func (s *DashboardService) DeleteWidget(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteWidget(ctx, id)
}

// UpdateWidgetLayout updates the positions of multiple widgets (for drag-and-drop)
func (s *DashboardService) UpdateWidgetLayout(ctx context.Context, updates map[uuid.UUID]json.RawMessage) error {
	return s.repo.UpdateWidgetPositions(ctx, updates)
}

// CreateShare creates a shareable link for a dashboard
func (s *DashboardService) CreateShare(ctx context.Context, dashboardID, createdBy uuid.UUID, expiresIn *time.Duration) (*domain.DashboardShare, error) {
	// Verify dashboard exists and is public or user has permission
	dashboard, err := s.repo.GetByID(ctx, dashboardID)
	if err != nil {
		return nil, err
	}

	// Generate random share token
	token, err := generateShareToken()
	if err != nil {
		s.logger.Error("failed to generate share token", zap.Error(err))
		return nil, err
	}

	share := &domain.DashboardShare{
		ID:          uuid.New(),
		DashboardID: dashboardID,
		ShareToken:  token,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}

	// Set expiration if provided
	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		share.ExpiresAt.Time = expiresAt
		share.ExpiresAt.Valid = true
	}

	if err := s.repo.CreateShare(ctx, share); err != nil {
		s.logger.Error("failed to create share", zap.Error(err))
		return nil, err
	}

	s.logger.Info("created dashboard share",
		zap.String("dashboard_id", dashboard.ID.String()),
		zap.String("share_token", token),
	)

	return share, nil
}

// GetDashboardByShare retrieves a dashboard using a share token
func (s *DashboardService) GetDashboardByShare(ctx context.Context, token string) (*domain.Dashboard, error) {
	share, err := s.repo.GetShareByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if share is expired
	if share.ExpiresAt.Valid && share.ExpiresAt.Time.Before(time.Now()) {
		return nil, domain.ErrNotFound
	}

	return s.repo.GetByID(ctx, share.DashboardID)
}

// ListShares lists all shares for a dashboard
func (s *DashboardService) ListShares(ctx context.Context, dashboardID uuid.UUID) ([]*domain.DashboardShare, error) {
	return s.repo.ListShares(ctx, dashboardID)
}

// DeleteShare deletes a share
func (s *DashboardService) DeleteShare(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteShare(ctx, id)
}

// CloneDashboard creates a copy of a dashboard
func (s *DashboardService) CloneDashboard(ctx context.Context, sourceID, newProjectID, createdBy uuid.UUID, name string) (*domain.Dashboard, error) {
	return s.repo.CloneDashboard(ctx, sourceID, newProjectID, createdBy, name)
}

// CloneTemplate creates a dashboard from a template
func (s *DashboardService) CloneTemplate(ctx context.Context, templateID, projectID, createdBy uuid.UUID, name string) (*domain.Dashboard, error) {
	// Verify template exists and is indeed a template
	template, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	if !template.IsTemplate {
		s.logger.Warn("attempted to clone non-template dashboard",
			zap.String("dashboard_id", templateID.String()),
		)
		return nil, domain.ErrValidation
	}

	return s.repo.CloneDashboard(ctx, templateID, projectID, createdBy, name)
}

// generateShareToken generates a random share token
func generateShareToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetDashboardWithWidgets retrieves a dashboard with all its widgets
func (s *DashboardService) GetDashboardWithWidgets(ctx context.Context, id uuid.UUID) (*domain.Dashboard, []*domain.DashboardWidget, error) {
	dashboard, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	widgets, err := s.repo.GetWidgets(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return dashboard, widgets, nil
}

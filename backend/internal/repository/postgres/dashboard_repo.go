package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// DashboardRepository handles dashboard persistence
type DashboardRepository struct {
	db *pgxpool.Pool
}

// NewDashboardRepository creates a new dashboard repository
func NewDashboardRepository(db *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// Create creates a new dashboard
func (r *DashboardRepository) Create(ctx context.Context, dashboard *domain.Dashboard) error {
	query := `
		INSERT INTO dashboards (
			id, project_id, name, description, layout, is_template, is_public, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(ctx, query,
		dashboard.ID,
		dashboard.ProjectID,
		dashboard.Name,
		dashboard.Description,
		dashboard.Layout,
		dashboard.IsTemplate,
		dashboard.IsPublic,
		dashboard.CreatedBy,
		dashboard.CreatedAt,
		dashboard.UpdatedAt,
	)
	return err
}

// GetByID retrieves a dashboard by ID
func (r *DashboardRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Dashboard, error) {
	var dashboard domain.Dashboard
	query := `
		SELECT id, project_id, name, description, layout, is_template, is_public, created_by, created_at, updated_at
		FROM dashboards
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := pgxscan.Get(ctx, r.db, &dashboard, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &dashboard, nil
}

// List retrieves dashboards for a project
func (r *DashboardRepository) List(ctx context.Context, projectID uuid.UUID, includeTemplates bool) ([]*domain.Dashboard, error) {
	var dashboards []*domain.Dashboard
	query := `
		SELECT id, project_id, name, description, layout, is_template, is_public, created_by, created_at, updated_at
		FROM dashboards
		WHERE (project_id = $1 OR (is_template = true AND $2 = true))
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	err := pgxscan.Select(ctx, r.db, &dashboards, query, projectID, includeTemplates)
	if err != nil {
		return nil, err
	}
	return dashboards, nil
}

// Update updates a dashboard
func (r *DashboardRepository) Update(ctx context.Context, dashboard *domain.Dashboard) error {
	query := `
		UPDATE dashboards
		SET name = $1, description = $2, layout = $3, is_public = $4, updated_at = $5
		WHERE id = $6 AND deleted_at IS NULL
	`
	result, err := r.db.Exec(ctx, query,
		dashboard.Name,
		dashboard.Description,
		dashboard.Layout,
		dashboard.IsPublic,
		dashboard.UpdatedAt,
		dashboard.ID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete soft deletes a dashboard
func (r *DashboardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE dashboards SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// CreateWidget creates a new dashboard widget
func (r *DashboardRepository) CreateWidget(ctx context.Context, widget *domain.DashboardWidget) error {
	query := `
		INSERT INTO dashboard_widgets (
			id, dashboard_id, widget_type, title, config, position, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		widget.ID,
		widget.DashboardID,
		widget.WidgetType,
		widget.Title,
		widget.Config,
		widget.Position,
		widget.CreatedAt,
		widget.UpdatedAt,
	)
	return err
}

// GetWidgets retrieves all widgets for a dashboard
func (r *DashboardRepository) GetWidgets(ctx context.Context, dashboardID uuid.UUID) ([]*domain.DashboardWidget, error) {
	var widgets []*domain.DashboardWidget
	query := `
		SELECT id, dashboard_id, widget_type, title, config, position, created_at, updated_at
		FROM dashboard_widgets
		WHERE dashboard_id = $1
		ORDER BY created_at ASC
	`
	err := pgxscan.Select(ctx, r.db, &widgets, query, dashboardID)
	if err != nil {
		return nil, err
	}
	return widgets, nil
}

// UpdateWidget updates a widget
func (r *DashboardRepository) UpdateWidget(ctx context.Context, widget *domain.DashboardWidget) error {
	query := `
		UPDATE dashboard_widgets
		SET widget_type = $1, title = $2, config = $3, position = $4, updated_at = $5
		WHERE id = $6
	`
	result, err := r.db.Exec(ctx, query,
		widget.WidgetType,
		widget.Title,
		widget.Config,
		widget.Position,
		widget.UpdatedAt,
		widget.ID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// DeleteWidget deletes a widget
func (r *DashboardRepository) DeleteWidget(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM dashboard_widgets WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// CreateShare creates a dashboard share
func (r *DashboardRepository) CreateShare(ctx context.Context, share *domain.DashboardShare) error {
	query := `
		INSERT INTO dashboard_shares (id, dashboard_id, share_token, expires_at, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		share.ID,
		share.DashboardID,
		share.ShareToken,
		share.ExpiresAt,
		share.CreatedBy,
		share.CreatedAt,
	)
	return err
}

// GetShareByToken retrieves a dashboard share by token
func (r *DashboardRepository) GetShareByToken(ctx context.Context, token string) (*domain.DashboardShare, error) {
	var share domain.DashboardShare
	query := `
		SELECT id, dashboard_id, share_token, expires_at, created_by, created_at
		FROM dashboard_shares
		WHERE share_token = $1
	`
	err := pgxscan.Get(ctx, r.db, &share, query, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &share, nil
}

// ListShares retrieves all shares for a dashboard
func (r *DashboardRepository) ListShares(ctx context.Context, dashboardID uuid.UUID) ([]*domain.DashboardShare, error) {
	var shares []*domain.DashboardShare
	query := `
		SELECT id, dashboard_id, share_token, expires_at, created_by, created_at
		FROM dashboard_shares
		WHERE dashboard_id = $1
		ORDER BY created_at DESC
	`
	err := pgxscan.Select(ctx, r.db, &shares, query, dashboardID)
	if err != nil {
		return nil, err
	}
	return shares, nil
}

// DeleteShare deletes a dashboard share
func (r *DashboardRepository) DeleteShare(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM dashboard_shares WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// CloneDashboard creates a copy of a dashboard
func (r *DashboardRepository) CloneDashboard(ctx context.Context, sourceID, newProjectID, createdBy uuid.UUID, name string) (*domain.Dashboard, error) {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Get source dashboard
	var source domain.Dashboard
	err = pgxscan.Get(ctx, tx, &source, `
		SELECT id, project_id, name, description, layout, is_template, is_public, created_by, created_at, updated_at
		FROM dashboards
		WHERE id = $1 AND deleted_at IS NULL
	`, sourceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	// Create new dashboard
	newDashboard := &domain.Dashboard{
		ID:          uuid.New(),
		ProjectID:   newProjectID,
		Name:        name,
		Description: source.Description,
		Layout:      source.Layout,
		IsTemplate:  false,
		IsPublic:    false,
		CreatedBy:   createdBy,
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO dashboards (id, project_id, name, description, layout, is_template, is_public, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, newDashboard.ID, newDashboard.ProjectID, newDashboard.Name, newDashboard.Description,
		newDashboard.Layout, newDashboard.IsTemplate, newDashboard.IsPublic, newDashboard.CreatedBy)
	if err != nil {
		return nil, err
	}

	// Copy widgets
	_, err = tx.Exec(ctx, `
		INSERT INTO dashboard_widgets (id, dashboard_id, widget_type, title, config, position)
		SELECT gen_random_uuid(), $1, widget_type, title, config, position
		FROM dashboard_widgets
		WHERE dashboard_id = $2
	`, newDashboard.ID, sourceID)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Return newly created dashboard
	return r.GetByID(ctx, newDashboard.ID)
}

// UpdateWidgetPositions updates multiple widget positions in one call
func (r *DashboardRepository) UpdateWidgetPositions(ctx context.Context, updates map[uuid.UUID]json.RawMessage) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for widgetID, position := range updates {
		_, err = tx.Exec(ctx, `
			UPDATE dashboard_widgets SET position = $1, updated_at = NOW() WHERE id = $2
		`, position, widgetID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

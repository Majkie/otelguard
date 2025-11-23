package postgres

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// ProjectRepository handles project data access
type ProjectRepository struct {
	db *pgxpool.Pool
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project
func (r *ProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	query := `
		INSERT INTO projects (id, organization_id, name, slug, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		project.ID,
		project.OrganizationID,
		project.Name,
		project.Slug,
		project.Settings,
		project.CreatedAt,
		project.UpdatedAt,
	)
	return err
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	var project domain.Project
	query := `
		SELECT id, organization_id, name, slug, settings, created_at, updated_at
		FROM projects
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := pgxscan.Get(ctx, r.db, &project, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &project, nil
}

// GetBySlug retrieves a project by organization ID and slug
func (r *ProjectRepository) GetBySlug(ctx context.Context, orgID, slug string) (*domain.Project, error) {
	var project domain.Project
	query := `
		SELECT id, organization_id, name, slug, settings, created_at, updated_at
		FROM projects
		WHERE organization_id = $1 AND slug = $2 AND deleted_at IS NULL
	`
	err := pgxscan.Get(ctx, r.db, &project, query, orgID, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &project, nil
}

// Update updates a project
func (r *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	query := `
		UPDATE projects
		SET name = $2, slug = $3, settings = $4, updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.Exec(ctx, query,
		project.ID,
		project.Name,
		project.Slug,
		project.Settings,
		project.UpdatedAt,
	)
	return err
}

// Delete soft-deletes a project
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE projects SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// ListByOrganizationID lists all projects in an organization
func (r *ProjectRepository) ListByOrganizationID(ctx context.Context, orgID string, limit, offset int) ([]*domain.Project, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM projects WHERE organization_id = $1 AND deleted_at IS NULL`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, orgID); err != nil {
		return nil, 0, err
	}

	var projects []*domain.Project
	query := `
		SELECT id, organization_id, name, slug, settings, created_at, updated_at
		FROM projects
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &projects, query, orgID, limit, offset); err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

// ListByUserID lists projects accessible to a user (via org membership or direct project membership)
func (r *ProjectRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Project, int, error) {
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT p.id)
		FROM projects p
		LEFT JOIN organization_members om ON p.organization_id = om.organization_id
		LEFT JOIN project_members pm ON p.id = pm.project_id
		WHERE (om.user_id = $1 OR pm.user_id = $1) AND p.deleted_at IS NULL
	`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, userID); err != nil {
		return nil, 0, err
	}

	var projects []*domain.Project
	query := `
		SELECT DISTINCT p.id, p.organization_id, p.name, p.slug, p.settings, p.created_at, p.updated_at
		FROM projects p
		LEFT JOIN organization_members om ON p.organization_id = om.organization_id
		LEFT JOIN project_members pm ON p.id = pm.project_id
		WHERE (om.user_id = $1 OR pm.user_id = $1) AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &projects, query, userID, limit, offset); err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

// AddMember adds a user to a project
func (r *ProjectRepository) AddMember(ctx context.Context, member *domain.ProjectMember) error {
	query := `
		INSERT INTO project_members (id, project_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		member.ID,
		member.ProjectID,
		member.UserID,
		member.Role,
		member.CreatedAt,
		member.UpdatedAt,
	)
	return err
}

// RemoveMember removes a user from a project
func (r *ProjectRepository) RemoveMember(ctx context.Context, projectID, userID string) error {
	query := `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, projectID, userID)
	return err
}

// GetMember gets a member's role in a project
func (r *ProjectRepository) GetMember(ctx context.Context, projectID, userID string) (*domain.ProjectMember, error) {
	var member domain.ProjectMember
	query := `
		SELECT id, project_id, user_id, role, created_at, updated_at
		FROM project_members
		WHERE project_id = $1 AND user_id = $2
	`
	err := pgxscan.Get(ctx, r.db, &member, query, projectID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &member, nil
}

// ListMembers lists all members of a project
func (r *ProjectRepository) ListMembers(ctx context.Context, projectID string, limit, offset int) ([]*domain.ProjectMember, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM project_members WHERE project_id = $1`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, projectID); err != nil {
		return nil, 0, err
	}

	var members []*domain.ProjectMember
	query := `
		SELECT id, project_id, user_id, role, created_at, updated_at
		FROM project_members
		WHERE project_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &members, query, projectID, limit, offset); err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

// GetUserRole gets the effective role of a user in a project (considers org membership too)
func (r *ProjectRepository) GetUserRole(ctx context.Context, projectID, userID string) (string, error) {
	// First check project-specific role
	var role string
	projectQuery := `SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2`
	err := pgxscan.Get(ctx, r.db, &role, projectQuery, projectID, userID)
	if err == nil {
		return role, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	// Fall back to org membership role
	orgQuery := `
		SELECT om.role
		FROM organization_members om
		INNER JOIN projects p ON p.organization_id = om.organization_id
		WHERE p.id = $1 AND om.user_id = $2
	`
	err = pgxscan.Get(ctx, r.db, &role, orgQuery, projectID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", err
	}
	return role, nil
}

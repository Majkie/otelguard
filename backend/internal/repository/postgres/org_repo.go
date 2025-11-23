package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// OrganizationRepository handles organization data access
type OrganizationRepository struct {
	db *pgxpool.Pool
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create creates a new organization
func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	query := `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query,
		org.ID,
		org.Name,
		org.Slug,
		org.CreatedAt,
		org.UpdatedAt,
	)
	return err
}

// GetByID retrieves an organization by ID
func (r *OrganizationRepository) GetByID(ctx context.Context, id string) (*domain.Organization, error) {
	var org domain.Organization
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := pgxscan.Get(ctx, r.db, &org, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

// GetBySlug retrieves an organization by slug
func (r *OrganizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	var org domain.Organization
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		WHERE slug = $1 AND deleted_at IS NULL
	`
	err := pgxscan.Get(ctx, r.db, &org, query, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

// Update updates an organization
func (r *OrganizationRepository) Update(ctx context.Context, org *domain.Organization) error {
	query := `
		UPDATE organizations
		SET name = $2, slug = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.Exec(ctx, query,
		org.ID,
		org.Name,
		org.Slug,
		org.UpdatedAt,
	)
	return err
}

// Delete soft-deletes an organization
func (r *OrganizationRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE organizations SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// ListByUserID lists organizations a user belongs to
func (r *OrganizationRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Organization, int, error) {
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT o.id)
		FROM organizations o
		INNER JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = $1 AND o.deleted_at IS NULL
	`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, userID); err != nil {
		return nil, 0, err
	}

	var orgs []*domain.Organization
	query := `
		SELECT o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM organizations o
		INNER JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = $1 AND o.deleted_at IS NULL
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &orgs, query, userID, limit, offset); err != nil {
		return nil, 0, err
	}

	return orgs, total, nil
}

// AddMember adds a user to an organization
func (r *OrganizationRepository) AddMember(ctx context.Context, member *domain.OrganizationMember) error {
	query := `
		INSERT INTO organization_members (id, organization_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query,
		member.ID,
		member.OrganizationID,
		member.UserID,
		member.Role,
		member.CreatedAt,
	)
	return err
}

// RemoveMember removes a user from an organization
func (r *OrganizationRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	query := `DELETE FROM organization_members WHERE organization_id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, orgID, userID)
	return err
}

// GetMember gets a member's role in an organization
func (r *OrganizationRepository) GetMember(ctx context.Context, orgID, userID string) (*domain.OrganizationMember, error) {
	var member domain.OrganizationMember
	query := `
		SELECT id, organization_id, user_id, role, created_at
		FROM organization_members
		WHERE organization_id = $1 AND user_id = $2
	`
	err := pgxscan.Get(ctx, r.db, &member, query, orgID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &member, nil
}

// UpdateMemberRole updates a member's role
func (r *OrganizationRepository) UpdateMemberRole(ctx context.Context, orgID, userID, role string) error {
	query := `
		UPDATE organization_members
		SET role = $3
		WHERE organization_id = $1 AND user_id = $2
	`
	_, err := r.db.Exec(ctx, query, orgID, userID, role)
	return err
}

// ListMembers lists all members of an organization
func (r *OrganizationRepository) ListMembers(ctx context.Context, orgID string, limit, offset int) ([]*domain.OrganizationMember, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM organization_members WHERE organization_id = $1`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, orgID); err != nil {
		return nil, 0, err
	}

	var members []*domain.OrganizationMember
	query := `
		SELECT id, organization_id, user_id, role, created_at
		FROM organization_members
		WHERE organization_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &members, query, orgID, limit, offset); err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

// CreatePasswordResetToken creates a new password reset token
func (r *OrganizationRepository) CreatePasswordResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.CreatedAt,
	)
	return err
}

// GetPasswordResetToken retrieves a password reset token by hash
func (r *OrganizationRepository) GetPasswordResetToken(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	var token domain.PasswordResetToken
	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
	`
	err := pgxscan.Get(ctx, r.db, &token, query, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &token, nil
}

// MarkPasswordResetTokenUsed marks a token as used
func (r *OrganizationRepository) MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE password_reset_tokens SET used_at = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, time.Now())
	return err
}

// InvalidatePasswordResetTokens invalidates all reset tokens for a user
func (r *OrganizationRepository) InvalidatePasswordResetTokens(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE password_reset_tokens SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL`
	_, err := r.db.Exec(ctx, query, userID, time.Now())
	return err
}

// CreateSession creates a new user session
func (r *OrganizationRepository) CreateSession(ctx context.Context, session *domain.UserSession) error {
	query := `
		INSERT INTO user_sessions (id, user_id, token_hash, user_agent, ip_address, last_active_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.UserAgent,
		session.IPAddress,
		session.LastActiveAt,
		session.ExpiresAt,
		session.CreatedAt,
	)
	return err
}

// GetSession retrieves a session by token hash
func (r *OrganizationRepository) GetSession(ctx context.Context, tokenHash string) (*domain.UserSession, error) {
	var session domain.UserSession
	query := `
		SELECT id, user_id, token_hash, user_agent, ip_address, last_active_at, expires_at, revoked_at, created_at
		FROM user_sessions
		WHERE token_hash = $1
	`
	err := pgxscan.Get(ctx, r.db, &session, query, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &session, nil
}

// UpdateSessionActivity updates the last active time of a session
func (r *OrganizationRepository) UpdateSessionActivity(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE user_sessions SET last_active_at = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, time.Now())
	return err
}

// RevokeSession revokes a session
func (r *OrganizationRepository) RevokeSession(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE user_sessions SET revoked_at = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, time.Now())
	return err
}

// RevokeAllUserSessions revokes all sessions for a user
func (r *OrganizationRepository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE user_sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.Exec(ctx, query, userID, time.Now())
	return err
}

// ListUserSessions lists active sessions for a user
func (r *OrganizationRepository) ListUserSessions(ctx context.Context, userID string, limit, offset int) ([]*domain.UserSession, int, error) {
	var total int
	countQuery := `
		SELECT COUNT(*) FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, userID); err != nil {
		return nil, 0, err
	}

	var sessions []*domain.UserSession
	query := `
		SELECT id, user_id, token_hash, user_agent, ip_address, last_active_at, expires_at, revoked_at, created_at
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY last_active_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &sessions, query, userID, limit, offset); err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// OrgService handles organization and project business logic
type OrgService struct {
	orgRepo     *postgres.OrganizationRepository
	projectRepo *postgres.ProjectRepository
	userRepo    *postgres.UserRepository
	logger      *zap.Logger
	bcryptCost  int
}

// NewOrgService creates a new organization service
func NewOrgService(
	orgRepo *postgres.OrganizationRepository,
	projectRepo *postgres.ProjectRepository,
	userRepo *postgres.UserRepository,
	logger *zap.Logger,
	bcryptCost int,
) *OrgService {
	return &OrgService{
		orgRepo:     orgRepo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
		logger:      logger,
		bcryptCost:  bcryptCost,
	}
}

// slugify converts a string to a URL-friendly slug
func slugify(s string) string {
	s = strings.ToLower(s)
	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile("[^a-z0-9-]+")
	s = reg.ReplaceAllString(s, "")
	// Remove consecutive hyphens
	reg = regexp.MustCompile("-+")
	s = reg.ReplaceAllString(s, "-")
	// Trim hyphens from ends
	return strings.Trim(s, "-")
}

// generateToken generates a random token
func generateToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := base64.URLEncoding.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(hash[:]), nil
}

// CreateOrganization creates a new organization with the creator as owner
func (s *OrgService) CreateOrganization(ctx context.Context, userID, name string) (*domain.Organization, error) {
	slug := slugify(name)

	// Check if slug is already taken
	existing, err := s.orgRepo.GetBySlug(ctx, slug)
	if err == nil && existing != nil {
		// Append a random suffix
		suffix := uuid.New().String()[:8]
		slug = slug + "-" + suffix
	}

	org := &domain.Organization{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.orgRepo.Create(ctx, org); err != nil {
		s.logger.Error("failed to create organization", zap.Error(err))
		return nil, errors.New("failed to create organization")
	}

	// Add creator as owner
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	member := &domain.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		UserID:         userUUID,
		Role:           domain.RoleOwner,
		CreatedAt:      time.Now(),
	}

	if err := s.orgRepo.AddMember(ctx, member); err != nil {
		s.logger.Error("failed to add owner to organization", zap.Error(err))
		// Don't fail the whole operation, but log the error
	}

	return org, nil
}

// GetOrganization retrieves an organization by ID
func (s *OrgService) GetOrganization(ctx context.Context, orgID string) (*domain.Organization, error) {
	return s.orgRepo.GetByID(ctx, orgID)
}

// UpdateOrganization updates an organization
func (s *OrgService) UpdateOrganization(ctx context.Context, orgID, name string) (*domain.Organization, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	org.Name = name
	org.Slug = slugify(name)
	org.UpdatedAt = time.Now()

	if err := s.orgRepo.Update(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

// DeleteOrganization deletes an organization
func (s *OrgService) DeleteOrganization(ctx context.Context, orgID string) error {
	return s.orgRepo.Delete(ctx, orgID)
}

// ListUserOrganizations lists organizations a user belongs to
func (s *OrgService) ListUserOrganizations(ctx context.Context, userID string, limit, offset int) ([]*domain.Organization, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.orgRepo.ListByUserID(ctx, userID, limit, offset)
}

// AddOrganizationMember adds a user to an organization
func (s *OrgService) AddOrganizationMember(ctx context.Context, orgID, userID, role string) error {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return errors.New("invalid organization ID")
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	member := &domain.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: orgUUID,
		UserID:         userUUID,
		Role:           role,
		CreatedAt:      time.Now(),
	}

	return s.orgRepo.AddMember(ctx, member)
}

// RemoveOrganizationMember removes a user from an organization
func (s *OrgService) RemoveOrganizationMember(ctx context.Context, orgID, userID string) error {
	return s.orgRepo.RemoveMember(ctx, orgID, userID)
}

// ListOrganizationMembers lists members of an organization
func (s *OrgService) ListOrganizationMembers(ctx context.Context, orgID string, limit, offset int) ([]*domain.OrganizationMember, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.orgRepo.ListMembers(ctx, orgID, limit, offset)
}

// GetUserOrgRole gets the user's role in an organization
func (s *OrgService) GetUserOrgRole(ctx context.Context, orgID, userID string) (string, error) {
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil {
		return "", err
	}
	return member.Role, nil
}

// CreateProject creates a new project in an organization
func (s *OrgService) CreateProject(ctx context.Context, orgID, name string) (*domain.Project, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, errors.New("invalid organization ID")
	}

	slug := slugify(name)

	// Check if slug is already taken in this org
	existing, err := s.projectRepo.GetBySlug(ctx, orgID, slug)
	if err == nil && existing != nil {
		suffix := uuid.New().String()[:8]
		slug = slug + "-" + suffix
	}

	project := &domain.Project{
		ID:             uuid.New(),
		OrganizationID: orgUUID,
		Name:           name,
		Slug:           slug,
		Settings:       []byte("{}"),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		s.logger.Error("failed to create project", zap.Error(err))
		return nil, errors.New("failed to create project")
	}

	return project, nil
}

// GetProject retrieves a project by ID
func (s *OrgService) GetProject(ctx context.Context, projectID string) (*domain.Project, error) {
	return s.projectRepo.GetByID(ctx, projectID)
}

// UpdateProject updates a project
func (s *OrgService) UpdateProject(ctx context.Context, projectID, name string, settings []byte) (*domain.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if name != "" {
		project.Name = name
		project.Slug = slugify(name)
	}
	if settings != nil {
		project.Settings = settings
	}
	project.UpdatedAt = time.Now()

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// DeleteProject deletes a project
func (s *OrgService) DeleteProject(ctx context.Context, projectID string) error {
	return s.projectRepo.Delete(ctx, projectID)
}

// ListOrgProjects lists projects in an organization
func (s *OrgService) ListOrgProjects(ctx context.Context, orgID string, limit, offset int) ([]*domain.Project, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.projectRepo.ListByOrganizationID(ctx, orgID, limit, offset)
}

// ListUserProjects lists projects accessible to a user
func (s *OrgService) ListUserProjects(ctx context.Context, userID string, limit, offset int) ([]*domain.Project, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.projectRepo.ListByUserID(ctx, userID, limit, offset)
}

// GetUserProjectRole gets the user's effective role in a project
func (s *OrgService) GetUserProjectRole(ctx context.Context, projectID, userID string) (string, error) {
	return s.projectRepo.GetUserRole(ctx, projectID, userID)
}

// CanUserAccessProject checks if a user can access a project with a specific permission
func (s *OrgService) CanUserAccessProject(ctx context.Context, projectID, userID string, perm domain.Permission) (bool, error) {
	role, err := s.projectRepo.GetUserRole(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return domain.HasPermission(role, perm), nil
}

// RequestPasswordReset creates a password reset token
func (s *OrgService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		return "", nil
	}

	token, tokenHash, err := generateToken()
	if err != nil {
		s.logger.Error("failed to generate reset token", zap.Error(err))
		return "", errors.New("failed to create reset token")
	}

	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour), // Token valid for 1 hour
		CreatedAt: time.Now(),
	}

	if err := s.orgRepo.CreatePasswordResetToken(ctx, resetToken); err != nil {
		s.logger.Error("failed to save reset token", zap.Error(err))
		return "", errors.New("failed to create reset token")
	}

	return token, nil
}

// ResetPassword resets a user's password using a reset token
func (s *OrgService) ResetPassword(ctx context.Context, token, newPassword string) error {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	resetToken, err := s.orgRepo.GetPasswordResetToken(ctx, tokenHash)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}

	if resetToken.IsExpired() {
		return errors.New("reset token has expired")
	}

	if resetToken.IsUsed() {
		return errors.New("reset token has already been used")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return errors.New("failed to reset password")
	}

	// Get user and update password
	user, err := s.userRepo.GetByID(ctx, resetToken.UserID.String())
	if err != nil {
		return errors.New("user not found")
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return errors.New("failed to update password")
	}

	// Mark token as used
	if err := s.orgRepo.MarkPasswordResetTokenUsed(ctx, resetToken.ID); err != nil {
		s.logger.Error("failed to mark token as used", zap.Error(err))
	}

	// Invalidate all other reset tokens for this user
	if err := s.orgRepo.InvalidatePasswordResetTokens(ctx, resetToken.UserID); err != nil {
		s.logger.Error("failed to invalidate other tokens", zap.Error(err))
	}

	return nil
}

// CreateSession creates a new user session
func (s *OrgService) CreateSession(ctx context.Context, userID, userAgent, ipAddress string, expiry time.Duration) (string, *domain.UserSession, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", nil, errors.New("invalid user ID")
	}

	token, tokenHash, err := generateToken()
	if err != nil {
		return "", nil, errors.New("failed to create session")
	}

	session := &domain.UserSession{
		ID:           uuid.New(),
		UserID:       userUUID,
		TokenHash:    tokenHash,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		LastActiveAt: time.Now(),
		ExpiresAt:    time.Now().Add(expiry),
		CreatedAt:    time.Now(),
	}

	if err := s.orgRepo.CreateSession(ctx, session); err != nil {
		s.logger.Error("failed to create session", zap.Error(err))
		return "", nil, errors.New("failed to create session")
	}

	return token, session, nil
}

// ValidateSession validates a session token and returns the session
func (s *OrgService) ValidateSession(ctx context.Context, token string) (*domain.UserSession, error) {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	session, err := s.orgRepo.GetSession(ctx, tokenHash)
	if err != nil {
		return nil, errors.New("invalid session")
	}

	if !session.IsValid() {
		return nil, errors.New("session expired or revoked")
	}

	// Update last active time
	if err := s.orgRepo.UpdateSessionActivity(ctx, session.ID); err != nil {
		s.logger.Error("failed to update session activity", zap.Error(err))
	}

	return session, nil
}

// RevokeSession revokes a user session
func (s *OrgService) RevokeSession(ctx context.Context, sessionID string) error {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return errors.New("invalid session ID")
	}
	return s.orgRepo.RevokeSession(ctx, sessionUUID)
}

// RevokeAllUserSessions revokes all sessions for a user
func (s *OrgService) RevokeAllUserSessions(ctx context.Context, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}
	return s.orgRepo.RevokeAllUserSessions(ctx, userUUID)
}

// ListUserSessions lists active sessions for a user
func (s *OrgService) ListUserSessions(ctx context.Context, userID string, limit, offset int) ([]*domain.UserSession, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	return s.orgRepo.ListUserSessions(ctx, userID, limit, offset)
}

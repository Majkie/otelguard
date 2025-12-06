package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo   *postgres.UserRepository
	apiKeyRepo *postgres.APIKeyRepository
	logger     *zap.Logger
	bcryptCost int
	apiKeySalt string
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo *postgres.UserRepository, apiKeyRepo *postgres.APIKeyRepository, logger *zap.Logger, bcryptCost int, apiKeySalt string) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		apiKeyRepo: apiKeyRepo,
		logger:     logger,
		bcryptCost: bcryptCost,
		apiKeySalt: apiKeySalt,
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, email, password, name string) (*domain.User, error) {
	// Check if user already exists
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, errors.New("failed to create account")
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Name:         name,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, errors.New("failed to create account")
	}

	return user, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// UpdateProfile updates a user's profile
func (s *AuthService) UpdateProfile(ctx context.Context, userID, name, avatarURL string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if name != "" {
		user.Name = name
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return errors.New("failed to update password")
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	return s.userRepo.Update(ctx, user)
}

// generateAPIKey generates a secure random API key
func (s *AuthService) generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "otg_" + hex.EncodeToString(bytes), nil
}

// hashAPIKey creates a hash of the API key for storage
// Uses SHA256 first to keep within bcrypt's 72-byte limit
func (s *AuthService) hashAPIKey(key string) (string, error) {
	// First hash with SHA256 to reduce length and add salt
	h := sha256.New()
	h.Write([]byte(key))
	h.Write([]byte(s.apiKeySalt))
	sha256Hash := h.Sum(nil)

	// Then use bcrypt on the SHA256 hash
	hash, err := bcrypt.GenerateFromPassword(sha256Hash, s.bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CreateAPIKey creates a new API key for a project
func (s *AuthService) CreateAPIKey(ctx context.Context, projectID uuid.UUID, name string, scopes []string, expiresAt *time.Time) (*domain.APIKey, string, error) {
	// Generate raw API key
	rawKey, err := s.generateAPIKey()
	if err != nil {
		s.logger.Error("failed to generate API key", zap.Error(err))
		return nil, "", errors.New("failed to generate API key")
	}

	// Hash the key for storage
	keyHash, err := s.hashAPIKey(rawKey)
	if err != nil {
		s.logger.Error("failed to hash API key", zap.Error(err))
		return nil, "", errors.New("failed to create API key")
	}

	// Create API key entity
	apiKey := &domain.APIKey{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: rawKey[:10], // Store first 10 chars for display (otg_ + 6 chars)
		Scopes:    scopes,
		CreatedAt: time.Now(),
	}

	if expiresAt != nil {
		apiKey.ExpiresAt.Valid = true
		apiKey.ExpiresAt.Time = *expiresAt
	}

	// Save to database
	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		s.logger.Error("failed to save API key", zap.Error(err))
		return nil, "", errors.New("failed to create API key")
	}

	return apiKey, rawKey, nil
}

// ListAPIKeys returns all API keys for a project
func (s *AuthService) ListAPIKeys(ctx context.Context, projectID uuid.UUID) ([]*domain.APIKey, error) {
	return s.apiKeyRepo.ListByProjectID(ctx, projectID)
}

// RevokeAPIKey deletes an API key
func (s *AuthService) RevokeAPIKey(ctx context.Context, keyID uuid.UUID) error {
	return s.apiKeyRepo.Delete(ctx, keyID)
}

// ValidateAPIKey validates an API key and returns the associated project
func (s *AuthService) ValidateAPIKey(ctx context.Context, rawKey string) (*domain.APIKey, error) {
	// Hash the provided key
	keyHash, err := s.hashAPIKey(rawKey)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Look up by hash
	apiKey, err := s.apiKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Check if expired
	if apiKey.ExpiresAt.Valid && apiKey.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("API key has expired")
	}

	// Update last used timestamp (async, don't wait for it)
	go func() {
		if err := s.apiKeyRepo.UpdateLastUsed(context.Background(), apiKey.ID); err != nil {
			s.logger.Warn("failed to update API key last used time", zap.Error(err))
		}
	}()

	return apiKey, nil
}

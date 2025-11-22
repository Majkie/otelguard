package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Organization represents a top-level organization
type Organization struct {
	ID        uuid.UUID      `db:"id" json:"id"`
	Name      string         `db:"name" json:"name"`
	Slug      string         `db:"slug" json:"slug"`
	CreatedAt time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time      `db:"updated_at" json:"updatedAt"`
	DeletedAt sql.NullTime   `db:"deleted_at" json:"-"`
}

// Project represents a project within an organization
type Project struct {
	ID             uuid.UUID      `db:"id" json:"id"`
	OrganizationID uuid.UUID      `db:"organization_id" json:"organizationId"`
	Name           string         `db:"name" json:"name"`
	Slug           string         `db:"slug" json:"slug"`
	Settings       []byte         `db:"settings" json:"settings,omitempty"`
	CreatedAt      time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time      `db:"updated_at" json:"updatedAt"`
	DeletedAt      sql.NullTime   `db:"deleted_at" json:"-"`
}

// User represents a user in the system
type User struct {
	ID           uuid.UUID    `db:"id" json:"id"`
	Email        string       `db:"email" json:"email"`
	PasswordHash string       `db:"password_hash" json:"-"`
	Name         string       `db:"name" json:"name"`
	AvatarURL    string       `db:"avatar_url" json:"avatarUrl,omitempty"`
	CreatedAt    time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time    `db:"updated_at" json:"updatedAt"`
	DeletedAt    sql.NullTime `db:"deleted_at" json:"-"`
}

// OrganizationMember represents a user's membership in an organization
type OrganizationMember struct {
	ID             uuid.UUID `db:"id" json:"id"`
	OrganizationID uuid.UUID `db:"organization_id" json:"organizationId"`
	UserID         uuid.UUID `db:"user_id" json:"userId"`
	Role           string    `db:"role" json:"role"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
}

// APIKey represents an API key for authentication
type APIKey struct {
	ID         uuid.UUID    `db:"id" json:"id"`
	ProjectID  uuid.UUID    `db:"project_id" json:"projectId"`
	Name       string       `db:"name" json:"name"`
	KeyHash    string       `db:"key_hash" json:"-"`
	KeyPrefix  string       `db:"key_prefix" json:"keyPrefix"`
	Scopes     []string     `db:"scopes" json:"scopes"`
	LastUsedAt sql.NullTime `db:"last_used_at" json:"lastUsedAt,omitempty"`
	ExpiresAt  sql.NullTime `db:"expires_at" json:"expiresAt,omitempty"`
	CreatedAt  time.Time    `db:"created_at" json:"createdAt"`
}

// Prompt represents a prompt template
type Prompt struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	ProjectID   uuid.UUID    `db:"project_id" json:"projectId"`
	Name        string       `db:"name" json:"name"`
	Description string       `db:"description" json:"description,omitempty"`
	Tags        []string     `db:"tags" json:"tags,omitempty"`
	CreatedAt   time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time    `db:"updated_at" json:"updatedAt"`
	DeletedAt   sql.NullTime `db:"deleted_at" json:"-"`
}

// PromptVersion represents a version of a prompt
type PromptVersion struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	PromptID  uuid.UUID  `db:"prompt_id" json:"promptId"`
	Version   int        `db:"version" json:"version"`
	Content   string     `db:"content" json:"content"`
	Config    []byte     `db:"config" json:"config,omitempty"`
	Labels    []string   `db:"labels" json:"labels,omitempty"`
	CreatedBy *uuid.UUID `db:"created_by" json:"createdBy,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"createdAt"`
}

// GuardrailPolicy represents a guardrail policy
type GuardrailPolicy struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	ProjectID   uuid.UUID  `db:"project_id" json:"projectId"`
	Name        string     `db:"name" json:"name"`
	Description string     `db:"description" json:"description,omitempty"`
	Enabled     bool       `db:"enabled" json:"enabled"`
	Priority    int        `db:"priority" json:"priority"`
	Triggers    []byte     `db:"triggers" json:"triggers"`
	CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updatedAt"`
}

// GuardrailRule represents a rule within a guardrail policy
type GuardrailRule struct {
	ID           uuid.UUID `db:"id" json:"id"`
	PolicyID     uuid.UUID `db:"policy_id" json:"policyId"`
	Type         string    `db:"type" json:"type"`
	Config       []byte    `db:"config" json:"config"`
	Action       string    `db:"action" json:"action"`
	ActionConfig []byte    `db:"action_config" json:"actionConfig,omitempty"`
	OrderIndex   int       `db:"order_index" json:"orderIndex"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
}

// Roles
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// API Key Scopes
const (
	ScopeTraceWrite = "trace:write"
	ScopeTraceRead  = "trace:read"
	ScopePromptRead = "prompt:read"
	ScopeAll        = "*"
)

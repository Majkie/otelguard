package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Organization represents a top-level organization
type Organization struct {
	ID        uuid.UUID    `db:"id" json:"id"`
	Name      string       `db:"name" json:"name"`
	Slug      string       `db:"slug" json:"slug"`
	CreatedAt time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time    `db:"updated_at" json:"updatedAt"`
	DeletedAt sql.NullTime `db:"deleted_at" json:"-"`
}

// Project represents a project within an organization
type Project struct {
	ID             uuid.UUID    `db:"id" json:"id"`
	OrganizationID uuid.UUID    `db:"organization_id" json:"organizationId"`
	Name           string       `db:"name" json:"name"`
	Slug           string       `db:"slug" json:"slug"`
	Settings       []byte       `db:"settings" json:"settings,omitempty"`
	CreatedAt      time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time    `db:"updated_at" json:"updatedAt"`
	DeletedAt      sql.NullTime `db:"deleted_at" json:"-"`
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
	ID          uuid.UUID `db:"id" json:"id"`
	ProjectID   uuid.UUID `db:"project_id" json:"projectId"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description,omitempty"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	Priority    int       `db:"priority" json:"priority"`
	Triggers    []byte    `db:"triggers" json:"triggers"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
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

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        uuid.UUID    `db:"id" json:"id"`
	UserID    uuid.UUID    `db:"user_id" json:"userId"`
	TokenHash string       `db:"token_hash" json:"-"`
	ExpiresAt time.Time    `db:"expires_at" json:"expiresAt"`
	UsedAt    sql.NullTime `db:"used_at" json:"-"`
	CreatedAt time.Time    `db:"created_at" json:"createdAt"`
}

// IsExpired returns true if the token has expired
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has been used
func (t *PasswordResetToken) IsUsed() bool {
	return t.UsedAt.Valid
}

// UserSession represents a user session
type UserSession struct {
	ID           uuid.UUID    `db:"id" json:"id"`
	UserID       uuid.UUID    `db:"user_id" json:"userId"`
	TokenHash    string       `db:"token_hash" json:"-"`
	UserAgent    string       `db:"user_agent" json:"userAgent,omitempty"`
	IPAddress    string       `db:"ip_address" json:"ipAddress,omitempty"`
	LastActiveAt time.Time    `db:"last_active_at" json:"lastActiveAt"`
	ExpiresAt    time.Time    `db:"expires_at" json:"expiresAt"`
	RevokedAt    sql.NullTime `db:"revoked_at" json:"-"`
	CreatedAt    time.Time    `db:"created_at" json:"createdAt"`
}

// IsExpired returns true if the session has expired
func (s *UserSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsRevoked returns true if the session has been revoked
func (s *UserSession) IsRevoked() bool {
	return s.RevokedAt.Valid
}

// IsValid returns true if the session is valid (not expired and not revoked)
func (s *UserSession) IsValid() bool {
	return !s.IsExpired() && !s.IsRevoked()
}

// ProjectMember represents a user's membership in a project
type ProjectMember struct {
	ID        uuid.UUID `db:"id" json:"id"`
	ProjectID uuid.UUID `db:"project_id" json:"projectId"`
	UserID    uuid.UUID `db:"user_id" json:"userId"`
	Role      string    `db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// Invitation represents an invitation to join an organization or project
type Invitation struct {
	ID             uuid.UUID    `db:"id" json:"id"`
	OrganizationID *uuid.UUID   `db:"organization_id" json:"organizationId,omitempty"`
	ProjectID      *uuid.UUID   `db:"project_id" json:"projectId,omitempty"`
	Email          string       `db:"email" json:"email"`
	Role           string       `db:"role" json:"role"`
	TokenHash      string       `db:"token_hash" json:"-"`
	InvitedBy      uuid.UUID    `db:"invited_by" json:"invitedBy"`
	AcceptedAt     sql.NullTime `db:"accepted_at" json:"acceptedAt,omitempty"`
	ExpiresAt      time.Time    `db:"expires_at" json:"expiresAt"`
	CreatedAt      time.Time    `db:"created_at" json:"createdAt"`
}

// IsExpired returns true if the invitation has expired
func (i *Invitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsAccepted returns true if the invitation has been accepted
func (i *Invitation) IsAccepted() bool {
	return i.AcceptedAt.Valid
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

// Permission constants for RBAC
type Permission string

const (
	PermissionOrgRead          Permission = "org:read"
	PermissionOrgWrite         Permission = "org:write"
	PermissionOrgDelete        Permission = "org:delete"
	PermissionOrgManageMembers Permission = "org:manage_members"
	PermissionProjectRead      Permission = "project:read"
	PermissionProjectWrite     Permission = "project:write"
	PermissionProjectDelete    Permission = "project:delete"
	PermissionProjectManageAPI Permission = "project:manage_api_keys"
	PermissionTraceRead        Permission = "trace:read"
	PermissionTraceWrite       Permission = "trace:write"
	PermissionPromptRead       Permission = "prompt:read"
	PermissionPromptWrite      Permission = "prompt:write"
	PermissionGuardrailRead    Permission = "guardrail:read"
	PermissionGuardrailWrite   Permission = "guardrail:write"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[string][]Permission{
	RoleOwner: {
		PermissionOrgRead, PermissionOrgWrite, PermissionOrgDelete, PermissionOrgManageMembers,
		PermissionProjectRead, PermissionProjectWrite, PermissionProjectDelete, PermissionProjectManageAPI,
		PermissionTraceRead, PermissionTraceWrite,
		PermissionPromptRead, PermissionPromptWrite,
		PermissionGuardrailRead, PermissionGuardrailWrite,
	},
	RoleAdmin: {
		PermissionOrgRead, PermissionOrgWrite, PermissionOrgManageMembers,
		PermissionProjectRead, PermissionProjectWrite, PermissionProjectManageAPI,
		PermissionTraceRead, PermissionTraceWrite,
		PermissionPromptRead, PermissionPromptWrite,
		PermissionGuardrailRead, PermissionGuardrailWrite,
	},
	RoleMember: {
		PermissionOrgRead,
		PermissionProjectRead, PermissionProjectWrite,
		PermissionTraceRead, PermissionTraceWrite,
		PermissionPromptRead, PermissionPromptWrite,
		PermissionGuardrailRead, PermissionGuardrailWrite,
	},
	RoleViewer: {
		PermissionOrgRead,
		PermissionProjectRead,
		PermissionTraceRead,
		PermissionPromptRead,
		PermissionGuardrailRead,
	},
}

// HasPermission checks if a role has a specific permission
func HasPermission(role string, perm Permission) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// LLM Provider Types
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGoogle    = "google"
	ProviderOllama    = "ollama"
)

// LLMModel represents an LLM model configuration
type LLMModel struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	ModelID      string   `json:"modelId"`
	ContextSize  int      `json:"contextSize"`
	Pricing      Pricing  `json:"pricing"`
	Capabilities []string `json:"capabilities"`
}

// Pricing represents cost information for a model
type Pricing struct {
	InputTokens  float64 `json:"inputTokens"`  // Cost per 1K input tokens
	OutputTokens float64 `json:"outputTokens"` // Cost per 1K output tokens
	Currency     string  `json:"currency"`
}

// LLMRequest represents a request to execute an LLM
type LLMRequest struct {
	Provider    string                 `json:"provider"`
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt"`
	MaxTokens   int                    `json:"maxTokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// LLMResponse represents a response from an LLM
type LLMResponse struct {
	Text         string     `json:"text"`
	Usage        TokenUsage `json:"usage"`
	FinishReason string     `json:"finishReason,omitempty"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// PlaygroundExecution represents a playground execution record
type PlaygroundExecution struct {
	ID            uuid.UUID     `json:"id"`
	ProjectID     uuid.UUID     `json:"projectId"`
	UserID        *uuid.UUID    `json:"userId,omitempty"`
	PromptID      *uuid.UUID    `json:"promptId,omitempty"`
	Request       LLMRequest    `json:"request"`
	Response      *LLMResponse  `json:"response,omitempty"`
	Error         string        `json:"error,omitempty"`
	ExecutionTime time.Duration `json:"executionTime"`
	Cost          float64       `json:"cost"`
	CreatedAt     time.Time     `json:"createdAt"`
}

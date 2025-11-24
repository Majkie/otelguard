package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// MustParseUUID parses a UUID string and panics if invalid
func MustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}

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
	ID             uuid.UUID `db:"id" json:"id"`
	ProjectID      uuid.UUID `db:"project_id" json:"projectId"`
	Name           string    `db:"name" json:"name"`
	Description    string    `db:"description" json:"description,omitempty"`
	Enabled        bool      `db:"enabled" json:"enabled"`
	Priority       int       `db:"priority" json:"priority"`
	Triggers       []byte    `db:"triggers" json:"triggers"`
	CurrentVersion int       `db:"current_version" json:"currentVersion"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}

// GuardrailPolicyVersion represents a version snapshot of a guardrail policy
type GuardrailPolicyVersion struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	PolicyID    uuid.UUID    `db:"policy_id" json:"policyId"`
	Version     int          `db:"version" json:"version"`
	Name        string       `db:"name" json:"name"`
	Description string       `db:"description" json:"description,omitempty"`
	Enabled     bool         `db:"enabled" json:"enabled"`
	Priority    int          `db:"priority" json:"priority"`
	Triggers    []byte       `db:"triggers" json:"triggers"`
	Rules       []byte       `db:"rules" json:"rules"` // Snapshot of rules at this version
	ChangeNotes string       `db:"change_notes" json:"changeNotes,omitempty"`
	CreatedBy   uuid.UUID    `db:"created_by" json:"createdBy"`
	CreatedAt   time.Time    `db:"created_at" json:"createdAt"`
	DeletedAt   sql.NullTime `db:"deleted_at" json:"-"`
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

// ScoreConfig represents a scoring configuration for annotations
type ScoreConfig struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	ProjectID   uuid.UUID       `db:"project_id" json:"projectId"`
	Name        string          `db:"name" json:"name"`
	DataType    string          `db:"data_type" json:"dataType"`
	Description string          `db:"description" json:"description,omitempty"`
	MinValue    sql.NullFloat64 `db:"min_value" json:"minValue,omitempty"`
	MaxValue    sql.NullFloat64 `db:"max_value" json:"maxValue,omitempty"`
	Categories  []string        `db:"categories" json:"categories"`
	CreatedAt   time.Time       `db:"created_at" json:"createdAt"`
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

// AnnotationQueue represents a queue of items to be annotated
type AnnotationQueue struct {
	ID                    uuid.UUID    `db:"id" json:"id"`
	ProjectID             uuid.UUID    `db:"project_id" json:"projectId"`
	Name                  string       `db:"name" json:"name"`
	Description           string       `db:"description" json:"description,omitempty"`
	ScoreConfigs          []byte       `db:"score_configs" json:"scoreConfigs"`
	Config                []byte       `db:"config" json:"config"`
	ItemSource            string       `db:"item_source" json:"itemSource"`
	ItemSourceConfig      []byte       `db:"item_source_config" json:"itemSourceConfig"`
	AssignmentStrategy    string       `db:"assignment_strategy" json:"assignmentStrategy"`
	MaxAnnotationsPerItem int          `db:"max_annotations_per_item" json:"maxAnnotationsPerItem"`
	Instructions          string       `db:"instructions" json:"instructions,omitempty"`
	IsActive              bool         `db:"is_active" json:"isActive"`
	CreatedAt             time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt             time.Time    `db:"updated_at" json:"updatedAt"`
	DeletedAt             sql.NullTime `db:"deleted_at" json:"-"`
}

// AnnotationQueueItem represents an item in an annotation queue
type AnnotationQueueItem struct {
	ID             uuid.UUID `db:"id" json:"id"`
	QueueID        uuid.UUID `db:"queue_id" json:"queueId"`
	ItemType       string    `db:"item_type" json:"itemType"`
	ItemID         string    `db:"item_id" json:"itemId"`
	ItemData       []byte    `db:"item_data" json:"itemData,omitempty"`
	Metadata       []byte    `db:"metadata" json:"metadata"`
	Priority       int       `db:"priority" json:"priority"`
	MaxAnnotations int       `db:"max_annotations" json:"maxAnnotations"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}

// AnnotationAssignment represents the assignment of a queue item to a user
type AnnotationAssignment struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	QueueItemID uuid.UUID    `db:"queue_item_id" json:"queueItemId"`
	UserID      uuid.UUID    `db:"user_id" json:"userId"`
	Status      string       `db:"status" json:"status"`
	AssignedAt  time.Time    `db:"assigned_at" json:"assignedAt"`
	StartedAt   sql.NullTime `db:"started_at" json:"startedAt,omitempty"`
	CompletedAt sql.NullTime `db:"completed_at" json:"completedAt,omitempty"`
	SkippedAt   sql.NullTime `db:"skipped_at" json:"skippedAt,omitempty"`
	Notes       string       `db:"notes" json:"notes,omitempty"`
	CreatedAt   time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time    `db:"updated_at" json:"updatedAt"`
}

// Annotation represents a completed human annotation
type Annotation struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	AssignmentID    uuid.UUID       `db:"assignment_id" json:"assignmentId"`
	QueueID         uuid.UUID       `db:"queue_id" json:"queueId"`
	QueueItemID     uuid.UUID       `db:"queue_item_id" json:"queueItemId"`
	UserID          uuid.UUID       `db:"user_id" json:"userId"`
	Scores          []byte          `db:"scores" json:"scores"`
	Labels          []string        `db:"labels" json:"labels"`
	Notes           string          `db:"notes" json:"notes,omitempty"`
	ConfidenceScore sql.NullFloat64 `db:"confidence_score" json:"confidenceScore,omitempty"`
	AnnotationTime  sql.NullString  `db:"annotation_time" json:"annotationTime,omitempty"`
	CreatedAt       time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updatedAt"`
}

// InterAnnotatorAgreement represents agreement metrics between annotators
type InterAnnotatorAgreement struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	QueueID         uuid.UUID       `db:"queue_id" json:"queueId"`
	QueueItemID     uuid.UUID       `db:"queue_item_id" json:"queueItemId"`
	ScoreConfigName string          `db:"score_config_name" json:"scoreConfigName"`
	AgreementType   string          `db:"agreement_type" json:"agreementType"`
	AgreementValue  sql.NullFloat64 `db:"agreement_value" json:"agreementValue,omitempty"`
	AnnotatorCount  int             `db:"annotator_count" json:"annotatorCount"`
	CalculatedAt    time.Time       `db:"calculated_at" json:"calculatedAt"`
}

// AnnotationQueueCreate represents data for creating a new annotation queue
type AnnotationQueueCreate struct {
	ProjectID             uuid.UUID              `json:"projectId" validate:"required"`
	Name                  string                 `json:"name" validate:"required,min=1,max=255"`
	Description           string                 `json:"description,omitempty"`
	ScoreConfigs          []ScoreConfig          `json:"scoreConfigs,omitempty"`
	Config                map[string]interface{} `json:"config,omitempty"`
	ItemSource            string                 `json:"itemSource,omitempty"`
	ItemSourceConfig      map[string]interface{} `json:"itemSourceConfig,omitempty"`
	AssignmentStrategy    string                 `json:"assignmentStrategy,omitempty"`
	MaxAnnotationsPerItem int                    `json:"maxAnnotationsPerItem,omitempty"`
	Instructions          string                 `json:"instructions,omitempty"`
}

// AnnotationQueueUpdate represents data for updating an annotation queue
type AnnotationQueueUpdate struct {
	Name                  *string                 `json:"name,omitempty"`
	Description           *string                 `json:"description,omitempty"`
	ScoreConfigs          *[]ScoreConfig          `json:"scoreConfigs,omitempty"`
	Config                *map[string]interface{} `json:"config,omitempty"`
	ItemSource            *string                 `json:"itemSource,omitempty"`
	ItemSourceConfig      *map[string]interface{} `json:"itemSourceConfig,omitempty"`
	AssignmentStrategy    *string                 `json:"assignmentStrategy,omitempty"`
	MaxAnnotationsPerItem *int                    `json:"maxAnnotationsPerItem,omitempty"`
	Instructions          *string                 `json:"instructions,omitempty"`
	IsActive              *bool                   `json:"isActive,omitempty"`
}

// AnnotationQueueItemCreate represents data for creating a new queue item
type AnnotationQueueItemCreate struct {
	QueueID        uuid.UUID              `json:"queueId" validate:"required"`
	ItemType       string                 `json:"itemType" validate:"required"`
	ItemID         string                 `json:"itemId" validate:"required"`
	ItemData       map[string]interface{} `json:"itemData,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Priority       int                    `json:"priority,omitempty"`
	MaxAnnotations int                    `json:"maxAnnotations,omitempty"`
}

// AnnotationAssignmentCreate represents data for creating a new assignment
type AnnotationAssignmentCreate struct {
	QueueItemID uuid.UUID `json:"queueItemId" validate:"required"`
	UserID      uuid.UUID `json:"userId" validate:"required"`
}

// AnnotationCreate represents data for creating a new annotation
type AnnotationCreate struct {
	AssignmentID    uuid.UUID              `json:"assignmentId" validate:"required"`
	Scores          map[string]interface{} `json:"scores,omitempty"`
	Labels          []string               `json:"labels,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	ConfidenceScore *float64               `json:"confidenceScore,omitempty"`
	AnnotationTime  *string                `json:"annotationTime,omitempty"`
}

// AnnotationAssignmentUpdate represents data for updating an assignment status
type AnnotationAssignmentUpdate struct {
	Status *string `json:"status,omitempty"`
	Notes  *string `json:"notes,omitempty"`
}

// FeedbackScoreMapping represents how feedback translates to evaluation scores
type FeedbackScoreMapping struct {
	ID          uuid.UUID `db:"id" json:"id"`
	ProjectID   uuid.UUID `db:"project_id" json:"projectId"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description,omitempty"`
	ItemType    string    `db:"item_type" json:"itemType"` // 'trace', 'session', 'span', 'prompt'
	Enabled     bool      `db:"enabled" json:"enabled"`
	Config      []byte    `db:"config" json:"config"` // JSON configuration for mapping rules
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// FeedbackScoreMappingConfig represents the configuration for feedback to score mapping
type FeedbackScoreMappingConfig struct {
	// Thumbs up/down to score mappings
	ThumbsUpScore   *ScoreMapping `json:"thumbsUpScore,omitempty"`
	ThumbsDownScore *ScoreMapping `json:"thumbsDownScore,omitempty"`

	// Rating to score mappings (1-5 stars)
	RatingScores map[int]*ScoreMapping `json:"ratingScores,omitempty"`

	// Comment analysis (future: sentiment analysis, keyword matching)
	CommentAnalysis *CommentAnalysisConfig `json:"commentAnalysis,omitempty"`
}

// ScoreMapping defines how feedback maps to a score
type ScoreMapping struct {
	ScoreConfigID uuid.UUID `json:"scoreConfigId"`
	Value         float64   `json:"value"`
	Comment       string    `json:"comment,omitempty"`
}

// CommentAnalysisConfig defines how to analyze comments for scoring
type CommentAnalysisConfig struct {
	Enabled         bool             `json:"enabled"`
	SentimentScore  *ScoreMapping    `json:"sentimentScore,omitempty"`
	KeywordMappings []KeywordMapping `json:"keywordMappings,omitempty"`
}

// KeywordMapping maps keywords to scores
type KeywordMapping struct {
	Keywords []string      `json:"keywords"`
	Mapping  *ScoreMapping `json:"mapping"`
}

// FeedbackScoreMappingCreate represents data for creating a feedback score mapping
type FeedbackScoreMappingCreate struct {
	ProjectID   uuid.UUID                   `json:"projectId" validate:"required"`
	Name        string                      `json:"name" validate:"required,min=1,max=255"`
	Description string                      `json:"description,omitempty"`
	ItemType    string                      `json:"itemType" validate:"required,oneof=trace session span prompt"`
	Enabled     *bool                       `json:"enabled,omitempty"`
	Config      *FeedbackScoreMappingConfig `json:"config,omitempty"`
}

// FeedbackScoreMappingUpdate represents data for updating a feedback score mapping
type FeedbackScoreMappingUpdate struct {
	Name        *string                     `json:"name,omitempty"`
	Description *string                     `json:"description,omitempty"`
	Enabled     *bool                       `json:"enabled,omitempty"`
	Config      *FeedbackScoreMappingConfig `json:"config,omitempty"`
}

// UserFeedback represents user feedback on traces, sessions, or other items
type UserFeedback struct {
	ID        uuid.UUID     `db:"id" json:"id"`
	ProjectID uuid.UUID     `db:"project_id" json:"projectId"`
	UserID    *uuid.UUID    `db:"user_id" json:"userId,omitempty"`
	SessionID *string       `db:"session_id" json:"sessionId,omitempty"`
	TraceID   *string       `db:"trace_id" json:"traceId,omitempty"`
	SpanID    *string       `db:"span_id" json:"spanId,omitempty"`
	ItemType  string        `db:"item_type" json:"itemType"` // 'trace', 'session', 'span', 'prompt'
	ItemID    string        `db:"item_id" json:"itemId"`
	ThumbsUp  sql.NullBool  `db:"thumbs_up" json:"thumbsUp,omitempty"`
	Rating    sql.NullInt32 `db:"rating" json:"rating,omitempty"` // 1-5 stars
	Comment   string        `db:"comment" json:"comment,omitempty"`
	Metadata  []byte        `db:"metadata" json:"metadata,omitempty"`
	UserAgent string        `db:"user_agent" json:"userAgent,omitempty"`
	IPAddress string        `db:"ip_address" json:"ipAddress,omitempty"`
	CreatedAt time.Time     `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time     `db:"updated_at" json:"updatedAt"`
}

// UserFeedbackCreate represents data for creating user feedback
type UserFeedbackCreate struct {
	ProjectID uuid.UUID              `json:"projectId" validate:"required"`
	UserID    *uuid.UUID             `json:"userId,omitempty"`
	SessionID *string                `json:"sessionId,omitempty"`
	TraceID   *string                `json:"traceId,omitempty"`
	SpanID    *string                `json:"spanId,omitempty"`
	ItemType  string                 `json:"itemType" validate:"required,oneof=trace session span prompt"`
	ItemID    string                 `json:"itemId" validate:"required"`
	ThumbsUp  *bool                  `json:"thumbsUp,omitempty"`
	Rating    *int                   `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	Comment   string                 `json:"comment,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// UserFeedbackUpdate represents data for updating user feedback
type UserFeedbackUpdate struct {
	ThumbsUp *bool                   `json:"thumbsUp,omitempty"`
	Rating   *int                    `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	Comment  *string                 `json:"comment,omitempty"`
	Metadata *map[string]interface{} `json:"metadata,omitempty"`
}

// FeedbackFilter represents filters for querying feedback
type FeedbackFilter struct {
	ProjectID string    `json:"projectId,omitempty"`
	UserID    string    `json:"userId,omitempty"`
	ItemType  string    `json:"itemType,omitempty"`
	ItemID    string    `json:"itemId,omitempty"`
	TraceID   string    `json:"traceId,omitempty"`
	SessionID string    `json:"sessionId,omitempty"`
	ThumbsUp  *bool     `json:"thumbsUp,omitempty"`
	Rating    *int      `json:"rating,omitempty"`
	StartDate time.Time `json:"startDate,omitempty"`
	EndDate   time.Time `json:"endDate,omitempty"`
	OrderBy   string    `json:"orderBy,omitempty"`
	OrderDesc bool      `json:"orderDesc,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

// FeedbackAnalytics represents aggregated feedback analytics
type FeedbackAnalytics struct {
	ProjectID       uuid.UUID       `json:"projectId"`
	ItemType        string          `json:"itemType"`
	TotalFeedback   int64           `json:"totalFeedback"`
	ThumbsUpCount   int64           `json:"thumbsUpCount"`
	ThumbsDownCount int64           `json:"thumbsDownCount"`
	AverageRating   sql.NullFloat64 `json:"averageRating"`
	RatingCounts    map[int]int64   `json:"ratingCounts"`
	CommentCount    int64           `json:"commentCount"`
	DateRange       string          `json:"dateRange"`
	Trends          []FeedbackTrend `json:"trends,omitempty"`
}

// FeedbackTrend represents feedback trends over time
type FeedbackTrend struct {
	Date          string  `json:"date"`
	TotalFeedback int64   `json:"totalFeedback"`
	ThumbsUpRate  float64 `json:"thumbsUpRate"`
	AverageRating float64 `json:"averageRating"`
	CommentCount  int64   `json:"commentCount"`
}

// Dataset represents a collection of test cases for evaluation
type Dataset struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	ProjectID   uuid.UUID    `db:"project_id" json:"projectId"`
	Name        string       `db:"name" json:"name"`
	Description string       `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time    `db:"updated_at" json:"updatedAt"`
	DeletedAt   sql.NullTime `db:"deleted_at" json:"-"`
}

// DatasetItem represents a single test case within a dataset
type DatasetItem struct {
	ID             uuid.UUID `db:"id" json:"id"`
	DatasetID      uuid.UUID `db:"dataset_id" json:"datasetId"`
	Input          []byte    `db:"input" json:"input"`
	ExpectedOutput []byte    `db:"expected_output" json:"expectedOutput,omitempty"`
	Metadata       []byte    `db:"metadata" json:"metadata,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
}

// DatasetCreate represents data for creating a new dataset
type DatasetCreate struct {
	ProjectID   uuid.UUID `json:"projectId" validate:"required"`
	Name        string    `json:"name" validate:"required,min=1,max=255"`
	Description string    `json:"description,omitempty"`
}

// DatasetUpdate represents data for updating a dataset
type DatasetUpdate struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty"`
}

// DatasetItemCreate represents data for creating a dataset item
type DatasetItemCreate struct {
	DatasetID      uuid.UUID              `json:"datasetId" validate:"required"`
	Input          map[string]interface{} `json:"input" validate:"required"`
	ExpectedOutput map[string]interface{} `json:"expectedOutput,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// DatasetItemUpdate represents data for updating a dataset item
type DatasetItemUpdate struct {
	Input          *map[string]interface{} `json:"input,omitempty"`
	ExpectedOutput *map[string]interface{} `json:"expectedOutput,omitempty"`
	Metadata       *map[string]interface{} `json:"metadata,omitempty"`
}

// DatasetImport represents data for bulk importing dataset items
type DatasetImport struct {
	DatasetID uuid.UUID          `json:"datasetId" validate:"required"`
	Format    string             `json:"format" validate:"required,oneof=json csv"`
	Items     []DatasetItemInput `json:"items,omitempty"`
	Data      string             `json:"data,omitempty"` // CSV or JSON string
}

// DatasetItemInput represents a simplified dataset item for import
type DatasetItemInput struct {
	Input          map[string]interface{} `json:"input"`
	ExpectedOutput map[string]interface{} `json:"expectedOutput,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Experiment represents an evaluation experiment configuration
type Experiment struct {
	ID          uuid.UUID `db:"id" json:"id"`
	ProjectID   uuid.UUID `db:"project_id" json:"projectId"`
	DatasetID   uuid.UUID `db:"dataset_id" json:"datasetId"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description,omitempty"`
	Config      []byte    `db:"config" json:"config"` // Prompt version, model config, evaluators
	Status      string    `db:"status" json:"status"` // 'pending', 'running', 'completed', 'failed'
	CreatedBy   uuid.UUID `db:"created_by" json:"createdBy"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// ExperimentConfig represents the configuration for an experiment
type ExperimentConfig struct {
	PromptID      *uuid.UUID             `json:"promptId,omitempty"`
	PromptVersion *int                   `json:"promptVersion,omitempty"`
	Model         string                 `json:"model" validate:"required"`
	Provider      string                 `json:"provider" validate:"required"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Evaluators    []uuid.UUID            `json:"evaluators,omitempty"` // ScoreConfig IDs or Evaluator IDs
	Timeout       int                    `json:"timeout,omitempty"`    // seconds
}

// ExperimentRun represents a single execution of an experiment
type ExperimentRun struct {
	ID           uuid.UUID    `db:"id" json:"id"`
	ExperimentID uuid.UUID    `db:"experiment_id" json:"experimentId"`
	RunNumber    int          `db:"run_number" json:"runNumber"`
	Status       string       `db:"status" json:"status"` // 'pending', 'running', 'completed', 'failed'
	StartedAt    time.Time    `db:"started_at" json:"startedAt"`
	CompletedAt  sql.NullTime `db:"completed_at" json:"completedAt,omitempty"`
	TotalItems   int          `db:"total_items" json:"totalItems"`
	CompletedItems int        `db:"completed_items" json:"completedItems"`
	FailedItems  int          `db:"failed_items" json:"failedItems"`
	TotalCost    float64      `db:"total_cost" json:"totalCost"`
	TotalLatency int64        `db:"total_latency_ms" json:"totalLatencyMs"` // milliseconds
	Error        string       `db:"error" json:"error,omitempty"`
	CreatedAt    time.Time    `db:"created_at" json:"createdAt"`
}

// ExperimentResult represents the result of running an experiment on a dataset item
type ExperimentResult struct {
	ID            uuid.UUID `db:"id" json:"id"`
	RunID         uuid.UUID `db:"run_id" json:"runId"`
	DatasetItemID uuid.UUID `db:"dataset_item_id" json:"datasetItemId"`
	TraceID       *string   `db:"trace_id" json:"traceId,omitempty"`
	Output        []byte    `db:"output" json:"output,omitempty"`
	Scores        []byte    `db:"scores" json:"scores,omitempty"` // JSON map of score name to value
	LatencyMs     int64     `db:"latency_ms" json:"latencyMs"`
	TokensUsed    int       `db:"tokens_used" json:"tokensUsed"`
	Cost          float64   `db:"cost" json:"cost"`
	Status        string    `db:"status" json:"status"` // 'success', 'error'
	Error         string    `db:"error" json:"error,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
}

// ExperimentCreate represents data for creating a new experiment
type ExperimentCreate struct {
	ProjectID   uuid.UUID         `json:"projectId" validate:"required"`
	DatasetID   uuid.UUID         `json:"datasetId" validate:"required"`
	Name        string            `json:"name" validate:"required,min=1,max=255"`
	Description string            `json:"description,omitempty"`
	Config      *ExperimentConfig `json:"config" validate:"required"`
	CreatedBy   uuid.UUID         `json:"createdBy" validate:"required"`
}

// ExperimentExecute represents data for executing an experiment
type ExperimentExecute struct {
	ExperimentID uuid.UUID `json:"experimentId" validate:"required"`
	Async        bool      `json:"async,omitempty"` // Run in background
}

// ExperimentComparison represents comparison data between multiple experiment runs
type ExperimentComparison struct {
	RunIDs []uuid.UUID                   `json:"runIds"`
	Runs   []*ExperimentRun              `json:"runs"`
	Metrics map[string]*ComparisonMetrics `json:"metrics"`
}

// ComparisonMetrics represents aggregated metrics for experiment comparison
type ComparisonMetrics struct {
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	StdDev float64 `json:"stdDev"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	N      int     `json:"n"` // Sample size
}

// PairwiseComparison represents statistical comparison between two experiment runs
type PairwiseComparison struct {
	Run1ID         uuid.UUID `json:"run1Id"`
	Run2ID         uuid.UUID `json:"run2Id"`
	Run1Name       string    `json:"run1Name"`
	Run2Name       string    `json:"run2Name"`
	MetricName     string    `json:"metricName"`
	TStatistic     float64   `json:"tStatistic"`
	PValue         float64   `json:"pValue"`
	DegreesOfFreedom int     `json:"degreesOfFreedom"`
	SignificantAt05  bool    `json:"significantAt05"` // p < 0.05
	SignificantAt01  bool    `json:"significantAt01"` // p < 0.01
	MeanDifference   float64 `json:"meanDifference"`
	EffectSize       float64 `json:"effectSize"` // Cohen's d
}

// StatisticalComparison represents complete statistical analysis of experiment runs
type StatisticalComparison struct {
	ExperimentComparison
	PairwiseTests map[string][]*PairwiseComparison `json:"pairwiseTests"` // metric name -> comparisons
}

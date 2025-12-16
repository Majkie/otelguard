package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// EvaluatorType constants
const (
	EvaluatorTypeLLMJudge = "llm_judge"
	EvaluatorTypeCustom   = "custom"
)

// EvaluationJobStatus constants
const (
	EvaluationJobStatusPending   = "pending"
	EvaluationJobStatusRunning   = "running"
	EvaluationJobStatusCompleted = "completed"
	EvaluationJobStatusFailed    = "failed"
	EvaluationJobStatusCancelled = "cancelled"
)

// Evaluator represents an LLM-as-a-Judge evaluator configuration
type Evaluator struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	ProjectID   uuid.UUID    `db:"project_id" json:"projectId"`
	Name        string       `db:"name" json:"name"`
	Description string       `db:"description" json:"description,omitempty"`
	Type        string       `db:"type" json:"type"` // llm_judge, custom
	Provider    string       `db:"provider" json:"provider"`
	Model       string       `db:"model" json:"model"`
	Template    string       `db:"template" json:"template"`      // Evaluation prompt template
	Config      []byte       `db:"config" json:"config"`          // JSON config
	OutputType  string       `db:"output_type" json:"outputType"` // numeric, boolean, categorical
	MinValue    *float64     `json:"minValue,omitempty"`
	MaxValue    *float64     `json:"maxValue,omitempty"`
	Categories  []string     `db:"categories" json:"categories,omitempty"`
	Enabled     bool         `db:"enabled" json:"enabled"`
	CreatedAt   time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time    `db:"updated_at" json:"updatedAt"`
	DeletedAt   sql.NullTime `db:"deleted_at" json:"-"`
}

// EvaluatorConfig represents the JSON configuration for an evaluator
type EvaluatorConfig struct {
	// LLM settings
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"maxTokens,omitempty"`

	// Output parsing
	OutputFormat   string `json:"outputFormat,omitempty"`   // json, text, structured
	ScoreExtractor string `json:"scoreExtractor,omitempty"` // regex or json path to extract score

	// Variables mapping - how to extract data from traces/spans
	Variables map[string]string `json:"variables,omitempty"` // e.g., {"input": "trace.input", "output": "trace.output"}

	// Retry settings
	MaxRetries int `json:"maxRetries,omitempty"`
	RetryDelay int `json:"retryDelay,omitempty"` // milliseconds

	// Cost limits
	MaxCostPerEval float64 `json:"maxCostPerEval,omitempty"`
}

// EvaluationJob represents an async evaluation job
type EvaluationJob struct {
	ID          uuid.UUID `db:"id" json:"id"`
	ProjectID   uuid.UUID `db:"project_id" json:"projectId"`
	EvaluatorID uuid.UUID `db:"evaluator_id" json:"evaluatorId"`
	Status      string    `db:"status" json:"status"`

	// Target - what to evaluate
	TargetType string      `db:"target_type" json:"targetType"` // trace, span, batch
	TargetIDs  []uuid.UUID `db:"target_ids" json:"targetIds"`

	// Progress tracking
	TotalItems int `db:"total_items" json:"totalItems"`
	Completed  int `db:"completed" json:"completed"`
	Failed     int `db:"failed" json:"failed"`

	// Timing
	StartedAt   sql.NullTime `db:"started_at" json:"startedAt,omitempty"`
	CompletedAt sql.NullTime `db:"completed_at" json:"completedAt,omitempty"`

	// Cost tracking
	TotalCost   float64 `db:"total_cost" json:"totalCost"`
	TotalTokens int     `db:"total_tokens" json:"totalTokens"`

	// Error info
	ErrorMessage *string `db:"error_message" json:"errorMessage,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// EvaluationResult represents the result of a single evaluation
type EvaluationResult struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	JobID       *uuid.UUID `db:"job_id" json:"jobId,omitempty"`
	EvaluatorID uuid.UUID  `db:"evaluator_id" json:"evaluatorId"`
	ProjectID   uuid.UUID  `db:"project_id" json:"projectId"`

	// Target
	TraceID uuid.UUID  `db:"trace_id" json:"traceId"`
	SpanID  *uuid.UUID `db:"span_id" json:"spanId,omitempty"`

	// Result
	Score       float64 `db:"score" json:"score"`
	StringValue *string `db:"string_value" json:"stringValue,omitempty"`
	Reasoning   *string `db:"reasoning" json:"reasoning,omitempty"`
	RawResponse string  `db:"raw_response" json:"rawResponse,omitempty"`

	// Cost and usage
	PromptTokens     int             `db:"prompt_tokens" json:"promptTokens"`
	CompletionTokens int             `db:"completion_tokens" json:"completionTokens"`
	Cost             decimal.Decimal `db:"cost" json:"cost"`
	LatencyMs        int             `db:"latency_ms" json:"latencyMs"`

	// Status
	Status       string  `db:"status" json:"status"` // success, error
	ErrorMessage *string `db:"error_message" json:"errorMessage,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}

// EvaluatorTemplate represents a built-in evaluation template
type EvaluatorTemplate struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"` // quality, safety, accuracy, helpfulness
	Template    string          `json:"template"`
	Variables   []string        `json:"variables"`
	OutputType  string          `json:"outputType"`
	MinValue    float64         `json:"minValue,omitempty"`
	MaxValue    float64         `json:"maxValue,omitempty"`
	Categories  []string        `json:"categories,omitempty"`
	Config      EvaluatorConfig `json:"config"`
}

// EvaluatorCreate represents data for creating a new evaluator
type EvaluatorCreate struct {
	ProjectID   uuid.UUID        `json:"projectId" validate:"required"`
	Name        string           `json:"name" validate:"required,min=1,max=255"`
	Description string           `json:"description,omitempty"`
	Type        string           `json:"type" validate:"required,oneof=llm_judge custom"`
	Provider    string           `json:"provider" validate:"required"`
	Model       string           `json:"model" validate:"required"`
	Template    string           `json:"template" validate:"required"`
	Config      *EvaluatorConfig `json:"config,omitempty"`
	OutputType  string           `json:"outputType" validate:"required,oneof=numeric boolean categorical"`
	MinValue    *float64         `json:"minValue,omitempty"`
	MaxValue    *float64         `json:"maxValue,omitempty"`
	Categories  []string         `json:"categories,omitempty"`
	Enabled     *bool            `json:"enabled,omitempty"`
}

// EvaluatorUpdate represents data for updating an evaluator
type EvaluatorUpdate struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Provider    *string          `json:"provider,omitempty"`
	Model       *string          `json:"model,omitempty"`
	Template    *string          `json:"template,omitempty"`
	Config      *EvaluatorConfig `json:"config,omitempty"`
	OutputType  *string          `json:"outputType,omitempty"`
	MinValue    *float64         `json:"minValue,omitempty"`
	MaxValue    *float64         `json:"maxValue,omitempty"`
	Categories  *[]string        `json:"categories,omitempty"`
	Enabled     *bool            `json:"enabled,omitempty"`
}

// EvaluationJobCreate represents data for creating an evaluation job
type EvaluationJobCreate struct {
	ProjectID   uuid.UUID   `json:"projectId" validate:"required"`
	EvaluatorID uuid.UUID   `json:"evaluatorId" validate:"required"`
	TargetType  string      `json:"targetType" validate:"required,oneof=trace span batch"`
	TargetIDs   []uuid.UUID `json:"targetIds" validate:"required,min=1"`
}

// RunEvaluationRequest represents a request to run a single evaluation
type RunEvaluationRequest struct {
	EvaluatorID uuid.UUID  `json:"evaluatorId" validate:"required"`
	TraceID     uuid.UUID  `json:"traceId" validate:"required"`
	SpanID      *uuid.UUID `json:"spanId,omitempty"`
	Async       bool       `json:"async,omitempty"` // If true, return job ID instead of waiting
}

// BatchEvaluationRequest represents a request to run batch evaluations
type BatchEvaluationRequest struct {
	EvaluatorID uuid.UUID   `json:"evaluatorId" validate:"required"`
	TraceIDs    []uuid.UUID `json:"traceIds" validate:"required,min=1,max=1000"`
	Async       bool        `json:"async,omitempty"` // If true, return job ID instead of waiting
}

// EvaluatorFilter represents filters for listing evaluators
type EvaluatorFilter struct {
	ProjectID  string `json:"projectId,omitempty"`
	Type       string `json:"type,omitempty"`
	Provider   string `json:"provider,omitempty"`
	OutputType string `json:"outputType,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
	Search     string `json:"search,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

// EvaluationJobFilter represents filters for listing evaluation jobs
type EvaluationJobFilter struct {
	ProjectID   string `json:"projectId,omitempty"`
	EvaluatorID string `json:"evaluatorId,omitempty"`
	Status      string `json:"status,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// EvaluationResultFilter represents filters for listing evaluation results
type EvaluationResultFilter struct {
	ProjectID   string    `json:"projectId,omitempty"`
	EvaluatorID string    `json:"evaluatorId,omitempty"`
	JobID       string    `json:"jobId,omitempty"`
	TraceID     string    `json:"traceId,omitempty"`
	SpanID      string    `json:"spanId,omitempty"`
	Status      string    `json:"status,omitempty"`
	StartDate   time.Time `json:"startDate,omitempty"`
	EndDate     time.Time `json:"endDate,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	Offset      int       `json:"offset,omitempty"`
}

// EvaluationStats represents aggregated evaluation statistics
type EvaluationStats struct {
	EvaluatorID      uuid.UUID `json:"evaluatorId"`
	TotalEvaluations int64     `json:"totalEvaluations"`
	SuccessCount     int64     `json:"successCount"`
	ErrorCount       int64     `json:"errorCount"`
	AvgScore         float64   `json:"avgScore"`
	MinScore         float64   `json:"minScore"`
	MaxScore         float64   `json:"maxScore"`
	TotalCost        float64   `json:"totalCost"`
	TotalTokens      int64     `json:"totalTokens"`
	AvgLatencyMs     float64   `json:"avgLatencyMs"`
}

// EvaluationCostSummary represents cost tracking for evaluations
type EvaluationCostSummary struct {
	EvaluatorID    uuid.UUID `json:"evaluatorId"`
	EvaluatorName  string    `json:"evaluatorName"`
	TotalCost      float64   `json:"totalCost"`
	TotalTokens    int       `json:"totalTokens"`
	EvalCount      int       `json:"evalCount"`
	AvgCostPerEval float64   `json:"avgCostPerEval"`
}

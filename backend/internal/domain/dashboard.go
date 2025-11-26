package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Dashboard represents a custom dashboard
type Dashboard struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	ProjectID   uuid.UUID    `db:"project_id" json:"projectId"`
	Name        string       `db:"name" json:"name"`
	Description string       `db:"description" json:"description,omitempty"`
	Layout      []byte       `db:"layout" json:"layout"` // JSON layout configuration
	IsTemplate  bool         `db:"is_template" json:"isTemplate"`
	IsPublic    bool         `db:"is_public" json:"isPublic"`
	CreatedBy   uuid.UUID    `db:"created_by" json:"createdBy"`
	CreatedAt   time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time    `db:"updated_at" json:"updatedAt"`
	DeletedAt   sql.NullTime `db:"deleted_at" json:"-"`
}

// DashboardWidget represents a widget on a dashboard
type DashboardWidget struct {
	ID          uuid.UUID `db:"id" json:"id"`
	DashboardID uuid.UUID `db:"dashboard_id" json:"dashboardId"`
	WidgetType  string    `db:"widget_type" json:"widgetType"` // line_chart, bar_chart, metric_card, etc.
	Title       string    `db:"title" json:"title"`
	Config      []byte    `db:"config" json:"config"`     // JSON widget configuration
	Position    []byte    `db:"position" json:"position"` // JSON position {x, y, w, h}
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// DashboardShare represents a shared dashboard
type DashboardShare struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	DashboardID uuid.UUID    `db:"dashboard_id" json:"dashboardId"`
	ShareToken  string       `db:"share_token" json:"shareToken"`
	ExpiresAt   sql.NullTime `db:"expires_at" json:"expiresAt,omitempty"`
	CreatedBy   uuid.UUID    `db:"created_by" json:"createdBy"`
	CreatedAt   time.Time    `db:"created_at" json:"createdAt"`
}

// WidgetType constants
const (
	WidgetTypeLineChart   = "line_chart"
	WidgetTypeBarChart    = "bar_chart"
	WidgetTypePieChart    = "pie_chart"
	WidgetTypeMetricCard  = "metric_card"
	WidgetTypeTable       = "table"
	WidgetTypeHeatmap     = "heatmap"
	WidgetTypeMarkdown    = "markdown"
)

// WidgetConfig represents the configuration for different widget types
type WidgetConfig struct {
	// Data source
	DataSource string                 `json:"dataSource"` // metrics, traces, scores, etc.
	Query      map[string]interface{} `json:"query"`

	// Visualization
	ChartType  string   `json:"chartType,omitempty"`
	XAxis      string   `json:"xAxis,omitempty"`
	YAxis      []string `json:"yAxis,omitempty"`
	Colors     []string `json:"colors,omitempty"`

	// Filters
	TimeRange  string                 `json:"timeRange,omitempty"`
	Filters    map[string]interface{} `json:"filters,omitempty"`

	// Formatting
	FormatX    string `json:"formatX,omitempty"`
	FormatY    string `json:"formatY,omitempty"`
	ShowLegend bool   `json:"showLegend,omitempty"`
	ShowGrid   bool   `json:"showGrid,omitempty"`
}

// WidgetPosition represents the position and size of a widget
type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"` // width in grid units
	H int `json:"h"` // height in grid units
}

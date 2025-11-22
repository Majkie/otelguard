package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

// AttributeStore handles high-cardinality attribute storage
type AttributeStore struct {
	conn driver.Conn
}

// NewAttributeStore creates a new attribute store
func NewAttributeStore(conn driver.Conn) *AttributeStore {
	return &AttributeStore{conn: conn}
}

// AttributeType defines the type of attribute
type AttributeType string

const (
	AttributeTypeString  AttributeType = "string"
	AttributeTypeInt     AttributeType = "int"
	AttributeTypeFloat   AttributeType = "float"
	AttributeTypeBool    AttributeType = "bool"
	AttributeTypeJSON    AttributeType = "json"
)

// TraceAttribute represents a single trace attribute
type TraceAttribute struct {
	TraceID     uuid.UUID     `json:"traceId"`
	SpanID      *uuid.UUID    `json:"spanId,omitempty"`
	ProjectID   uuid.UUID     `json:"projectId"`
	Key         string        `json:"key"`
	ValueType   AttributeType `json:"valueType"`
	StringValue *string       `json:"stringValue,omitempty"`
	IntValue    *int64        `json:"intValue,omitempty"`
	FloatValue  *float64      `json:"floatValue,omitempty"`
	BoolValue   *bool         `json:"boolValue,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

// AttributeBatch represents a batch of attributes for efficient storage
type AttributeBatch struct {
	Attributes []TraceAttribute
	ProjectID  uuid.UUID
}

// StoreAttributes stores a batch of attributes
func (s *AttributeStore) StoreAttributes(ctx context.Context, batch *AttributeBatch) error {
	if len(batch.Attributes) == 0 {
		return nil
	}

	stmt, err := s.conn.PrepareBatch(ctx, `
		INSERT INTO trace_attributes (
			trace_id, span_id, project_id, key, value_type,
			string_value, int_value, float_value, bool_value, timestamp
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, attr := range batch.Attributes {
		spanID := ""
		if attr.SpanID != nil {
			spanID = attr.SpanID.String()
		}

		stringVal := ""
		if attr.StringValue != nil {
			stringVal = *attr.StringValue
		}

		var intVal int64
		if attr.IntValue != nil {
			intVal = *attr.IntValue
		}

		var floatVal float64
		if attr.FloatValue != nil {
			floatVal = *attr.FloatValue
		}

		var boolVal bool
		if attr.BoolValue != nil {
			boolVal = *attr.BoolValue
		}

		if err := stmt.Append(
			attr.TraceID,
			spanID,
			attr.ProjectID,
			attr.Key,
			string(attr.ValueType),
			stringVal,
			intVal,
			floatVal,
			boolVal,
			attr.Timestamp,
		); err != nil {
			return fmt.Errorf("failed to append attribute: %w", err)
		}
	}

	return stmt.Send()
}

// ExtractAttributes extracts attributes from a JSON metadata string
func ExtractAttributes(traceID uuid.UUID, spanID *uuid.UUID, projectID uuid.UUID, metadata string, timestamp time.Time) []TraceAttribute {
	if metadata == "" || metadata == "{}" {
		return nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &m); err != nil {
		return nil
	}

	var attrs []TraceAttribute
	for key, value := range m {
		attr := TraceAttribute{
			TraceID:   traceID,
			SpanID:    spanID,
			ProjectID: projectID,
			Key:       key,
			Timestamp: timestamp,
		}

		switch v := value.(type) {
		case string:
			attr.ValueType = AttributeTypeString
			attr.StringValue = &v
		case float64:
			// JSON numbers are float64
			if v == float64(int64(v)) {
				intVal := int64(v)
				attr.ValueType = AttributeTypeInt
				attr.IntValue = &intVal
			} else {
				attr.ValueType = AttributeTypeFloat
				attr.FloatValue = &v
			}
		case bool:
			attr.ValueType = AttributeTypeBool
			attr.BoolValue = &v
		case map[string]interface{}, []interface{}:
			// Store complex types as JSON strings
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				continue
			}
			jsonStr := string(jsonBytes)
			attr.ValueType = AttributeTypeJSON
			attr.StringValue = &jsonStr
		default:
			continue
		}

		attrs = append(attrs, attr)
	}

	return attrs
}

// QueryAttributeOptions contains options for querying attributes
type QueryAttributeOptions struct {
	ProjectID uuid.UUID
	TraceID   *uuid.UUID
	SpanID    *uuid.UUID
	KeyPrefix string
	Keys      []string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// QueryAttributes queries attributes with filtering
func (s *AttributeStore) QueryAttributes(ctx context.Context, opts *QueryAttributeOptions) ([]TraceAttribute, error) {
	query := `
		SELECT
			trace_id, span_id, project_id, key, value_type,
			string_value, int_value, float_value, bool_value, timestamp
		FROM trace_attributes
		WHERE project_id = ?
	`
	args := []interface{}{opts.ProjectID}

	if opts.TraceID != nil {
		query += " AND trace_id = ?"
		args = append(args, *opts.TraceID)
	}

	if opts.SpanID != nil {
		query += " AND span_id = ?"
		args = append(args, opts.SpanID.String())
	}

	if opts.KeyPrefix != "" {
		query += " AND key LIKE ?"
		args = append(args, opts.KeyPrefix+"%")
	}

	if len(opts.Keys) > 0 {
		placeholders := make([]string, len(opts.Keys))
		for i := range opts.Keys {
			placeholders[i] = "?"
			args = append(args, opts.Keys[i])
		}
		query += " AND key IN (" + strings.Join(placeholders, ",") + ")"
	}

	if !opts.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, opts.StartTime)
	}

	if !opts.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, opts.EndTime)
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query attributes: %w", err)
	}
	defer rows.Close()

	var attrs []TraceAttribute
	for rows.Next() {
		var attr TraceAttribute
		var spanID, stringVal, valueType string
		var intVal int64
		var floatVal float64
		var boolVal bool

		if err := rows.Scan(
			&attr.TraceID, &spanID, &attr.ProjectID, &attr.Key, &valueType,
			&stringVal, &intVal, &floatVal, &boolVal, &attr.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attribute: %w", err)
		}

		attr.ValueType = AttributeType(valueType)

		if spanID != "" {
			parsed, _ := uuid.Parse(spanID)
			attr.SpanID = &parsed
		}

		if stringVal != "" {
			attr.StringValue = &stringVal
		}
		if intVal != 0 {
			attr.IntValue = &intVal
		}
		if floatVal != 0 {
			attr.FloatValue = &floatVal
		}
		attr.BoolValue = &boolVal

		attrs = append(attrs, attr)
	}

	return attrs, nil
}

// GetDistinctAttributeKeys returns distinct attribute keys for a project
func (s *AttributeStore) GetDistinctAttributeKeys(ctx context.Context, projectID uuid.UUID, limit int) ([]string, error) {
	query := `
		SELECT DISTINCT key
		FROM trace_attributes
		WHERE project_id = ?
		ORDER BY key
		LIMIT ?
	`

	rows, err := s.conn.Query(ctx, query, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// GetAttributeCardinality returns the cardinality of an attribute key
func (s *AttributeStore) GetAttributeCardinality(ctx context.Context, projectID uuid.UUID, key string) (int, error) {
	query := `
		SELECT uniqExact(string_value)
		FROM trace_attributes
		WHERE project_id = ? AND key = ? AND value_type = 'string'
	`

	var count uint64
	if err := s.conn.QueryRow(ctx, query, projectID, key).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to query cardinality: %w", err)
	}

	return int(count), nil
}

package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Mock objects and helpers for testing extraction
func TestExtractAgent(t *testing.T) {
	logger := zap.NewNop()
	svc := NewOTLPTraceService(nil, nil, logger)

	traceID := uuid.New()
	spanID := uuid.New()
	projectID := uuid.New()

	now := time.Now()

	tests := []struct {
		name     string
		span     *domain.Span
		attrs    map[string]string
		expected *domain.Agent
	}{
		{
			name: "Explicit Agent Type",
			span: &domain.Span{
				ID:        spanID,
				TraceID:   traceID,
				ProjectID: projectID,
				Name:      "Test Agent",
				StartTime: now,
				EndTime:   now.Add(time.Second),
			},
			attrs: map[string]string{
				"agent.type": "orchestrator",
				"agent.role": "Main Planner",
			},
			expected: &domain.Agent{
				ID:        uuid.NewSHA1(uuid.NameSpaceOID, spanID[:]),
				ProjectID: projectID,
				TraceID:   traceID,
				SpanID:    spanID,
				Name:      "Test Agent",
				Type:      "orchestrator",
				Role:      "Main Planner",
			},
		},
		{
			name: "Implicit Agent from Span Type",
			span: &domain.Span{
				ID:        spanID,
				TraceID:   traceID,
				ProjectID: projectID,
				Name:      "Worker Agent",
				Type:      domain.SpanTypeAgent,
			},
			attrs: map[string]string{},
			expected: &domain.Agent{
				ID:   uuid.NewSHA1(uuid.NameSpaceOID, spanID[:]),
				Name: "Worker Agent",
				Type: "custom",
			},
		},
		{
			name: "Not an Agent",
			span: &domain.Span{
				ID:   spanID,
				Name: "Regular Span",
				Type: domain.SpanTypeLLM,
			},
			attrs:    map[string]string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractAgent(tt.span, tt.attrs)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Type, result.Type)
				assert.Equal(t, tt.expected.Name, result.Name)
				if tt.expected.Role != "" {
					assert.Equal(t, tt.expected.Role, result.Role)
				}
			}
		})
	}
}

func TestExtractToolCall(t *testing.T) {
	logger := zap.NewNop()
	svc := NewOTLPTraceService(nil, nil, logger)

	traceID := uuid.New()
	spanID := uuid.New()
	projectID := uuid.New()

	tests := []struct {
		name     string
		span     *domain.Span
		attrs    map[string]string
		expected *domain.ToolCall
	}{
		{
			name: "Explicit Tool Call Attribute",
			span: &domain.Span{
				ID:        spanID,
				TraceID:   traceID,
				ProjectID: projectID,
				Name:      "Call Weather API",
			},
			attrs: map[string]string{
				"tool.name": "weather_api",
			},
			expected: &domain.ToolCall{
				Name: "weather_api",
			},
		},
		{
			name: "Implicit Tool Call from Span Type",
			span: &domain.Span{
				ID:        spanID,
				TraceID:   traceID,
				ProjectID: projectID,
				Name:      "calculator",
				Type:      domain.SpanTypeTool,
			},
			attrs: map[string]string{},
			expected: &domain.ToolCall{
				Name: "calculator",
			},
		},
		{
			name: "Not a Tool Call",
			span: &domain.Span{
				ID:   spanID,
				Name: "Normal Span",
				Type: domain.SpanTypeCustom,
			},
			attrs:    map[string]string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractToolCall(tt.span, tt.attrs)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Name, result.Name)
			}
		})
	}
}

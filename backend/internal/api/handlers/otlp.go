package handlers

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// OTLPHandler handles OpenTelemetry Protocol trace ingestion
type OTLPHandler struct {
	traceService *service.TraceService
	logger       *zap.Logger
}

// NewOTLPHandler creates a new OTLP handler
func NewOTLPHandler(traceService *service.TraceService, logger *zap.Logger) *OTLPHandler {
	return &OTLPHandler{
		traceService: traceService,
		logger:       logger,
	}
}

// OTLPTraceRequest represents the OTLP JSON trace export request
// Based on OpenTelemetry proto format converted to JSON
type OTLPTraceRequest struct {
	ResourceSpans []ResourceSpan `json:"resourceSpans"`
}

// ResourceSpan represents a resource with its spans
type ResourceSpan struct {
	Resource   Resource    `json:"resource"`
	ScopeSpans []ScopeSpan `json:"scopeSpans"`
}

// Resource represents the resource producing spans
type Resource struct {
	Attributes []Attribute `json:"attributes"`
}

// ScopeSpan represents an instrumentation scope with spans
type ScopeSpan struct {
	Scope InstrumentationScope `json:"scope"`
	Spans []OTLPSpan           `json:"spans"`
}

// InstrumentationScope represents the instrumentation library
type InstrumentationScope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// OTLPSpan represents an OpenTelemetry span
type OTLPSpan struct {
	TraceID           string      `json:"traceId"`
	SpanID            string      `json:"spanId"`
	ParentSpanID      string      `json:"parentSpanId"`
	Name              string      `json:"name"`
	Kind              int         `json:"kind"`
	StartTimeUnixNano string      `json:"startTimeUnixNano"`
	EndTimeUnixNano   string      `json:"endTimeUnixNano"`
	Attributes        []Attribute `json:"attributes"`
	Status            SpanStatus  `json:"status"`
	Events            []SpanEvent `json:"events"`
}

// Attribute represents a key-value attribute
type Attribute struct {
	Key   string         `json:"key"`
	Value AttributeValue `json:"value"`
}

// AttributeValue represents an attribute value
type AttributeValue struct {
	StringValue string      `json:"stringValue,omitempty"`
	IntValue    string      `json:"intValue,omitempty"`
	DoubleValue float64     `json:"doubleValue,omitempty"`
	BoolValue   bool        `json:"boolValue,omitempty"`
	ArrayValue  *ArrayValue `json:"arrayValue,omitempty"`
}

// ArrayValue represents an array attribute value
type ArrayValue struct {
	Values []AttributeValue `json:"values"`
}

// SpanStatus represents the span status
type SpanStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SpanEvent represents a span event
type SpanEvent struct {
	TimeUnixNano string      `json:"timeUnixNano"`
	Name         string      `json:"name"`
	Attributes   []Attribute `json:"attributes"`
}

// LLM semantic convention attribute keys
const (
	AttrLLMSystem           = "gen_ai.system"
	AttrLLMRequestModel     = "gen_ai.request.model"
	AttrLLMResponseModel    = "gen_ai.response.model"
	AttrLLMPromptTokens     = "gen_ai.usage.prompt_tokens"
	AttrLLMCompletionTokens = "gen_ai.usage.completion_tokens"
	AttrLLMTotalTokens      = "gen_ai.usage.total_tokens"
	AttrLLMPrompt           = "gen_ai.prompt"
	AttrLLMCompletion       = "gen_ai.completion"
	AttrLLMTemperature      = "gen_ai.request.temperature"
	AttrLLMMaxTokens        = "gen_ai.request.max_tokens"

	// Langfuse/Langchain conventions
	AttrLangfuseInput     = "langfuse.input"
	AttrLangfuseOutput    = "langfuse.output"
	AttrLangfuseUserId    = "langfuse.user_id"
	AttrLangfuseSessionId = "langfuse.session_id"
	AttrLangfuseMetadata  = "langfuse.metadata"
	AttrLangfuseCost      = "langfuse.cost"

	// Service attributes
	AttrServiceName    = "service.name"
	AttrServiceVersion = "service.version"
	AttrDeploymentEnv  = "deployment.environment"
)

// SpanKind constants
const (
	SpanKindUnspecified = 0
	SpanKindInternal    = 1
	SpanKindServer      = 2
	SpanKindClient      = 3
	SpanKindProducer    = 4
	SpanKindConsumer    = 5
)

// IngestTraces handles OTLP HTTP trace ingestion
// Supports both JSON and Protobuf (with Content-Type header)
// POST /v1/traces (OTLP standard endpoint)
func (h *OTLPHandler) IngestTraces(c *gin.Context) {
	projectID := c.GetString(middleware.ContextProjectID)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_project",
			"message": "Project ID is required",
		})
		return
	}

	// Handle gzip compressed requests
	var reader io.Reader = c.Request.Body
	if c.GetHeader("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_encoding",
				"message": "Failed to decode gzip content",
			})
			return
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Read the request body
	body, err := io.ReadAll(reader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "read_error",
			"message": "Failed to read request body",
		})
		return
	}

	contentType := c.GetHeader("Content-Type")

	var req OTLPTraceRequest

	// Handle different content types
	if strings.Contains(contentType, "application/x-protobuf") {
		// For protobuf, we'd need to decode using proto
		// For now, return an error suggesting JSON
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error":   "unsupported_media_type",
			"message": "Protobuf format not yet supported. Please use application/json",
		})
		return
	}

	// Default to JSON parsing
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("failed to parse OTLP request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_json",
			"message": "Failed to parse OTLP trace data",
		})
		return
	}

	// Convert OTLP spans to our domain model
	projectUUID, _ := uuid.Parse(projectID)
	traces, spans := h.convertOTLPToTraces(projectUUID, &req)

	// Ingest traces
	if len(traces) > 0 {
		if err := h.traceService.IngestBatch(c.Request.Context(), traces); err != nil {
			h.logger.Error("failed to ingest OTLP traces", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "ingestion_failed",
				"message": "Failed to ingest traces",
			})
			return
		}
	}

	// Ingest spans
	for _, span := range spans {
		if err := h.traceService.IngestSpan(c.Request.Context(), span); err != nil {
			h.logger.Error("failed to ingest OTLP span", zap.Error(err), zap.String("span_id", span.ID.String()))
		}
	}

	h.logger.Info("ingested OTLP traces",
		zap.Int("trace_count", len(traces)),
		zap.Int("span_count", len(spans)),
		zap.String("project_id", projectID),
	)

	// OTLP exporters expect an empty response on success
	c.Status(http.StatusOK)
}

// convertOTLPToTraces converts OTLP spans to our domain traces and spans
func (h *OTLPHandler) convertOTLPToTraces(projectID uuid.UUID, req *OTLPTraceRequest) ([]*domain.Trace, []*domain.Span) {
	var traces []*domain.Trace
	var spans []*domain.Span

	// Track which trace IDs we've seen (to create trace records)
	traceMap := make(map[string]*domain.Trace)

	for _, resourceSpan := range req.ResourceSpans {
		// Extract resource attributes
		resourceAttrs := h.extractAttributes(resourceSpan.Resource.Attributes)

		for _, scopeSpan := range resourceSpan.ScopeSpans {
			for _, otlpSpan := range scopeSpan.Spans {
				// Extract span attributes
				spanAttrs := h.extractAttributes(otlpSpan.Attributes)

				// Merge resource attributes with span attributes
				for k, v := range resourceAttrs {
					if _, exists := spanAttrs[k]; !exists {
						spanAttrs[k] = v
					}
				}

				// Parse times
				startTime := h.parseNanoTime(otlpSpan.StartTimeUnixNano)
				endTime := h.parseNanoTime(otlpSpan.EndTimeUnixNano)
				latencyMs := uint32(endTime.Sub(startTime).Milliseconds())

				// Determine span type based on attributes and kind
				spanType := h.determineSpanType(otlpSpan, spanAttrs)

				// Extract LLM-specific attributes
				model := h.getStringAttr(spanAttrs, AttrLLMRequestModel, AttrLLMResponseModel)
				// Apply truncation to large input/output fields
				input := truncateString(h.getStringAttr(spanAttrs, AttrLLMPrompt, AttrLangfuseInput), MaxInputOutputSize)
				output := truncateString(h.getStringAttr(spanAttrs, AttrLLMCompletion, AttrLangfuseOutput), MaxInputOutputSize)

				promptTokens := h.getIntAttr(spanAttrs, AttrLLMPromptTokens)
				completionTokens := h.getIntAttr(spanAttrs, AttrLLMCompletionTokens)
				totalTokens := h.getIntAttr(spanAttrs, AttrLLMTotalTokens)
				if totalTokens == 0 {
					totalTokens = promptTokens + completionTokens
				}

				cost := h.getFloatAttr(spanAttrs, AttrLangfuseCost)

				// Extract session and user IDs
				sessionID := h.getStringAttr(spanAttrs, AttrLangfuseSessionId)
				userID := h.getStringAttr(spanAttrs, AttrLangfuseUserId)

				// Determine status
				status := domain.StatusSuccess
				var errorMsg *string
				if otlpSpan.Status.Code == 2 { // ERROR
					status = domain.StatusError
					if otlpSpan.Status.Message != "" {
						errorMsg = &otlpSpan.Status.Message
					}
				}

				// Create span UUID from hex trace/span IDs
				spanUUID := h.hexToUUID(otlpSpan.SpanID)
				traceUUID := h.hexToUUID(otlpSpan.TraceID)

				// Build metadata JSON
				metadata := h.buildMetadata(spanAttrs, scopeSpan.Scope)

				// Create or update trace record for root spans (no parent)
				if otlpSpan.ParentSpanID == "" {
					// This is a root span, create a trace
					trace := &domain.Trace{
						ID:               traceUUID,
						ProjectID:        projectID,
						Name:             otlpSpan.Name,
						Input:            input,
						Output:           output,
						Metadata:         metadata,
						StartTime:        startTime,
						EndTime:          endTime,
						LatencyMs:        latencyMs,
						TotalTokens:      uint32(totalTokens),
						PromptTokens:     uint32(promptTokens),
						CompletionTokens: uint32(completionTokens),
						Cost:             decimal.NewFromFloat(cost),
						Model:            model,
						Status:           status,
						ErrorMessage:     errorMsg,
					}
					if sessionID != "" {
						trace.SessionID = &sessionID
					}
					if userID != "" {
						trace.UserID = &userID
					}

					traceMap[otlpSpan.TraceID] = trace
					traces = append(traces, trace)
				} else {
					// Create span record
					span := &domain.Span{
						ID:           spanUUID,
						TraceID:      traceUUID,
						ProjectID:    projectID,
						Name:         otlpSpan.Name,
						Type:         spanType,
						Input:        input,
						Output:       output,
						Metadata:     metadata,
						StartTime:    startTime,
						EndTime:      endTime,
						LatencyMs:    latencyMs,
						Tokens:       uint32(totalTokens),
						Cost:         cost,
						Status:       status,
						ErrorMessage: errorMsg,
					}

					if model != "" {
						span.Model = &model
					}

					parentUUID := h.hexToUUID(otlpSpan.ParentSpanID)
					span.ParentSpanID = &parentUUID

					spans = append(spans, span)
				}
			}
		}
	}

	return traces, spans
}

// extractAttributes converts OTLP attributes to a map
func (h *OTLPHandler) extractAttributes(attrs []Attribute) map[string]interface{} {
	result := make(map[string]interface{})
	for _, attr := range attrs {
		result[attr.Key] = h.getAttributeValue(attr.Value)
	}
	return result
}

// getAttributeValue extracts the value from an AttributeValue
func (h *OTLPHandler) getAttributeValue(v AttributeValue) interface{} {
	if v.StringValue != "" {
		return v.StringValue
	}
	if v.IntValue != "" {
		return v.IntValue
	}
	if v.DoubleValue != 0 {
		return v.DoubleValue
	}
	if v.BoolValue {
		return v.BoolValue
	}
	if v.ArrayValue != nil {
		var arr []interface{}
		for _, val := range v.ArrayValue.Values {
			arr = append(arr, h.getAttributeValue(val))
		}
		return arr
	}
	return nil
}

// parseNanoTime parses a nanosecond timestamp string to time.Time
func (h *OTLPHandler) parseNanoTime(nanoStr string) time.Time {
	if nanoStr == "" {
		return time.Now()
	}
	var nanos int64
	for _, c := range nanoStr {
		nanos = nanos*10 + int64(c-'0')
	}
	return time.Unix(0, nanos)
}

// hexToUUID converts a hex string to UUID
func (h *OTLPHandler) hexToUUID(hex string) uuid.UUID {
	// OTLP trace IDs are 32 hex chars, span IDs are 16 hex chars
	// Pad span IDs to make valid UUIDs
	if len(hex) == 16 {
		hex = "00000000" + hex + "00000000"
	}
	if len(hex) != 32 {
		return uuid.New()
	}

	// Insert hyphens to make UUID format
	uuidStr := hex[:8] + "-" + hex[8:12] + "-" + hex[12:16] + "-" + hex[16:20] + "-" + hex[20:32]

	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.New()
	}
	return parsed
}

// determineSpanType determines the span type from attributes
func (h *OTLPHandler) determineSpanType(span OTLPSpan, attrs map[string]interface{}) string {
	// Check for LLM-specific attributes
	if _, ok := attrs[AttrLLMSystem]; ok {
		return domain.SpanTypeLLM
	}
	if _, ok := attrs[AttrLLMRequestModel]; ok {
		return domain.SpanTypeLLM
	}

	// Check span name patterns
	name := strings.ToLower(span.Name)
	if strings.Contains(name, "llm") || strings.Contains(name, "chat") ||
		strings.Contains(name, "completion") || strings.Contains(name, "generate") {
		return domain.SpanTypeLLM
	}
	if strings.Contains(name, "embed") {
		return domain.SpanTypeEmbedding
	}
	if strings.Contains(name, "retriev") || strings.Contains(name, "search") ||
		strings.Contains(name, "vector") {
		return domain.SpanTypeRetrieval
	}
	if strings.Contains(name, "tool") || strings.Contains(name, "function") {
		return domain.SpanTypeTool
	}
	if strings.Contains(name, "agent") {
		return domain.SpanTypeAgent
	}

	// Default based on span kind
	switch span.Kind {
	case SpanKindClient:
		return domain.SpanTypeLLM
	case SpanKindServer:
		return domain.SpanTypeAgent
	default:
		return domain.SpanTypeCustom
	}
}

// getStringAttr gets a string attribute value
func (h *OTLPHandler) getStringAttr(attrs map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := attrs[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

// getIntAttr gets an integer attribute value
func (h *OTLPHandler) getIntAttr(attrs map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if val, ok := attrs[key]; ok {
			switch v := val.(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			case string:
				var i int
				for _, c := range v {
					i = i*10 + int(c-'0')
				}
				return i
			}
		}
	}
	return 0
}

// getFloatAttr gets a float attribute value
func (h *OTLPHandler) getFloatAttr(attrs map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		if val, ok := attrs[key]; ok {
			switch v := val.(type) {
			case float64:
				return v
			case int:
				return float64(v)
			case int64:
				return float64(v)
			}
		}
	}
	return 0
}

// buildMetadata builds metadata JSON from attributes
func (h *OTLPHandler) buildMetadata(attrs map[string]interface{}, scope InstrumentationScope) string {
	metadata := make(map[string]interface{})

	// Add scope info
	if scope.Name != "" {
		metadata["instrumentation_scope"] = map[string]string{
			"name":    scope.Name,
			"version": scope.Version,
		}
	}

	// Add relevant attributes (excluding already processed ones)
	skipKeys := map[string]bool{
		AttrLLMPrompt: true, AttrLLMCompletion: true,
		AttrLangfuseInput: true, AttrLangfuseOutput: true,
		AttrLangfuseUserId: true, AttrLangfuseSessionId: true,
		AttrLLMPromptTokens: true, AttrLLMCompletionTokens: true,
		AttrLLMTotalTokens: true, AttrLangfuseCost: true,
		AttrLLMRequestModel: true, AttrLLMResponseModel: true,
	}

	for k, v := range attrs {
		if !skipKeys[k] {
			metadata[k] = v
		}
	}

	if len(metadata) == 0 {
		return "{}"
	}

	jsonBytes, err := json.Marshal(metadata)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

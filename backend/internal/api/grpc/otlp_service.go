package grpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"github.com/shopspring/decimal"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OTLPTraceService implements the OTLP gRPC trace collector service
type OTLPTraceService struct {
	collectortrace.UnimplementedTraceServiceServer
	traceService *service.TraceService
	logger       *zap.Logger
}

// NewOTLPTraceService creates a new OTLP trace service
func NewOTLPTraceService(traceService *service.TraceService, logger *zap.Logger) *OTLPTraceService {
	return &OTLPTraceService{
		traceService: traceService,
		logger:       logger,
	}
}

// Export implements the OTLP TraceService Export RPC
func (s *OTLPTraceService) Export(ctx context.Context, req *collectortrace.ExportTraceServiceRequest) (*collectortrace.ExportTraceServiceResponse, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		s.logger.Info("incoming metadata", zap.Any("md", md))
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	var traces []*domain.Trace

	for _, resourceSpans := range req.GetResourceSpans() {
		// Extract resource attributes
		resourceAttrs := extractAttributes(resourceSpans.GetResource().GetAttributes())

		for _, scopeSpans := range resourceSpans.GetScopeSpans() {
			for _, span := range scopeSpans.GetSpans() {
				trace, err := s.convertSpanToTrace(span, resourceAttrs)
				if err != nil {
					s.logger.Warn("failed to convert span",
						zap.Error(err),
						zap.String("span_id", hex.EncodeToString(span.GetSpanId())),
					)
					continue
				}
				traces = append(traces, trace)
			}
		}
	}

	if len(traces) > 0 {
		if err := s.traceService.IngestBatch(ctx, traces); err != nil {
			s.logger.Error("failed to ingest traces via gRPC", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to ingest traces")
		}
	}

	s.logger.Debug("ingested traces via gRPC", zap.Int("count", len(traces)))

	return &collectortrace.ExportTraceServiceResponse{}, nil
}

// convertSpanToTrace converts an OTLP span to a domain trace
func (s *OTLPTraceService) convertSpanToTrace(span *tracev1.Span, resourceAttrs map[string]string) (*domain.Trace, error) {
	attrs := extractAttributes(span.GetAttributes())

	// Merge resource attributes
	for k, v := range resourceAttrs {
		if _, exists := attrs[k]; !exists {
			attrs[k] = v
		}
	}

	// Extract trace and span IDs
	traceID := hex.EncodeToString(span.GetTraceId())
	spanID := hex.EncodeToString(span.GetSpanId())

	// Generate UUID from trace ID or create new one
	var id uuid.UUID
	if len(traceID) >= 32 {
		// Try to parse as UUID
		parsed, err := uuid.Parse(traceID[:8] + "-" + traceID[8:12] + "-" + traceID[12:16] + "-" + traceID[16:20] + "-" + traceID[20:32])
		if err != nil {
			id = uuid.New()
		} else {
			id = parsed
		}
	} else {
		id = uuid.New()
	}

	// Extract project ID from attributes or use default
	projectID := getAttrString(attrs, "project.id", "service.name", "langfuse.project_id")
	if projectID == "" {
		projectID = "default"
	}
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		// Generate a deterministic UUID from project name
		projectUUID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(projectID))
	}

	// Extract timestamps
	startTime := time.Unix(0, int64(span.GetStartTimeUnixNano()))
	endTime := time.Unix(0, int64(span.GetEndTimeUnixNano()))
	latencyMs := int(endTime.Sub(startTime).Milliseconds())

	// Extract LLM-specific attributes
	model := getAttrString(attrs, "gen_ai.request.model", "llm.model", "model")
	input := getMessageList(attrs, "gen_ai.prompt", "langfuse.input", "llm.input", "input")
	output := getMessageList(attrs, "gen_ai.completion", "langfuse.output", "llm.output", "output")

	inputJSON, err := json.Marshal(map[string]any{"messages": input})
	if err != nil {
		inputJSON = []byte(`{"messages":[]}`)
	}
	outputJSON, err := json.Marshal(map[string]any{"messages": output})
	if err != nil {
		outputJSON = []byte(`{"messages":[]}`)
	}

	// Extract token counts
	promptTokens := getAttrInt(attrs, "gen_ai.usage.prompt_tokens", "llm.prompt_tokens", "prompt_tokens")
	completionTokens := getAttrInt(attrs, "gen_ai.usage.completion_tokens", "llm.completion_tokens", "completion_tokens")
	totalTokens := promptTokens + completionTokens
	if t := getAttrInt(attrs, "gen_ai.usage.total_tokens", "llm.total_tokens", "total_tokens"); t > 0 {
		totalTokens = t
	}

	// Calculate cost
	cost := getAttrFloat(attrs, "gen_ai.usage.cost", "llm.cost", "cost")

	// Extract session and user IDs
	sessionID := getAttrString(attrs, "session.id", "langfuse.session_id", "session_id")
	userID := getAttrString(attrs, "user.id", "langfuse.user_id", "user_id")

	// Determine status
	traceStatus := "success"
	if span.GetStatus().GetCode() == tracev1.Status_STATUS_CODE_ERROR {
		traceStatus = "error"
	}

	// Extract error message
	var errorMsg *string
	if traceStatus == "error" {
		msg := span.GetStatus().GetMessage()
		if msg != "" {
			errorMsg = &msg
		}
	}

	// Build metadata from remaining attributes
	metadata := buildMetadataJSON(attrs, spanID)

	// Extract tags
	tags := extractTags(attrs)

	trace := &domain.Trace{
		ID:               id,
		ProjectID:        projectUUID,
		Name:             span.GetName(),
		Input:            truncateString(string(inputJSON), 500000),
		Output:           truncateString(string(outputJSON), 500000),
		Metadata:         metadata,
		StartTime:        startTime,
		EndTime:          endTime,
		LatencyMs:        uint32(latencyMs),
		TotalTokens:      uint32(totalTokens),
		PromptTokens:     uint32(promptTokens),
		CompletionTokens: uint32(completionTokens),
		Cost:             decimal.NewFromFloat(cost),
		Model:            model,
		Tags:             tags,
		Status:           traceStatus,
		ErrorMessage:     errorMsg,
	}

	if sessionID != "" {
		trace.SessionID = &sessionID
	}
	if userID != "" {
		trace.UserID = &userID
	}

	return trace, nil
}

// extractAttributes converts OTLP attributes to a string map
func extractAttributes(attrs []*v1.KeyValue) map[string]string {
	result := make(map[string]string)
	for _, attr := range attrs {
		key := attr.GetKey()
		value := attr.GetValue()
		if value == nil {
			continue
		}

		switch v := value.Value.(type) {
		case *v1.AnyValue_StringValue:
			result[key] = v.StringValue
		case *v1.AnyValue_IntValue:
			result[key] = strconv.FormatInt(v.IntValue, 10)
		case *v1.AnyValue_DoubleValue:
			// use 'g' to avoid trailing zeros, or 'f' with precision if needed
			result[key] = strconv.FormatFloat(v.DoubleValue, 'g', -1, 64)
		case *v1.AnyValue_BoolValue:
			if v.BoolValue {
				result[key] = "true"
			} else {
				result[key] = "false"
			}
		case *v1.AnyValue_BytesValue:
			// encode bytes as hex/base64 if you want; here use hex:
			result[key] = fmt.Sprintf("%x", v.BytesValue)
		default:
			// fallback: try to stringify
			result[key] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// getAttrString gets a string attribute by trying multiple keys
func getAttrString(attrs map[string]string, keys ...string) string {
	for _, key := range keys {
		if v, ok := attrs[key]; ok && v != "" {
			return v
		}
	}
	return ""
}

// getAttrInt gets an integer attribute by trying multiple keys
func getAttrInt(attrs map[string]string, keys ...string) int {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			var i int
			if _, err := parseIntFromString(v, &i); err == nil {
				return i
			}
		}
	}
	return 0
}

// getAttrFloat gets a float attribute by trying multiple keys
func getAttrFloat(attrs map[string]string, keys ...string) float64 {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			var f float64
			if _, err := parseFloatFromString(v, &f); err == nil {
				return f
			}
		}
	}
	return 0
}

// parseIntFromString parses an int from a string
func parseIntFromString(s string, result *int) (bool, error) {
	var i int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			i = i*10 + int(c-'0')
		} else {
			break
		}
	}
	*result = i
	return true, nil
}

// parseFloatFromString parses a float from a string
func parseFloatFromString(s string, result *float64) (bool, error) {
	var f float64
	var isDecimal bool
	var decimalPlace = 0.1
	for _, c := range s {
		if c == '.' {
			isDecimal = true
		} else if c >= '0' && c <= '9' {
			if isDecimal {
				f += float64(c-'0') * decimalPlace
				decimalPlace *= 0.1
			} else {
				f = f*10 + float64(c-'0')
			}
		} else {
			break
		}
	}
	*result = f
	return true, nil
}

func getMessageList(attrs map[string]string, prefixes ...string) []ChatMessage {
	type partial struct {
		Role    string
		Content string
	}

	messages := map[int]*partial{}

	// 1) Structured keys e.g. gen_ai.prompt.0.content
	for _, p := range prefixes {
		prefix := p + "."
		for key, val := range attrs {
			if !strings.HasPrefix(key, prefix) {
				continue
			}

			rest := strings.TrimPrefix(key, prefix)
			parts := strings.Split(rest, ".")

			// Accept formats like:
			//   0.content           -> parts == ["0","content"]
			//   0.content.value     -> parts == ["0","content","value"]  (we take first and last)
			if len(parts) < 2 {
				continue
			}

			// parse index (first element)
			idx, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}

			// field is the last element (handles both 2 and 3+ parts)
			field := parts[len(parts)-1]

			if _, ok := messages[idx]; !ok {
				messages[idx] = &partial{}
			}

			switch field {
			case "content":
				messages[idx].Content = val
			case "role":
				messages[idx].Role = val
			default:
				// ignore other fields (finish_reason, name, etc.) for now
			}
		}
	}

	// If structured messages found, build ordered slice
	if len(messages) > 0 {
		keys := make([]int, 0, len(messages))
		for k := range messages {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		out := make([]ChatMessage, 0, len(keys))
		for _, i := range keys {
			m := messages[i]
			if m == nil || m.Content == "" {
				// skip empty content entries
				continue
			}
			role := m.Role
			if role == "" {
				role = "user"
			}
			out = append(out, ChatMessage{
				Role:    role,
				Content: m.Content,
			})
		}
		// ensure we return a non-nil slice (could be empty)
		if out == nil {
			return []ChatMessage{}
		}
		return out
	}

	// 2) Fallback single-string attributes: e.g. "gen_ai.prompt" or "llm.input"
	for _, p := range prefixes {
		if v, ok := attrs[p]; ok && v != "" {
			return []ChatMessage{{Role: "user", Content: v}}
		}
	}

	// nothing found — return empty slice (not nil) so JSON becomes [] not null
	return []ChatMessage{}
}

// buildMetadataJSON builds a JSON string from attributes
func buildMetadataJSON(attrs map[string]string, spanID string) string {
	m := map[string]string{"span_id": spanID}
	for k, v := range attrs {
		if k == "gen_ai.prompt" || k == "gen_ai.completion" ||
			k == "langfuse.input" || k == "langfuse.output" {
			continue
		}
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// extractTags extracts tags from attributes
func extractTags(attrs map[string]string) []string {
	var tags []string
	if t, ok := attrs["tags"]; ok && t != "" {
		// Simple comma-separated parsing
		var tag string
		for _, c := range t {
			if c == ',' {
				if tag != "" {
					tags = append(tags, tag)
					tag = ""
				}
			} else {
				tag += string(c)
			}
		}
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// truncateString truncates a string to maxLen
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	suffix := "\n...[truncated]"
	truncateAt := maxLen - len(suffix)
	if truncateAt < 0 {
		truncateAt = 0
	}
	for truncateAt > 0 && s[truncateAt]&0xC0 == 0x80 {
		truncateAt--
	}
	return s[:truncateAt] + suffix
}

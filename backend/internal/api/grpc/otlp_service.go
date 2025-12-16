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

// InstrumentationScope represents the instrumentation library
type InstrumentationScope struct {
	Name    string
	Version string
}

// OTLPTraceService implements the OTLP gRPC trace collector service
type OTLPTraceService struct {
	collectortrace.UnimplementedTraceServiceServer
	traceService *service.TraceService
	agentService *service.AgentService
	logger       *zap.Logger
}

// NewOTLPTraceService creates a new OTLP trace service
func NewOTLPTraceService(traceService *service.TraceService, agentService *service.AgentService, logger *zap.Logger) *OTLPTraceService {
	return &OTLPTraceService{
		traceService: traceService,
		agentService: agentService,
		logger:       logger,
	}
}

// Export implements the OTLP TraceService Export RPC
func (s *OTLPTraceService) Export(ctx context.Context, req *collectortrace.ExportTraceServiceRequest) (*collectortrace.ExportTraceServiceResponse, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		s.logger.Debug("incoming metadata", zap.Any("md", md))
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	var traces []*domain.Trace
	var spans []*domain.Span
	traceMap := make(map[string]*domain.Trace)

	for _, resourceSpans := range req.GetResourceSpans() {
		// Extract resource attributes
		resourceAttrs := extractAttributes(resourceSpans.GetResource().GetAttributes())

		for _, scopeSpans := range resourceSpans.GetScopeSpans() {
			scope := scopeSpans.GetScope()
			scopeInfo := InstrumentationScope{
				Name:    scope.GetName(),
				Version: scope.GetVersion(),
			}

			for _, span := range scopeSpans.GetSpans() {
				// Convert to domain span and trace (if root)
				t, sp, err := s.convertOTLPSpan(span, resourceAttrs, scopeInfo)
				if err != nil {
					s.logger.Warn("failed to convert span",
						zap.Error(err),
						zap.String("span_id", hex.EncodeToString(span.GetSpanId())),
					)
					continue
				}

				if t != nil {
					// Only add trace if we haven't seen it yet for this batch?
					// Or strictly if it's a root span.
					// OTLP batches might contain fragments of a trace.
					// But our domain model creates a Trace record mainly from the Root Span.
					// If we receive a root span, we create the Trace.
					traceID := hex.EncodeToString(span.GetTraceId())
					if _, exists := traceMap[traceID]; !exists {
						traceMap[traceID] = t
						traces = append(traces, t)
					}
				}
				if sp != nil {
					spans = append(spans, sp)
				}
			}
		}
	}

	// Ingest Traces
	if len(traces) > 0 {
		if err := s.traceService.IngestBatch(ctx, traces); err != nil {
			s.logger.Error("failed to ingest traces via gRPC", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to ingest traces")
		}
	}

	// Ingest Spans
	if len(spans) > 0 {
		for _, span := range spans {
			if err := s.traceService.IngestSpan(ctx, span); err != nil {
				// Log but don't fail the whole request?
				s.logger.Error("failed to ingest span", zap.Error(err), zap.String("span_id", span.ID.String()))
			}
		}
	}

	// Extract and Ingest Agents & Tool Calls
	s.extractAndIngestAgentsAndTools(ctx, traces, spans, req)

	s.logger.Debug("ingested OTLP batch",
		zap.Int("trace_count", len(traces)),
		zap.Int("span_count", len(spans)),
	)

	return &collectortrace.ExportTraceServiceResponse{}, nil
}

// convertOTLPSpan converts an OTLP span to domain trace and span
func (s *OTLPTraceService) convertOTLPSpan(
	span *tracev1.Span,
	resourceAttrs map[string]string,
	scope InstrumentationScope,
) (*domain.Trace, *domain.Span, error) {
	attrs := extractAttributes(span.GetAttributes())

	// Merge resource attributes
	for k, v := range resourceAttrs {
		if _, exists := attrs[k]; !exists {
			attrs[k] = v
		}
	}

	// Extract IDs
	traceIDHex := hex.EncodeToString(span.GetTraceId())
	spanIDHex := hex.EncodeToString(span.GetSpanId())
	parentSpanIDHex := hex.EncodeToString(span.GetParentSpanId())

	traceUUID := s.hexToUUID(traceIDHex)
	spanUUID := s.hexToUUID(spanIDHex)

	var parentSpanUUID *uuid.UUID
	// Check if parent span ID is valid (non-empty and not all zeros)
	if len(parentSpanIDHex) > 0 {
		isValid := false
		for _, c := range parentSpanIDHex {
			if c != '0' {
				isValid = true
				break
			}
		}
		if isValid {
			id := s.hexToUUID(parentSpanIDHex)
			parentSpanUUID = &id
		}
	}

	// Extract project ID
	projectID := getAttrString(attrs, "project.id", "service.name", "langfuse.project_id")
	if projectID == "" {
		projectID = "default"
	}
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		projectUUID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(projectID))
	}

	// Extract timestamps
	startTime := time.Unix(0, int64(span.GetStartTimeUnixNano()))
	endTime := time.Unix(0, int64(span.GetEndTimeUnixNano()))
	latencyMs := uint32(endTime.Sub(startTime).Milliseconds())

	// Extract common LLM attributes
	model := getAttrString(attrs, "gen_ai.request.model", "llm.model", "model")
	input := getMessageList(attrs, "gen_ai.prompt", "langfuse.input", "llm.input", "input")
	output := getMessageList(attrs, "gen_ai.completion", "langfuse.output", "llm.output", "output")

	inputJSON, _ := json.Marshal(map[string]any{"messages": input})
	outputJSON, _ := json.Marshal(map[string]any{"messages": output})

	promptTokens := getAttrInt(attrs, "gen_ai.usage.prompt_tokens", "llm.prompt_tokens", "prompt_tokens")
	completionTokens := getAttrInt(attrs, "gen_ai.usage.completion_tokens", "llm.completion_tokens", "completion_tokens")
	totalTokens := promptTokens + completionTokens
	if t := getAttrInt(attrs, "gen_ai.usage.total_tokens", "llm.total_tokens", "total_tokens"); t > 0 {
		totalTokens = t
	}

	cost := getAttrDecimal(attrs, "gen_ai.usage.cost", "llm.cost", "cost")
	s.logger.Info("cost: ", zap.Any("cost", cost), zap.String("trace_id", traceIDHex), zap.String("span_id", spanIDHex))

	// Determine Span Type
	spanType := s.determineSpanType(span, attrs)

	// Determine Status
	statusMessage := "success"
	if span.GetStatus().GetCode() == tracev1.Status_STATUS_CODE_ERROR {
		statusMessage = "error"
	}
	var errorMsg *string
	if statusMessage == "error" {
		msg := span.GetStatus().GetMessage()
		if msg != "" {
			errorMsg = &msg
		}
	}

	// Build Metadata
	metadataStr := buildMetadataJSON(attrs, scope, spanIDHex)

	// Create Span
	domainSpan := &domain.Span{
		ID:           spanUUID,
		TraceID:      traceUUID,
		ParentSpanID: parentSpanUUID,
		ProjectID:    projectUUID,
		Name:         span.GetName(),
		Type:         spanType,
		Input:        truncateString(string(inputJSON), 500000),
		Output:       truncateString(string(outputJSON), 500000),
		Metadata:     metadataStr,
		StartTime:    startTime,
		EndTime:      endTime,
		LatencyMs:    latencyMs,
		Tokens:       uint32(totalTokens),
		Cost:         cost,
		Status:       statusMessage,
		ErrorMessage: errorMsg,
	}
	if model != "" {
		domainSpan.Model = &model
	}

	// Create Trace ONLY if it is a root span
	var domainTrace *domain.Trace
	if parentSpanUUID == nil {
		sessionID := getAttrString(attrs, "session.id", "langfuse.session_id", "session_id")
		userID := getAttrString(attrs, "user.id", "langfuse.user_id", "user_id")
		tags := extractTags(attrs)

		domainTrace = &domain.Trace{
			ID:               traceUUID,
			ProjectID:        projectUUID,
			Name:             span.GetName(),
			Input:            domainSpan.Input,
			Output:           domainSpan.Output,
			Metadata:         metadataStr,
			StartTime:        startTime,
			EndTime:          endTime,
			LatencyMs:        latencyMs,
			TotalTokens:      uint32(totalTokens),
			PromptTokens:     uint32(promptTokens),
			CompletionTokens: uint32(completionTokens),
			Cost:             cost,
			Model:            model,
			Tags:             tags,
			Status:           statusMessage,
			ErrorMessage:     errorMsg,
		}
		if sessionID != "" {
			domainTrace.SessionID = &sessionID
		}
		if userID != "" {
			domainTrace.UserID = &userID
		}
	}

	return domainTrace, domainSpan, nil
}

// determineSpanType determines the span type from attributes
func (s *OTLPTraceService) determineSpanType(span *tracev1.Span, attrs map[string]string) string {
	// PRIORITY 1: Check for explicit span.type attribute
	if spanType := getAttrString(attrs, "span.type"); spanType != "" {
		switch spanType {
		case "agent":
			return domain.SpanTypeAgent
		case "tool":
			return domain.SpanTypeTool
		case "llm":
			return domain.SpanTypeLLM
		default:
			return domain.SpanTypeCustom
		}
	}

	// PRIORITY 2: Check for LLM-specific attributes
	if _, ok := attrs["gen_ai.system"]; ok {
		return domain.SpanTypeLLM
	}
	if _, ok := attrs["gen_ai.request.model"]; ok {
		return domain.SpanTypeLLM
	}

	// Check span name patterns
	name := strings.ToLower(span.GetName())
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
	switch span.GetKind() {
	case tracev1.Span_SPAN_KIND_CLIENT:
		return domain.SpanTypeLLM
	case tracev1.Span_SPAN_KIND_SERVER:
		return domain.SpanTypeAgent
	default:
		return domain.SpanTypeCustom
	}
}

// Fixed function structure after paste error above
// extractAndIngestAgentsAndTools helper to process agents and tool calls
func (s *OTLPTraceService) extractAndIngestAgentsAndTools(
	ctx context.Context,
	traces []*domain.Trace,
	spans []*domain.Span,
	req *collectortrace.ExportTraceServiceRequest,
) {
	// Create lookup maps
	spanMap := make(map[uuid.UUID]*domain.Span)
	for _, span := range spans {
		spanMap[span.ID] = span
	}

	var agents []*domain.Agent
	var toolCalls []*domain.ToolCall

	// Re-iterate over OTLP spans to have access to raw attributes if needed,
	// but we can also work with domain spans.
	// Working with OTLP spans is better because we have full attributes before filtering/mapping.
	// However, we just converted them.
	// Let's use the domain spans we just created, but we need raw attributes for deeper inspection?
	// Actually, domain spans have Metadata JSON which might be hard to parse back efficiently.
	// A better approach is to do extraction DURING conversion or iterate OTLP spans again.
	// Iterating OTLP spans again is safer.

	for _, resourceSpans := range req.GetResourceSpans() {
		resourceAttrs := extractAttributes(resourceSpans.GetResource().GetAttributes())
		for _, scopeSpans := range resourceSpans.GetScopeSpans() {
			for _, span := range scopeSpans.GetSpans() {
				spanIDHex := hex.EncodeToString(span.GetSpanId())
				spanUUID := s.hexToUUID(spanIDHex)

				// Find domain span to verify it exists and get ProjectID
				dSpan, ok := spanMap[spanUUID]
				if !ok {
					continue
				}

				attrs := extractAttributes(span.GetAttributes())
				for k, v := range resourceAttrs {
					if _, exists := attrs[k]; !exists {
						attrs[k] = v
					}
				}

				// Extract Agent
				if agent := s.extractAgent(dSpan, attrs); agent != nil {
					agents = append(agents, agent)
				}

				// Extract Tool Call
				if tc := s.extractToolCall(dSpan, attrs); tc != nil {
					toolCalls = append(toolCalls, tc)
				}
			}
		}
	}

	// Ingest Agents
	if len(agents) > 0 {
		// After creating agents, establish parent relationships
		agentsBySpanID := make(map[uuid.UUID]*domain.Agent)
		for _, agent := range agents {
			agentsBySpanID[agent.SpanID] = agent
		}

		// Set parent_agent_id based on parent span relationships
		for _, agent := range agents {
			if span := spanMap[agent.SpanID]; span != nil && span.ParentSpanID != nil {
				if parentAgent, exists := agentsBySpanID[*span.ParentSpanID]; exists {
					agent.ParentAgent = &parentAgent.ID
				}
			}
		}

		if err := s.agentService.CreateAgentsBatch(ctx, agents); err != nil {
			s.logger.Error("failed to create agents", zap.Error(err))
		}

		// Create Agent Relationships
		relationships := s.createAgentRelationships(agents)
		for _, rel := range relationships {
			if err := s.agentService.CreateAgentRelationship(ctx, rel); err != nil {
				s.logger.Error("failed to create agent relationship", zap.Error(err))
			}
		}
	}

	// Ingest Tool Calls
	if len(toolCalls) > 0 {
		// We need to link tool calls to agents if possible
		// This might require a second pass or relying on parent span ID
		for _, tc := range toolCalls {
			if dSpan, ok := spanMap[tc.SpanID]; ok && dSpan.ParentSpanID != nil {
				// See if parent span is an agent
				// This is tricky without looking up the parent agent ID.
				// For now, we ingest as is; generic linking happens later or if ParentSpanID matches an Agent's SpanID.
				// The domain logic `DetectAndStoreToolCalls` does this mapping.
				// Since we are doing it batch-wise, we might miss the parent agent if it was processed in a previous batch.
				// But we can try to look at `agents` in this batch.
				for _, a := range agents {
					if a.SpanID == *dSpan.ParentSpanID {
						tc.AgentID = &a.ID
						break
					}
				}
			}
		}
		if err := s.agentService.CreateToolCallsBatch(ctx, toolCalls); err != nil {
			s.logger.Error("failed to create tool calls", zap.Error(err))
		}
	}
}

func (s *OTLPTraceService) createAgentRelationships(
	agents []*domain.Agent,
) []*domain.AgentRelationship {
	var relationships []*domain.AgentRelationship

	// Better approach mirroring the user snippet roughly:
	// We just set agent.ParentAgent using SpanID matchup.
	// Now we want to create relationship objects.

	// Create map by Agent ID for quick lookup if needed, but we have ParentAgent ID directly in agent struct now.
	// But we need the parent agent's details (like ProjectID, TraceID) to create relationship?
	// Actually agent.ProjectID and agent.TraceID are same for parent usually.

	for _, agent := range agents {
		if agent.ParentAgent != nil {
			// Create relationship
			rel := &domain.AgentRelationship{
				ID:            uuid.New(),
				ProjectID:     agent.ProjectID,
				TraceID:       agent.TraceID,
				SourceAgentID: *agent.ParentAgent,
				TargetAgentID: agent.ID,
				RelationType:  "delegates_to",
				Timestamp:     agent.StartTime,
				CreatedAt:     time.Now(),
			}
			relationships = append(relationships, rel)
		}
	}

	return relationships
}

func (s *OTLPTraceService) extractAgent(
	span *domain.Span,
	attrs map[string]string,
) *domain.Agent {
	agentType := getAttrString(attrs,
		"agent.type",
		"langfuse.agent_type",
		"gen_ai.agent.type")

	// If no explicit type, check if it's implicitly an agent
	if agentType == "" {
		if span.Type == domain.SpanTypeAgent {
			agentType = "custom"
		} else {
			return nil // Not an agent
		}
	}

	// Deterministic ID
	agentID := uuid.NewSHA1(uuid.NameSpaceOID, span.ID[:])

	agent := &domain.Agent{
		ID:           agentID,
		ProjectID:    span.ProjectID,
		TraceID:      span.TraceID,
		SpanID:       span.ID,
		ParentSpanID: span.ParentSpanID,
		Name:         span.Name,
		Type:         agentType,
		StartTime:    span.StartTime,
		EndTime:      span.EndTime,
		LatencyMs:    span.LatencyMs,
		TotalTokens:  span.Tokens,
		Cost:         span.Cost,
		Status:       span.Status,
		ErrorMessage: span.ErrorMessage,
		Metadata:     span.Metadata,
		CreatedAt:    time.Now(),
	}

	// Copy Model from span
	if span.Model != nil {
		agent.Model = span.Model
	}

	// Role
	if role := getAttrString(attrs, "agent.role", "gen_ai.agent.role"); role != "" {
		agent.Role = role
	}

	return agent
}

func (s *OTLPTraceService) extractToolCall(
	span *domain.Span,
	attrs map[string]string,
) *domain.ToolCall {
	// Check for tool call attributes
	// Often represented as a span with specific attributes or name
	isTool := false
	if getAttrString(attrs, "tool.name", "function.name", "gen_ai.tool.name") != "" {
		isTool = true
	} else if span.Type == domain.SpanTypeTool {
		isTool = true
	}

	if !isTool {
		return nil
	}

	toolName := getAttrString(attrs, "tool.name", "function.name", "gen_ai.tool.name")
	if toolName == "" {
		toolName = span.Name
	}

	// Deterministic ID
	tcID := uuid.NewSHA1(uuid.NameSpaceOID, span.ID[:])

	tc := &domain.ToolCall{
		ID:           tcID,
		ProjectID:    span.ProjectID,
		TraceID:      span.TraceID,
		SpanID:       span.ID,
		Name:         toolName,
		Input:        span.Input,
		Output:       span.Output,
		StartTime:    span.StartTime,
		EndTime:      span.EndTime,
		LatencyMs:    span.LatencyMs,
		Status:       span.Status,
		ErrorMessage: span.ErrorMessage,
		Metadata:     span.Metadata,
		CreatedAt:    time.Now(),
	}

	return tc
}

// hexToUUID converts a hex string to UUID
func (s *OTLPTraceService) hexToUUID(hexStr string) uuid.UUID {
	// OTLP trace IDs are 32 hex chars, span IDs are 16 hex chars
	// Pad span IDs to make valid UUIDs
	if len(hexStr) == 16 {
		hexStr = "00000000" + hexStr + "00000000"
	}
	if len(hexStr) != 32 {
		return uuid.New()
	}

	// Insert hyphens to make UUID format
	uuidStr := hexStr[:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:32]

	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.New()
	}
	return parsed
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

func getAttrDecimal(attrs map[string]string, keys ...string) decimal.Decimal {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			num := extractNumericPrefix(v)
			if num == "" {
				continue
			}
			if d, err := decimal.NewFromString(num); err == nil {
				return d
			}
		}
	}
	return decimal.Zero
}

func extractNumericPrefix(s string) string {
	out := make([]rune, 0, len(s))
	decimalSeen := false

	for _, c := range s {
		if c >= '0' && c <= '9' {
			out = append(out, c)
		} else if c == '.' && !decimalSeen {
			out = append(out, c)
			decimalSeen = true
		} else {
			break
		}
	}

	return string(out)
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
func buildMetadataJSON(attrs map[string]string, scope InstrumentationScope, spanID string) string {
	m := map[string]interface{}{"span_id": spanID}

	// Add scope info
	if scope.Name != "" {
		m["instrumentation_scope"] = map[string]string{
			"name":    scope.Name,
			"version": scope.Version,
		}
	}

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

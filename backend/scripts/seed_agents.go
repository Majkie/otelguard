package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Project struct {
	ID   string
	Name string
	Slug string
}

type User struct {
	ID    string
	Email string
	Name  string
}

type Trace struct {
	ID        string
	ProjectID string
	StartTime time.Time
}

type Span struct {
	ID        string
	TraceID   string
	ProjectID string
	Name      string
	StartTime time.Time
	EndTime   time.Time
}

func main() {
	// Connect to PostgreSQL
	pgDSN := "host=localhost port=5432 user=otelguard password=otelguard dbname=otelguard sslmode=disable"
	pgPool, err := pgxpool.New(context.Background(), pgDSN)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	// Connect to ClickHouse
	chConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
	})
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer chConn.Close()

	// Seed agent data
	if err := seedAgentData(pgPool, chConn); err != nil {
		log.Fatalf("Failed to seed agent data: %v", err)
	}

	log.Println("Agent data seeding completed successfully!")
}

func seedAgentData(pgPool *pgxpool.Pool, chConn clickhouse.Conn) error {
	ctx := context.Background()

	// Get existing projects
	var projects []Project
	rows, err := pgPool.Query(ctx, "SELECT id, name, slug FROM projects")
	if err != nil {
		return fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug); err != nil {
			return fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, p)
	}

	if len(projects) == 0 {
		return fmt.Errorf("no projects found")
	}

	// Get existing traces
	var traces []Trace
	chRows, err := chConn.Query(ctx, "SELECT id, project_id, start_time FROM traces LIMIT 10")
	if err != nil {
		return fmt.Errorf("failed to query traces: %w", err)
	}
	defer chRows.Close()

	for chRows.Next() {
		var t Trace
		if err := chRows.Scan(&t.ID, &t.ProjectID, &t.StartTime); err != nil {
			return fmt.Errorf("failed to scan trace: %w", err)
		}
		traces = append(traces, t)
	}

	if len(traces) == 0 {
		log.Println("No traces found, creating sample traces first...")
		// Create some sample traces
		for i := 0; i < 5; i++ {
			traceID := uuid.New().String()
			project := projects[rand.Intn(len(projects))]
			startTime := time.Now().Add(-time.Duration(rand.Intn(30)) * time.Minute)

			err := chConn.Exec(ctx, `
				INSERT INTO traces (id, project_id, name, service_name, service_version, start_time, end_time, latency_ms, status, metadata, tags)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				traceID, project.ID, fmt.Sprintf("Sample Trace %d", i+1), "otelguard", "1.0.0",
				startTime, startTime.Add(time.Duration(rand.Intn(60)+10)*time.Second),
				rand.Intn(5000)+100, "success", "{}", []string{"agent-demo"})
			if err != nil {
				return fmt.Errorf("failed to insert sample trace: %w", err)
			}

			traces = append(traces, Trace{
				ID:        traceID,
				ProjectID: project.ID,
				StartTime: startTime,
			})
		}
	}

	// Seed agents
	agentTypes := []string{"orchestrator", "worker", "tool_caller", "planner", "executor"}
	agentNames := []string{"AgentCoordinator", "TaskExecutor", "DataAnalyzer", "ResponseGenerator", "CodeReviewer"}
	models := []string{"gpt-4", "gpt-3.5-turbo", "claude-3", "llama-2-70b", "codellama"}

	var agents []map[string]interface{}

	for i := 0; i < 10; i++ {
		trace := traces[rand.Intn(len(traces))]
		agentID := uuid.New().String()
		agentType := agentTypes[rand.Intn(len(agentTypes))]
		startTime := trace.StartTime.Add(time.Duration(rand.Intn(30)) * time.Second)
		endTime := startTime.Add(time.Duration(rand.Intn(120)+30) * time.Second)
		latency := uint32(endTime.Sub(startTime).Milliseconds())

		agent := map[string]interface{}{
			"id":           agentID,
			"project_id":   trace.ProjectID,
			"trace_id":     trace.ID,
			"span_id":      uuid.New().String(),
			"name":         agentNames[rand.Intn(len(agentNames))],
			"agent_type":   agentType,
			"role":         fmt.Sprintf("%s Agent", agentType),
			"model":        models[rand.Intn(len(models))],
			"start_time":   startTime,
			"end_time":     endTime,
			"latency_ms":   latency,
			"total_tokens": uint32(rand.Intn(10000) + 1000),
			"cost":         big.NewFloat(rand.Float64() * 0.01),
			"status":       "success",
			"metadata":     fmt.Sprintf(`{"model": "%s", "temperature": %.2f}`, models[rand.Intn(len(models))], rand.Float64()),
			"tags":         []string{"demo", "multi-agent", agentType},
		}

		agents = append(agents, agent)
	}

	// Insert agents in batch - omit cost column to use default
	batch, err := chConn.PrepareBatch(ctx, "INSERT INTO agents (id, project_id, trace_id, span_id, parent_agent_id, name, agent_type, role, model, system_prompt, start_time, end_time, latency_ms, total_tokens, status, error_message, metadata, tags, created_at)")
	if err != nil {
		return fmt.Errorf("failed to prepare batch for agents: %w", err)
	}

	for _, agent := range agents {
		err := batch.Append(
			agent["id"],
			agent["project_id"],
			agent["trace_id"],
			agent["span_id"],
			nil, // parent_agent_id
			agent["name"],
			agent["agent_type"],
			agent["role"],
			agent["model"],
			nil, // system_prompt
			agent["start_time"],
			agent["end_time"],
			agent["latency_ms"],
			agent["total_tokens"],
			agent["status"],
			nil, // error_message
			agent["metadata"],
			agent["tags"],
			time.Now(), // created_at
		)
		if err != nil {
			return fmt.Errorf("failed to append agent to batch: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send agent batch: %w", err)
	}

	log.Printf("Inserted %d agents", len(agents))

	// Seed agent relationships
	var relationships []map[string]interface{}
	for i, agent := range agents {
		if rand.Float32() < 0.7 { // 70% chance of having a relationship
			targetAgent := agents[rand.Intn(len(agents))]
			if agent["id"] != targetAgent["id"] {
				relationship := map[string]interface{}{
					"id":              uuid.New().String(),
					"project_id":      agent["project_id"],
					"trace_id":        agent["trace_id"],
					"source_agent_id": agent["id"],
					"target_agent_id": targetAgent["id"],
					"relation_type":   []string{"delegates_to", "calls", "responds_to", "supervises", "collaborates"}[rand.Intn(5)],
					"timestamp":       agent["start_time"].(time.Time).Add(time.Duration(rand.Intn(60)) * time.Second),
					"metadata":        fmt.Sprintf(`{"sequence": %d}`, i),
				}
				relationships = append(relationships, relationship)
			}
		}
	}

	// Insert relationships
	relBatch, err := chConn.PrepareBatch(ctx, "INSERT INTO agent_relationships")
	if err != nil {
		return fmt.Errorf("failed to prepare batch for relationships: %w", err)
	}

	for _, rel := range relationships {
		err := relBatch.Append(
			rel["id"],
			rel["project_id"],
			rel["trace_id"],
			rel["source_agent_id"],
			rel["target_agent_id"],
			rel["relation_type"],
			rel["timestamp"],
			rel["metadata"],
			time.Now(), // created_at
		)
		if err != nil {
			return fmt.Errorf("failed to append relationship to batch: %w", err)
		}
	}

	if err := relBatch.Send(); err != nil {
		return fmt.Errorf("failed to send relationship batch: %w", err)
	}

	log.Printf("Inserted %d agent relationships", len(relationships))

	// Seed tool calls
	toolNames := []string{"web_search", "database_query", "file_read", "api_call", "code_execution"}
	var toolCalls []map[string]interface{}

	for _, agent := range agents {
		if rand.Float32() < 0.8 { // 80% chance of tool usage
			numTools := rand.Intn(3) + 1
			for j := 0; j < numTools; j++ {
				startTime := agent["start_time"].(time.Time).Add(time.Duration(rand.Intn(60)) * time.Second)
				endTime := startTime.Add(time.Duration(rand.Intn(30)+5) * time.Second)
				latency := uint32(endTime.Sub(startTime).Milliseconds())

				toolCall := map[string]interface{}{
					"id":          uuid.New().String(),
					"project_id":  agent["project_id"],
					"trace_id":    agent["trace_id"],
					"span_id":     agent["span_id"],
					"agent_id":    agent["id"],
					"name":        toolNames[rand.Intn(len(toolNames))],
					"description": fmt.Sprintf("Tool call for %s", toolNames[rand.Intn(len(toolNames))]),
					"input":       fmt.Sprintf(`{"query": "sample input %d"}`, j),
					"output":      fmt.Sprintf(`{"result": "sample output %d", "status": "success"}`, j),
					"start_time":  startTime,
					"end_time":    endTime,
					"latency_ms":  latency,
					"status":      "success",
					"metadata":    fmt.Sprintf(`{"tool_version": "1.%d"}`, rand.Intn(5)+1),
				}
				toolCalls = append(toolCalls, toolCall)
			}
		}
	}

	// Insert tool calls
	toolBatch, err := chConn.PrepareBatch(ctx, "INSERT INTO tool_calls")
	if err != nil {
		return fmt.Errorf("failed to prepare batch for tool calls: %w", err)
	}

	for _, tool := range toolCalls {
		err := toolBatch.Append(
			tool["id"],
			tool["project_id"],
			tool["trace_id"],
			tool["span_id"],
			tool["agent_id"],
			tool["name"],
			tool["description"],
			tool["input"],
			tool["output"],
			tool["start_time"],
			tool["end_time"],
			tool["latency_ms"],
			tool["status"],
			nil, // error_message
			0,   // retry_count
			tool["metadata"],
			time.Now(), // created_at
		)
		if err != nil {
			return fmt.Errorf("failed to append tool call to batch: %w", err)
		}
	}

	if err := toolBatch.Send(); err != nil {
		return fmt.Errorf("failed to send tool call batch: %w", err)
	}

	log.Printf("Inserted %d tool calls", len(toolCalls))

	// Seed agent messages
	messageTypes := []string{"request", "response", "notification"}
	roles := []string{"user", "assistant", "system"}
	var messages []map[string]interface{}

	for _, agent := range agents {
		if rand.Float32() < 0.9 { // 90% chance of messages
			numMessages := rand.Intn(5) + 1
			parentMsgID := uuid.Nil.String()

			for j := 0; j < numMessages; j++ {
				fromAgent := agent["id"]
				toAgent := agents[rand.Intn(len(agents))]["id"]
				if fromAgent == toAgent {
					continue
				}

				timestamp := agent["start_time"].(time.Time).Add(time.Duration(j*10+rand.Intn(30)) * time.Second)

				message := map[string]interface{}{
					"id":            uuid.New().String(),
					"project_id":    agent["project_id"],
					"trace_id":      agent["trace_id"],
					"span_id":       agent["span_id"],
					"from_agent_id": fromAgent,
					"to_agent_id":   toAgent,
					"message_type":  messageTypes[rand.Intn(len(messageTypes))],
					"role":          roles[rand.Intn(len(roles))],
					"content":       fmt.Sprintf("Sample message content %d from %s", j+1, agent["name"]),
					"sequence_num":  int32(j + 1),
					"parent_msg_id": parentMsgID,
					"token_count":   uint32(rand.Intn(500) + 50),
					"timestamp":     timestamp,
					"metadata":      fmt.Sprintf(`{"confidence": %.2f}`, rand.Float64()),
				}

				messages = append(messages, message)
				parentMsgID = message["id"].(string)
			}
		}
	}

	// Insert messages
	msgBatch, err := chConn.PrepareBatch(ctx, "INSERT INTO agent_messages")
	if err != nil {
		return fmt.Errorf("failed to prepare batch for messages: %w", err)
	}

	for _, msg := range messages {
		err := msgBatch.Append(
			msg["id"],
			msg["project_id"],
			msg["trace_id"],
			msg["span_id"],
			msg["from_agent_id"],
			msg["to_agent_id"],
			msg["message_type"],
			msg["role"],
			msg["content"],
			"text", // content_type
			msg["sequence_num"],
			msg["parent_msg_id"],
			msg["token_count"],
			msg["timestamp"],
			msg["metadata"],
			time.Now(), // created_at
		)
		if err != nil {
			return fmt.Errorf("failed to append message to batch: %w", err)
		}
	}

	if err := msgBatch.Send(); err != nil {
		return fmt.Errorf("failed to send message batch: %w", err)
	}

	log.Printf("Inserted %d agent messages", len(messages))

	// Seed agent states
	states := []string{"initializing", "planning", "executing", "waiting", "thinking", "completed"}
	var agentStates []map[string]interface{}

	for _, agent := range agents {
		numStates := rand.Intn(4) + 2
		currentTime := agent["start_time"].(time.Time)

		for j := 0; j < numStates; j++ {
			state := map[string]interface{}{
				"id":           uuid.New().String(),
				"project_id":   agent["project_id"],
				"trace_id":     agent["trace_id"],
				"agent_id":     agent["id"],
				"span_id":      agent["span_id"],
				"sequence_num": int32(j + 1),
				"state":        states[rand.Intn(len(states))],
				"variables":    fmt.Sprintf(`{"step": %d, "progress": %.1f}`, j+1, float64(j+1)/float64(numStates)),
				"memory":       fmt.Sprintf(`{"context": "step_%d"}`, j+1),
				"plan":         fmt.Sprintf("Step %d of execution plan", j+1),
				"reasoning":    fmt.Sprintf("Reasoning for step %d", j+1),
				"timestamp":    currentTime,
				"metadata":     fmt.Sprintf(`{"phase": "execution", "step": %d}`, j+1),
			}

			agentStates = append(agentStates, state)
			currentTime = currentTime.Add(time.Duration(rand.Intn(20)+5) * time.Second)
		}
	}

	// Insert states
	stateBatch, err := chConn.PrepareBatch(ctx, "INSERT INTO agent_states")
	if err != nil {
		return fmt.Errorf("failed to prepare batch for states: %w", err)
	}

	for _, state := range agentStates {
		err := stateBatch.Append(
			state["id"],
			state["project_id"],
			state["trace_id"],
			state["agent_id"],
			state["span_id"],
			state["sequence_num"],
			state["state"],
			state["variables"],
			state["memory"],
			state["plan"],
			state["reasoning"],
			state["timestamp"],
			state["metadata"],
			time.Now(), // created_at
		)
		if err != nil {
			return fmt.Errorf("failed to append state to batch: %w", err)
		}
	}

	if err := stateBatch.Send(); err != nil {
		return fmt.Errorf("failed to send state batch: %w", err)
	}

	log.Printf("Inserted %d agent states", len(agentStates))

	return nil
}

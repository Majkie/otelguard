package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Data structures to hold Postgres data
type Organization struct {
	ID   string
	Name string
	Slug string
}

type Project struct {
	ID             string
	OrganizationID string
	Name           string
	Slug           string
}

type User struct {
	ID    string
	Email string
	Name  string
}

type Prompt struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	Tags        []string
}

type GuardrailPolicy struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	Enabled     bool
}

// Sample data for generating realistic traces
var (
	models       = []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}
	spanTypes    = []string{"llm", "retrieval", "tool", "agent", "embedding", "custom"}
	tags         = []string{"production", "staging", "development", "test", "experiment", "v1", "v2", "critical", "background"}
	eventTypes   = []string{"log", "exception", "custom", "user_action", "system"}
	levels       = []string{"debug", "info", "warn", "error", "fatal"}
	sources      = []string{"api", "worker", "frontend", "scheduler", "monitor"}
	environments = []string{"production", "staging", "development"}
)

func main() {
	log.Println("OTelGuard Postgres to ClickHouse Migration")
	log.Println("==========================================")

	// Environment variables
	pgHost := getEnv("POSTGRES_HOST", "localhost")
	pgPort := getEnv("POSTGRES_PORT", "5432")
	pgUser := getEnv("POSTGRES_USER", "otelguard")
	pgPass := getEnv("POSTGRES_PASSWORD", "otelguard")
	pgDB := getEnv("POSTGRES_DB", "otelguard")

	chHost := getEnv("CLICKHOUSE_HOST", "localhost")
	chPort := getEnv("CLICKHOUSE_PORT", "9000")
	chDB := getEnv("CLICKHOUSE_DB", "default")

	// Connect to PostgreSQL
	pgDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		pgHost, pgPort, pgUser, pgPass, pgDB)

	pgConfig, err := pgxpool.ParseConfig(pgDSN)
	if err != nil {
		log.Fatalf("Failed to parse PostgreSQL config: %v", err)
	}

	pgConn, err := pgxpool.NewWithConfig(context.Background(), pgConfig)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgConn.Close()
	log.Println("✓ Connected to PostgreSQL")

	// Connect to ClickHouse
	chConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", chHost, chPort)},
		Auth: clickhouse.Auth{
			Database: chDB,
		},
	})
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer chConn.Close()
	log.Println("✓ Connected to ClickHouse")

	ctx := context.Background()

	// Query existing data from Postgres
	log.Println("\nQuerying existing PostgreSQL data...")
	orgs := queryOrganizations(ctx, pgConn)
	projects := queryProjects(ctx, pgConn)
	users := queryUsers(ctx, pgConn)
	prompts := queryPrompts(ctx, pgConn)
	policies := queryGuardrailPolicies(ctx, pgConn)

	log.Printf("Found %d organizations, %d projects, %d users, %d prompts, %d guardrail policies",
		len(orgs), len(projects), len(users), len(prompts), len(policies))

	// Generate and insert ClickHouse data
	log.Println("\nGenerating ClickHouse data...")
	seedClickHouse(ctx, chConn, projects, users)

	log.Println("\n==========================================")
	log.Println("Migration complete!")
}

func queryOrganizations(ctx context.Context, db *pgxpool.Pool) []Organization {
	rows, err := db.Query(ctx, "SELECT id, name, slug FROM organizations ORDER BY created_at")
	if err != nil {
		log.Fatalf("Failed to query organizations: %v", err)
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var org Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug); err != nil {
			log.Printf("Warning: Could not scan organization: %v", err)
			continue
		}
		orgs = append(orgs, org)
	}
	return orgs
}

func queryProjects(ctx context.Context, db *pgxpool.Pool) []Project {
	rows, err := db.Query(ctx, "SELECT id, organization_id, name, slug FROM projects ORDER BY created_at")
	if err != nil {
		log.Fatalf("Failed to query projects: %v", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		if err := rows.Scan(&project.ID, &project.OrganizationID, &project.Name, &project.Slug); err != nil {
			log.Printf("Warning: Could not scan project: %v", err)
			continue
		}
		projects = append(projects, project)
	}
	return projects
}

func queryUsers(ctx context.Context, db *pgxpool.Pool) []User {
	rows, err := db.Query(ctx, "SELECT id, email, name FROM users ORDER BY created_at")
	if err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name); err != nil {
			log.Printf("Warning: Could not scan user: %v", err)
			continue
		}
		users = append(users, user)
	}
	return users
}

func queryPrompts(ctx context.Context, db *pgxpool.Pool) []Prompt {
	rows, err := db.Query(ctx, "SELECT id, project_id, name, description, tags FROM prompts ORDER BY created_at")
	if err != nil {
		log.Fatalf("Failed to query prompts: %v", err)
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		var prompt Prompt
		if err := rows.Scan(&prompt.ID, &prompt.ProjectID, &prompt.Name, &prompt.Description, &prompt.Tags); err != nil {
			log.Printf("Warning: Could not scan prompt: %v", err)
			continue
		}
		prompts = append(prompts, prompt)
	}
	return prompts
}

func queryGuardrailPolicies(ctx context.Context, db *pgxpool.Pool) []GuardrailPolicy {
	rows, err := db.Query(ctx, "SELECT id, project_id, name, description, enabled FROM guardrail_policies ORDER BY created_at")
	if err != nil {
		log.Fatalf("Failed to query guardrail policies: %v", err)
	}
	defer rows.Close()

	var policies []GuardrailPolicy
	for rows.Next() {
		var policy GuardrailPolicy
		if err := rows.Scan(&policy.ID, &policy.ProjectID, &policy.Name, &policy.Description, &policy.Enabled); err != nil {
			log.Printf("Warning: Could not scan policy: %v", err)
			continue
		}
		policies = append(policies, policy)
	}
	return policies
}

func seedClickHouse(ctx context.Context, conn clickhouse.Conn, projects []Project, users []User) {
	if len(projects) == 0 {
		log.Println("No projects found, skipping ClickHouse seeding")
		return
	}

	now := time.Now()
	totalTraces := 0
	totalSpans := 0
	totalScores := 0
	totalEvents := 0
	totalAttributes := 0
	totalGuardrailEvents := 0

	for _, project := range projects {
		projectUUID, err := uuid.Parse(project.ID)
		if err != nil {
			log.Printf("Warning: Invalid project UUID %s: %v", project.ID, err)
			continue
		}

		log.Printf("  Processing project: %s", project.Name)

		// Generate traces for this project (50-200 traces per project)
		numTraces := 50 + rand.Intn(150)
		traceIDs := generateTraces(ctx, conn, projectUUID, users, numTraces, now)
		totalTraces += len(traceIDs)

		// Generate spans for these traces (2-5 spans per trace)
		spansGenerated := generateSpans(ctx, conn, projectUUID, traceIDs, now)
		totalSpans += spansGenerated

		// Generate scores for traces (60% of traces get scores)
		scoresGenerated := generateScores(ctx, conn, projectUUID, traceIDs, now)
		totalScores += scoresGenerated

		// Generate events (10-50 events per project)
		eventsGenerated := generateEvents(ctx, conn, projectUUID, users, traceIDs, 10+rand.Intn(40), now)
		totalEvents += eventsGenerated

		// Generate attributes for traces and spans
		attrsGenerated := generateAttributes(ctx, conn, projectUUID, traceIDs, now)
		totalAttributes += attrsGenerated

		// Generate guardrail events (5-20 per project)
		guardrailGenerated := generateGuardrailEvents(ctx, conn, projectUUID, traceIDs, 5+rand.Intn(15), now)
		totalGuardrailEvents += guardrailGenerated
	}

	log.Printf("  ✓ Created %d traces", totalTraces)
	log.Printf("  ✓ Created %d spans", totalSpans)
	log.Printf("  ✓ Created %d scores", totalScores)
	log.Printf("  ✓ Created %d events", totalEvents)
	log.Printf("  ✓ Created %d attributes", totalAttributes)
	log.Printf("  ✓ Created %d guardrail events", totalGuardrailEvents)
}

func generateTraces(ctx context.Context, conn clickhouse.Conn, projectID uuid.UUID, users []User, numTraces int, now time.Time) []uuid.UUID {
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO traces (
			id, project_id, session_id, user_id, name,
			input, output, metadata, start_time, end_time,
			latency_ms, total_tokens, prompt_tokens, completion_tokens,
			model, tags, status, error_message
		)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare trace batch: %v", err)
	}

	traceIDs := make([]uuid.UUID, numTraces)
	for i := 0; i < numTraces; i++ {
		traceID := uuid.New()
		traceIDs[i] = traceID

		// Random time in last 30 days
		startTime := now.Add(-time.Duration(rand.Intn(30*24)) * time.Hour)
		latencyMs := uint32(500 + rand.Intn(4500)) // 500ms to 5s
		endTime := startTime.Add(time.Duration(latencyMs) * time.Millisecond)

		promptTokens := uint32(100 + rand.Intn(900))    // 100-1000
		completionTokens := uint32(50 + rand.Intn(450)) // 50-500
		totalTokens := promptTokens + completionTokens

		model := models[rand.Intn(len(models))]

		// Cost calculation

		// Random tags
		numTags := rand.Intn(4)
		traceTags := make([]string, numTags)
		for j := 0; j < numTags; j++ {
			traceTags[j] = tags[rand.Intn(len(tags))]
		}

		// Status (90% success)
		status := "success"
		errorMsg := ""
		if rand.Float32() < 0.1 {
			status = "error"
			errorMsg = "Rate limit exceeded"
		}

		// Random user
		userID := ""
		if len(users) > 0 && rand.Float32() < 0.8 {
			userID = users[rand.Intn(len(users))].ID
		}

		sessionID := fmt.Sprintf("session-%d", rand.Intn(20)+1)

		name := []string{"chat-completion", "code-review", "text-summarization", "sentiment-analysis",
			"translation", "question-answering", "content-generation", "data-extraction"}[rand.Intn(8)]

		err = batch.Append(
			traceID,
			projectID,
			sessionID,
			userID,
			name,
			fmt.Sprintf(`{"messages": [{"role": "user", "content": "Sample input for %s request #%d"}]}`, name, i),
			fmt.Sprintf(`{"content": "Sample response for %s request #%d", "finish_reason": "stop"}`, name, i),
			"{}",
			startTime,
			endTime,
			latencyMs,
			totalTokens,
			promptTokens,
			completionTokens,
			model,
			traceTags,
			status,
			errorMsg,
		)
		if err != nil {
			log.Printf("Warning: Could not append trace: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		log.Fatalf("Failed to send trace batch: %v", err)
	}

	return traceIDs
}

func generateSpans(ctx context.Context, conn clickhouse.Conn, projectID uuid.UUID, traceIDs []uuid.UUID, now time.Time) int {
	if len(traceIDs) == 0 {
		return 0
	}

	spanBatch, err := conn.PrepareBatch(ctx, `
		INSERT INTO spans (
			id, trace_id, parent_span_id, project_id, name, span_type,
			input, output, metadata, start_time, end_time,
			latency_ms, tokens, model, status, error_message
		)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare span batch: %v", err)
	}

	totalSpans := 0
	for _, traceID := range traceIDs {
		// 2-5 spans per trace
		numSpans := 2 + rand.Intn(4)
		startTime := now.Add(-time.Duration(rand.Intn(30*24)) * time.Hour)
		var parentSpanID *uuid.UUID

		for j := 0; j < numSpans; j++ {
			spanID := uuid.New()
			spanType := spanTypes[rand.Intn(len(spanTypes))]
			latencyMs := uint32(50 + rand.Intn(950))
			endTime := startTime.Add(time.Duration(latencyMs) * time.Millisecond)

			tokens := uint32(20 + rand.Intn(480))

			model := ""
			if spanType == "llm" || spanType == "embedding" {
				model = models[rand.Intn(len(models))]
			}

			var parentID *uuid.UUID
			if parentSpanID != nil {
				parentID = parentSpanID
			}

			err = spanBatch.Append(
				spanID,
				traceID,
				parentID,
				projectID,
				fmt.Sprintf("%s-span-%d", spanType, j),
				spanType,
				fmt.Sprintf(`{"query": "Sample input for span %d"}`, j),
				fmt.Sprintf(`{"result": "Sample output for span %d"}`, j),
				"{}",
				startTime,
				endTime,
				latencyMs,
				tokens,
				model,
				"success",
				"",
			)
			if err != nil {
				log.Printf("Warning: Could not append span: %v", err)
			}

			parentSpanID = &spanID
			startTime = endTime
			totalSpans++
		}
	}

	if err := spanBatch.Send(); err != nil {
		log.Fatalf("Failed to send span batch: %v", err)
	}

	return totalSpans
}

func generateScores(ctx context.Context, conn clickhouse.Conn, projectID uuid.UUID, traceIDs []uuid.UUID, now time.Time) int {
	if len(traceIDs) == 0 {
		return 0
	}

	scoreBatch, err := conn.PrepareBatch(ctx, `
		INSERT INTO scores (
			id, project_id, trace_id, span_id, name, value,
			string_value, data_type, source, config_id, comment, created_at
		)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare score batch: %v", err)
	}

	scoreNames := []string{"relevance", "coherence", "accuracy", "helpfulness", "toxicity", "safety"}
	totalScores := 0

	for _, traceID := range traceIDs {
		// 60% chance of having scores
		if rand.Float32() < 0.6 {
			numScores := 1 + rand.Intn(3) // 1-3 scores per trace
			for j := 0; j < numScores; j++ {
				scoreID := uuid.New()
				scoreName := scoreNames[rand.Intn(len(scoreNames))]
				value := 0.6 + rand.Float64()*0.4 // 0.6 to 1.0

				var spanID *uuid.UUID   // nil for no span association
				var configID *uuid.UUID // nil for no config association
				err = scoreBatch.Append(
					scoreID,
					projectID,
					traceID,
					spanID,
					scoreName,
					value,
					"",
					"numeric",
					"llm_judge",
					configID,
					"",
					now.Add(-time.Duration(rand.Intn(24))*time.Hour),
				)
				if err != nil {
					log.Printf("Warning: Could not append score: %v", err)
				}
				totalScores++
			}
		}
	}

	if err := scoreBatch.Send(); err != nil {
		log.Fatalf("Failed to send score batch: %v", err)
	}

	return totalScores
}

func generateEvents(ctx context.Context, conn clickhouse.Conn, projectID uuid.UUID, users []User, traceIDs []uuid.UUID, numEvents int, now time.Time) int {
	eventBatch, err := conn.PrepareBatch(ctx, `
		INSERT INTO events (
			id, project_id, trace_id, span_id, session_id, user_id,
			name, event_type, level, message, data, exception_type,
			exception_message, exception_stacktrace, source, environment,
			version, tags, attributes, timestamp, created_at
		)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare event batch: %v", err)
	}

	for i := 0; i < numEvents; i++ {
		eventID := uuid.New()

		// Random time in last 30 days
		timestamp := now.Add(-time.Duration(rand.Intn(30*24)) * time.Hour)

		// Sometimes associate with a trace
		var traceID *uuid.UUID
		if len(traceIDs) > 0 && rand.Float32() < 0.3 {
			randomTraceID := traceIDs[rand.Intn(len(traceIDs))]
			traceID = &randomTraceID
		}

		// Sometimes associate with a user
		userID := ""
		if len(users) > 0 && rand.Float32() < 0.4 {
			userID = users[rand.Intn(len(users))].ID
		}

		sessionID := ""
		if rand.Float32() < 0.5 {
			sessionID = fmt.Sprintf("session-%d", rand.Intn(20)+1)
		}

		eventType := eventTypes[rand.Intn(len(eventTypes))]
		level := levels[rand.Intn(len(levels))]
		source := sources[rand.Intn(len(sources))]
		environment := environments[rand.Intn(len(environments))]

		name := fmt.Sprintf("%s-event-%d", eventType, i)
		message := fmt.Sprintf("Sample %s event message %d", eventType, i)

		// Exception fields
		exceptionType := ""
		exceptionMessage := ""
		exceptionStacktrace := ""
		if eventType == "exception" {
			exceptionType = "RuntimeError"
			exceptionMessage = "Something went wrong"
			exceptionStacktrace = "at line 42 in main.go"
		}

		// Random tags
		numTags := rand.Intn(3)
		eventTags := make([]string, numTags)
		for j := 0; j < numTags; j++ {
			eventTags[j] = tags[rand.Intn(len(tags))]
		}

		var spanID *uuid.UUID // nil for no span association
		err = eventBatch.Append(
			eventID,
			projectID,
			traceID,
			spanID,
			sessionID,
			userID,
			name,
			eventType,
			level,
			message,
			"{}",
			exceptionType,
			exceptionMessage,
			exceptionStacktrace,
			source,
			environment,
			"1.0.0",
			eventTags,
			map[string]string{"component": source, "env": environment},
			timestamp,
			timestamp,
		)
		if err != nil {
			log.Printf("Warning: Could not append event: %v", err)
		}
	}

	if err := eventBatch.Send(); err != nil {
		log.Fatalf("Failed to send event batch: %v", err)
	}

	return numEvents
}

func generateAttributes(ctx context.Context, conn clickhouse.Conn, projectID uuid.UUID, traceIDs []uuid.UUID, now time.Time) int {
	if len(traceIDs) == 0 {
		return 0
	}

	attributeBatch, err := conn.PrepareBatch(ctx, `
		INSERT INTO trace_attributes (
			trace_id, span_id, project_id, key, value_type,
			string_value, int_value, float_value, bool_value, timestamp
		)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare attribute batch: %v", err)
	}

	attributeKeys := []string{"model_version", "temperature", "max_tokens", "user_agent", "ip_address", "region", "request_id"}
	totalAttributes := 0

	for _, traceID := range traceIDs {
		// 1-5 attributes per trace
		numAttrs := 1 + rand.Intn(5)
		for j := 0; j < numAttrs; j++ {
			key := attributeKeys[rand.Intn(len(attributeKeys))]
			timestamp := now.Add(-time.Duration(rand.Intn(30*24)) * time.Hour)

			// Generate different value types
			valueType := []string{"string", "int", "float", "bool"}[rand.Intn(4)]
			stringValue := ""
			intValue := 0
			floatValue := 0.0
			boolValue := 0

			switch valueType {
			case "string":
				stringValue = fmt.Sprintf("value-%d", rand.Intn(100))
			case "int":
				intValue = rand.Intn(1000)
			case "float":
				floatValue = rand.Float64() * 100
			case "bool":
				if rand.Float32() < 0.5 {
					boolValue = 1
				}
			}

			var spanID *uuid.UUID // nil for trace-level attributes
			err = attributeBatch.Append(
				traceID,
				spanID,
				projectID,
				key,
				valueType,
				stringValue,
				intValue,
				floatValue,
				boolValue,
				timestamp,
			)
			if err != nil {
				log.Printf("Warning: Could not append attribute: %v", err)
			}
			totalAttributes++
		}
	}

	if err := attributeBatch.Send(); err != nil {
		log.Fatalf("Failed to send attribute batch: %v", err)
	}

	return totalAttributes
}

func generateGuardrailEvents(ctx context.Context, conn clickhouse.Conn, projectID uuid.UUID, traceIDs []uuid.UUID, numEvents int, now time.Time) int {
	if len(traceIDs) == 0 {
		return 0
	}

	guardrailBatch, err := conn.PrepareBatch(ctx, `
		INSERT INTO guardrail_events (
			id, project_id, trace_id, span_id, policy_id, rule_id,
			rule_type, triggered, action, action_taken, input_text,
			output_text, detection_result, latency_ms, created_at
		)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare guardrail event batch: %v", err)
	}

	ruleTypes := []string{"toxicity", "jailbreak", "pii", "bias", "safety"}
	actions := []string{"block", "warn", "log", "modify"}

	for i := 0; i < numEvents; i++ {
		eventID := uuid.New()
		traceID := traceIDs[rand.Intn(len(traceIDs))]
		timestamp := now.Add(-time.Duration(rand.Intn(30*24)) * time.Hour)

		policyID := uuid.New()
		ruleID := uuid.New()
		ruleType := ruleTypes[rand.Intn(len(ruleTypes))]
		action := actions[rand.Intn(len(actions))]

		// 30% chance of being triggered
		triggered := uint8(0)
		actionTaken := uint8(0)
		if rand.Float32() < 0.3 {
			triggered = 1
			if rand.Float32() < 0.7 { // 70% of triggered events result in action
				actionTaken = 1
			}
		}

		var spanID *uuid.UUID // nil for no span association
		err = guardrailBatch.Append(
			eventID,
			projectID,
			traceID,
			spanID,
			policyID,
			ruleID,
			ruleType,
			triggered,
			action,
			actionTaken,
			"Sample input text for guardrail check",
			"Sample output text from model",
			fmt.Sprintf("Detected %s pattern", ruleType),
			uint32(10+rand.Intn(100)), // 10-110ms latency
			timestamp,
		)
		if err != nil {
			log.Printf("Warning: Could not append guardrail event: %v", err)
		}
	}

	if err := guardrailBatch.Send(); err != nil {
		log.Fatalf("Failed to send guardrail event batch: %v", err)
	}

	return numEvents
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

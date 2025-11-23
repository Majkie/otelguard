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
	"golang.org/x/crypto/bcrypt"
)

// Sample data
var (
	models = []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}
	names  = []string{
		"chat-completion", "code-review", "text-summarization", "sentiment-analysis",
		"translation", "question-answering", "content-generation", "data-extraction",
		"embedding-search", "agent-task", "rag-query", "classification",
	}
	spanTypes = []string{"llm", "retrieval", "tool", "agent", "embedding", "custom"}
	tags      = []string{"production", "staging", "development", "test", "experiment", "v1", "v2", "critical", "background"}
)

func main() {
	log.Println("OTelGuard Database Seeder")
	log.Println("=========================")

	// Environment variables with defaults
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

	// Seed PostgreSQL
	log.Println("\nSeeding PostgreSQL...")
	orgID, projectID, userID, promptIDs := seedPostgres(ctx, pgConn)

	// Seed ClickHouse
	log.Println("\nSeeding ClickHouse...")
	seedClickHouse(ctx, chConn, projectID, promptIDs)

	log.Println("\n=========================")
	log.Println("Seeding complete!")
	log.Printf("Organization ID: %s", orgID)
	log.Printf("Project ID: %s", projectID)
	log.Printf("User ID: %s", userID)
	log.Println("\nTest credentials:")
	log.Println("  Email: demo@otelguard.dev")
	log.Println("  Password: demo1234")
}

func seedPostgres(ctx context.Context, db *pgxpool.Pool) (string, string, string, []uuid.UUID) {
	// Create organization
	orgID := uuid.New()
	_, err := db.Exec(ctx, `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (slug) DO NOTHING
	`, orgID, "Demo Organization", "demo-org", time.Now())
	if err != nil {
		log.Printf("Warning: Could not create organization: %v", err)
	} else {
		log.Println("  ✓ Created organization: Demo Organization")
	}

	// Create project
	projectID := uuid.New()
	_, err = db.Exec(ctx, `
		INSERT INTO projects (id, organization_id, name, slug, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (organization_id, slug) DO NOTHING
	`, projectID, orgID, "Demo Project", "demo-project", "{}", time.Now())
	if err != nil {
		log.Printf("Warning: Could not create project: %v", err)
	} else {
		log.Println("  ✓ Created project: Demo Project")
	}

	// Create demo user
	userID := uuid.New()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("demo1234"), 10)
	_, err = db.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash
	`, userID, "demo@otelguard.dev", string(hashedPassword), "Demo User", time.Now())
	if err != nil {
		log.Printf("Warning: Could not create user: %v", err)
	} else {
		log.Println("  ✓ Created user: demo@otelguard.dev")
	}

	// Add user to organization
	memberID := uuid.New()
	_, err = db.Exec(ctx, `
		INSERT INTO organization_members (id, organization_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (organization_id, user_id) DO NOTHING
	`, memberID, orgID, userID, "admin", time.Now())
	if err != nil {
		log.Printf("Warning: Could not add member: %v", err)
	} else {
		log.Println("  ✓ Added user to organization as admin")
	}

	// Create sample prompts
	promptTags := []string{"support", "code", "docs"}
	promptIDs := make([]uuid.UUID, 3)
	for i, promptName := range []string{"Customer Support Assistant", "Code Review Helper", "Documentation Generator"} {
		promptID := uuid.New()
		promptIDs[i] = promptID
		_, err = db.Exec(ctx, `
			INSERT INTO prompts (id, project_id, name, description, tags, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6)
		`, promptID, projectID, promptName, fmt.Sprintf("Sample prompt for %s", promptName),
			fmt.Sprintf("{%s}", promptTags[i]), time.Now())
		if err != nil {
			log.Printf("Warning: Could not create prompt %s: %v", promptName, err)
		}
	}
	log.Println("  ✓ Created 3 sample prompts")

	// Create sample guardrail policy
	policyID := uuid.New()
	_, err = db.Exec(ctx, `
		INSERT INTO guardrail_policies (id, project_id, name, description, enabled, priority, triggers, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
	`, policyID, projectID, "Default Safety Policy", "Standard safety guardrails for production",
		true, 1, `{"models": ["*"], "environments": ["production"]}`, time.Now())
	if err != nil {
		log.Printf("Warning: Could not create guardrail policy: %v", err)
	} else {
		log.Println("  ✓ Created sample guardrail policy")
	}

	return orgID.String(), projectID.String(), userID.String(), promptIDs
}

func seedClickHouse(ctx context.Context, conn clickhouse.Conn, projectID string, promptIDs []uuid.UUID) {
	projectUUID, _ := uuid.Parse(projectID)

	// Generate traces for the last 7 days
	now := time.Now()
	numTraces := 100
	numSpansPerTrace := 3

	log.Printf("  Generating %d traces with %d spans each...", numTraces, numSpansPerTrace)

	// Insert traces
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO traces (
			id, project_id, session_id, user_id, name,
			input, output, metadata, start_time, end_time,
			latency_ms, total_tokens, prompt_tokens, completion_tokens,
			model, tags, status, error_message, prompt_id, prompt_version
		)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare trace batch: %v", err)
	}

	traceIDs := make([]uuid.UUID, numTraces)
	for i := 0; i < numTraces; i++ {
		traceID := uuid.New()
		traceIDs[i] = traceID

		// Random time in last 7 days
		startTime := now.Add(-time.Duration(rand.Intn(7*24)) * time.Hour)
		latencyMs := uint32(rand.Intn(5000) + 100)
		endTime := startTime.Add(time.Duration(latencyMs) * time.Millisecond)

		promptTokens := uint32(rand.Intn(1000) + 50)
		completionTokens := uint32(rand.Intn(500) + 20)
		totalTokens := promptTokens + completionTokens

		model := models[rand.Intn(len(models))]
		name := names[rand.Intn(len(names))]

		// Random tags
		numTags := rand.Intn(3) + 1
		traceTags := make([]string, numTags)
		for j := 0; j < numTags; j++ {
			traceTags[j] = tags[rand.Intn(len(tags))]
		}

		// Status (90% success, 10% error)
		status := "success"
		errorMsg := ""
		if rand.Float32() < 0.1 {
			status = "error"
			errorMsg = "Rate limit exceeded"
		}

		sessionID := fmt.Sprintf("session-%d", rand.Intn(20)+1)
		userID := fmt.Sprintf("user-%d", rand.Intn(10)+1)

		// Assign prompt data to ~60% of traces
		var promptID *uuid.UUID
		var promptVersion *int32
		if rand.Float32() < 0.6 && len(promptIDs) > 0 {
			selectedPrompt := promptIDs[rand.Intn(len(promptIDs))]
			promptID = &selectedPrompt
			// Random version between 1-3
			version := int32(rand.Intn(3) + 1)
			promptVersion = &version
		}

		err = batch.Append(
			traceID,
			projectUUID,
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
			promptID,
			promptVersion,
		)
		if err != nil {
			log.Printf("Warning: Could not append trace: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		log.Fatalf("Failed to send trace batch: %v", err)
	}
	log.Printf("  ✓ Created %d traces", numTraces)

	// Insert spans for each trace
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
		startTime := now.Add(-time.Duration(rand.Intn(7*24)) * time.Hour)
		var parentSpanID *uuid.UUID

		for j := 0; j < numSpansPerTrace; j++ {
			spanID := uuid.New()
			spanType := spanTypes[rand.Intn(len(spanTypes))]
			latencyMs := uint32(rand.Intn(1000) + 50)
			endTime := startTime.Add(time.Duration(latencyMs) * time.Millisecond)

			tokens := uint32(rand.Intn(500) + 20)

			model := ""
			if spanType == "llm" || spanType == "embedding" {
				model = models[rand.Intn(len(models))]
			}

			err = spanBatch.Append(
				spanID,
				traceID,
				parentSpanID, // already a *uuid.UUID
				projectUUID,
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
	log.Printf("  ✓ Created %d spans", totalSpans)

	// Insert some scores
	scoreBatch, err := conn.PrepareBatch(ctx, `
		INSERT INTO scores (
			id, project_id, trace_id, span_id, name, value,
			string_value, data_type, source, config_id, comment, created_at
		)
	`)
	if err != nil {
		log.Fatalf("Failed to prepare score batch: %v", err)
	}

	scoreNames := []string{"relevance", "coherence", "accuracy", "helpfulness"}
	totalScores := 0
	for _, traceID := range traceIDs[:min(50, len(traceIDs))] { // Score first 50 traces
		for _, scoreName := range scoreNames {
			if rand.Float32() < 0.6 { // 60% chance of having each score
				scoreID := uuid.New()
				value := rand.Float64()*0.4 + 0.6 // 0.6 to 1.0

				var spanID *uuid.UUID   // nil for no span association
				var configID *uuid.UUID // nil for no config association
				err = scoreBatch.Append(
					scoreID,
					projectUUID,
					traceID,
					spanID,
					scoreName,
					value,
					"",
					"numeric",
					"llm_judge",
					configID,
					"",
					time.Now().Add(-time.Duration(rand.Intn(24))*time.Hour),
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
	log.Printf("  ✓ Created %d scores", totalScores)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

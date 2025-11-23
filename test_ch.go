package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func main() {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
		},
	})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	// Test basic query
	var count uint64
	if err := conn.QueryRow(ctx, "SELECT COUNT(*) FROM traces").Scan(&count); err != nil {
		log.Fatalf("Failed to query traces: %v", err)
	}
	fmt.Printf("Total traces: %d\n", count)

	// Test users query with exact same structure as the code
	query := `
		SELECT
			user_id,
			project_id,
			count() as trace_count,
			uniqExact(session_id) as session_count,
			sum(latency_ms) as total_latency_ms,
			avg(latency_ms) as avg_latency_ms,
			sum(total_tokens) as total_tokens,
			toFloat64(sum(cost)) as total_cost,
			countIf(status = 'success') as success_count,
			countIf(status = 'error') as error_count,
			if(count() > 0, countIf(status = 'success') / count(), 0) as success_rate,
			toString(min(start_time)) as first_seen_time,
			toString(max(start_time)) as last_seen_time,
			groupUniqArray(model) as models
		FROM traces
		WHERE user_id IS NOT NULL AND length(user_id) > 0
		GROUP BY user_id, project_id
		ORDER BY last_seen_time DESC
		LIMIT 3
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}
	defer rows.Close()

	fmt.Println("Users:")
	for rows.Next() {
		var userID, projectID, firstSeen, lastSeen string
		var traceCount, sessionCount, totalLatency, totalTokens, successCount, errorCount uint64
		var avgLatency, totalCost, successRate float64
		var models []string

		if err := rows.Scan(
			&userID, &projectID,
			&traceCount, &sessionCount, &totalLatency, &avgLatency,
			&totalTokens, &totalCost,
			&successCount, &errorCount, &successRate,
			&firstSeen, &lastSeen, &models,
		); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		fmt.Printf("  User: %s, Project: %s, Traces: %d, Sessions: %d\n", userID, projectID, traceCount, sessionCount)
	}
}

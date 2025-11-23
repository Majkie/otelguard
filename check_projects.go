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

	// Check project IDs in traces
	rows, err := conn.Query(ctx, "SELECT DISTINCT project_id FROM traces LIMIT 10")
	if err != nil {
		log.Fatalf("Failed to query project IDs: %v", err)
	}
	defer rows.Close()

	fmt.Println("Project IDs in traces:")
	for rows.Next() {
		var projectID string
		if err := rows.Scan(&projectID); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		fmt.Printf("  %s\n", projectID)
	}
}

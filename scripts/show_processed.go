package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	dsn := "postgresql://gen_user:uzShH%3CA8S%3B7c.e@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	ctx := context.Background()

	rows, err := db.QueryContext(ctx, "SELECT message_id, processed_at FROM processed_emails ORDER BY processed_at DESC")
	if err != nil {
		log.Fatal("Failed to query:", err)
	}
	defer rows.Close()

	fmt.Println("Обработанные письма:")
	for rows.Next() {
		var messageID string
		var processedAt string
		rows.Scan(&messageID, &processedAt)
		fmt.Printf("%s | %s\n", processedAt, messageID)
	}
}

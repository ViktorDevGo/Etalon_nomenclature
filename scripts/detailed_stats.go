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

	// Check all tables
	fmt.Println("=== Статистика по всем таблицам ===\n")

	// price_tires
	var priceCount int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires").Scan(&priceCount)
	fmt.Printf("price_tires: %d записей\n", priceCount)

	rows, _ := db.QueryContext(ctx, `
		SELECT provider, COUNT(*) as count
		FROM price_tires
		GROUP BY provider
		ORDER BY count DESC
	`)
	for rows.Next() {
		var provider string
		var count int
		rows.Scan(&provider, &count)
		fmt.Printf("  - %s: %d\n", provider, count)
	}
	rows.Close()

	// etalon_nomenclature
	var nomenclatureCount int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM etalon_nomenclature").Scan(&nomenclatureCount)
	fmt.Printf("\netalon_nomenclature: %d записей\n", nomenclatureCount)

	// processed_emails
	var emailCount int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM processed_emails").Scan(&emailCount)
	fmt.Printf("\nprocessed_emails: %d записей\n", emailCount)

	// Show all processed emails
	emailRows, _ := db.QueryContext(ctx, `
		SELECT message_id, processed_at
		FROM processed_emails
		ORDER BY processed_at DESC
	`)
	fmt.Println("\nОбработанные письма:")
	for emailRows.Next() {
		var messageID string
		var processedAt string
		emailRows.Scan(&messageID, &processedAt)
		fmt.Printf("  %s | %s\n", processedAt, messageID)
	}
	emailRows.Close()
}

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgresql://gen_user:uzShH%3CA8S%3B7c.e@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Check price_tires table
	var priceCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires").Scan(&priceCount)
	if err != nil {
		log.Fatal("Failed to count price_tires:", err)
	}
	fmt.Printf("✓ price_tires: %d записей\n", priceCount)

	// Show sample price data
	if priceCount > 0 {
		rows, err := db.QueryContext(ctx, `
			SELECT provider, COUNT(*) as count
			FROM price_tires
			GROUP BY provider
		`)
		if err != nil {
			log.Fatal("Failed to query providers:", err)
		}
		defer rows.Close()

		fmt.Println("\nПо поставщикам:")
		for rows.Next() {
			var provider string
			var count int
			rows.Scan(&provider, &count)
			fmt.Printf("  - %s: %d\n", provider, count)
		}

		// Show sample rows
		fmt.Println("\nПримеры данных:")
		sampleRows, _ := db.QueryContext(ctx, `
			SELECT article, price, balance, store, provider
			FROM price_tires
			ORDER BY created_at DESC
			LIMIT 5
		`)
		defer sampleRows.Close()

		for sampleRows.Next() {
			var article, store, provider string
			var price float64
			var balance int
			sampleRows.Scan(&article, &price, &balance, &store, &provider)
			fmt.Printf("  %s | %.2f₽ | остаток: %d | склад: %s | %s\n",
				article, price, balance, store, provider)
		}
	}

	// Check nomenclature table
	var nomenclatureCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM etalon_nomenclature").Scan(&nomenclatureCount)
	if err != nil {
		log.Fatal("Failed to count nomenclature:", err)
	}
	fmt.Printf("\n✓ etalon_nomenclature: %d записей\n", nomenclatureCount)

	// Check processed emails
	var emailCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM processed_emails").Scan(&emailCount)
	if err != nil {
		log.Fatal("Failed to count emails:", err)
	}
	fmt.Printf("✓ processed_emails: %d записей\n", emailCount)

	// Clear processed emails if requested
	if len(os.Args) > 1 && os.Args[1] == "clear" {
		fmt.Println("\n🗑️  Очистка processed_emails...")
		result, err := db.ExecContext(ctx, "DELETE FROM processed_emails WHERE message_id LIKE '%sibzapaska%' OR message_id LIKE '%brinex%' OR message_id LIKE '%bigm%'")
		if err != nil {
			log.Fatal("Failed to clear:", err)
		}
		deleted, _ := result.RowsAffected()
		fmt.Printf("✓ Удалено %d записей из processed_emails\n", deleted)

		// Also clear price_tires for clean test
		fmt.Println("🗑️  Очистка price_tires...")
		result2, err := db.ExecContext(ctx, "DELETE FROM price_tires")
		if err != nil {
			log.Fatal("Failed to clear price_tires:", err)
		}
		deleted2, _ := result2.RowsAffected()
		fmt.Printf("✓ Удалено %d записей из price_tires\n", deleted2)
	}
}

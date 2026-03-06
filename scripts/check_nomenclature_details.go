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

	// Total count
	var total int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM etalon_nomenclature").Scan(&total)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("📊 Всего записей: %d\n\n", total)

	// Check for duplicates by article
	var duplicateCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM (
			SELECT article
			FROM etalon_nomenclature
			GROUP BY article
			HAVING COUNT(*) > 1
		) duplicates
	`).Scan(&duplicateCount)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("🔄 Артикулов с дубликатами: %d\n", duplicateCount)

	// Distinct count
	var distinct int
	err = db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT article) FROM etalon_nomenclature").Scan(&distinct)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✨ Уникальных артикулов: %d\n\n", distinct)

	// Show some duplicates if any
	if duplicateCount > 0 {
		fmt.Println("Примеры дублированных артикулов:")
		rows, _ := db.QueryContext(ctx, `
			SELECT article, COUNT(*) as count
			FROM etalon_nomenclature
			GROUP BY article
			HAVING COUNT(*) > 1
			ORDER BY count DESC
			LIMIT 10
		`)
		defer rows.Close()

		for rows.Next() {
			var article string
			var count int
			rows.Scan(&article, &count)
			fmt.Printf("  %s: %d раз(а)\n", article, count)
		}
	}

	// Check date range
	fmt.Println("\n📅 Когда были добавлены записи:")
	rows, _ := db.QueryContext(ctx, `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM etalon_nomenclature
		GROUP BY DATE(created_at)
		ORDER BY date DESC
		LIMIT 10
	`)
	defer rows.Close()

	for rows.Next() {
		var date string
		var count int
		rows.Scan(&date, &count)
		fmt.Printf("  %s: %d записей\n", date, count)
	}
}

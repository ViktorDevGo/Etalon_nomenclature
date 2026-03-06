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

	// Count by provider in price_tires
	fmt.Println("📊 Статистика price_tires по поставщикам:")
	fmt.Println(string(make([]byte, 60)))

	rows, err := db.QueryContext(ctx, `
		SELECT provider, COUNT(*) as count
		FROM price_tires
		GROUP BY provider
		ORDER BY count DESC
	`)
	if err != nil {
		log.Fatal("Failed to query:", err)
	}
	defer rows.Close()

	totalPrice := 0
	for rows.Next() {
		var provider string
		var count int
		if err := rows.Scan(&provider, &count); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %s: %d записей\n", provider, count)
		totalPrice += count
	}
	fmt.Printf("\n  ИТОГО: %d записей\n\n", totalPrice)

	// Count etalon_nomenclature
	var nomenclatureCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM etalon_nomenclature").Scan(&nomenclatureCount)
	if err != nil {
		log.Fatal("Failed to count nomenclature:", err)
	}

	fmt.Printf("📦 etalon_nomenclature: %d записей\n\n", nomenclatureCount)

	// Show БИГМАШИН samples
	fmt.Println("🔍 Примеры данных от БИГМАШИН (первые 10 строк):")
	fmt.Println(string(make([]byte, 60)))

	rows2, err := db.QueryContext(ctx, `
		SELECT article, price, balance, store
		FROM price_tires
		WHERE provider = 'БИГМАШИН'
		ORDER BY created_at
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("Failed to query:", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var article, store string
		var price float64
		var balance int
		if err := rows2.Scan(&article, &price, &balance, &store); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %s | %.2f₽ | остаток: %d | склад: %s\n",
			article, price, balance, store)
	}

	// Count unique stores for БИГМАШИН
	fmt.Println("\n🏪 Склады БИГМАШИН:")
	fmt.Println(string(make([]byte, 60)))

	rows3, err := db.QueryContext(ctx, `
		SELECT store, COUNT(*) as count
		FROM price_tires
		WHERE provider = 'БИГМАШИН'
		GROUP BY store
		ORDER BY count DESC
	`)
	if err != nil {
		log.Fatal("Failed to query:", err)
	}
	defer rows3.Close()

	for rows3.Next() {
		var store string
		var count int
		if err := rows3.Scan(&store, &count); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %s: %d позиций\n", store, count)
	}
}

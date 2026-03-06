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

	// Show ЗАПАСКА samples
	rows, _ := db.QueryContext(ctx, `
		SELECT article, price, balance, store, provider
		FROM price_tires
		WHERE provider = 'ЗАПАСКА'
		ORDER BY created_at
		LIMIT 10
	`)

	fmt.Println("📦 Примеры данных от ЗАПАСКА (первые 10 строк):")
	for rows.Next() {
		var article, store, provider string
		var price float64
		var balance int
		rows.Scan(&article, &price, &balance, &store, &provider)
		fmt.Printf("  %s | %.2f₽ | остаток: %d | склад: %s\n",
			article, price, balance, store)
	}
	rows.Close()
}

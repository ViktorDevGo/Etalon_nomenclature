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

	fmt.Println("🔍 АНАЛИЗ ПРОБЛЕМНЫХ ЦЕН")
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println()

	// Записи с ценами > 1 млн
	fmt.Println("📊 Топ-20 записей с самыми высокими ценами:")
	fmt.Println(string(make([]byte, 70)))
	rows, err := db.QueryContext(ctx, `
		SELECT article, price, balance, store, provider
		FROM price_tires
		WHERE price > 1000000
		ORDER BY price DESC
		LIMIT 20
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var article, store, provider string
		var price float64
		var balance int
		rows.Scan(&article, &price, &balance, &store, &provider)
		fmt.Printf("  %-20s | %20.2f₽ | %5d шт | %-30s | %s\n",
			article, price, balance, store, provider)
	}

	// Статистика по диапазонам цен
	fmt.Println("\n💰 Статистика по диапазонам цен:")
	fmt.Println(string(make([]byte, 70)))

	ranges := []struct {
		label string
		min   float64
		max   float64
	}{
		{"Нормальные (< 100k)", 0, 100000},
		{"Высокие (100k-1M)", 100000, 1000000},
		{"Подозрительные (1M-10M)", 1000000, 10000000},
		{"Аномальные (> 10M)", 10000000, 999999999999},
	}

	for _, r := range ranges {
		var count int
		err = db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM price_tires
			WHERE price >= $1 AND price < $2
		`, r.min, r.max).Scan(&count)
		if err != nil {
			log.Fatal(err)
		}
		percentage := float64(count) / 189178.0 * 100
		fmt.Printf("  %-30s: %7d записей (%.2f%%)\n", r.label, count, percentage)
	}

	// Поиск паттернов в проблемных ценах
	fmt.Println("\n🔎 Примеры проблемных значений (цена > 100k):")
	fmt.Println(string(make([]byte, 70)))
	rows2, err := db.QueryContext(ctx, `
		SELECT DISTINCT article, price, provider
		FROM price_tires
		WHERE price > 100000
		ORDER BY price DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var article, provider string
		var price float64
		rows2.Scan(&article, &price, &provider)
		fmt.Printf("  Артикул: %-20s | Цена: %20.2f | Поставщик: %s\n",
			article, price, provider)
	}

	// Проверка по поставщикам
	fmt.Println("\n🏢 Проблемные цены по поставщикам:")
	fmt.Println(string(make([]byte, 70)))
	rows3, err := db.QueryContext(ctx, `
		SELECT provider, COUNT(*) as bad_count, MAX(price) as max_price, AVG(price) as avg_price
		FROM price_tires
		WHERE price > 100000
		GROUP BY provider
		ORDER BY bad_count DESC
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows3.Close()

	for rows3.Next() {
		var provider string
		var badCount int
		var maxPrice, avgPrice float64
		rows3.Scan(&provider, &badCount, &maxPrice, &avgPrice)
		fmt.Printf("  %-20s: %5d записей (max: %.2f₽, avg: %.2f₽)\n",
			provider, badCount, maxPrice, avgPrice)
	}

	fmt.Println("\n" + string(make([]byte, 70)))
}

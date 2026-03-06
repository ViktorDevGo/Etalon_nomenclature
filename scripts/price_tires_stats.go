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

	fmt.Println("📊 СТАТИСТИКА ТАБЛИЦЫ price_tires")
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println()

	// Общее количество записей
	var total int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires").Scan(&total)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("📦 Всего записей: %d\n\n", total)

	// По поставщикам
	fmt.Println("📈 По поставщикам:")
	fmt.Println(string(make([]byte, 70)))
	rows, err := db.QueryContext(ctx, `
		SELECT provider, COUNT(*) as count
		FROM price_tires
		GROUP BY provider
		ORDER BY count DESC
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		var count int
		rows.Scan(&provider, &count)
		percentage := float64(count) / float64(total) * 100
		fmt.Printf("  %-20s: %7d записей (%.1f%%)\n", provider, count, percentage)
	}

	// По датам email
	fmt.Println("\n📅 По датам получения (email_date):")
	fmt.Println(string(make([]byte, 70)))
	rows2, err := db.QueryContext(ctx, `
		SELECT
			CASE
				WHEN email_date IS NULL THEN 'NULL (старые данные)'
				ELSE TO_CHAR(email_date, 'YYYY-MM-DD')
			END as date,
			COUNT(*) as count
		FROM price_tires
		GROUP BY date
		ORDER BY date DESC NULLS LAST
		LIMIT 10
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var date string
		var count int
		rows2.Scan(&date, &count)
		fmt.Printf("  %s: %7d записей\n", date, count)
	}

	// Последние обновления по поставщикам
	fmt.Println("\n🔄 Последние обновления по поставщикам:")
	fmt.Println(string(make([]byte, 70)))
	rows3, err := db.QueryContext(ctx, `
		SELECT
			provider,
			COALESCE(TO_CHAR(MAX(email_date), 'YYYY-MM-DD HH24:MI'), 'нет данных') as last_update,
			COUNT(*) as count
		FROM price_tires
		GROUP BY provider
		ORDER BY MAX(email_date) DESC NULLS LAST
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows3.Close()

	for rows3.Next() {
		var provider, lastUpdate string
		var count int
		rows3.Scan(&provider, &lastUpdate, &count)
		fmt.Printf("  %-20s: %s (%d записей)\n", provider, lastUpdate, count)
	}

	// Уникальные артикулы
	var uniqueArticles int
	err = db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT article) FROM price_tires").Scan(&uniqueArticles)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n📝 Дополнительная информация:")
	fmt.Println(string(make([]byte, 70)))
	fmt.Printf("  Уникальных артикулов: %d\n", uniqueArticles)
	fmt.Printf("  Среднее кол-во записей на артикул: %.2f\n", float64(total)/float64(uniqueArticles))

	// Склады БИГМАШИН
	var bigmCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires WHERE provider = 'БИГМАШИН'").Scan(&bigmCount)
	if err == nil && bigmCount > 0 {
		fmt.Println("\n🏪 Склады БИГМАШИН:")
		fmt.Println(string(make([]byte, 70)))
		rows4, err := db.QueryContext(ctx, `
			SELECT store, COUNT(*) as count
			FROM price_tires
			WHERE provider = 'БИГМАШИН'
			GROUP BY store
			ORDER BY count DESC
		`)
		if err == nil {
			defer rows4.Close()
			for rows4.Next() {
				var store string
				var count int
				rows4.Scan(&store, &count)
				fmt.Printf("  %-30s: %5d позиций\n", store, count)
			}
		}
	}

	// Диапазон цен
	var minPrice, maxPrice, avgPrice float64
	err = db.QueryRowContext(ctx, `
		SELECT MIN(price), MAX(price), AVG(price)
		FROM price_tires
		WHERE price > 0
	`).Scan(&minPrice, &maxPrice, &avgPrice)
	if err == nil {
		fmt.Println("\n💰 Диапазон цен:")
		fmt.Println(string(make([]byte, 70)))
		fmt.Printf("  Минимальная: %.2f₽\n", minPrice)
		fmt.Printf("  Максимальная: %.2f₽\n", maxPrice)
		fmt.Printf("  Средняя: %.2f₽\n", avgPrice)
	}

	fmt.Println("\n" + string(make([]byte, 70)))
}

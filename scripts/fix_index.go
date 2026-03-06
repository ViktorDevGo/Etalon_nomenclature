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

	fmt.Println("🔧 ИСПРАВЛЕНИЕ ИНДЕКСА price_tires.article")
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println()

	fmt.Println("Проблема: B-tree индекс не поддерживает очень длинные артикулы")
	fmt.Println("Решение: Заменить на HASH индекс (без ограничений по длине)")
	fmt.Println()

	// Drop existing index
	fmt.Println("🗑️  Удаляю старый B-tree индекс...")
	_, err = db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_price_tires_article")
	if err != nil {
		log.Fatal("Failed to drop index:", err)
	}
	fmt.Println("✅ Старый индекс удален")

	// Create new HASH index
	fmt.Println("\n🔨 Создаю новый HASH индекс...")
	_, err = db.ExecContext(ctx, "CREATE INDEX idx_price_tires_article ON price_tires USING HASH (article)")
	if err != nil {
		log.Fatal("Failed to create index:", err)
	}
	fmt.Println("✅ Новый HASH индекс создан")

	// Also fix other indexes if needed
	fmt.Println("\n🔨 Проверяю другие индексы...")

	// Recreate provider index as B-tree (short values, safe)
	_, err = db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_price_tires_provider")
	if err == nil {
		_, err = db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_price_tires_provider ON price_tires(provider)")
		if err == nil {
			fmt.Println("✅ Индекс provider готов")
		}
	}

	// Recreate created_at index
	_, err = db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_price_tires_created_at")
	if err == nil {
		_, err = db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_price_tires_created_at ON price_tires(created_at)")
		if err == nil {
			fmt.Println("✅ Индекс created_at готов")
		}
	}

	fmt.Println()
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println("✅ Индексы исправлены!")
	fmt.Println()
	fmt.Println("Теперь можно запустить парсинг заново.")
}

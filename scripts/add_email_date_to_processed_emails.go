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

	fmt.Println("🔧 ДОБАВЛЕНИЕ КОЛОНКИ email_date В processed_emails")
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println()

	// Check if column already exists
	var exists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_name='processed_emails'
			AND column_name='email_date'
		)
	`).Scan(&exists)

	if err != nil {
		log.Fatal("Failed to check column existence:", err)
	}

	if exists {
		fmt.Println("✅ Колонка email_date уже существует")
		return
	}

	fmt.Println("➕ Добавляем колонку email_date...")

	// Add column
	_, err = db.ExecContext(ctx, `
		ALTER TABLE processed_emails
		ADD COLUMN email_date TIMESTAMP
	`)

	if err != nil {
		log.Fatal("Failed to add column:", err)
	}

	fmt.Println("✅ Колонка email_date добавлена")

	// Add index
	fmt.Println("➕ Добавляем индекс на email_date...")

	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_processed_emails_email_date
		ON processed_emails(email_date)
	`)

	if err != nil {
		log.Fatal("Failed to add index:", err)
	}

	fmt.Println("✅ Индекс добавлен")

	// Check table structure
	fmt.Println()
	fmt.Println("📋 Структура таблицы processed_emails:")

	rows, err := db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'processed_emails'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatal("Failed to get table structure:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var colName, dataType, nullable string
		rows.Scan(&colName, &dataType, &nullable)
		fmt.Printf("  - %s (%s, nullable: %s)\n", colName, dataType, nullable)
	}

	fmt.Println()
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println("✅ Миграция завершена успешно!")
}

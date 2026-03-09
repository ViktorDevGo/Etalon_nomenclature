package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	dsn := "postgresql://gen_user:uzShH%3CA8S%3B7c.e@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✓ Connected to database")

	// Check current count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM processed_emails").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to count: %v", err)
	}

	fmt.Printf("\nТекущее количество: %d записей\n", count)

	// Delete all
	fmt.Println("\n=== Очистка processed_emails ===")
	result, err := db.Exec("DELETE FROM processed_emails")
	if err != nil {
		log.Fatalf("Failed to delete: %v", err)
	}

	rows, _ := result.RowsAffected()
	fmt.Printf("✓ Удалено %d записей\n", rows)

	// Verify
	err = db.QueryRow("SELECT COUNT(*) FROM processed_emails").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to verify: %v", err)
	}

	fmt.Printf("\nОсталось записей: %d\n", count)
	fmt.Println("\n✅ Готово! Письма будут обработаны заново с новой атомарной логикой!")
}

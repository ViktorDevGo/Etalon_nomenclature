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

	// Clear ЗАПАСКА from processed_emails
	result, err := db.ExecContext(ctx, "DELETE FROM processed_emails WHERE message_id LIKE '%sibzapaska%'")
	if err != nil {
		log.Fatal("Failed to clear:", err)
	}
	deleted, _ := result.RowsAffected()
	fmt.Printf("✓ Удалено %d записей от ЗАПАСКА\n", deleted)
}

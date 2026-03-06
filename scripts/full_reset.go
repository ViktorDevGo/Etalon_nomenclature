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

	fmt.Println("🗑️  Полная очистка всех таблиц...")

	// Clear processed_emails
	result1, _ := db.ExecContext(ctx, "DELETE FROM processed_emails")
	deleted1, _ := result1.RowsAffected()
	fmt.Printf("✓ Удалено %d записей из processed_emails\n", deleted1)

	// Clear price_tires
	result2, _ := db.ExecContext(ctx, "DELETE FROM price_tires")
	deleted2, _ := result2.RowsAffected()
	fmt.Printf("✓ Удалено %d записей из price_tires\n", deleted2)

	// Clear etalon_nomenclature
	result3, _ := db.ExecContext(ctx, "DELETE FROM etalon_nomenclature")
	deleted3, _ := result3.RowsAffected()
	fmt.Printf("✓ Удалено %d записей из etalon_nomenclature\n", deleted3)

	fmt.Println("\n✅ БД полностью очищена и готова к тесту")
}

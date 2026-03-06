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

	fmt.Println("🔧 Удаление дубликата номенклатуры...")

	// Delete the specific processed email that has nomenclature
	result, err := db.ExecContext(ctx, `
		DELETE FROM processed_emails
		WHERE message_id = '<00ff01dcaaf5$0876dff0$19649fd0$@sibzapaska.ru>'
	`)
	if err != nil {
		log.Fatal("Failed to delete:", err)
	}

	rows, _ := result.RowsAffected()
	fmt.Printf("✓ Удалено %d записей из processed_emails\n", rows)

	// Clear etalon_nomenclature
	result, err = db.ExecContext(ctx, "TRUNCATE etalon_nomenclature")
	if err != nil {
		log.Fatal("Failed to truncate:", err)
	}
	fmt.Println("✓ Очищена таблица etalon_nomenclature")

	fmt.Println("\n✅ Готово! Теперь email с номенклатурой будет обработан заново")
}

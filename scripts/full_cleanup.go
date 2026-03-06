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

	fmt.Println("🧹 ПОЛНАЯ ОЧИСТКА БАЗЫ ДАННЫХ")
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println()

	// Count records before cleanup
	var countNomenclature, countPrices, countProcessed int

	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM etalon_nomenclature").Scan(&countNomenclature)
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires").Scan(&countPrices)
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM processed_emails").Scan(&countProcessed)

	fmt.Println("📊 Текущее состояние:")
	fmt.Printf("  etalon_nomenclature: %d записей\n", countNomenclature)
	fmt.Printf("  price_tires: %d записей\n", countPrices)
	fmt.Printf("  processed_emails: %d записей\n", countProcessed)
	fmt.Println()

	fmt.Println("⚠️  ВНИМАНИЕ: Будут удалены ВСЕ данные из всех таблиц!")
	fmt.Println()

	// Delete all data
	fmt.Println("🗑️  Выполняю очистку...")

	result1, err := db.ExecContext(ctx, "DELETE FROM etalon_nomenclature")
	if err != nil {
		log.Fatal("Failed to delete from etalon_nomenclature:", err)
	}
	rows1, _ := result1.RowsAffected()
	fmt.Printf("✅ Удалено из etalon_nomenclature: %d записей\n", rows1)

	result2, err := db.ExecContext(ctx, "DELETE FROM price_tires")
	if err != nil {
		log.Fatal("Failed to delete from price_tires:", err)
	}
	rows2, _ := result2.RowsAffected()
	fmt.Printf("✅ Удалено из price_tires: %d записей\n", rows2)

	result3, err := db.ExecContext(ctx, "DELETE FROM processed_emails")
	if err != nil {
		log.Fatal("Failed to delete from processed_emails:", err)
	}
	rows3, _ := result3.RowsAffected()
	fmt.Printf("✅ Удалено из processed_emails: %d записей\n", rows3)

	// Verify cleanup
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM etalon_nomenclature").Scan(&countNomenclature)
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires").Scan(&countPrices)
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM processed_emails").Scan(&countProcessed)

	fmt.Println()
	fmt.Println("📊 Состояние после очистки:")
	fmt.Printf("  etalon_nomenclature: %d записей\n", countNomenclature)
	fmt.Printf("  price_tires: %d записей\n", countPrices)
	fmt.Printf("  processed_emails: %d записей\n", countProcessed)

	fmt.Println()
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println("✅ Полная очистка завершена!")
	fmt.Println()
	fmt.Println("Готово к запуску парсинга.")
}

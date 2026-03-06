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

	fmt.Println("🧹 ОЧИСТКА ТАБЛИЦЫ ПРАЙСОВ")
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println()

	// Count before
	var countBefore int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires").Scan(&countBefore)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("📊 Записей в price_tires до очистки: %d\n", countBefore)

	// Count processed emails with price files
	var processedPriceEmails int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM processed_emails
		WHERE message_id IN (
			SELECT DISTINCT message_id FROM processed_emails
			-- We'll delete all processed_emails to reprocess everything
		)
	`).Scan(&processedPriceEmails)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("📧 Обработанных email: %d\n", processedPriceEmails)

	fmt.Println()
	fmt.Println("⚠️  ВНИМАНИЕ: Сейчас будут удалены:")
	fmt.Println("   1. Все записи из price_tires")
	fmt.Println("   2. Все записи из processed_emails")
	fmt.Println()
	fmt.Println("Это позволит перезапустить обработку всех прайсов с исправленной логикой.")
	fmt.Println()
	fmt.Print("Продолжить? (yes/no): ")

	var answer string
	fmt.Scanln(&answer)

	if answer != "yes" {
		fmt.Println("❌ Отменено пользователем")
		return
	}

	fmt.Println()
	fmt.Println("🗑️  Выполняю очистку...")

	// Delete from price_tires
	result, err := db.ExecContext(ctx, "DELETE FROM price_tires")
	if err != nil {
		log.Fatal("Failed to delete from price_tires:", err)
	}
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✅ Удалено из price_tires: %d записей\n", rowsAffected)

	// Delete from processed_emails
	result2, err := db.ExecContext(ctx, "DELETE FROM processed_emails")
	if err != nil {
		log.Fatal("Failed to delete from processed_emails:", err)
	}
	rowsAffected2, _ := result2.RowsAffected()
	fmt.Printf("✅ Удалено из processed_emails: %d записей\n", rowsAffected2)

	// Count after
	var countAfter int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires").Scan(&countAfter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n📊 Записей в price_tires после очистки: %d\n", countAfter)

	fmt.Println()
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println("✅ Очистка завершена!")
	fmt.Println()
	fmt.Println("Теперь запустите сервис:")
	fmt.Println("  go run cmd/app/main.go")
	fmt.Println()
	fmt.Println("Он заново обработает все письма с исправленной логикой парсинга цен.")
}

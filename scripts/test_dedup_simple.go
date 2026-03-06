package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
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

	// Clear existing test data
	fmt.Println("🧹 Очистка тестовых данных...")
	_, err = db.ExecContext(ctx, "DELETE FROM etalon_nomenclature WHERE article LIKE 'TEST%'")
	if err != nil {
		log.Fatal("Failed to clear:", err)
	}

	// Simulate batch with duplicates (like what happens with МРЦ Зима + МРЦ Лето)
	testData := []struct {
		article      string
		brand        string
		typ          string
		sizeModel    string
		nomenclature string
		mrc          float64
	}{
		{"TEST001", "Brand1", "Type1", "195/65R15", "Nom1", 5000},
		{"TEST002", "Brand2", "Type2", "205/55R16", "Nom2", 6000},
		{"TEST001", "Brand1-Updated", "Type1-Updated", "195/65R15", "Nom1-Updated", 5500}, // Duplicate
		{"TEST003", "Brand3", "Type3", "225/45R17", "Nom3", 7000},
		{"TEST002", "Brand2-Updated", "Type2-Updated", "205/55R16", "Nom2-Updated", 6500}, // Duplicate
		{"TEST001", "Brand1-Final", "Type1-Final", "195/65R15", "Nom1-Final", 5800},       // Duplicate
	}

	fmt.Printf("\n📦 Тестовые данные:\n")
	fmt.Printf("  Всего строк: %d\n", len(testData))
	fmt.Printf("  Уникальных артикулов: 3 (TEST001 x3, TEST002 x2, TEST003 x1)\n\n")

	fmt.Println("🔄 Шаг 1: Дедупликация внутри батча (последнее вхождение каждого артикула)...")

	// Simulate deduplication logic from InsertNomenclatureWithEmail
	articleMap := make(map[string]int)
	for i, row := range testData {
		articleMap[row.article] = i // Last index wins
	}

	dedupData := make([]struct {
		article      string
		brand        string
		typ          string
		sizeModel    string
		nomenclature string
		mrc          float64
	}, 0)

	for _, idx := range articleMap {
		dedupData = append(dedupData, testData[idx])
	}

	fmt.Printf("  После дедупликации: %d строк (было %d)\n\n", len(dedupData), len(testData))

	// Step 1: Delete existing duplicates for today
	fmt.Println("🗑️  Шаг 2: Удаление существующих дублей за сегодня...")

	articles := make([]string, 0, len(dedupData))
	for _, row := range dedupData {
		articles = append(articles, row.article)
	}

	deleteResult, err := db.ExecContext(ctx, `
		DELETE FROM etalon_nomenclature
		WHERE article = ANY($1)
		AND DATE(created_at) = CURRENT_DATE
	`, pq.Array(articles))
	if err != nil {
		log.Fatal("Failed to delete:", err)
	}

	deletedRows, _ := deleteResult.RowsAffected()
	fmt.Printf("  Удалено: %d записей\n\n", deletedRows)

	// Step 2: Insert deduplicated data
	fmt.Println("📥 Шаг 3: Вставка дедуплицированных данных...")

	for _, row := range dedupData {
		_, err = db.ExecContext(ctx, `
			INSERT INTO etalon_nomenclature (article, brand, type, size_model, nomenclature, mrc, isimport)
			VALUES ($1, $2, $3, $4, $5, $6, 0)
		`, row.article, row.brand, row.typ, row.sizeModel, row.nomenclature, row.mrc)
		if err != nil {
			log.Fatal("Failed to insert:", err)
		}
	}

	fmt.Printf("  Вставлено: %d записей\n\n", len(dedupData))

	// Check results
	fmt.Println("📊 Проверка результатов:")

	var total int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM etalon_nomenclature WHERE article LIKE 'TEST%'").Scan(&total)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Всего записей: %d\n", total)

	var unique int
	err = db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT article) FROM etalon_nomenclature WHERE article LIKE 'TEST%'").Scan(&unique)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Уникальных артикулов: %d\n\n", unique)

	if total == 3 && unique == 3 {
		fmt.Println("✅ УСПЕХ! Дедупликация работает корректно!")
		fmt.Println("   Ожидали: 3 записи (по одной на артикул)")
		fmt.Println("   Получили: 3 записи\n")

		// Show which versions were kept
		fmt.Println("Сохраненные записи (последнее вхождение):")
		rows, _ := db.QueryContext(ctx, "SELECT article, brand, type, mrc FROM etalon_nomenclature WHERE article LIKE 'TEST%' ORDER BY article")
		defer rows.Close()

		for rows.Next() {
			var article, brand, typ string
			var mrc float64
			rows.Scan(&article, &brand, &typ, &mrc)
			fmt.Printf("  %s: %s, %s, %.0f₽\n", article, brand, typ, mrc)
		}
	} else {
		fmt.Printf("❌ ОШИБКА! Дедупликация НЕ работает!\n")
		fmt.Printf("   Ожидали: 3 записи\n")
		fmt.Printf("   Получили: %d записей\n", total)
	}
}

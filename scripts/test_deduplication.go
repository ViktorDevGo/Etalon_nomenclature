package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/prokoleso/etalon-nomenclature/internal/db"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	dsn := "postgresql://gen_user:uzShH%3CA8S%3B7c.e@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer sqlDB.Close()

	// Create a simple database wrapper that implements the interface we need
	type testDB struct {
		*sql.DB
		logger *zap.Logger
	}

	database := &testDB{DB: sqlDB, logger: logger}

	ctx := context.Background()

	// Clear existing data
	fmt.Println("🧹 Очистка etalon_nomenclature...")
	_, err = database.Exec("DELETE FROM etalon_nomenclature")
	if err != nil {
		log.Fatal("Failed to clear:", err)
	}

	// Create test data with duplicates
	rows := []db.NomenclatureRow{
		{Article: "TEST001", Brand: "Brand1", Type: "Type1", SizeModel: "195/65R15", Nomenclature: "Nom1", MRC: 5000},
		{Article: "TEST002", Brand: "Brand2", Type: "Type2", SizeModel: "205/55R16", Nomenclature: "Nom2", MRC: 6000},
		{Article: "TEST001", Brand: "Brand1-Updated", Type: "Type1-Updated", SizeModel: "195/65R15", Nomenclature: "Nom1-Updated", MRC: 5500}, // Duplicate!
		{Article: "TEST003", Brand: "Brand3", Type: "Type3", SizeModel: "225/45R17", Nomenclature: "Nom3", MRC: 7000},
		{Article: "TEST002", Brand: "Brand2-Updated", Type: "Type2-Updated", SizeModel: "205/55R16", Nomenclature: "Nom2-Updated", MRC: 6500}, // Duplicate!
		{Article: "TEST001", Brand: "Brand1-Final", Type: "Type1-Final", SizeModel: "195/65R15", Nomenclature: "Nom1-Final", MRC: 5800},      // Duplicate again!
	}

	fmt.Printf("\n📦 Тестовые данные:\n")
	fmt.Printf("  Всего строк: %d\n", len(rows))
	fmt.Printf("  Уникальных артикулов: 3 (TEST001, TEST002, TEST003)\n")
	fmt.Printf("  Дубликатов: 3 (TEST001 x3, TEST002 x2, TEST003 x1)\n\n")

	// Insert with deduplication
	err = database.InsertNomenclatureWithEmail(ctx, rows, "<test@example.com>")
	if err != nil {
		log.Fatal("Failed to insert:", err)
	}

	// Check results
	fmt.Println("\n📊 Проверка результатов:")

	var total int
	err = database.QueryRow("SELECT COUNT(*) FROM etalon_nomenclature").Scan(&total)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Всего записей: %d\n", total)

	var unique int
	err = database.QueryRow("SELECT COUNT(DISTINCT article) FROM etalon_nomenclature").Scan(&unique)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Уникальных артикулов: %d\n\n", unique)

	if total == 3 && unique == 3 {
		fmt.Println("✅ УСПЕХ! Дедупликация работает корректно!")
		fmt.Println("   Ожидали: 3 записи (по одной на артикул)")
		fmt.Println("   Получили: 3 записи\n")

		// Show which versions were kept (should be last occurrence)
		fmt.Println("Сохраненные записи (последнее вхождение каждого артикула):")
		rows, _ := database.Query("SELECT article, brand, type, mrc FROM etalon_nomenclature ORDER BY article")
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

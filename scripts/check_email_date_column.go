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

	fmt.Println("📋 Проверка столбца email_date в таблицах:")
	fmt.Println("==========================================\n")

	// Check etalon_nomenclature
	var hasEmailDateNom bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'etalon_nomenclature' AND column_name = 'email_date'
		)
	`).Scan(&hasEmailDateNom)
	if err != nil {
		log.Fatal(err)
	}

	if hasEmailDateNom {
		fmt.Println("✅ etalon_nomenclature.email_date - добавлен")
	} else {
		fmt.Println("❌ etalon_nomenclature.email_date - отсутствует")
	}

	// Check price_tires
	var hasEmailDatePrice bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'price_tires' AND column_name = 'email_date'
		)
	`).Scan(&hasEmailDatePrice)
	if err != nil {
		log.Fatal(err)
	}

	if hasEmailDatePrice {
		fmt.Println("✅ price_tires.email_date - добавлен")
	} else {
		fmt.Println("❌ price_tires.email_date - отсутствует")
	}

	fmt.Println("\n==========================================")

	if hasEmailDateNom && hasEmailDatePrice {
		fmt.Println("✅ ВСЕ СТОЛБЦЫ УСПЕШНО ДОБАВЛЕНЫ!")
	} else {
		fmt.Println("⚠️  Некоторые столбцы не добавлены")
	}
}

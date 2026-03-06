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

	fmt.Println("📊 ПОЛНЫЙ АНАЛИЗ БАЗЫ ДАННЫХ")
	fmt.Println("=" + string(make([]byte, 80)))
	fmt.Println()

	// ============ ETALON_NOMENCLATURE ============
	fmt.Println("📋 ТАБЛИЦА: etalon_nomenclature")
	fmt.Println(string(make([]byte, 80)))

	var totalNom int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM etalon_nomenclature").Scan(&totalNom)
	fmt.Printf("📦 Всего записей: %d\n\n", totalNom)

	if totalNom > 0 {
		// По датам
		fmt.Println("📅 По датам получения:")
		fmt.Println(string(make([]byte, 80)))
		rows, _ := db.QueryContext(ctx, `
			SELECT
				CASE
					WHEN email_date IS NULL THEN 'NULL (старые данные)'
					ELSE TO_CHAR(email_date, 'YYYY-MM-DD')
				END as date,
				COUNT(*) as count
			FROM etalon_nomenclature
			GROUP BY date
			ORDER BY date DESC NULLS LAST
		`)
		defer rows.Close()
		for rows.Next() {
			var date string
			var count int
			rows.Scan(&date, &count)
			percentage := float64(count) / float64(totalNom) * 100
			fmt.Printf("  %-25s: %7d записей (%.1f%%)\n", date, count, percentage)
		}

		// Уникальные артикулы
		var uniqueArticles int
		db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT article) FROM etalon_nomenclature").Scan(&uniqueArticles)

		// Уникальные бренды
		var uniqueBrands int
		db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT brand) FROM etalon_nomenclature WHERE brand != ''").Scan(&uniqueBrands)

		fmt.Println("\n📝 Дополнительная информация:")
		fmt.Println(string(make([]byte, 80)))
		fmt.Printf("  Уникальных артикулов: %d\n", uniqueArticles)
		fmt.Printf("  Уникальных брендов: %d\n", uniqueBrands)
		fmt.Printf("  Среднее записей на артикул: %.2f\n", float64(totalNom)/float64(uniqueArticles))

		// Топ брендов
		fmt.Println("\n🏆 Топ-10 брендов:")
		fmt.Println(string(make([]byte, 80)))
		rows2, _ := db.QueryContext(ctx, `
			SELECT brand, COUNT(*) as count
			FROM etalon_nomenclature
			WHERE brand != ''
			GROUP BY brand
			ORDER BY count DESC
			LIMIT 10
		`)
		defer rows2.Close()
		for rows2.Next() {
			var brand string
			var count int
			rows2.Scan(&brand, &count)
			fmt.Printf("  %-30s: %5d позиций\n", brand, count)
		}

		// Статистика МРЦ
		var minMRC, maxMRC, avgMRC float64
		db.QueryRowContext(ctx, `
			SELECT MIN(mrc), MAX(mrc), AVG(mrc)
			FROM etalon_nomenclature
			WHERE mrc > 0
		`).Scan(&minMRC, &maxMRC, &avgMRC)

		fmt.Println("\n💰 Диапазон МРЦ:")
		fmt.Println(string(make([]byte, 80)))
		fmt.Printf("  Минимальная: %.2f₽\n", minMRC)
		fmt.Printf("  Максимальная: %.2f₽\n", maxMRC)
		fmt.Printf("  Средняя: %.2f₽\n", avgMRC)

		// Sample records
		fmt.Println("\n📄 Примеры записей (первые 5):")
		fmt.Println(string(make([]byte, 80)))
		rows3, _ := db.QueryContext(ctx, `
			SELECT article, brand, type, size_model, nomenclature, mrc
			FROM etalon_nomenclature
			LIMIT 5
		`)
		defer rows3.Close()
		for rows3.Next() {
			var article, brand, typ, sizeModel, nomenclature string
			var mrc float64
			rows3.Scan(&article, &brand, &typ, &sizeModel, &nomenclature, &mrc)
			fmt.Printf("\n  Артикул: %s\n", article)
			fmt.Printf("  Бренд: %s | Тип: %s | Размер: %s\n", brand, typ, sizeModel)
			fmt.Printf("  Номенклатура: %s\n", nomenclature)
			fmt.Printf("  МРЦ: %.2f₽\n", mrc)
		}
	}

	fmt.Println("\n" + string(make([]byte, 80)))
	fmt.Println()

	// ============ PRICE_TIRES ============
	fmt.Println("📋 ТАБЛИЦА: price_tires")
	fmt.Println(string(make([]byte, 80)))

	var totalPrice int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires").Scan(&totalPrice)
	fmt.Printf("📦 Всего записей: %d\n\n", totalPrice)

	if totalPrice > 0 {
		// По поставщикам
		fmt.Println("📈 По поставщикам:")
		fmt.Println(string(make([]byte, 80)))
		rows4, _ := db.QueryContext(ctx, `
			SELECT provider, COUNT(*) as count
			FROM price_tires
			GROUP BY provider
			ORDER BY count DESC
		`)
		defer rows4.Close()
		for rows4.Next() {
			var provider string
			var count int
			rows4.Scan(&provider, &count)
			percentage := float64(count) / float64(totalPrice) * 100
			fmt.Printf("  %-20s: %7d записей (%.1f%%)\n", provider, count, percentage)
		}

		// По датам
		fmt.Println("\n📅 По датам получения:")
		fmt.Println(string(make([]byte, 80)))
		rows5, _ := db.QueryContext(ctx, `
			SELECT
				CASE
					WHEN email_date IS NULL THEN 'NULL (старые данные)'
					ELSE TO_CHAR(email_date, 'YYYY-MM-DD')
				END as date,
				COUNT(*) as count
			FROM price_tires
			GROUP BY date
			ORDER BY date DESC NULLS LAST
			LIMIT 10
		`)
		defer rows5.Close()
		for rows5.Next() {
			var date string
			var count int
			rows5.Scan(&date, &count)
			fmt.Printf("  %s: %7d записей\n", date, count)
		}

		// Последние обновления по поставщикам
		fmt.Println("\n🔄 Последние обновления по поставщикам:")
		fmt.Println(string(make([]byte, 80)))
		rows6, _ := db.QueryContext(ctx, `
			SELECT
				provider,
				COALESCE(TO_CHAR(MAX(email_date), 'YYYY-MM-DD HH24:MI'), 'нет данных') as last_update,
				COUNT(*) as count
			FROM price_tires
			GROUP BY provider
			ORDER BY MAX(email_date) DESC NULLS LAST
		`)
		defer rows6.Close()
		for rows6.Next() {
			var provider, lastUpdate string
			var count int
			rows6.Scan(&provider, &lastUpdate, &count)
			fmt.Printf("  %-20s: %s (%d записей)\n", provider, lastUpdate, count)
		}

		// Уникальные артикулы
		var uniquePriceArticles int
		db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT article) FROM price_tires").Scan(&uniquePriceArticles)

		fmt.Println("\n📝 Дополнительная информация:")
		fmt.Println(string(make([]byte, 80)))
		fmt.Printf("  Уникальных артикулов: %d\n", uniquePriceArticles)
		fmt.Printf("  Среднее записей на артикул: %.2f\n", float64(totalPrice)/float64(uniquePriceArticles))

		// Склады БИГМАШИН
		var bigmCount int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires WHERE provider = 'БИГМАШИН'").Scan(&bigmCount)
		if bigmCount > 0 {
			fmt.Println("\n🏪 Склады БИГМАШИН:")
			fmt.Println(string(make([]byte, 80)))
			rows7, _ := db.QueryContext(ctx, `
				SELECT store, COUNT(*) as count
				FROM price_tires
				WHERE provider = 'БИГМАШИН'
				GROUP BY store
				ORDER BY count DESC
			`)
			defer rows7.Close()
			for rows7.Next() {
				var store string
				var count int
				rows7.Scan(&store, &count)
				fmt.Printf("  %-30s: %5d позиций\n", store, count)
			}
		}

		// Диапазон цен
		var minPrice, maxPrice, avgPrice float64
		db.QueryRowContext(ctx, `
			SELECT MIN(price), MAX(price), AVG(price)
			FROM price_tires
			WHERE price > 0
		`).Scan(&minPrice, &maxPrice, &avgPrice)

		fmt.Println("\n💰 Диапазон цен:")
		fmt.Println(string(make([]byte, 80)))
		fmt.Printf("  Минимальная: %.2f₽\n", minPrice)
		fmt.Printf("  Максимальная: %.2f₽\n", maxPrice)
		fmt.Printf("  Средняя: %.2f₽\n", avgPrice)

		// Проверка проблемных цен
		var badPriceCount int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires WHERE price > 500000").Scan(&badPriceCount)
		if badPriceCount > 0 {
			fmt.Printf("\n⚠️  ВНИМАНИЕ: Найдено %d записей с ценой > 500,000₽\n", badPriceCount)
		}

		// Проверка номенклатуры в артикуле
		var badArticleCount int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_tires WHERE article LIKE 'Автошина%'").Scan(&badArticleCount)
		if badArticleCount > 0 {
			fmt.Printf("⚠️  ВНИМАНИЕ: Найдено %d записей с номенклатурой в поле article\n", badArticleCount)
		}

		if badPriceCount == 0 && badArticleCount == 0 {
			fmt.Println("\n✅ Все данные корректны!")
		}

		// Sample records
		fmt.Println("\n📄 Примеры записей (первые 5 по каждому поставщику):")
		fmt.Println(string(make([]byte, 80)))
		rows8, _ := db.QueryContext(ctx, `
			WITH ranked AS (
				SELECT
					article, price, balance, store, provider,
					ROW_NUMBER() OVER (PARTITION BY provider ORDER BY article) as rn
				FROM price_tires
			)
			SELECT article, price, balance, store, provider
			FROM ranked
			WHERE rn <= 3
			ORDER BY provider, rn
		`)
		defer rows8.Close()
		currentProvider := ""
		for rows8.Next() {
			var article, store, provider string
			var price float64
			var balance int
			rows8.Scan(&article, &price, &balance, &store, &provider)

			if provider != currentProvider {
				fmt.Printf("\n  %s:\n", provider)
				currentProvider = provider
			}
			fmt.Printf("    %-25s | %10.2f₽ | %4d шт | %s\n", article, price, balance, store)
		}
	}

	// ============ PROCESSED_EMAILS ============
	fmt.Println("\n" + string(make([]byte, 80)))
	fmt.Println()
	fmt.Println("📋 ТАБЛИЦА: processed_emails")
	fmt.Println(string(make([]byte, 80)))

	var totalEmails int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM processed_emails").Scan(&totalEmails)
	fmt.Printf("📦 Всего обработано email: %d\n", totalEmails)

	if totalEmails > 0 {
		rows9, _ := db.QueryContext(ctx, `
			SELECT message_id, TO_CHAR(processed_at, 'YYYY-MM-DD HH24:MI:SS') as processed
			FROM processed_emails
			ORDER BY processed_at DESC
			LIMIT 10
		`)
		defer rows9.Close()
		fmt.Println("\nПоследние обработанные письма:")
		for rows9.Next() {
			var messageId, processed string
			rows9.Scan(&messageId, &processed)
			if len(messageId) > 50 {
				messageId = messageId[:47] + "..."
			}
			fmt.Printf("  %s - %s\n", processed, messageId)
		}
	}

	fmt.Println("\n" + string(make([]byte, 80)))
	fmt.Println("✅ Анализ завершен!")
}

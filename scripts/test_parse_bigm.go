package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/prokoleso/etalon-nomenclature/internal/parser"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run test_parse_bigm.go <file_path>")
	}

	filePath := os.Args[1]
	fmt.Printf("📄 Парсинг файла: %s\n", filePath)
	fmt.Println("=" + string(make([]byte, 80)))

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed to read file:", err)
	}

	// Create parser
	priceParser := parser.NewPriceParser(logger)

	// Parse
	fmt.Println("\n⚙️  Запускаю парсер...")
	rows, err := priceParser.Parse(content, "test.xls", "БИГМАШИН", time.Now())
	if err != nil {
		log.Fatalf("❌ Ошибка парсинга: %v", err)
	}

	fmt.Printf("\n✅ Успешно распарсено: %d строк\n", len(rows))

	// Show first 10 rows
	fmt.Println("\n📊 Первые 10 записей:")
	fmt.Println(string(make([]byte, 120)))
	fmt.Printf("%-25s | %-10s | %-8s | %-30s\n", "АРТИКУЛ", "ЦЕНА", "ОСТАТОК", "СКЛАД")
	fmt.Println(string(make([]byte, 120)))

	maxRows := 10
	if len(rows) < maxRows {
		maxRows = len(rows)
	}

	for i := 0; i < maxRows; i++ {
		row := rows[i]
		article := row.Article
		if len(article) > 25 {
			article = article[:22] + "..."
		}
		store := row.Store
		if len(store) > 30 {
			store = store[:27] + "..."
		}

		fmt.Printf("%-25s | %10.2f | %8d | %-30s\n",
			article, row.Price, row.Balance, store)
	}

	// Check for problems
	fmt.Println("\n🔍 Проверка на проблемы:")
	fmt.Println(string(make([]byte, 80)))

	badPriceCount := 0
	badArticleCount := 0
	goodCount := 0

	for _, row := range rows {
		// Check price
		if row.Price > 500000 {
			badPriceCount++
			if badPriceCount <= 3 {
				fmt.Printf("  ⚠️  Высокая цена: %s = %.2f₽\n", row.Article, row.Price)
			}
		}

		// Check article
		if len(row.Article) > 30 || (len(row.Article) > 10 && row.Article[:8] == "Автошина") {
			badArticleCount++
			if badArticleCount <= 3 {
				fmt.Printf("  ⚠️  Номенклатура в article: '%s'\n", row.Article)
			}
		}

		if row.Price <= 500000 && len(row.Article) <= 30 {
			goodCount++
		}
	}

	fmt.Printf("\n📊 Статистика:\n")
	fmt.Printf("  ✅ Корректных: %d (%.1f%%)\n", goodCount, float64(goodCount)/float64(len(rows))*100)
	fmt.Printf("  ⚠️  Проблемных цен (>500k): %d\n", badPriceCount)
	fmt.Printf("  ⚠️  Номенклатур в article: %d\n", badArticleCount)

	if badPriceCount == 0 && badArticleCount == 0 {
		fmt.Println("\n🎉 ВСЕ ОТЛИЧНО! Все данные корректны!")
	} else {
		fmt.Println("\n❌ ПРОБЛЕМА! Найдены некорректные данные")
	}

	fmt.Println("\n" + string(make([]byte, 80)))
}

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/prokoleso/etalon-nomenclature/internal/parser"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run check_other_sheets.go <file_path>")
	}

	filePath := os.Args[1]
	fmt.Printf("📄 Проверка всех листов: %s\n", filePath)
	fmt.Println("=" + strings.Repeat("=", 79))

	// Create logger (silent)
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	logger, _ := config.Build()
	defer logger.Sync()

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed to read file:", err)
	}

	// Parse (it will handle xls conversion internally)
	priceParser := parser.NewPriceParser(logger)
	rows, err := priceParser.Parse(content, "test.xls", "БИГМАШИН", time.Now())
	if err != nil {
		log.Fatalf("❌ Ошибка парсинга: %v", err)
	}

	// Group by first word in article to identify patterns
	fmt.Printf("\n✅ Всего распарсено: %d строк\n", len(rows))

	// Check for problems
	problemCount := 0
	problems := make(map[string]int)

	for _, row := range rows {
		article := strings.TrimSpace(row.Article)

		// Check if looks like nomenclature
		if strings.HasPrefix(strings.ToLower(article), "автошина") ||
			strings.HasPrefix(strings.ToLower(article), "а/ш") ||
			strings.HasPrefix(strings.ToLower(article), "на r") ||
			strings.HasPrefix(strings.ToLower(article), "шина") ||
			len(article) > 60 {
			problemCount++

			// Truncate for display
			display := article
			if len(display) > 60 {
				display = display[:57] + "..."
			}
			problems[display]++
		}
	}

	fmt.Printf("⚠️  Проблемных артикулов: %d\n", problemCount)

	if problemCount > 0 {
		fmt.Println("\n🔍 ПРИМЕРЫ ПРОБЛЕМНЫХ АРТИКУЛОВ:")
		fmt.Println(strings.Repeat("-", 80))

		count := 0
		for article, occurrences := range problems {
			if count >= 15 {
				break
			}
			fmt.Printf("  [%2d раз] %s\n", occurrences, article)
			count++
		}

		fmt.Println("\n❌ ПРОБЛЕМА: Парсер пропустил эти строки!")
	} else {
		fmt.Println("\n🎉 ВСЕ ОТЛИЧНО! Нет проблемных артикулов!")
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

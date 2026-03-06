package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/prokoleso/etalon-nomenclature/internal/parser"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run compare_parsed_vs_original.go <file_path>")
	}

	filePath := os.Args[1]
	fmt.Printf("📄 Детальное сравнение: %s\n", filePath)
	fmt.Println("=" + strings.Repeat("=", 79))

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed to read file:", err)
	}

	// Parse with our parser
	fmt.Println("\n⚙️  ПАРСИНГ С НАШИМ ПАРСЕРОМ")
	fmt.Println(strings.Repeat("-", 80))

	priceParser := parser.NewPriceParser(logger)
	parsedRows, err := priceParser.Parse(content, "test.xls", "БИГМАШИН", time.Now())
	if err != nil {
		log.Fatalf("❌ Ошибка парсинга: %v", err)
	}

	fmt.Printf("✅ Успешно распарсено: %d строк\n", len(parsedRows))

	// Collect unique articles from parsed data
	parsedArticles := make(map[string]bool)
	problemArticles := make(map[string]int) // article -> count

	for _, row := range parsedRows {
		parsedArticles[row.Article] = true

		// Check if article looks like nomenclature
		article := strings.TrimSpace(row.Article)
		if strings.HasPrefix(strings.ToLower(article), "автошина") ||
			strings.HasPrefix(strings.ToLower(article), "а/ш") ||
			strings.HasPrefix(strings.ToLower(article), "на r") ||
			len(article) > 60 {
			problemArticles[article]++
		}
	}

	fmt.Printf("📊 Уникальных артикулов: %d\n", len(parsedArticles))
	fmt.Printf("⚠️  Проблемных артикулов (номенклатура): %d\n", len(problemArticles))

	if len(problemArticles) > 0 {
		fmt.Println("\n🔍 ПРИМЕРЫ ПРОБЛЕМНЫХ АРТИКУЛОВ (распарсено нашим парсером):")
		fmt.Println(strings.Repeat("-", 80))
		count := 0
		for article, occurrences := range problemArticles {
			if count >= 10 {
				break
			}
			displayArticle := article
			if len(displayArticle) > 70 {
				displayArticle = displayArticle[:67] + "..."
			}
			fmt.Printf("  [%2d раз] %s\n", occurrences, displayArticle)
			count++
		}
	}

	// Now read Excel directly and compare
	fmt.Println("\n⚙️  ЧТЕНИЕ ОРИГИНАЛЬНОГО ФАЙЛА")
	fmt.Println(strings.Repeat("-", 80))

	converted, err := parser.ConvertXLStoXLSX(content)
	if err != nil {
		log.Fatal("Failed to convert:", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(converted))
	if err != nil {
		log.Fatal("Failed to open:", err)
	}
	defer f.Close()

	// Analyze "Зимние" sheet
	sheetName := "Зимние"
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("📋 Лист '%s': %d строк\n", sheetName, len(rows))

	// Find header row (should be at row 11, index 10)
	headerRow := -1
	articleCol := -1
	priceCol := -1
	nomenclatureCol := -1

	for i := 0; i < 20 && i < len(rows); i++ {
		for j, col := range rows[i] {
			normalized := strings.ToLower(strings.TrimSpace(col))
			if strings.Contains(normalized, "артикул производителя") ||
			   (strings.Contains(normalized, "артикул") && !strings.Contains(normalized, "поставщика")) {
				headerRow = i
				articleCol = j
			}
			if strings.Contains(normalized, "оптовая") && strings.Contains(normalized, "цена") {
				priceCol = j
			}
			if strings.Contains(normalized, "номенклатура") {
				nomenclatureCol = j
			}
		}
		if headerRow >= 0 && articleCol >= 0 && priceCol >= 0 {
			break
		}
	}

	if headerRow < 0 {
		log.Fatal("Header row not found")
	}

	fmt.Printf("📍 Заголовки найдены в строке %d:\n", headerRow+1)
	fmt.Printf("   Артикул: колонка [%d]\n", articleCol)
	fmt.Printf("   Номенклатура: колонка [%d]\n", nomenclatureCol)
	fmt.Printf("   Цена: колонка [%d]\n", priceCol)

	// Now check data rows and compare
	fmt.Println("\n🔍 СРАВНЕНИЕ: ЧТО В ОРИГИНАЛЕ vs ЧТО РАСПАРСИЛОСЬ")
	fmt.Println(strings.Repeat("-", 80))

	// Look for rows where article column contains nomenclature
	problemCount := 0
	for i := headerRow + 1; i < len(rows) && problemCount < 15; i++ {
		if len(rows[i]) <= articleCol {
			continue
		}

		article := strings.TrimSpace(rows[i][articleCol])
		if article == "" {
			continue
		}

		// Check if article looks like nomenclature
		isNomenclature := strings.HasPrefix(strings.ToLower(article), "автошина") ||
			strings.HasPrefix(strings.ToLower(article), "а/ш") ||
			strings.HasPrefix(strings.ToLower(article), "на r") ||
			len(article) > 60

		// Or check if it's in our problem list
		_, isParsedProblem := problemArticles[article]

		if isNomenclature || isParsedProblem {
			problemCount++

			fmt.Printf("\n📌 Строка %d в оригинальном файле:\n", i+1)

			// Show all relevant columns
			if articleCol < len(rows[i]) {
				val := rows[i][articleCol]
				if len(val) > 60 {
					val = val[:57] + "..."
				}
				fmt.Printf("   [%d] Артикул: '%s'\n", articleCol, val)
			}

			if nomenclatureCol >= 0 && nomenclatureCol < len(rows[i]) {
				val := rows[i][nomenclatureCol]
				if len(val) > 60 {
					val = val[:57] + "..."
				}
				fmt.Printf("   [%d] Номенклатура: '%s'\n", nomenclatureCol, val)
			}

			if priceCol < len(rows[i]) {
				fmt.Printf("   [%d] Цена: '%s'\n", priceCol, rows[i][priceCol])
			}

			// Was it parsed?
			if parsedArticles[article] {
				fmt.Printf("   ❌ ПРОБЛЕМА: Эта строка ПРОШЛА валидацию и попала в БД!\n")
			} else {
				fmt.Printf("   ✅ Эта строка отфильтрована парсером\n")
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ Сравнение завершено")
}

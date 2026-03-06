package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/prokoleso/etalon-nomenclature/internal/parser"
	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run check_letnie_sheet.go <file_path>")
	}

	filePath := os.Args[1]
	fmt.Printf("📄 Анализ листа 'Летние': %s\n", filePath)
	fmt.Println("=" + strings.Repeat("=", 79))

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed to read file:", err)
	}

	// Convert
	converted, err := parser.ConvertXLStoXLSX(content)
	if err != nil {
		log.Fatal("Failed to convert:", err)
	}

	// Open
	f, err := excelize.OpenReader(bytes.NewReader(converted))
	if err != nil {
		log.Fatal("Failed to open:", err)
	}
	defer f.Close()

	sheetName := "Летние"
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n📋 Лист '%s': %d строк\n", sheetName, len(rows))

	// Find header
	headerRow := -1
	articleCol := -1

	for i := 0; i < 20 && i < len(rows); i++ {
		for j, col := range rows[i] {
			normalized := strings.ToLower(strings.TrimSpace(col))
			if strings.Contains(normalized, "артикул") {
				headerRow = i
				articleCol = j
				break
			}
		}
		if headerRow >= 0 {
			break
		}
	}

	if headerRow < 0 {
		log.Fatal("Header not found")
	}

	fmt.Printf("📍 Заголовки в строке %d, артикул в колонке [%d]\n", headerRow+1, articleCol)

	// Find rows with "Автошина" in article column
	fmt.Println("\n🔍 Строки где артикул начинается с 'Автошина':")
	fmt.Println(strings.Repeat("-", 80))

	count := 0
	for i := headerRow + 1; i < len(rows) && count < 15; i++ {
		if len(rows[i]) <= articleCol {
			continue
		}

		article := strings.TrimSpace(rows[i][articleCol])
		if strings.HasPrefix(strings.ToLower(article), "автошина") {
			count++
			displayArticle := article
			if len(displayArticle) > 65 {
				displayArticle = displayArticle[:62] + "..."
			}
			fmt.Printf("  Строка %4d: '%s'\n", i+1, displayArticle)
		}
	}

	if count == 0 {
		fmt.Println("  ✅ Не найдено!")
	} else {
		fmt.Printf("\n❌ Найдено %d строк с 'Автошина' в артикуле\n", count)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

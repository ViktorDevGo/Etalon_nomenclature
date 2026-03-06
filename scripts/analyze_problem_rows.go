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
		log.Fatal("Usage: go run analyze_problem_rows.go <file_path>")
	}

	filePath := os.Args[1]

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed to read file:", err)
	}

	// Convert xls to xlsx
	converted, err := parser.ConvertXLStoXLSX(content)
	if err != nil {
		log.Fatal("Failed to convert:", err)
	}

	// Open xlsx
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

	fmt.Printf("📄 Анализ проблемных строк в листе '%s'\n", sheetName)
	fmt.Println("=" + strings.Repeat("=", 79))

	// Headers should be at row 11 (index 10)
	fmt.Println("\n📋 Заголовочная строка (строка 11):")
	if len(rows) > 10 {
		for i, col := range rows[10] {
			if i > 10 {
				break
			}
			fmt.Printf("  [%2d] '%s'\n", i, col)
		}
	}

	// Check rows around line 150 (where errors occur)
	// Also check some earlier rows to see when the shift starts
	problemRows := []int{11, 12, 13, 50, 100, 145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 164}

	fmt.Println("\n🔍 Проблемные строки (примеры из логов):")
	fmt.Println(strings.Repeat("-", 80))

	for _, rowIdx := range problemRows {
		if rowIdx >= len(rows) {
			continue
		}

		row := rows[rowIdx]
		fmt.Printf("\n📌 Строка %d (%d колонок):\n", rowIdx+1, len(row))

		// Show first 8 columns
		maxCols := 8
		if len(row) < maxCols {
			maxCols = len(row)
		}

		for i := 0; i < maxCols; i++ {
			value := row[i]
			if len(value) > 60 {
				value = value[:57] + "..."
			}
			value = strings.ReplaceAll(value, "\n", "\\n")

			marker := ""
			switch i {
			case 0:
				marker = " <- article"
			case 1:
				marker = " <- nomenclature"
			case 5:
				marker = " <- price"
			}

			fmt.Printf("  [%d] '%s'%s\n", i, value, marker)
		}

		// Analyze structure
		isEmpty := len(row) == 0 || (len(row) > 0 && strings.TrimSpace(row[0]) == "")
		hasFewerCols := len(row) < 8

		if isEmpty {
			fmt.Println("  ⚠️  Пустая строка")
		} else if hasFewerCols {
			fmt.Printf("  ⚠️  Мало колонок: %d (ожидается >= 8)\n", len(row))
		} else if len(row) > 0 && strings.Contains(strings.ToLower(row[0]), "автошина") {
			fmt.Println("  ⚠️  Номенклатура в колонке [0] вместо артикула!")
		} else if len(row) > 1 && len(row[1]) < 10 {
			fmt.Println("  ⚠️  Подзаголовок? (короткая номенклатура)")
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

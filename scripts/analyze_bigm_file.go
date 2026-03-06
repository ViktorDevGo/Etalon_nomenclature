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
		log.Fatal("Usage: go run analyze_bigm_file.go <file_path>")
	}

	filePath := os.Args[1]
	fmt.Printf("📄 Анализ файла: %s\n", filePath)
	fmt.Println("=" + strings.Repeat("=", 79))

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed to read file:", err)
	}

	// Convert xls to xlsx
	fmt.Println("\n🔄 Конвертирую .xls в .xlsx...")
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

	// Get sheets
	sheets := f.GetSheetList()
	fmt.Printf("✅ Листов: %d\n", len(sheets))
	for i, sheet := range sheets {
		fmt.Printf("  %d. %s\n", i+1, sheet)
	}

	// Analyze first tire sheet
	var tireSheet string
	for _, sheet := range sheets {
		normalized := strings.ToLower(strings.TrimSpace(sheet))
		if normalized == "зимние" || normalized == "летние" {
			tireSheet = sheet
			break
		}
	}

	if tireSheet == "" {
		log.Fatal("No tire sheet found")
	}

	fmt.Printf("\n📋 Анализирую лист: '%s'\n", tireSheet)
	fmt.Println(strings.Repeat("-", 80))

	rows, err := f.GetRows(tireSheet)
	if err != nil {
		log.Fatal(err)
	}

	// Show first 15 rows
	fmt.Printf("\nПервые 15 строк:\n\n")
	for i := 0; i < 15 && i < len(rows); i++ {
		if len(rows[i]) == 0 {
			fmt.Printf("  Строка %2d: [пустая]\n", i+1)
			continue
		}

		fmt.Printf("  Строка %2d (%d колонок):\n", i+1, len(rows[i]))

		// Show up to 12 columns
		maxCols := 12
		if len(rows[i]) < maxCols {
			maxCols = len(rows[i])
		}

		for j := 0; j < maxCols; j++ {
			value := rows[i][j]
			if len(value) > 50 {
				value = value[:47] + "..."
			}
			// Replace newlines with \n for display
			value = strings.ReplaceAll(value, "\n", "\\n")
			fmt.Printf("    [%2d] '%s'\n", j, value)
		}

		if len(rows[i]) > maxCols {
			fmt.Printf("    ... еще %d колонок\n", len(rows[i])-maxCols)
		}
		fmt.Println()
	}

	// Find header row
	fmt.Println("🔎 Поиск заголовочной строки:")
	fmt.Println(strings.Repeat("-", 80))

	for i := 0; i < 20 && i < len(rows); i++ {
		hasArticle := false
		hasPrice := false
		hasBalance := false

		articleCol := -1
		priceCol := -1
		balanceCols := []int{}

		for j, col := range rows[i] {
			normalized := strings.ToLower(strings.TrimSpace(col))

			if strings.Contains(normalized, "артикул") {
				hasArticle = true
				articleCol = j
			}
			if strings.Contains(normalized, "оптовая") && strings.Contains(normalized, "цена") {
				hasPrice = true
				priceCol = j
			}
			if strings.Contains(normalized, "остаток") {
				hasBalance = true
				balanceCols = append(balanceCols, j)
			}
		}

		if hasArticle || hasPrice || hasBalance {
			fmt.Printf("\nСтрока %d:\n", i+1)
			if hasArticle {
				fmt.Printf("  ✓ Артикул в колонке [%d]: '%s'\n", articleCol, rows[i][articleCol])
			}
			if hasPrice {
				fmt.Printf("  ✓ Цена в колонке [%d]: '%s'\n", priceCol, rows[i][priceCol])
			}
			if hasBalance {
				fmt.Printf("  ✓ Остаток в колонках: %v\n", balanceCols)
				for _, idx := range balanceCols {
					val := rows[i][idx]
					val = strings.ReplaceAll(val, "\n", "\\n")
					fmt.Printf("      [%d] '%s'\n", idx, val)
				}
			}
		}

		if hasArticle && hasPrice && hasBalance {
			fmt.Printf("\n✅ ПОЛНЫЙ ЗАГОЛОВОК НАЙДЕН В СТРОКЕ %d\n", i+1)

			// Show data rows after header
			fmt.Printf("\nПримеры данных (строки %d-%d):\n", i+2, i+6)
			fmt.Println(strings.Repeat("-", 80))

			for k := i + 1; k < i + 6 && k < len(rows); k++ {
				if len(rows[k]) <= articleCol || len(rows[k]) <= priceCol {
					continue
				}

				article := rows[k][articleCol]
				price := rows[k][priceCol]

				fmt.Printf("\nСтрока %d:\n", k+1)
				fmt.Printf("  Артикул [%d]: '%s'\n", articleCol, article)
				fmt.Printf("  Цена [%d]: '%s'\n", priceCol, price)

				for _, bIdx := range balanceCols {
					if bIdx < len(rows[k]) {
						fmt.Printf("  Остаток [%d]: '%s'\n", bIdx, rows[k][bIdx])
					}
				}
			}

			break
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ Анализ завершен")
}

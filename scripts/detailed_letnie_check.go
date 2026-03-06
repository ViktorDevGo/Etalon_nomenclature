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
		log.Fatal("Usage: go run detailed_letnie_check.go <file_path>")
	}

	filePath := os.Args[1]
	fmt.Printf("📄 ДЕТАЛЬНАЯ ПРОВЕРКА листа 'Летние': %s\n", filePath)
	fmt.Println("=" + strings.Repeat("=", 79))

	// Read and convert
	content, _ := os.ReadFile(filePath)
	converted, _ := parser.ConvertXLStoXLSX(content)
	f, _ := excelize.OpenReader(bytes.NewReader(converted))
	defer f.Close()

	sheetName := "Летние"
	rows, _ := f.GetRows(sheetName)

	// Find header
	headerRow := -1
	for i := 0; i < 20 && i < len(rows); i++ {
		for _, col := range rows[i] {
			normalized := strings.ToLower(strings.TrimSpace(col))
			if strings.Contains(normalized, "артикул производителя") {
				headerRow = i
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

	fmt.Printf("\n📋 Заголовочная строка (строка %d):\n", headerRow+1)
	for i := 0; i < 8 && i < len(rows[headerRow]); i++ {
		fmt.Printf("  [%d] '%s'\n", i, rows[headerRow][i])
	}

	// Show several data rows
	fmt.Printf("\n📊 Примеры строк данных (15-25):\n")
	fmt.Println(strings.Repeat("-", 80))

	for i := 15; i < 25 && i < len(rows); i++ {
		fmt.Printf("\nСтрока %d:\n", i+1)

		// Show first 8 columns
		for j := 0; j < 8 && j < len(rows[i]); j++ {
			val := rows[i][j]
			if len(val) > 50 {
				val = val[:47] + "..."
			}

			marker := ""
			switch j {
			case 0:
				marker = " <- Артикул"
			case 1:
				marker = " <- Номенклатура"
			case 5:
				marker = " <- Цена"
			}

			fmt.Printf("  [%d] '%s'%s\n", j, val, marker)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

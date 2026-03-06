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
		log.Fatal("Usage: go run find_row_18.go <file_path>")
	}

	filePath := os.Args[1]
	fmt.Printf("🔍 Поиск строки 18 с артикулом 2PO1954H1\n")
	fmt.Println("=" + strings.Repeat("=", 79))

	content, _ := os.ReadFile(filePath)
	converted, _ := parser.ConvertXLStoXLSX(content)
	f, _ := excelize.OpenReader(bytes.NewReader(converted))
	defer f.Close()

	sheetName := "Летние"
	rows, _ := f.GetRows(sheetName)

	// Find row with article "2PO1954H1"
	fmt.Println("\n🔍 Ищу строку с артикулом '2PO1954H1':")
	for i := 0; i < len(rows); i++ {
		for j := 0; j < len(rows[i]); j++ {
			if strings.Contains(rows[i][j], "2PO1954H1") {
				fmt.Printf("\n✅ Найдено в массиве: rows[%d][%d] (Excel строка %d)\n", i, j, i+1)
				fmt.Printf("Показываю всю строку:\n")
				for k := 0; k < 8 && k < len(rows[i]); k++ {
					val := rows[i][k]
					if len(val) > 50 {
						val = val[:47] + "..."
					}
					fmt.Printf("  [%d] '%s'\n", k, val)
				}
				break
			}
		}
	}

	// Find row with article "PO1956H1"
	fmt.Println("\n🔍 Ищу строку с артикулом 'PO1956H1':")
	for i := 0; i < len(rows); i++ {
		for j := 0; j < len(rows[i]); j++ {
			if rows[i][j] == "PO1956H1" || strings.Contains(rows[i][j], "PO1956H1") {
				fmt.Printf("\n✅ Найдено в массиве: rows[%d][%d] (Excel строка %d)\n", i, j, i+1)
				fmt.Printf("Показываю всю строку:\n")
				for k := 0; k < 8 && k < len(rows[i]); k++ {
					val := rows[i][k]
					if len(val) > 50 {
						val = val[:47] + "..."
					}
					fmt.Printf("  [%d] '%s'\n", k, val)
				}
				break
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

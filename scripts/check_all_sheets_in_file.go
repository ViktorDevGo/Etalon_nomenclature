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
		log.Fatal("Usage: go run check_all_sheets_in_file.go <file_path>")
	}

	filePath := os.Args[1]
	fmt.Printf("📄 Проверка всех листов: %s\n", filePath)
	fmt.Println("=" + strings.Repeat("=", 79))

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed to read file:", err)
	}

	// Convert to xlsx
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

	sheets := f.GetSheetList()
	fmt.Printf("\n📋 Всего листов: %d\n", len(sheets))

	for _, sheet := range sheets {
		rows, _ := f.GetRows(sheet)
		fmt.Printf("\n📄 Лист: '%s' (%d строк)\n", sheet, len(rows))
	}

	fmt.Println("\n" + strings.Repeat("=", 79))
}

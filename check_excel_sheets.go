package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run check_excel_sheets.go <excel_file_path>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		fmt.Printf("Error opening Excel: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	fmt.Printf("\n📋 File: %s\n", filePath)
	fmt.Printf("Total sheets: %d\n\n", len(sheets))

	for i, sheetName := range sheets {
		fmt.Printf("%d. '%s'\n", i+1, sheetName)

		// Check if it would be processed by disk parser
		normalized := sheetName
		if normalized == "Автодиски" || normalized == "автодиски" {
			fmt.Printf("   ✅ БУДЕТ ОБРАБОТАН парсером дисков (БРИНЕКС)\n")
		} else {
			fmt.Printf("   ❌ НЕ будет обработан парсером дисков (БРИНЕКС)\n")
		}
	}
	fmt.Println()
}

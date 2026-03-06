package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	filename := "/Users/viktor/Desktop/Прайс-лист ЗАПАСКА от 04.03.2026.xlsx"

	f, err := excelize.OpenFile(filename)
	if err != nil {
		log.Fatal("Failed to open:", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	fmt.Printf("📋 Листы: %v\n\n", sheets)

	for _, sheetName := range sheets {
		fmt.Printf("=== Лист: %s ===\n", sheetName)

		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Printf("Error reading sheet %s: %v\n", sheetName, err)
			continue
		}

		fmt.Printf("Всего строк: %d\n\n", len(rows))

		// Show first 20 rows with full details
		for i := 0; i < 20 && i < len(rows); i++ {
			fmt.Printf("Строка %d (%d колонок): %v\n", i+1, len(rows[i]), rows[i])
		}

		fmt.Println()
	}
}

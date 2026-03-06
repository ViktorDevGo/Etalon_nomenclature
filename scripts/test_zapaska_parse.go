package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/prokoleso/etalon-nomenclature/internal/parser"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	filename := "/Users/viktor/Desktop/Прайс-лист ЗАПАСКА от 04.03.2026.xlsx"

	// Read file
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("Failed to open:", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatal("Failed to read:", err)
	}

	// Parse with PriceParser
	priceParser := parser.NewPriceParser(logger)
	rows, err := priceParser.Parse(content, filename, "ЗАПАСКА")
	if err != nil {
		log.Fatal("Failed to parse:", err)
	}

	fmt.Printf("\n✅ Успешно распарсено: %d строк\n\n", len(rows))

	// Show first 10 rows
	fmt.Println("Первые 10 строк:")
	for i := 0; i < 10 && i < len(rows); i++ {
		row := rows[i]
		fmt.Printf("%d. %s | %.2f₽ | остаток: %d | склад: %s | %s\n",
			i+1, row.Article, row.Price, row.Balance, row.Store, row.Provider)
	}

	// Show last 5 rows
	if len(rows) > 10 {
		fmt.Println("\nПоследние 5 строк:")
		for i := len(rows) - 5; i < len(rows); i++ {
			row := rows[i]
			fmt.Printf("%d. %s | %.2f₽ | остаток: %d | склад: %s | %s\n",
				i+1, row.Article, row.Price, row.Balance, row.Store, row.Provider)
		}
	}
}

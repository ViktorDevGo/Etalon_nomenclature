package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/prokoleso/etalon-nomenclature/config"
	"github.com/prokoleso/etalon-nomenclature/internal/imap"
	"github.com/prokoleso/etalon-nomenclature/internal/parser"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	mailbox := cfg.Mailboxes[0]
	fmt.Printf("📧 Подключаюсь к %s...\n", mailbox.Email)
	client := imap.NewClient(mailbox, logger)

	ctx := context.Background()
	emails, err := client.FetchTodayEmails(ctx)
	if err != nil {
		log.Fatal("Failed to fetch emails:", err)
	}

	// Find БИГМАШИН email and xls file
	for _, email := range emails {
		if email.From != "m.timoshenkova@bigm.pro" {
			continue
		}

		for _, attachment := range email.Attachments {
			filename := strings.ToLower(attachment.Filename)
			if !strings.Contains(filename, "прайс") || !strings.Contains(filename, "шины") {
				continue
			}

			fmt.Printf("\n📄 БИГМАШИН Файл: %s (%.2f KB)\n", attachment.Filename, float64(attachment.Size)/1024)
			fmt.Println(strings.Repeat("=", 80))

			// Convert xls to xlsx
			content := attachment.Content
			if strings.HasSuffix(filename, ".xls") && !strings.HasSuffix(filename, ".xlsx") {
				fmt.Println("🔄 Конвертирую .xls в .xlsx...")
				converted, err := parser.ConvertXLStoXLSX(content)
				if err != nil {
					log.Fatal("Failed to convert:", err)
				}
				content = converted
			}

			// Open Excel
			f, err := excelize.OpenReader(bytes.NewReader(content))
			if err != nil {
				log.Fatal("Failed to open:", err)
			}
			defer f.Close()

			sheets := f.GetSheetList()
			fmt.Printf("\n📊 Листов в файле: %d\n", len(sheets))
			for _, sheet := range sheets {
				fmt.Printf("  - %s\n", sheet)
			}

			// Analyze first sheet
			sheetName := sheets[0]
			fmt.Printf("\n📋 Анализ листа: '%s'\n", sheetName)
			fmt.Println(strings.Repeat("-", 80))

			rows, err := f.GetRows(sheetName)
			if err != nil {
				log.Fatal(err)
			}

			// Show first 15 rows
			maxRows := 15
			if len(rows) < maxRows {
				maxRows = len(rows)
			}

			fmt.Printf("\nПервые %d строк:\n\n", maxRows)
			for i := 0; i < maxRows; i++ {
				if len(rows[i]) == 0 {
					fmt.Printf("  Строка %2d: [пустая]\n", i+1)
					continue
				}

				fmt.Printf("  Строка %2d (%d колонок):\n", i+1, len(rows[i]))

				// Show all columns
				maxCols := 15
				if len(rows[i]) < maxCols {
					maxCols = len(rows[i])
				}

				for j := 0; j < maxCols; j++ {
					value := rows[i][j]
					if len(value) > 60 {
						value = value[:57] + "..."
					}
					fmt.Printf("    [%2d] '%s'\n", j, value)
				}

				if len(rows[i]) > maxCols {
					fmt.Printf("    ... и еще %d колонок\n", len(rows[i])-maxCols)
				}
				fmt.Println()
			}

			// Find header
			fmt.Println("🔎 Поиск заголовков:")
			fmt.Println(strings.Repeat("-", 80))
			for i := 0; i < 10 && i < len(rows); i++ {
				for j, col := range rows[i] {
					norm := strings.ToLower(strings.TrimSpace(col))
					if strings.Contains(norm, "артикул") {
						fmt.Printf("  Строка %d, колонка [%d]: 'Артикул' = '%s'\n", i+1, j, col)
					}
					if strings.Contains(norm, "номенклатура") {
						fmt.Printf("  Строка %d, колонка [%d]: 'Номенклатура' = '%s'\n", i+1, j, col)
					}
					if strings.Contains(norm, "цена") {
						fmt.Printf("  Строка %d, колонка [%d]: 'Цена' = '%s'\n", i+1, j, col)
					}
					if strings.Contains(norm, "остаток") {
						fmt.Printf("  Строка %d, колонка [%d]: 'Остаток' = '%s'\n", i+1, j, col)
					}
				}
			}

			return // Analyze only first БИГМАШИН file
		}
	}

	fmt.Println("❌ БИГМАШИН файл не найден")
}

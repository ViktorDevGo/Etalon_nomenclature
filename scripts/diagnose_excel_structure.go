package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/prokoleso/etalon-nomenclature/config"
	"github.com/prokoleso/etalon-nomenclature/internal/imap"
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

	fmt.Println("🔍 ДИАГНОСТИКА СТРУКТУРЫ EXCEL БИГМАШИН")
	fmt.Println("=" + string(make([]byte, 80)))
	fmt.Println()

	// Use first mailbox (all emails come there)
	if len(cfg.Mailboxes) == 0 {
		log.Fatal("No mailboxes configured")
	}

	mailbox := cfg.Mailboxes[0]
	fmt.Printf("📧 Подключаюсь к %s...\n", mailbox.Email)
	client := imap.NewClient(mailbox, logger)

	ctx := context.Background()
	emails, err := client.FetchTodayEmails(ctx)
	if err != nil {
		log.Fatal("Failed to fetch emails:", err)
	}

	fmt.Printf("✅ Найдено писем с вложениями: %d\n\n", len(emails))

	if len(emails) == 0 {
		fmt.Println("⚠️  Нет писем для анализа")
		return
	}

	// Analyze first email with price file
	for _, email := range emails {
		for _, attachment := range email.Attachments {
			filename := strings.ToLower(attachment.Filename)
			if !strings.Contains(filename, "прайс") {
				continue
			}

			fmt.Printf("📄 Файл: %s (%.2f KB)\n", attachment.Filename, float64(attachment.Size)/1024)
			fmt.Println(strings.Repeat("=", 80))

			// Open Excel
			f, err := excelize.OpenReader(bytes.NewReader(attachment.Content))
			if err != nil {
				fmt.Printf("❌ Ошибка открытия: %v\n", err)
				continue
			}
			defer f.Close()

			sheets := f.GetSheetList()
			fmt.Printf("\n📊 Листов в файле: %d\n", len(sheets))

			// Analyze first relevant sheet
			for _, sheetName := range sheets {
				normalized := strings.ToLower(strings.TrimSpace(sheetName))
				if normalized != "автошины" && normalized != "зимние" && normalized != "летние" {
					continue
				}

				fmt.Printf("\n📋 Лист: '%s'\n", sheetName)
				fmt.Println(strings.Repeat("-", 80))

				rows, err := f.GetRows(sheetName)
				if err != nil {
					fmt.Printf("❌ Ошибка чтения: %v\n", err)
					continue
				}

				// Show first 10 rows
				maxRows := 10
				if len(rows) < maxRows {
					maxRows = len(rows)
				}

				fmt.Printf("\nПервые %d строк:\n", maxRows)
				for i := 0; i < maxRows; i++ {
					if len(rows[i]) == 0 {
						fmt.Printf("  Строка %2d: [пустая]\n", i+1)
						continue
					}

					fmt.Printf("\n  Строка %2d (%d колонок):\n", i+1, len(rows[i]))

					// Show first 10 columns
					maxCols := 10
					if len(rows[i]) < maxCols {
						maxCols = len(rows[i])
					}

					for j := 0; j < maxCols; j++ {
						value := rows[i][j]
						if len(value) > 50 {
							value = value[:47] + "..."
						}
						fmt.Printf("    [%2d] = '%s'\n", j, value)
					}

					if len(rows[i]) > maxCols {
						fmt.Printf("    ... и еще %d колонок\n", len(rows[i])-maxCols)
					}
				}

				// Try to find header row
				fmt.Println("\n🔎 Поиск заголовков колонок:")
				for i := 0; i < maxRows && i < len(rows); i++ {
					hasArticle := false
					hasPrice := false
					hasBalance := false

					for j, col := range rows[i] {
						normalized := strings.ToLower(strings.TrimSpace(col))

						if strings.Contains(normalized, "артикул") {
							fmt.Printf("  Строка %d, колонка %d: 'Артикул' найден\n", i+1, j)
							hasArticle = true
						}
						if strings.Contains(normalized, "оптовая") ||
						   (strings.Contains(normalized, "цена") && !strings.Contains(normalized, "розн")) {
							fmt.Printf("  Строка %d, колонка %d: 'Цена' найдена ('%s')\n", i+1, j, col)
							hasPrice = true
						}
						if strings.Contains(normalized, "остаток") {
							fmt.Printf("  Строка %d, колонка %d: 'Остаток' найден ('%s')\n", i+1, j, col)
							hasBalance = true
						}
					}

					if hasArticle && hasPrice && hasBalance {
						fmt.Printf("\n✅ ЗАГОЛОВОК НАЙДЕН В СТРОКЕ %d\n", i+1)
						break
					}
				}

				fmt.Println()
				return // Analyze only first relevant sheet
			}
		}
	}
}

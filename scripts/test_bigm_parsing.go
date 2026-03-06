package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/prokoleso/etalon-nomenclature/config"
	"github.com/prokoleso/etalon-nomenclature/internal/imap"
	"github.com/prokoleso/etalon-nomenclature/internal/parser"
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

	// Find БИГМАШИН email
	for _, email := range emails {
		if email.From != "m.timoshenkova@bigm.pro" {
			continue
		}

		fmt.Printf("\n📧 Email от: %s\n", email.From)
		fmt.Printf("   Тема: %s\n", email.Subject)
		fmt.Printf("   Дата: %s\n", email.Date.Format("2006-01-02 15:04"))

		// Find price file
		for _, attachment := range email.Attachments {
			filename := strings.ToLower(attachment.Filename)
			if !strings.Contains(filename, "прайс") || !strings.Contains(filename, "шины") {
				continue
			}

			fmt.Printf("\n📄 Парсим файл: %s\n", attachment.Filename)
			fmt.Println(strings.Repeat("=", 80))

			// Parse
			priceParser := parser.NewPriceParser(logger)
			detector := parser.NewDetector(logger)
			provider := detector.DetectProvider(email.From)

			rows, err := priceParser.Parse(attachment.Content, attachment.Filename, string(provider), time.Now())
			if err != nil {
				fmt.Printf("❌ ОШИБКА парсинга: %v\n", err)
				return
			}

			fmt.Printf("\n✅ Успешно распарсено: %d строк\n", len(rows))

			// Show first 10 rows
			fmt.Println("\n📊 Первые 10 записей:")
			fmt.Println(strings.Repeat("-", 120))
			fmt.Printf("%-25s | %-10s | %-8s | %-30s | %s\n", "АРТИКУЛ", "ЦЕНА", "ОСТАТОК", "СКЛАД", "ПОСТАВЩИК")
			fmt.Println(strings.Repeat("-", 120))

			maxRows := 10
			if len(rows) < maxRows {
				maxRows = len(rows)
			}

			for i := 0; i < maxRows; i++ {
				row := rows[i]
				article := row.Article
				if len(article) > 25 {
					article = article[:22] + "..."
				}
				store := row.Store
				if len(store) > 30 {
					store = store[:27] + "..."
				}

				fmt.Printf("%-25s | %10.2f | %8d | %-30s | %s\n",
					article, row.Price, row.Balance, store, row.Provider)
			}

			fmt.Println("\n✅ Проверка:")
			validCount := 0
			invalidCount := 0

			for _, row := range rows {
				// Check if article looks like nomenclature (starts with "Автошина")
				if strings.HasPrefix(row.Article, "Автошина") {
					invalidCount++
				} else {
					validCount++
				}

				// Check if price is negative or too high
				if row.Price < 0 {
					fmt.Printf("⚠️  Отрицательная цена: article='%s', price=%.2f\n", row.Article, row.Price)
					invalidCount++
				}
				if row.Price > 500000 {
					fmt.Printf("⚠️  Слишком высокая цена: article='%s', price=%.2f\n", row.Article, row.Price)
					invalidCount++
				}
			}

			fmt.Printf("\nСтатистика:\n")
			fmt.Printf("  Валидных записей: %d (%.1f%%)\n", validCount, float64(validCount)/float64(len(rows))*100)
			fmt.Printf("  Проблемных записей: %d (%.1f%%)\n", invalidCount, float64(invalidCount)/float64(len(rows))*100)

			if invalidCount == 0 && validCount > 0 {
				fmt.Println("\n🎉 ВСЕ ОТЛИЧНО! Парсинг работает корректно!")
			} else if invalidCount > len(rows)/2 {
				fmt.Println("\n❌ ПРОБЛЕМА! Больше половины записей некорректны!")
			}

			return // Test only first file
		}
	}

	fmt.Println("❌ БИГМАШИН файл не найден")
}

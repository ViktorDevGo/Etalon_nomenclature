package main

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	_ "github.com/emersion/go-message/charset"
)

func main() {
	// Connect to IMAP server
	c, err := client.DialTLS("mail.hosting.reg.ru:993", nil)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer c.Logout()

	// Login
	if err := c.Login("zakupki@etalon-shina.ru", "S69Y1ypojVLCZHO8"); err != nil {
		log.Fatal("Failed to login:", err)
	}

	// Select INBOX
	_, err = c.Select("INBOX", false)
	if err != nil {
		log.Fatal("Failed to select INBOX:", err)
	}

	// Search for БИГМАШИН emails
	since := time.Now().AddDate(0, 0, -3)
	criteria := imap.NewSearchCriteria()
	criteria.Since = since

	ids, err := c.Search(criteria)
	if err != nil {
		log.Fatal("Failed to search:", err)
	}

	// Fetch messages
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(ids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchRFC822}, messages)
	}()

	fmt.Println("🔍 Анализ писем от m.timoshenkova@bigm.pro:")
	fmt.Println(string(make([]byte, 80)))

	count := 0
	for msg := range messages {
		if msg.Envelope == nil {
			continue
		}

		from := "unknown"
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].Address()
		}

		// Only process БИГМАШИН emails
		if from != "m.timoshenkova@bigm.pro" {
			continue
		}

		count++
		fmt.Printf("\n📧 Email #%d: %s\n", count, msg.Envelope.Subject)
		fmt.Printf("   От: %s\n", from)
		fmt.Printf("   Дата: %s\n", msg.Envelope.Date)
		fmt.Printf("   Message-ID: %s\n\n", msg.Envelope.MessageId)

		// Parse message
		section := &imap.BodySectionName{}
		r := msg.GetBody(section)
		if r == nil {
			fmt.Println("   ❌ Тело письма пустое")
			continue
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			fmt.Printf("   ❌ Ошибка создания reader: %v\n", err)
			continue
		}

		fmt.Println("   📎 Части письма:")
		partNum := 0
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Printf("   ❌ Ошибка чтения части: %v\n", err)
				break
			}

			partNum++

			switch h := part.Header.(type) {
			case *mail.AttachmentHeader:
				filename, _ := h.Filename()
				fmt.Printf("   %d. [AttachmentHeader] Файл: %s\n", partNum, filename)
				fmt.Printf("      Content-Type: %s\n", h.Get("Content-Type"))
				fmt.Printf("      Content-Disposition: %s\n", h.Get("Content-Disposition"))

			case *mail.InlineHeader:
				contentType := h.Get("Content-Type")
				disposition := h.Get("Content-Disposition")
				fmt.Printf("   %d. [InlineHeader]\n", partNum)
				fmt.Printf("      Content-Type: %s\n", contentType)
				fmt.Printf("      Content-Disposition: %s\n", disposition)

				// Try to extract filename from Content-Disposition or Content-Type
				filename := extractFilename(contentType, disposition)
				if filename != "" {
					fmt.Printf("      Filename: %s\n", filename)
					if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
						fmt.Printf("      ⚠️  EXCEL ФАЙЛ, НО С InlineHeader (пропускается текущим кодом!)\n")
					}
				}

			default:
				contentType := part.Header.Get("Content-Type")
				disposition := part.Header.Get("Content-Disposition")
				fmt.Printf("   %d. [%T] Content-Type: %s\n", partNum, h, contentType)
				if disposition != "" {
					fmt.Printf("      Content-Disposition: %s\n", disposition)
				}
			}
		}
	}

	if err := <-done; err != nil {
		log.Fatal("Failed to fetch:", err)
	}

	fmt.Printf("\n\nВсего писем от БИГМАШИН: %d\n", count)
}

func extractFilename(contentType, disposition string) string {
	// Try Content-Disposition first
	if disposition != "" {
		parts := strings.Split(disposition, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(strings.ToLower(part), "filename=") {
				filename := strings.TrimPrefix(part, "filename=")
				filename = strings.TrimPrefix(filename, "\"")
				filename = strings.TrimSuffix(filename, "\"")
				return filename
			}
		}
	}

	// Try Content-Type
	if contentType != "" {
		parts := strings.Split(contentType, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(strings.ToLower(part), "name=") {
				filename := strings.TrimPrefix(part, "name=")
				filename = strings.TrimPrefix(filename, "\"")
				filename = strings.TrimSuffix(filename, "\"")
				return filename
			}
		}
	}

	return ""
}

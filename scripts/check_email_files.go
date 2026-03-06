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

	// Search for messages since 3 days ago
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

	targetMessageID := "<00ff01dcaaf5$0876dff0$19649fd0$@sibzapaska.ru>"

	for msg := range messages {
		if msg.Envelope == nil {
			continue
		}

		if msg.Envelope.MessageId != targetMessageID {
			continue
		}

		fmt.Printf("📧 Email: %s\n", msg.Envelope.Subject)
		fmt.Printf("   Message-ID: %s\n", msg.Envelope.MessageId)

		from := "unknown"
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].Address()
		}
		fmt.Printf("   От: %s\n\n", from)

		// Parse message
		section := &imap.BodySectionName{}
		r := msg.GetBody(section)
		if r == nil {
			fmt.Println("   ❌ Тело письма пустое")
			continue
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			fmt.Printf("   ❌ Ошибка: %v\n", err)
			continue
		}

		fmt.Println("   📎 Вложения:")
		partNum := 0
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Printf("   ❌ Ошибка: %v\n", err)
				break
			}

			switch h := part.Header.(type) {
			case *mail.AttachmentHeader:
				filename, _ := h.Filename()
				lowerFilename := strings.ToLower(filename)

				partNum++
				fmt.Printf("   %d. %s\n", partNum, filename)

				// Check file type detection
				if strings.Contains(lowerFilename, "прайс") {
					fmt.Println("      → Тип: PRICE (прайс-лист)")
				} else {
					fmt.Println("      → Тип: NOMENCLATURE (номенклатура)")
				}
			}
		}
	}

	if err := <-done; err != nil {
		log.Fatal("Failed to fetch:", err)
	}
}

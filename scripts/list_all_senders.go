package main

import (
	"fmt"
	"log"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
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

	fmt.Printf("📧 Найдено писем за последние 3 дня: %d\n\n", len(ids))

	if len(ids) == 0 {
		return
	}

	// Fetch all messages
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(ids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBodyStructure}, messages)
	}()

	// Count senders
	senderCounts := make(map[string]int)
	withAttachments := make(map[string]int)
	var allMessages []*imap.Message

	fmt.Println("Список всех отправителей:")
	fmt.Println(string(make([]byte, 80)))

	i := 0
	for msg := range messages {
		allMessages = append(allMessages, msg)
		i++

		if msg.Envelope == nil {
			continue
		}

		from := "unknown"
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].Address()
		}

		subject := msg.Envelope.Subject
		senderCounts[from]++

		// Check for attachments
		hasAttachment := hasAttachments(msg.BodyStructure)
		if hasAttachment {
			withAttachments[from]++
		}

		attachmentMark := ""
		if hasAttachment {
			attachmentMark = "📎"
		}

		fmt.Printf("%d. %s %s\n   От: %s\n   Тема: %s\n\n",
			i, attachmentMark, from, from, subject)
	}

	if err := <-done; err != nil {
		log.Fatal("Failed to fetch:", err)
	}

	fmt.Println(string(make([]byte, 80)))
	fmt.Println("\n📊 Статистика по отправителям:")
	for sender, count := range senderCounts {
		attachCount := withAttachments[sender]
		fmt.Printf("  %s: %d писем (с вложениями: %d)\n", sender, count, attachCount)
	}
}

func hasAttachments(bs *imap.BodyStructure) bool {
	if bs == nil {
		return false
	}

	// Check if multipart
	if len(bs.Parts) > 0 {
		for _, part := range bs.Parts {
			if hasAttachments(part) {
				return true
			}
		}
	}

	// Check disposition
	if bs.Disposition == "attachment" {
		return true
	}

	// Check if it's an application type (likely attachment)
	if bs.MIMEType == "application" {
		return true
	}

	return false
}

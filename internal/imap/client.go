package imap

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/prokoleso/etalon-nomenclature/config"
	"go.uber.org/zap"
)

const (
	maxRetries    = 3
	retryDelay    = 5 * time.Second
	maxAttachment = 10 * 1024 * 1024 // 10 MB
	lookbackDays  = 3                 // Number of days to look back for emails
)

// Client represents an IMAP client wrapper
type Client struct {
	config config.MailboxConfig
	logger *zap.Logger
}

// Email represents an email with attachments
type Email struct {
	MessageID   string
	Subject     string
	From        string
	Date        time.Time
	Attachments []Attachment
}

// Attachment represents an email attachment
type Attachment struct {
	Filename string
	Content  []byte
	Size     int64
}

// NewClient creates a new IMAP client
func NewClient(cfg config.MailboxConfig, logger *zap.Logger) *Client {
	return &Client{
		config: cfg,
		logger: logger,
	}
}

// FetchTodayEmails fetches emails from the last N days with Excel attachments
// The lookback period is defined by the lookbackDays constant
func (c *Client) FetchTodayEmails(ctx context.Context) ([]Email, error) {
	var emails []Email
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Info("Retrying IMAP connection",
				zap.String("email", c.config.Email),
				zap.Int("attempt", attempt+1))

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		emails, lastErr = c.fetchEmails(ctx)
		if lastErr == nil {
			return emails, nil
		}

		c.logger.Warn("IMAP operation failed",
			zap.String("email", c.config.Email),
			zap.Error(lastErr),
			zap.Int("attempt", attempt+1))
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (c *Client) fetchEmails(ctx context.Context) ([]Email, error) {
	// Connect to IMAP server
	imapClient, err := client.DialTLS(
		fmt.Sprintf("%s:%d", c.config.Host, c.config.Port),
		&tls.Config{ServerName: c.config.Host},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer imapClient.Logout()

	// Login
	if err := imapClient.Login(c.config.Email, c.config.Password); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	// Select INBOX
	mbox, err := imapClient.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	if mbox.Messages == 0 {
		c.logger.Debug("No messages in mailbox", zap.String("email", c.config.Email))
		return []Email{}, nil
	}

	// Search for emails from the last N days
	sinceDate := time.Now().Add(-lookbackDays * 24 * time.Hour)
	criteria := imap.NewSearchCriteria()
	criteria.Since = sinceDate

	uids, err := imapClient.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(uids) == 0 {
		c.logger.Debug("No messages found in the last N days",
			zap.String("email", c.config.Email),
			zap.Int("days", lookbackDays),
			zap.String("since", sinceDate.Format("02-Jan-2006")))
		return []Email{}, nil
	}

	c.logger.Info("Found messages",
		zap.String("email", c.config.Email),
		zap.Int("count", len(uids)),
		zap.Int("lookback_days", lookbackDays),
		zap.String("since", sinceDate.Format("02-Jan-2006")))

	// Fetch messages
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	messages := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)

	go func() {
		done <- imapClient.Fetch(seqset, []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchRFC822,
		}, messages)
	}()

	var emails []Email
	for msg := range messages {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		email, err := c.parseMessage(msg)
		if err != nil {
			c.logger.Warn("Failed to parse message",
				zap.Error(err),
				zap.Uint32("uid", msg.Uid))
			continue
		}

		if email != nil && len(email.Attachments) > 0 {
			emails = append(emails, *email)
		}
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return emails, nil
}

func (c *Client) parseMessage(msg *imap.Message) (*Email, error) {
	if msg.Envelope == nil {
		return nil, fmt.Errorf("message envelope is nil")
	}

	section := &imap.BodySectionName{}
	r := msg.GetBody(section)
	if r == nil {
		return nil, fmt.Errorf("message body is nil")
	}

	mr, err := mail.CreateReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create mail reader: %w", err)
	}

	email := &Email{
		MessageID: msg.Envelope.MessageId,
		Subject:   msg.Envelope.Subject,
		Date:      msg.Envelope.Date,
	}

	if len(msg.Envelope.From) > 0 {
		email.From = msg.Envelope.From[0].Address()
	}

	// Process message parts
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read part: %w", err)
		}

		switch h := part.Header.(type) {
		case *mail.AttachmentHeader:
			filename, _ := h.Filename()
			if !strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
				continue
			}

			// Read attachment
			content, err := io.ReadAll(part.Body)
			if err != nil {
				c.logger.Warn("Failed to read attachment",
					zap.String("filename", filename),
					zap.Error(err))
				continue
			}

			size := int64(len(content))
			if size > maxAttachment {
				c.logger.Warn("Attachment too large, skipping",
					zap.String("filename", filename),
					zap.Int64("size", size),
					zap.Int64("max_size", maxAttachment))
				continue
			}

			email.Attachments = append(email.Attachments, Attachment{
				Filename: filename,
				Content:  content,
				Size:     size,
			})

			c.logger.Debug("Found Excel attachment",
				zap.String("filename", filename),
				zap.Int64("size", size))
		}
	}

	if len(email.Attachments) == 0 {
		return nil, nil
	}

	return email, nil
}

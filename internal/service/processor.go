package service

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/prokoleso/etalon-nomenclature/config"
	"github.com/prokoleso/etalon-nomenclature/internal/db"
	"github.com/prokoleso/etalon-nomenclature/internal/imap"
	"github.com/prokoleso/etalon-nomenclature/internal/parser"
	"go.uber.org/zap"
)

// Processor handles the main email processing logic
type Processor struct {
	config *config.Config
	db     *db.Database
	parser *parser.Parser
	logger *zap.Logger
}

// NewProcessor creates a new processor
func NewProcessor(cfg *config.Config, database *db.Database, logger *zap.Logger) *Processor {
	return &Processor{
		config: cfg,
		db:     database,
		parser: parser.New(logger),
		logger: logger,
	}
}

// Run starts the main processing loop
func (p *Processor) Run(ctx context.Context) {
	p.logger.Info("Processor started")
	defer p.logger.Info("Processor stopped")

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	// Process immediately on start
	p.processWithRecovery(ctx)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Received shutdown signal")
			return
		case <-ticker.C:
			p.processWithRecovery(ctx)
		}
	}
}

// processWithRecovery wraps processEmails with panic recovery
func (p *Processor) processWithRecovery(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("Panic recovered in processor",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()

	if err := p.processEmails(ctx); err != nil {
		p.logger.Error("Failed to process emails", zap.Error(err))
	}
}

// processEmails processes emails from all configured mailboxes
func (p *Processor) processEmails(ctx context.Context) error {
	p.logger.Info("Starting email processing cycle")

	for _, mailboxCfg := range p.config.Mailboxes {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := p.processMailbox(ctx, mailboxCfg); err != nil {
			p.logger.Error("Failed to process mailbox",
				zap.String("email", mailboxCfg.Email),
				zap.Error(err))
			// Continue with other mailboxes
		}
	}

	p.logger.Info("Email processing cycle completed")
	return nil
}

// processMailbox processes emails from a single mailbox
func (p *Processor) processMailbox(ctx context.Context, mailboxCfg config.MailboxConfig) error {
	p.logger.Info("Processing mailbox", zap.String("email", mailboxCfg.Email))

	client := imap.NewClient(mailboxCfg, p.logger)

	emails, err := client.FetchTodayEmails(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch emails: %w", err)
	}

	if len(emails) == 0 {
		p.logger.Debug("No emails with attachments found",
			zap.String("email", mailboxCfg.Email))
		return nil
	}

	p.logger.Info("Found emails with attachments",
		zap.String("email", mailboxCfg.Email),
		zap.Int("count", len(emails)))

	processed := 0
	skipped := 0

	for _, email := range emails {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := p.processEmail(ctx, email); err != nil {
			p.logger.Error("Failed to process email",
				zap.String("message_id", email.MessageID),
				zap.String("subject", email.Subject),
				zap.Error(err))
			continue
		}

		processed++
	}

	p.logger.Info("Mailbox processing completed",
		zap.String("email", mailboxCfg.Email),
		zap.Int("processed", processed),
		zap.Int("skipped", skipped))

	return nil
}

// processEmail processes a single email
func (p *Processor) processEmail(ctx context.Context, email imap.Email) error {
	// Check blacklisted domains (system emails that should never be processed)
	blacklistedDomains := []string{
		"@bitrix24.com",
		"@noreply",
		"@no-reply",
		"@mailer-daemon",
		"@postmaster",
		"@mail-daemon",
	}

	// Check both From address and Message-ID
	emailLower := strings.ToLower(email.From)
	messageIDLower := strings.ToLower(email.MessageID)

	for _, blacklisted := range blacklistedDomains {
		if strings.Contains(emailLower, blacklisted) {
			p.logger.Debug("Email from blacklisted domain, skipping",
				zap.String("from", email.From),
				zap.String("reason", "Blacklisted sender address"))
			return nil
		}
		if strings.Contains(messageIDLower, blacklisted) {
			p.logger.Debug("Email with blacklisted Message-ID, skipping",
				zap.String("from", email.From),
				zap.String("message_id", email.MessageID),
				zap.String("reason", "Blacklisted message ID domain"))
			return nil
		}
	}

	// Check if sender is allowed (if filter is configured)
	if len(p.config.AllowedSenders) > 0 {
		allowed := false
		for _, sender := range p.config.AllowedSenders {
			if email.From == sender {
				allowed = true
				break
			}
		}
		if !allowed {
			p.logger.Debug("Email from non-allowed sender, skipping",
				zap.String("from", email.From),
				zap.Strings("allowed_senders", p.config.AllowedSenders))
			return nil
		}
	}

	// Check if already processed
	processed, err := p.db.IsEmailProcessed(ctx, email.MessageID)
	if err != nil {
		return fmt.Errorf("failed to check if email is processed: %w", err)
	}

	if processed {
		p.logger.Debug("Email already processed, skipping",
			zap.String("message_id", email.MessageID),
			zap.String("subject", email.Subject))
		return nil
	}

	p.logger.Info("Processing email",
		zap.String("message_id", email.MessageID),
		zap.String("subject", email.Subject),
		zap.String("from", email.From),
		zap.Int("attachments", len(email.Attachments)))

	// Create detector and parsers
	detector := parser.NewDetector(p.logger)
	priceParser := parser.NewPriceParser(p.logger)
	diskParser := parser.NewDiskParser(p.logger)

	var allRows []db.NomenclatureRow
	var allPriceRows []db.PriceTireRow
	var allDiskRows []db.PriceDiskRow

	// Process each attachment
	for _, attachment := range email.Attachments {
		p.logger.Info("Processing attachment",
			zap.String("filename", attachment.Filename),
			zap.Int64("size", attachment.Size))

		// Detect file type
		fileType := detector.DetectFileType(attachment.Filename)

		if fileType == parser.FileTypeNomenclature {
			// Parse nomenclature file
			rows, err := p.parser.Parse(attachment.Content, attachment.Filename, email.Date)
			if err != nil {
				p.logger.Error("Failed to parse nomenclature attachment",
					zap.String("filename", attachment.Filename),
					zap.Error(err))
				continue
			}

			p.logger.Info("Parsed nomenclature attachment",
				zap.String("filename", attachment.Filename),
				zap.Int("rows", len(rows)))

			allRows = append(allRows, rows...)

		} else if fileType == parser.FileTypePrice {
			// Detect provider and parse price file
			provider := detector.DetectProvider(email.From)

			// Parse tires from price file
			priceRows, err := priceParser.Parse(attachment.Content, attachment.Filename, string(provider), email.Date)
			if err != nil {
				p.logger.Error("Failed to parse price attachment",
					zap.String("filename", attachment.Filename),
					zap.String("provider", string(provider)),
					zap.Error(err))
			} else {
				p.logger.Info("Parsed tire price attachment",
					zap.String("filename", attachment.Filename),
					zap.String("provider", string(provider)),
					zap.Int("rows", len(priceRows)))
				allPriceRows = append(allPriceRows, priceRows...)
			}

			// For ЗАПАСКА and БРИНЕКС, also try to parse disks from the same file
			if provider == parser.ProviderZapaska || provider == parser.ProviderBrinex {
				diskRows, err := diskParser.Parse(attachment.Content, attachment.Filename, string(provider), email.Date)
				if err != nil {
					p.logger.Warn("Failed to parse disk section from price file (may not contain disks)",
						zap.String("filename", attachment.Filename),
						zap.String("provider", string(provider)),
						zap.Error(err))
				} else if len(diskRows) > 0 {
					p.logger.Info("Parsed disk section from price attachment",
						zap.String("filename", attachment.Filename),
						zap.String("provider", string(provider)),
						zap.Int("rows", len(diskRows)))
					allDiskRows = append(allDiskRows, diskRows...)
				}
			}

		} else if fileType == parser.FileTypeDisk {
			// Detect provider and parse disk file
			provider := detector.DetectProvider(email.From)
			diskRows, err := diskParser.Parse(attachment.Content, attachment.Filename, string(provider), email.Date)
			if err != nil {
				p.logger.Error("Failed to parse disk attachment",
					zap.String("filename", attachment.Filename),
					zap.String("provider", string(provider)),
					zap.Error(err))
				continue
			}

			p.logger.Info("Parsed disk attachment",
				zap.String("filename", attachment.Filename),
				zap.String("provider", string(provider)),
				zap.Int("rows", len(diskRows)))

			allDiskRows = append(allDiskRows, diskRows...)
		}
	}

	if len(allRows) == 0 && len(allPriceRows) == 0 && len(allDiskRows) == 0 {
		p.logger.Error("No data extracted from attachments - NOT marking as processed",
			zap.String("message_id", email.MessageID),
			zap.String("subject", email.Subject),
			zap.String("from", email.From))
		// DO NOT mark as processed - we want to retry when parser is fixed
		return fmt.Errorf("failed to extract any data from attachments")
	}

	// Log sample data for debugging
	if len(allRows) > 0 {
		sampleSize := 3
		if len(allRows) < sampleSize {
			sampleSize = len(allRows)
		}
		for i := 0; i < sampleSize; i++ {
			row := allRows[i]
			p.logger.Debug("Sample nomenclature row",
				zap.Int("row_index", i),
				zap.String("article", row.Article),
				zap.String("brand", row.Brand),
				zap.String("nomenclature", row.Nomenclature))
		}
	}

	if len(allPriceRows) > 0 {
		sampleSize := 3
		if len(allPriceRows) < sampleSize {
			sampleSize = len(allPriceRows)
		}
		for i := 0; i < sampleSize; i++ {
			row := allPriceRows[i]
			p.logger.Debug("Sample price row",
				zap.Int("row_index", i),
				zap.String("article", row.Article),
				zap.String("provider", row.Provider))
		}
	}

	if len(allDiskRows) > 0 {
		sampleSize := 3
		if len(allDiskRows) < sampleSize {
			sampleSize = len(allDiskRows)
		}
		for i := 0; i < sampleSize; i++ {
			row := allDiskRows[i]
			p.logger.Debug("Sample disk row",
				zap.Int("row_index", i),
				zap.String("article", row.Article),
				zap.String("manufacturer", row.Manufacturer),
				zap.String("provider", row.Provider))
		}
	}

	// Save ALL data in a SINGLE atomic transaction
	// Email is marked as processed ONLY if ALL data saves successfully!
	p.logger.Info("Saving all email data in atomic transaction",
		zap.String("message_id", email.MessageID),
		zap.Int("nomenclature_rows", len(allRows)),
		zap.Int("price_rows", len(allPriceRows)),
		zap.Int("disk_rows", len(allDiskRows)))

	if err := p.db.InsertAllEmailDataWithTransaction(ctx, allRows, allPriceRows, allDiskRows, email.MessageID, email.Date); err != nil {
		p.logger.Error("Failed to save email data (transaction rolled back)",
			zap.String("message_id", email.MessageID),
			zap.Error(err))
		return fmt.Errorf("failed to save email data: %w", err)
	}

	p.logger.Info("✅ Successfully processed email and saved ALL data atomically",
		zap.String("message_id", email.MessageID),
		zap.Int("nomenclature_rows", len(allRows)),
		zap.Int("price_rows", len(allPriceRows)),
		zap.Int("disk_rows", len(allDiskRows)))

	return nil
}

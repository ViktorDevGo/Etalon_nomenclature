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

	emailLower := strings.ToLower(email.From)
	for _, blacklisted := range blacklistedDomains {
		if strings.Contains(emailLower, blacklisted) {
			p.logger.Debug("Email from blacklisted domain, skipping",
				zap.String("from", email.From),
				zap.String("reason", "System/automated email"))
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

	// Create detector and price parser
	detector := parser.NewDetector(p.logger)
	priceParser := parser.NewPriceParser(p.logger)

	var allRows []db.NomenclatureRow
	var allPriceRows []db.PriceTireRow

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
			priceRows, err := priceParser.Parse(attachment.Content, attachment.Filename, string(provider), email.Date)
			if err != nil {
				p.logger.Error("Failed to parse price attachment",
					zap.String("filename", attachment.Filename),
					zap.String("provider", string(provider)),
					zap.Error(err))
				continue
			}

			p.logger.Info("Parsed price attachment",
				zap.String("filename", attachment.Filename),
				zap.String("provider", string(provider)),
				zap.Int("rows", len(priceRows)))

			allPriceRows = append(allPriceRows, priceRows...)
		}
	}

	if len(allRows) == 0 && len(allPriceRows) == 0 {
		p.logger.Error("No data extracted from attachments - NOT marking as processed",
			zap.String("message_id", email.MessageID),
			zap.String("subject", email.Subject),
			zap.String("from", email.From))
		// DO NOT mark as processed - we want to retry when parser is fixed
		return fmt.Errorf("failed to extract any data from attachments")
	}

	// Save nomenclature data if present
	if len(allRows) > 0 {
		p.logger.Info("Preparing to save nomenclature data to database",
			zap.String("message_id", email.MessageID),
			zap.Int("total_rows", len(allRows)))

		// Log sample of first few rows for debugging
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
				zap.String("type", row.Type),
				zap.String("size_model", row.SizeModel),
				zap.String("nomenclature", row.Nomenclature),
				zap.Float64("mrc", row.MRC))
		}

		if err := p.db.InsertNomenclatureWithEmail(ctx, allRows, email.MessageID); err != nil {
			p.logger.Error("Failed to save nomenclature data",
				zap.String("message_id", email.MessageID),
				zap.Int("total_rows", len(allRows)),
				zap.Error(err))
			return fmt.Errorf("failed to save nomenclature: %w", err)
		}

		p.logger.Info("Successfully saved nomenclature data",
			zap.String("message_id", email.MessageID),
			zap.Int("total_rows", len(allRows)))
	}

	// Save price data if present
	if len(allPriceRows) > 0 {
		p.logger.Info("Preparing to save price data to database",
			zap.String("message_id", email.MessageID),
			zap.Int("total_rows", len(allPriceRows)))

		// Log sample of first few price rows
		sampleSize := 3
		if len(allPriceRows) < sampleSize {
			sampleSize = len(allPriceRows)
		}
		for i := 0; i < sampleSize; i++ {
			row := allPriceRows[i]
			p.logger.Debug("Sample price row",
				zap.Int("row_index", i),
				zap.String("article", row.Article),
				zap.Float64("price", row.Price),
				zap.Int("balance", row.Balance),
				zap.String("store", row.Store),
				zap.String("provider", row.Provider))
		}

		if err := p.db.InsertPriceTiresWithEmail(ctx, allPriceRows, email.MessageID); err != nil {
			p.logger.Error("Failed to save price data",
				zap.String("message_id", email.MessageID),
				zap.Int("total_rows", len(allPriceRows)),
				zap.Error(err))
			return fmt.Errorf("failed to save prices: %w", err)
		}

		p.logger.Info("Successfully saved price data",
			zap.String("message_id", email.MessageID),
			zap.Int("total_rows", len(allPriceRows)))
	}

	p.logger.Info("Successfully processed email and saved to database",
		zap.String("message_id", email.MessageID),
		zap.Int("nomenclature_rows", len(allRows)),
		zap.Int("price_rows", len(allPriceRows)))

	return nil
}

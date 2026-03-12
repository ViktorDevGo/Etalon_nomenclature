package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/prokoleso/etalon-nomenclature/config"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// migrationSQL contains the initial database schema
const migrationSQL = `
-- Table: etalon_nomenclature
-- Stores nomenclature data extracted from Excel files
CREATE TABLE IF NOT EXISTS etalon_nomenclature (
    id SERIAL PRIMARY KEY,
    article TEXT,
    brand TEXT,
    type TEXT,
    size_model TEXT,
    nomenclature TEXT,
    mrc NUMERIC,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now()
);

-- Add indices for common queries
CREATE INDEX IF NOT EXISTS idx_etalon_nomenclature_article ON etalon_nomenclature(article);
CREATE INDEX IF NOT EXISTS idx_etalon_nomenclature_brand ON etalon_nomenclature(brand);
CREATE INDEX IF NOT EXISTS idx_etalon_nomenclature_isimport ON etalon_nomenclature(isimport);
CREATE INDEX IF NOT EXISTS idx_etalon_nomenclature_created_at ON etalon_nomenclature(created_at);

-- Table: processed_emails
-- Tracks processed emails to prevent duplicate processing
CREATE TABLE IF NOT EXISTS processed_emails (
    id SERIAL PRIMARY KEY,
    message_id TEXT NOT NULL,
    email_date TIMESTAMP,
    processed_at TIMESTAMP DEFAULT now()
);

-- Unique index on message_id to prevent duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_processed_emails_message_id
ON processed_emails(message_id);

-- Index on email_date for queries
CREATE INDEX IF NOT EXISTS idx_processed_emails_email_date
ON processed_emails(email_date);

-- Table: price_tires
-- Stores tire prices from suppliers
CREATE TABLE IF NOT EXISTS price_tires (
    id SERIAL PRIMARY KEY,
    article TEXT NOT NULL,
    price NUMERIC,
    balance INTEGER,
    store TEXT,
    provider TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now()
);

-- Indices for fast lookups
CREATE INDEX IF NOT EXISTS idx_price_tires_article ON price_tires(article);
CREATE INDEX IF NOT EXISTS idx_price_tires_provider ON price_tires(provider);
CREATE INDEX IF NOT EXISTS idx_price_tires_created_at ON price_tires(created_at);

-- Composite index for deduplication: check if exact record already exists
-- This dramatically speeds up duplicate detection during batch inserts
CREATE INDEX IF NOT EXISTS idx_price_tires_dedup ON price_tires(article, price, balance, store);

-- Table: price_disks
-- Stores disk/wheel prices from suppliers
CREATE TABLE IF NOT EXISTS price_disks (
    id SERIAL PRIMARY KEY,
    article TEXT NOT NULL,
    manufacturer TEXT,
    model TEXT,
    width NUMERIC,
    diameter NUMERIC,
    drilling TEXT,
    radius TEXT,
    central_hole TEXT,
    color TEXT,
    price NUMERIC,
    store TEXT,
    balance INTEGER DEFAULT 0,
    provider TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);

-- Index for fast article lookups
CREATE INDEX IF NOT EXISTS idx_price_disks_article ON price_disks(article);
CREATE INDEX IF NOT EXISTS idx_price_disks_provider ON price_disks(provider);
CREATE INDEX IF NOT EXISTS idx_price_disks_created_at ON price_disks(created_at);

-- Composite index for deduplication: check if exact record already exists
-- This dramatically speeds up duplicate detection during batch inserts
CREATE INDEX IF NOT EXISTS idx_price_disks_dedup ON price_disks(article, price, balance, store);
`

// Database represents the database connection
type Database struct {
	db     *sql.DB
	logger *zap.Logger
}

// NomenclatureRow represents a row in etalon_nomenclature table
type NomenclatureRow struct {
	Article      string
	Brand        string
	Type         string
	SizeModel    string
	Nomenclature string
	MRC          float64
	EmailDate    time.Time
}

// PriceTireRow represents a row in price_tires table
type PriceTireRow struct {
	Article   string
	Price     float64
	Balance   int
	Store     string
	Provider  string
	EmailDate time.Time
}

// PriceDiskRow represents a row in price_disks table
type PriceDiskRow struct {
	Article      string
	Manufacturer string
	Model        string
	Width        float64
	Diameter     float64
	Drilling     string
	Radius       string
	CentralHole  string
	Color        string
	Price        float64
	Store        string
	Balance      int
	Provider     string
	EmailDate    time.Time
}

// New creates a new database connection
func New(cfg config.DatabaseConfig, logger *zap.Logger) (*Database, error) {
	// Set SSL certificate if provided
	if cfg.SSLRootCert != "" {
		if err := os.Setenv("PGSSLROOTCERT", cfg.SSLRootCert); err != nil {
			return nil, fmt.Errorf("failed to set PGSSLROOTCERT: %w", err)
		}
	}

	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnMaxLifetime)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established successfully")

	database := &Database{
		db:     db,
		logger: logger,
	}

	// Check and apply migrations if needed
	if err := database.ensureSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure database schema: %w", err)
	}

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// ensureSchema checks if required tables exist and applies migrations if needed
func (d *Database) ensureSchema(ctx context.Context) error {
	d.logger.Info("Checking database schema...")

	exists, err := d.checkTablesExist(ctx)
	if err != nil {
		return fmt.Errorf("failed to check tables: %w", err)
	}

	if !exists {
		d.logger.Info("Required tables not found, applying migrations...")
		if err := d.applyMigrations(ctx); err != nil {
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
		d.logger.Info("Migrations applied successfully")
	}

	// Apply incremental migrations (add email_date column if missing)
	if err := d.applyIncrementalMigrations(ctx); err != nil {
		return fmt.Errorf("failed to apply incremental migrations: %w", err)
	}

	d.logger.Info("Database schema is up to date")
	return nil
}

// checkTablesExist verifies that required tables exist in the database
func (d *Database) checkTablesExist(ctx context.Context) (bool, error) {
	// Check for all required tables
	query := `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('etalon_nomenclature', 'processed_emails', 'price_tires', 'price_disks')
	`

	var count int
	if err := d.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to query tables: %w", err)
	}

	// All four tables should exist
	return count == 4, nil
}

// applyMigrations applies the database schema migrations
func (d *Database) applyMigrations(ctx context.Context) error {
	d.logger.Info("Applying database migrations...")

	// Execute the entire migration SQL in one go
	// PostgreSQL can handle multiple statements separated by semicolons
	if _, err := d.db.ExecContext(ctx, migrationSQL); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	d.logger.Info("Database migrations applied successfully")
	return nil
}

// applyIncrementalMigrations applies incremental schema updates
func (d *Database) applyIncrementalMigrations(ctx context.Context) error {
	// Migration 1: Add email_date column to etalon_nomenclature
	if err := d.addColumnIfNotExists(ctx, "etalon_nomenclature", "email_date", "TIMESTAMP"); err != nil {
		return err
	}

	// Migration 2: Add email_date column to price_tires
	if err := d.addColumnIfNotExists(ctx, "price_tires", "email_date", "TIMESTAMP"); err != nil {
		return err
	}

	// Migration 3: Add price column to price_disks (if missing)
	if err := d.addColumnIfNotExists(ctx, "price_disks", "price", "NUMERIC"); err != nil {
		return err
	}

	// Migration 4: Add deduplication composite index for price_tires
	if err := d.addIndexIfNotExists(ctx, "idx_price_tires_dedup",
		"CREATE INDEX IF NOT EXISTS idx_price_tires_dedup ON price_tires(article, price, balance, store)"); err != nil {
		return err
	}

	// Migration 5: Add deduplication composite index for price_disks
	if err := d.addIndexIfNotExists(ctx, "idx_price_disks_dedup",
		"CREATE INDEX IF NOT EXISTS idx_price_disks_dedup ON price_disks(article, price, balance, store)"); err != nil {
		return err
	}

	return nil
}

// addColumnIfNotExists adds a column to a table if it doesn't exist
func (d *Database) addColumnIfNotExists(ctx context.Context, tableName, columnName, columnType string) error {
	// Check if column exists
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = $1 AND column_name = $2
		)
	`

	var exists bool
	if err := d.db.QueryRowContext(ctx, checkQuery, tableName, columnName).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check column %s.%s: %w", tableName, columnName, err)
	}

	if exists {
		d.logger.Debug("Column already exists",
			zap.String("table", tableName),
			zap.String("column", columnName))
		return nil
	}

	// Add column
	alterQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, columnType)
	if _, err := d.db.ExecContext(ctx, alterQuery); err != nil {
		return fmt.Errorf("failed to add column %s.%s: %w", tableName, columnName, err)
	}

	d.logger.Info("Added column to table",
		zap.String("table", tableName),
		zap.String("column", columnName),
		zap.String("type", columnType))

	return nil
}

// addIndexIfNotExists adds an index if it doesn't exist
func (d *Database) addIndexIfNotExists(ctx context.Context, indexName, createSQL string) error {
	// Check if index exists
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE indexname = $1
		)
	`

	var exists bool
	if err := d.db.QueryRowContext(ctx, checkQuery, indexName).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check index %s: %w", indexName, err)
	}

	if exists {
		d.logger.Debug("Index already exists", zap.String("index", indexName))
		return nil
	}

	// Create index
	if _, err := d.db.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("failed to create index %s: %w", indexName, err)
	}

	d.logger.Info("Created index", zap.String("index", indexName))
	return nil
}

// IsEmailProcessed checks if an email with given message ID has been processed
func (d *Database) IsEmailProcessed(ctx context.Context, messageID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM processed_emails WHERE message_id = $1)`

	err := d.db.QueryRowContext(ctx, query, messageID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check processed email: %w", err)
	}

	return exists, nil
}

// MarkEmailAsProcessed marks an email as processed
func (d *Database) MarkEmailAsProcessed(ctx context.Context, messageID string, emailDate time.Time) error {
	query := `INSERT INTO processed_emails (message_id, email_date) VALUES ($1, $2) ON CONFLICT (message_id) DO NOTHING`

	_, err := d.db.ExecContext(ctx, query, messageID, emailDate)
	if err != nil {
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}

	return nil
}

// InsertNomenclature inserts nomenclature data into the database
func (d *Database) InsertNomenclature(ctx context.Context, rows []NomenclatureRow) error {
	if len(rows) == 0 {
		return nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO etalon_nomenclature
		(article, brand, type, size_model, nomenclature, mrc, isimport)
		VALUES ($1, $2, $3, $4, $5, $6, 0)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		_, err := stmt.ExecContext(ctx,
			row.Article,
			row.Brand,
			row.Type,
			row.SizeModel,
			row.Nomenclature,
			row.MRC,
		)
		if err != nil {
			return fmt.Errorf("failed to insert row: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	d.logger.Info("Inserted nomenclature data", zap.Int("rows", len(rows)))
	return nil
}

// InsertNomenclatureWithEmail inserts nomenclature data and marks email as processed in a transaction
func (d *Database) InsertNomenclatureWithEmail(ctx context.Context, rows []NomenclatureRow, messageID string) error {
	d.logger.Info("Starting InsertNomenclatureWithEmail",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		d.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	d.logger.Info("Transaction started successfully")

	// Insert nomenclature data in batches
	if len(rows) > 0 {
		// Step 0: Deduplicate rows within the batch (keep last occurrence of each article)
		articleMap := make(map[string]NomenclatureRow)
		for _, row := range rows {
			articleMap[row.Article] = row // Last occurrence wins
		}

		// Convert back to slice
		deduplicatedRows := make([]NomenclatureRow, 0, len(articleMap))
		for _, row := range articleMap {
			deduplicatedRows = append(deduplicatedRows, row)
		}

		originalCount := len(rows)
		rows = deduplicatedRows

		if originalCount > len(rows) {
			d.logger.Info("Removed duplicates within batch",
				zap.Int("original_count", originalCount),
				zap.Int("deduplicated_count", len(rows)),
				zap.Int("duplicates_removed", originalCount-len(rows)))
		}

		batchSize := 1000
		totalBatches := (len(rows) + batchSize - 1) / batchSize
		d.logger.Info("Preparing to insert data in batches",
			zap.Int("total_rows", len(rows)),
			zap.Int("batch_size", batchSize),
			zap.Int("total_batches", totalBatches))

		for i := 0; i < len(rows); i += batchSize {
			end := i + batchSize
			if end > len(rows) {
				end = len(rows)
			}
			batch := rows[i:end]
			batchNum := (i / batchSize) + 1

			d.logger.Info("Processing batch",
				zap.Int("batch_num", batchNum),
				zap.Int("total_batches", totalBatches),
				zap.Int("batch_start", i),
				zap.Int("batch_size", len(batch)))

			// Step 1: Delete existing duplicates for today
			// Collect unique articles in this batch
			articleMap := make(map[string]bool)
			for _, row := range batch {
				articleMap[row.Article] = true
			}
			articles := make([]string, 0, len(articleMap))
			for article := range articleMap {
				articles = append(articles, article)
			}

			// Delete existing records with same articles created today
			deleteQuery := `
				DELETE FROM etalon_nomenclature
				WHERE article = ANY($1)
				AND DATE(created_at) = CURRENT_DATE
			`
			deleteResult, err := tx.ExecContext(ctx, deleteQuery, pq.Array(articles))
			if err != nil {
				d.logger.Error("Failed to delete duplicates",
					zap.Int("batch_num", batchNum),
					zap.Error(err))
				return fmt.Errorf("failed to delete duplicates in batch %d: %w", batchNum, err)
			}

			deletedRows, _ := deleteResult.RowsAffected()
			if deletedRows > 0 {
				d.logger.Info("Deleted duplicate records for today",
					zap.Int("batch_num", batchNum),
					zap.Int64("deleted_rows", deletedRows),
					zap.Int("unique_articles", len(articles)))
			}

			// Step 2: Build VALUES clause for INSERT
			values := make([]interface{}, 0, len(batch)*7)
			placeholders := make([]string, 0, len(batch))

			for idx, row := range batch {
				placeholderStart := idx * 7
				placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, 0)",
					placeholderStart+1, placeholderStart+2, placeholderStart+3,
					placeholderStart+4, placeholderStart+5, placeholderStart+6, placeholderStart+7))
				values = append(values, row.Article, row.Brand, row.Type, row.SizeModel, row.Nomenclature, row.MRC, row.EmailDate)
			}

			query := fmt.Sprintf(`
				INSERT INTO etalon_nomenclature
				(article, brand, type, size_model, nomenclature, mrc, email_date, isimport)
				VALUES %s
			`, strings.Join(placeholders, ","))

			d.logger.Debug("Executing INSERT query",
				zap.Int("batch_num", batchNum),
				zap.Int("values_count", len(values)),
				zap.Int("placeholders_count", len(placeholders)))

			result, err := tx.ExecContext(ctx, query, values...)
			if err != nil {
				d.logger.Error("Failed to insert batch",
					zap.Int("batch_num", batchNum),
					zap.Int("batch_start", i),
					zap.Int("batch_size", len(batch)),
					zap.Error(err))
				return fmt.Errorf("failed to insert batch %d: %w", batchNum, err)
			}

			rowsAffected, _ := result.RowsAffected()
			d.logger.Info("Batch inserted successfully",
				zap.Int("batch_num", batchNum),
				zap.Int64("rows_affected", rowsAffected))
		}
	}

	d.logger.Info("All batches inserted, marking email as processed",
		zap.String("message_id", messageID))

	// Mark email as processed with email date from first row
	var emailDate time.Time
	if len(rows) > 0 {
		emailDate = rows[0].EmailDate
	}

	result, err := tx.ExecContext(ctx,
		`INSERT INTO processed_emails (message_id, email_date) VALUES ($1, $2) ON CONFLICT (message_id) DO NOTHING`,
		messageID, emailDate,
	)
	if err != nil {
		d.logger.Error("Failed to mark email as processed",
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	d.logger.Info("Email marked as processed",
		zap.String("message_id", messageID),
		zap.Int64("rows_affected", rowsAffected))

	d.logger.Info("Committing transaction",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	if err := tx.Commit(); err != nil {
		d.logger.Error("Failed to commit transaction",
			zap.Int("total_rows", len(rows)),
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	d.logger.Info("Transaction committed successfully - data saved to database",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	return nil
}

// InsertPriceTiresWithEmail inserts price tire data and marks email as processed in a transaction
func (d *Database) InsertPriceTiresWithEmail(ctx context.Context, rows []PriceTireRow, messageID string) error {
	d.logger.Info("Starting InsertPriceTiresWithEmail",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		d.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	d.logger.Info("Transaction started successfully")

	// Insert price tire data in batches
	if len(rows) > 0 {
		batchSize := 1000
		totalBatches := (len(rows) + batchSize - 1) / batchSize
		d.logger.Info("Preparing to insert price data in batches",
			zap.Int("total_rows", len(rows)),
			zap.Int("batch_size", batchSize),
			zap.Int("total_batches", totalBatches))

		for i := 0; i < len(rows); i += batchSize {
			end := i + batchSize
			if end > len(rows) {
				end = len(rows)
			}
			batch := rows[i:end]
			batchNum := (i / batchSize) + 1

			d.logger.Info("Processing batch",
				zap.Int("batch_num", batchNum),
				zap.Int("total_batches", totalBatches),
				zap.Int("batch_start", i),
				zap.Int("batch_size", len(batch)))

			// Build array parameters for deduplication
			articles := make([]string, len(batch))
			prices := make([]float64, len(batch))
			balances := make([]int, len(batch))
			stores := make([]string, len(batch))
			providers := make([]string, len(batch))
			emailDates := make([]time.Time, len(batch))

			for idx, row := range batch {
				articles[idx] = row.Article
				prices[idx] = row.Price
				balances[idx] = row.Balance
				stores[idx] = row.Store
				providers[idx] = row.Provider
				emailDates[idx] = row.EmailDate
			}

			// Use INSERT ... SELECT with deduplication via NOT EXISTS
			query := `
				WITH new_data AS (
					SELECT * FROM unnest(
						$1::text[], $2::numeric[], $3::integer[], $4::text[], $5::text[], $6::timestamp[]
					) AS t(article, price, balance, store, provider, email_date)
				)
				INSERT INTO price_tires (article, price, balance, store, provider, email_date, isimport)
				SELECT article, price, balance, store, provider, email_date, 0
				FROM new_data nd
				WHERE NOT EXISTS (
					SELECT 1 FROM price_tires pt
					WHERE pt.article = nd.article
					  AND pt.price = nd.price
					  AND pt.balance = nd.balance
					  AND pt.store = nd.store
				)
			`

			d.logger.Debug("Executing INSERT with deduplication",
				zap.Int("batch_num", batchNum),
				zap.Int("batch_size", len(batch)))

			result, err := tx.ExecContext(ctx, query,
				pq.Array(articles), pq.Array(prices), pq.Array(balances),
				pq.Array(stores), pq.Array(providers), pq.Array(emailDates))
			if err != nil {
				d.logger.Error("Failed to insert batch",
					zap.Int("batch_num", batchNum),
					zap.Int("batch_start", i),
					zap.Int("batch_size", len(batch)),
					zap.Error(err))
				return fmt.Errorf("failed to insert batch %d: %w", batchNum, err)
			}

			rowsAffected, _ := result.RowsAffected()
			skipped := int64(len(batch)) - rowsAffected
			d.logger.Info("Batch processed with deduplication",
				zap.Int("batch_num", batchNum),
				zap.Int("batch_size", len(batch)),
				zap.Int64("inserted", rowsAffected),
				zap.Int64("skipped_duplicates", skipped))
		}
	}

	d.logger.Info("All batches inserted, marking email as processed",
		zap.String("message_id", messageID))

	// Mark email as processed with email date from first row
	var emailDate time.Time
	if len(rows) > 0 {
		emailDate = rows[0].EmailDate
	}

	result, err := tx.ExecContext(ctx,
		`INSERT INTO processed_emails (message_id, email_date) VALUES ($1, $2) ON CONFLICT (message_id) DO NOTHING`,
		messageID, emailDate,
	)
	if err != nil {
		d.logger.Error("Failed to mark email as processed",
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	d.logger.Info("Email marked as processed",
		zap.String("message_id", messageID),
		zap.Int64("rows_affected", rowsAffected))

	d.logger.Info("Committing transaction",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	if err := tx.Commit(); err != nil {
		d.logger.Error("Failed to commit transaction",
			zap.Int("total_rows", len(rows)),
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	d.logger.Info("Transaction committed successfully - price data saved to database",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	return nil
}

// InsertPriceDisksWithEmail inserts price disk data and marks email as processed in a transaction
func (d *Database) InsertPriceDisksWithEmail(ctx context.Context, rows []PriceDiskRow, messageID string) error {
	d.logger.Info("Starting InsertPriceDisksWithEmail",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		d.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	d.logger.Info("Transaction started successfully")

	// Insert price disk data in batches
	if len(rows) > 0 {
		batchSize := 1000
		totalBatches := (len(rows) + batchSize - 1) / batchSize
		d.logger.Info("Preparing to insert disk price data in batches",
			zap.Int("total_rows", len(rows)),
			zap.Int("batch_size", batchSize),
			zap.Int("total_batches", totalBatches))

		for i := 0; i < len(rows); i += batchSize {
			end := i + batchSize
			if end > len(rows) {
				end = len(rows)
			}
			batch := rows[i:end]
			batchNum := (i / batchSize) + 1

			d.logger.Info("Processing batch",
				zap.Int("batch_num", batchNum),
				zap.Int("total_batches", totalBatches),
				zap.Int("batch_start", i),
				zap.Int("batch_size", len(batch)))

			// Build array parameters for deduplication
			articles := make([]string, len(batch))
			manufacturers := make([]string, len(batch))
			models := make([]string, len(batch))
			widths := make([]float64, len(batch))
			diameters := make([]float64, len(batch))
			drillings := make([]string, len(batch))
			radiuses := make([]string, len(batch))
			centralHoles := make([]string, len(batch))
			colors := make([]string, len(batch))
			prices := make([]float64, len(batch))
			stores := make([]string, len(batch))
			balances := make([]int, len(batch))
			providers := make([]string, len(batch))
			emailDates := make([]time.Time, len(batch))

			for idx, row := range batch {
				articles[idx] = row.Article
				manufacturers[idx] = row.Manufacturer
				models[idx] = row.Model
				widths[idx] = row.Width
				diameters[idx] = row.Diameter
				drillings[idx] = row.Drilling
				radiuses[idx] = row.Radius
				centralHoles[idx] = row.CentralHole
				colors[idx] = row.Color
				prices[idx] = row.Price
				stores[idx] = row.Store
				balances[idx] = row.Balance
				providers[idx] = row.Provider
				emailDates[idx] = row.EmailDate
			}

			// Use INSERT ... SELECT with deduplication via NOT EXISTS
			query := `
				WITH new_data AS (
					SELECT * FROM unnest(
						$1::text[], $2::text[], $3::text[], $4::numeric[], $5::numeric[],
						$6::text[], $7::text[], $8::text[], $9::text[], $10::numeric[],
						$11::text[], $12::integer[], $13::text[], $14::timestamp[]
					) AS t(article, manufacturer, model, width, diameter, drilling, radius,
					       central_hole, color, price, store, balance, provider, email_date)
				)
				INSERT INTO price_disks
				(article, manufacturer, model, width, diameter, drilling, radius,
				 central_hole, color, price, store, balance, provider, email_date, isimport)
				SELECT article, manufacturer, model, width, diameter, drilling, radius,
				       central_hole, color, price, store, balance, provider, email_date, 0
				FROM new_data nd
				WHERE NOT EXISTS (
					SELECT 1 FROM price_disks pd
					WHERE pd.article = nd.article
					  AND pd.price = nd.price
					  AND pd.balance = nd.balance
					  AND pd.store = nd.store
				)
			`

			d.logger.Debug("Executing INSERT with deduplication",
				zap.Int("batch_num", batchNum),
				zap.Int("batch_size", len(batch)))

			result, err := tx.ExecContext(ctx, query,
				pq.Array(articles), pq.Array(manufacturers), pq.Array(models),
				pq.Array(widths), pq.Array(diameters), pq.Array(drillings),
				pq.Array(radiuses), pq.Array(centralHoles), pq.Array(colors),
				pq.Array(prices), pq.Array(stores), pq.Array(balances),
				pq.Array(providers), pq.Array(emailDates))
			if err != nil {
				d.logger.Error("Failed to insert batch",
					zap.Int("batch_num", batchNum),
					zap.Int("batch_start", i),
					zap.Int("batch_size", len(batch)),
					zap.Error(err))
				return fmt.Errorf("failed to insert batch %d: %w", batchNum, err)
			}

			rowsAffected, _ := result.RowsAffected()
			skipped := int64(len(batch)) - rowsAffected
			d.logger.Info("Batch processed with deduplication",
				zap.Int("batch_num", batchNum),
				zap.Int("batch_size", len(batch)),
				zap.Int64("inserted", rowsAffected),
				zap.Int64("skipped_duplicates", skipped))
		}
	}

	d.logger.Info("All batches inserted, marking email as processed",
		zap.String("message_id", messageID))

	// Mark email as processed with email date from first row
	var emailDate time.Time
	if len(rows) > 0 {
		emailDate = rows[0].EmailDate
	}

	result, err := tx.ExecContext(ctx,
		`INSERT INTO processed_emails (message_id, email_date) VALUES ($1, $2) ON CONFLICT (message_id) DO NOTHING`,
		messageID, emailDate,
	)
	if err != nil {
		d.logger.Error("Failed to mark email as processed",
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	d.logger.Info("Email marked as processed",
		zap.String("message_id", messageID),
		zap.Int64("rows_affected", rowsAffected))

	d.logger.Info("Committing transaction",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	if err := tx.Commit(); err != nil {
		d.logger.Error("Failed to commit transaction",
			zap.Int("total_rows", len(rows)),
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	d.logger.Info("Transaction committed successfully - disk price data saved to database",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	return nil
}

// insertNomenclatureInTx inserts nomenclature data within an existing transaction
func (d *Database) insertNomenclatureInTx(ctx context.Context, tx *sql.Tx, rows []NomenclatureRow) error {
	if len(rows) == 0 {
		return nil
	}

	batchSize := 1000

	// Deduplicate rows within the batch
	articleMap := make(map[string]NomenclatureRow)
	for _, row := range rows {
		articleMap[row.Article] = row
	}
	deduplicatedRows := make([]NomenclatureRow, 0, len(articleMap))
	for _, row := range articleMap {
		deduplicatedRows = append(deduplicatedRows, row)
	}
	rows = deduplicatedRows

	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[i:end]
		batchNum := (i / batchSize) + 1

		// Delete existing duplicates for today
		articles := make([]string, 0, len(batch))
		for _, row := range batch {
			articles = append(articles, row.Article)
		}

		deleteQuery := `DELETE FROM etalon_nomenclature WHERE article = ANY($1) AND DATE(created_at) = CURRENT_DATE`
		_, err := tx.ExecContext(ctx, deleteQuery, pq.Array(articles))
		if err != nil {
			return fmt.Errorf("failed to delete duplicates in batch %d: %w", batchNum, err)
		}

		// Build VALUES clause
		values := make([]interface{}, 0, len(batch)*7)
		placeholders := make([]string, 0, len(batch))

		for idx, row := range batch {
			placeholderStart := idx * 7
			placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, 0)",
				placeholderStart+1, placeholderStart+2, placeholderStart+3,
				placeholderStart+4, placeholderStart+5, placeholderStart+6, placeholderStart+7))
			values = append(values, row.Article, row.Brand, row.Type, row.SizeModel, row.Nomenclature, row.MRC, row.EmailDate)
		}

		query := fmt.Sprintf(`INSERT INTO etalon_nomenclature
			(article, brand, type, size_model, nomenclature, mrc, email_date, isimport)
			VALUES %s`, strings.Join(placeholders, ","))

		_, err = tx.ExecContext(ctx, query, values...)
		if err != nil {
			return fmt.Errorf("failed to insert batch %d: %w", batchNum, err)
		}
	}

	return nil
}

// insertTiresInTx inserts tire price data within an existing transaction
func (d *Database) insertTiresInTx(ctx context.Context, tx *sql.Tx, rows []PriceTireRow) error {
	if len(rows) == 0 {
		return nil
	}

	batchSize := 1000
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[i:end]
		batchNum := (i / batchSize) + 1

		// Build array parameters for deduplication
		articles := make([]string, len(batch))
		prices := make([]float64, len(batch))
		balances := make([]int, len(batch))
		stores := make([]string, len(batch))
		providers := make([]string, len(batch))
		emailDates := make([]time.Time, len(batch))

		for idx, row := range batch {
			articles[idx] = row.Article
			prices[idx] = row.Price
			balances[idx] = row.Balance
			stores[idx] = row.Store
			providers[idx] = row.Provider
			emailDates[idx] = row.EmailDate
		}

		// Use INSERT ... SELECT with deduplication via NOT EXISTS
		query := `
			WITH new_data AS (
				SELECT * FROM unnest(
					$1::text[], $2::numeric[], $3::integer[], $4::text[], $5::text[], $6::timestamp[]
				) AS t(article, price, balance, store, provider, email_date)
			)
			INSERT INTO price_tires (article, price, balance, store, provider, email_date, isimport)
			SELECT article, price, balance, store, provider, email_date, 0
			FROM new_data nd
			WHERE NOT EXISTS (
				SELECT 1 FROM price_tires pt
				WHERE pt.article = nd.article
				  AND pt.price = nd.price
				  AND pt.balance = nd.balance
				  AND pt.store = nd.store
			)
		`

		_, err := tx.ExecContext(ctx, query,
			pq.Array(articles), pq.Array(prices), pq.Array(balances),
			pq.Array(stores), pq.Array(providers), pq.Array(emailDates))
		if err != nil {
			return fmt.Errorf("failed to insert batch %d: %w", batchNum, err)
		}
	}

	return nil
}

// insertDisksInTx inserts disk price data within an existing transaction
func (d *Database) insertDisksInTx(ctx context.Context, tx *sql.Tx, rows []PriceDiskRow) error {
	if len(rows) == 0 {
		return nil
	}

	batchSize := 1000
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[i:end]
		batchNum := (i / batchSize) + 1

		// Build array parameters for deduplication
		articles := make([]string, len(batch))
		manufacturers := make([]string, len(batch))
		models := make([]string, len(batch))
		widths := make([]float64, len(batch))
		diameters := make([]float64, len(batch))
		drillings := make([]string, len(batch))
		radiuses := make([]string, len(batch))
		centralHoles := make([]string, len(batch))
		colors := make([]string, len(batch))
		prices := make([]float64, len(batch))
		stores := make([]string, len(batch))
		balances := make([]int, len(batch))
		providers := make([]string, len(batch))
		emailDates := make([]time.Time, len(batch))

		for idx, row := range batch {
			articles[idx] = row.Article
			manufacturers[idx] = row.Manufacturer
			models[idx] = row.Model
			widths[idx] = row.Width
			diameters[idx] = row.Diameter
			drillings[idx] = row.Drilling
			radiuses[idx] = row.Radius
			centralHoles[idx] = row.CentralHole
			colors[idx] = row.Color
			prices[idx] = row.Price
			stores[idx] = row.Store
			balances[idx] = row.Balance
			providers[idx] = row.Provider
			emailDates[idx] = row.EmailDate
		}

		// Use INSERT ... SELECT with deduplication via NOT EXISTS
		query := `
			WITH new_data AS (
				SELECT * FROM unnest(
					$1::text[], $2::text[], $3::text[], $4::numeric[], $5::numeric[],
					$6::text[], $7::text[], $8::text[], $9::text[], $10::numeric[],
					$11::text[], $12::integer[], $13::text[], $14::timestamp[]
				) AS t(article, manufacturer, model, width, diameter, drilling, radius,
				       central_hole, color, price, store, balance, provider, email_date)
			)
			INSERT INTO price_disks
			(article, manufacturer, model, width, diameter, drilling, radius,
			 central_hole, color, price, store, balance, provider, email_date, isimport)
			SELECT article, manufacturer, model, width, diameter, drilling, radius,
			       central_hole, color, price, store, balance, provider, email_date, 0
			FROM new_data nd
			WHERE NOT EXISTS (
				SELECT 1 FROM price_disks pd
				WHERE pd.article = nd.article
				  AND pd.price = nd.price
				  AND pd.balance = nd.balance
				  AND pd.store = nd.store
			)
		`

		_, err := tx.ExecContext(ctx, query,
			pq.Array(articles), pq.Array(manufacturers), pq.Array(models),
			pq.Array(widths), pq.Array(diameters), pq.Array(drillings),
			pq.Array(radiuses), pq.Array(centralHoles), pq.Array(colors),
			pq.Array(prices), pq.Array(stores), pq.Array(balances),
			pq.Array(providers), pq.Array(emailDates))
		if err != nil {
			return fmt.Errorf("failed to insert batch %d: %w", batchNum, err)
		}
	}

	return nil
}

// InsertAllEmailDataWithTransaction inserts all email data (nomenclature, tires, disks)
// in a SINGLE atomic transaction. Email is marked as processed ONLY if ALL data saves successfully.
func (d *Database) InsertAllEmailDataWithTransaction(
	ctx context.Context,
	nomenclatureRows []NomenclatureRow,
	tireRows []PriceTireRow,
	diskRows []PriceDiskRow,
	messageID string,
	emailDate time.Time,
) error {
	d.logger.Info("Starting atomic email data transaction",
		zap.String("message_id", messageID),
		zap.Int("nomenclature_rows", len(nomenclatureRows)),
		zap.Int("tire_rows", len(tireRows)),
		zap.Int("disk_rows", len(diskRows)))

	// Begin single transaction for ALL data
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		d.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	d.logger.Info("Transaction started successfully")

	// 1. Insert nomenclature data if present
	if len(nomenclatureRows) > 0 {
		d.logger.Info("Inserting nomenclature data",
			zap.Int("rows", len(nomenclatureRows)))

		if err := d.insertNomenclatureInTx(ctx, tx, nomenclatureRows); err != nil {
			d.logger.Error("Failed to insert nomenclature data",
				zap.Error(err))
			return fmt.Errorf("failed to insert nomenclature: %w", err)
		}

		d.logger.Info("Nomenclature data inserted successfully",
			zap.Int("rows", len(nomenclatureRows)))
	}

	// 2. Insert tire price data if present
	if len(tireRows) > 0 {
		d.logger.Info("Inserting tire price data",
			zap.Int("rows", len(tireRows)))

		if err := d.insertTiresInTx(ctx, tx, tireRows); err != nil {
			d.logger.Error("Failed to insert tire data",
				zap.Error(err))
			return fmt.Errorf("failed to insert tires: %w", err)
		}

		d.logger.Info("Tire price data inserted successfully",
			zap.Int("rows", len(tireRows)))
	}

	// 3. Insert disk price data if present
	if len(diskRows) > 0 {
		d.logger.Info("Inserting disk price data",
			zap.Int("rows", len(diskRows)))

		if err := d.insertDisksInTx(ctx, tx, diskRows); err != nil {
			d.logger.Error("Failed to insert disk data",
				zap.Error(err))
			return fmt.Errorf("failed to insert disks: %w", err)
		}

		d.logger.Info("Disk price data inserted successfully",
			zap.Int("rows", len(diskRows)))
	}

	// 4. Mark email as processed - ONLY at the end after ALL data is saved!
	d.logger.Info("All data inserted successfully, marking email as processed",
		zap.String("message_id", messageID))

	result, err := tx.ExecContext(ctx,
		`INSERT INTO processed_emails (message_id, email_date) VALUES ($1, $2) ON CONFLICT (message_id) DO NOTHING`,
		messageID, emailDate,
	)
	if err != nil {
		d.logger.Error("Failed to mark email as processed",
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	d.logger.Info("Email marked as processed",
		zap.String("message_id", messageID),
		zap.Int64("rows_affected", rowsAffected))

	// 5. Commit entire transaction
	d.logger.Info("Committing atomic transaction",
		zap.String("message_id", messageID),
		zap.Int("total_nomenclature", len(nomenclatureRows)),
		zap.Int("total_tires", len(tireRows)),
		zap.Int("total_disks", len(diskRows)))

	if err := tx.Commit(); err != nil {
		d.logger.Error("Failed to commit transaction",
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	d.logger.Info("✅ Atomic transaction committed successfully - ALL data saved",
		zap.String("message_id", messageID),
		zap.Int("nomenclature_rows", len(nomenclatureRows)),
		zap.Int("tire_rows", len(tireRows)),
		zap.Int("disk_rows", len(diskRows)))

	return nil
}

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
-- Table: mrc_etalon
-- Stores nomenclature data extracted from Excel files
CREATE TABLE IF NOT EXISTS mrc_etalon (
    id SERIAL PRIMARY KEY,
    article TEXT,
    brand TEXT,
    type TEXT,
    size_model TEXT,
    nomenclature TEXT,
    mrc NUMERIC,
    isimport INTEGER DEFAULT 0,
    isimport_1С INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now()
);

-- Add indices for common queries
CREATE INDEX IF NOT EXISTS idx_mrc_etalon_article ON mrc_etalon(article);
CREATE INDEX IF NOT EXISTS idx_mrc_etalon_brand ON mrc_etalon(brand);
CREATE INDEX IF NOT EXISTS idx_mrc_etalon_isimport ON mrc_etalon(isimport);
CREATE INDEX IF NOT EXISTS idx_mrc_etalon_isimport_1С ON mrc_etalon(isimport_1С);
CREATE INDEX IF NOT EXISTS idx_mrc_etalon_created_at ON mrc_etalon(created_at);

-- Composite index for deduplication by (article, mrc)
-- This dramatically speeds up duplicate detection during batch inserts
CREATE INDEX IF NOT EXISTS idx_mrc_etalon_dedup ON mrc_etalon(article, mrc);

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

-- Table: tyres_prices_stock
-- Stores tire prices and stock from suppliers
CREATE TABLE IF NOT EXISTS tyres_prices_stock (
    id SERIAL PRIMARY KEY,
    cae TEXT NOT NULL,
    price NUMERIC,
    stock INTEGER,
    warehouse_name TEXT,
    provider TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);

-- Indices for fast lookups
CREATE INDEX IF NOT EXISTS idx_tyres_prices_stock_cae ON tyres_prices_stock(cae);
CREATE INDEX IF NOT EXISTS idx_tyres_prices_stock_provider ON tyres_prices_stock(provider);
CREATE INDEX IF NOT EXISTS idx_tyres_prices_stock_created_at ON tyres_prices_stock(created_at);

-- UNIQUE constraint for UPSERT logic: (cae, warehouse_name, provider)
-- Ensures one record per article+warehouse+provider combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_tyres_prices_stock_unique ON tyres_prices_stock(cae, warehouse_name, provider);

-- Table: rims_prices_stock
-- Stores rim/wheel prices and stock from suppliers
CREATE TABLE IF NOT EXISTS rims_prices_stock (
    id SERIAL PRIMARY KEY,
    cae TEXT NOT NULL,
    price NUMERIC,
    stock INTEGER,
    warehouse_name TEXT,
    provider TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);

-- Indices for fast lookups
CREATE INDEX IF NOT EXISTS idx_rims_prices_stock_cae ON rims_prices_stock(cae);
CREATE INDEX IF NOT EXISTS idx_rims_prices_stock_provider ON rims_prices_stock(provider);
CREATE INDEX IF NOT EXISTS idx_rims_prices_stock_created_at ON rims_prices_stock(created_at);

-- UNIQUE constraint for UPSERT logic: (cae, warehouse_name, provider)
-- Ensures one record per article+warehouse+provider combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_rims_prices_stock_unique ON rims_prices_stock(cae, warehouse_name, provider);

-- Table: nomenclature_rims
-- Stores rim nomenclature data (only for ЗАПАСКА provider and specific manufacturers)
CREATE TABLE IF NOT EXISTS nomenclature_rims (
    id SERIAL PRIMARY KEY,
    cae TEXT NOT NULL,
    name TEXT,
    width NUMERIC,
    diameter NUMERIC,
    bolts_count INTEGER,
    bolts_spacing NUMERIC,
    et TEXT,
    dia TEXT,
    model TEXT,
    brand TEXT,
    color TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_nomenclature_rims_cae ON nomenclature_rims(cae);
CREATE INDEX IF NOT EXISTS idx_nomenclature_rims_brand ON nomenclature_rims(brand);
CREATE INDEX IF NOT EXISTS idx_nomenclature_rims_created_at ON nomenclature_rims(created_at);

-- UNIQUE constraint on cae to prevent duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_nomenclature_rims_cae_unique ON nomenclature_rims(cae);
`

// Database represents the database connection
type Database struct {
	db     *sql.DB
	logger *zap.Logger
}

// NomenclatureRow represents a row in mrc_etalon table
type NomenclatureRow struct {
	Article      string
	Brand        string
	Type         string
	SizeModel    string
	Nomenclature string
	MRC          float64
	EmailDate    time.Time
}

// TyrePriceStockRow represents a row in tyres_prices_stock table
type TyrePriceStockRow struct {
	CAE           string
	Price         float64
	Stock         int
	WarehouseName string
	Provider      string
	EmailDate     time.Time
}

// RimPriceStockRow represents a row in rims_prices_stock table
type RimPriceStockRow struct {
	CAE           string
	Price         float64
	Stock         int
	WarehouseName string
	Provider      string
	EmailDate     time.Time
}

// NomenclatureRimRow represents a row in nomenclature_rims table
type NomenclatureRimRow struct {
	CAE          string
	Name         string
	Width        float64
	Diameter     float64
	BoltsCount   int
	BoltsSpacing float64
	ET           string
	DIA          string
	Model        string
	Brand        string
	Color        string
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
		AND table_name IN ('mrc_etalon', 'processed_emails', 'tyres_prices_stock', 'rims_prices_stock', 'nomenclature_rims')
	`

	var count int
	if err := d.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to query tables: %w", err)
	}

	// All five tables should exist
	return count == 5, nil
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
	// Migration 1: Add email_date column to mrc_etalon
	if err := d.addColumnIfNotExists(ctx, "mrc_etalon", "email_date", "TIMESTAMP"); err != nil {
		return err
	}

	// Migration 2: Add deduplication composite index for mrc_etalon
	if err := d.addIndexIfNotExists(ctx, "idx_mrc_etalon_dedup",
		"CREATE INDEX IF NOT EXISTS idx_mrc_etalon_dedup ON mrc_etalon(article, mrc)"); err != nil {
		return err
	}

	// Migration 3: Drop old price_tires table if it exists
	if err := d.dropTableIfExists(ctx, "price_tires"); err != nil {
		return err
	}

	// Migration 4: Drop old price_disks table if it exists
	if err := d.dropTableIfExists(ctx, "price_disks"); err != nil {
		return err
	}

	// Migration 5: Add isimport_1С column to mrc_etalon
	if err := d.addColumnIfNotExists(ctx, "mrc_etalon", "isimport_1С", "INTEGER DEFAULT 0"); err != nil {
		return err
	}

	// Migration 6: Add index on isimport_1С column
	if err := d.addIndexIfNotExists(ctx, "idx_mrc_etalon_isimport_1С",
		"CREATE INDEX IF NOT EXISTS idx_mrc_etalon_isimport_1С ON mrc_etalon(isimport_1С)"); err != nil {
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

// dropTableIfExists drops a table if it exists
func (d *Database) dropTableIfExists(ctx context.Context, tableName string) error {
	// Check if table exists
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`

	var exists bool
	if err := d.db.QueryRowContext(ctx, checkQuery, tableName).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check table %s: %w", tableName, err)
	}

	if !exists {
		d.logger.Debug("Table does not exist, skipping drop", zap.String("table", tableName))
		return nil
	}

	// Drop table
	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
	if _, err := d.db.ExecContext(ctx, dropSQL); err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}

	d.logger.Info("Dropped table", zap.String("table", tableName))
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
		INSERT INTO mrc_etalon
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

	// Insert nomenclature data in batches with deduplication by (article, mrc)
	if len(rows) > 0 {
		batchSize := 1000
		totalBatches := (len(rows) + batchSize - 1) / batchSize
		d.logger.Info("Preparing to insert data in batches with deduplication",
			zap.Int("total_rows", len(rows)),
			zap.Int("batch_size", batchSize),
			zap.Int("total_batches", totalBatches))

		totalInserted := int64(0)
		totalSkipped := int64(0)

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
			brands := make([]string, len(batch))
			types := make([]string, len(batch))
			sizeModels := make([]string, len(batch))
			nomenclatures := make([]string, len(batch))
			mrcs := make([]float64, len(batch))
			emailDates := make([]time.Time, len(batch))

			for idx, row := range batch {
				articles[idx] = strings.TrimSpace(row.Article) // Normalize: trim spaces
				brands[idx] = row.Brand
				types[idx] = row.Type
				sizeModels[idx] = row.SizeModel
				nomenclatures[idx] = row.Nomenclature
				mrcs[idx] = row.MRC
				emailDates[idx] = row.EmailDate
			}

			// Use INSERT ... SELECT with deduplication via NOT EXISTS
			// Deduplication check: article (trimmed) + mrc (numeric comparison)
			// Append-only: no DELETE, no UPDATE, only INSERT new records
			query := `
				WITH new_data AS (
					SELECT * FROM unnest(
						$1::text[], $2::text[], $3::text[], $4::text[],
						$5::text[], $6::numeric[], $7::timestamp[]
					) AS t(article, brand, type, size_model, nomenclature, mrc, email_date)
				)
				INSERT INTO mrc_etalon
				(article, brand, type, size_model, nomenclature, mrc, email_date, isimport)
				SELECT article, brand, type, size_model, nomenclature, mrc, email_date, 0
				FROM new_data nd
				WHERE NOT EXISTS (
					SELECT 1 FROM mrc_etalon en
					WHERE TRIM(en.article) = TRIM(nd.article)
					  AND en.mrc = nd.mrc
				)
			`

			d.logger.Debug("Executing INSERT with deduplication by (article, mrc)",
				zap.Int("batch_num", batchNum),
				zap.Int("batch_size", len(batch)))

			result, err := tx.ExecContext(ctx, query,
				pq.Array(articles), pq.Array(brands), pq.Array(types),
				pq.Array(sizeModels), pq.Array(nomenclatures), pq.Array(mrcs),
				pq.Array(emailDates))
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
			totalInserted += rowsAffected
			totalSkipped += skipped

			d.logger.Info("Batch processed with deduplication",
				zap.Int("batch_num", batchNum),
				zap.Int("batch_size", len(batch)),
				zap.Int64("inserted", rowsAffected),
				zap.Int64("skipped_duplicates", skipped))
		}

		d.logger.Info("All batches completed",
			zap.Int("total_rows", len(rows)),
			zap.Int64("total_inserted", totalInserted),
			zap.Int64("total_skipped", totalSkipped))
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

// InsertTyrePriceStockWithEmail inserts/updates tyre price stock data and marks email as processed in a transaction
// Logic:
// - If exact match (cae, price, stock, warehouse_name, provider) exists -> SKIP
// - If record exists with same (cae, warehouse_name, provider) but different price/stock -> UPDATE with isimport=0
// - If no record exists -> INSERT with isimport=0
func (d *Database) InsertTyrePriceStockWithEmail(ctx context.Context, rows []TyrePriceStockRow, messageID string) error {
	d.logger.Info("Starting InsertTyrePriceStockWithEmail",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		d.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	d.logger.Info("Transaction started successfully")

	// Insert/update tyre price stock data in batches
	if len(rows) > 0 {
		batchSize := 1000
		totalBatches := (len(rows) + batchSize - 1) / batchSize
		d.logger.Info("Preparing to upsert tyre price stock data in batches",
			zap.Int("total_rows", len(rows)),
			zap.Int("batch_size", batchSize),
			zap.Int("total_batches", totalBatches))

		totalInserted := int64(0)
		totalSkipped := int64(0)

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

			// Build array parameters
			caes := make([]string, len(batch))
			prices := make([]float64, len(batch))
			stocks := make([]int, len(batch))
			warehouseNames := make([]string, len(batch))
			providers := make([]string, len(batch))
			emailDates := make([]time.Time, len(batch))

			for idx, row := range batch {
				caes[idx] = row.CAE
				prices[idx] = row.Price
				stocks[idx] = row.Stock
				warehouseNames[idx] = row.WarehouseName
				providers[idx] = row.Provider
				emailDates[idx] = row.EmailDate
			}

			// UPSERT with conditional UPDATE:
			// - Insert new records with isimport=0
			// - Update existing records ONLY if price or stock changed, set isimport=0
			// - Skip if no changes (price and stock are the same)
			query := `
				WITH new_data AS (
					SELECT * FROM unnest(
						$1::text[], $2::numeric[], $3::integer[], $4::text[], $5::text[], $6::timestamp[]
					) AS t(cae, price, stock, warehouse_name, provider, email_date)
				)
				INSERT INTO tyres_prices_stock (cae, price, stock, warehouse_name, provider, email_date, isimport, created_at)
				SELECT cae, price, stock, warehouse_name, provider, email_date, 0, now()
				FROM new_data nd
				ON CONFLICT (cae, warehouse_name, provider)
				DO UPDATE SET
					price = EXCLUDED.price,
					stock = EXCLUDED.stock,
					email_date = EXCLUDED.email_date,
					isimport = 0,
					created_at = now()
				WHERE tyres_prices_stock.price != EXCLUDED.price
				   OR tyres_prices_stock.stock != EXCLUDED.stock
			`

			d.logger.Debug("Executing UPSERT with conditional update",
				zap.Int("batch_num", batchNum),
				zap.Int("batch_size", len(batch)))

			result, err := tx.ExecContext(ctx, query,
				pq.Array(caes), pq.Array(prices), pq.Array(stocks),
				pq.Array(warehouseNames), pq.Array(providers), pq.Array(emailDates))
			if err != nil {
				d.logger.Error("Failed to upsert batch",
					zap.Int("batch_num", batchNum),
					zap.Int("batch_start", i),
					zap.Int("batch_size", len(batch)),
					zap.Error(err))
				return fmt.Errorf("failed to upsert batch %d: %w", batchNum, err)
			}

			rowsAffected, _ := result.RowsAffected()
			// rowsAffected includes both INSERTs and UPDATEs
			// Rows with no changes are skipped (not counted)
			skipped := int64(len(batch)) - rowsAffected

			totalInserted += rowsAffected // This is insert + update count
			totalSkipped += skipped

			d.logger.Info("Batch processed with UPSERT",
				zap.Int("batch_num", batchNum),
				zap.Int("batch_size", len(batch)),
				zap.Int64("inserted_or_updated", rowsAffected),
				zap.Int64("skipped_no_changes", skipped))
		}

		d.logger.Info("All batches processed",
			zap.Int64("total_inserted_or_updated", totalInserted),
			zap.Int64("total_skipped", totalSkipped))
	}

	d.logger.Info("All batches processed, marking email as processed",
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

	d.logger.Info("Transaction committed successfully - tyre price stock data saved to database",
		zap.Int("total_rows", len(rows)),
		zap.String("message_id", messageID))

	return nil
}

// InsertPriceDisksWithEmail inserts price disk data and marks email as processed in a transaction
// insertNomenclatureInTx inserts nomenclature data within an existing transaction
// with deduplication by (article, mrc) - append-only, no deletes
func (d *Database) insertNomenclatureInTx(ctx context.Context, tx *sql.Tx, rows []NomenclatureRow) error {
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
		brands := make([]string, len(batch))
		types := make([]string, len(batch))
		sizeModels := make([]string, len(batch))
		nomenclatures := make([]string, len(batch))
		mrcs := make([]float64, len(batch))
		emailDates := make([]time.Time, len(batch))

		for idx, row := range batch {
			articles[idx] = strings.TrimSpace(row.Article) // Normalize: trim spaces
			brands[idx] = row.Brand
			types[idx] = row.Type
			sizeModels[idx] = row.SizeModel
			nomenclatures[idx] = row.Nomenclature
			mrcs[idx] = row.MRC
			emailDates[idx] = row.EmailDate
		}

		// Use INSERT ... SELECT with deduplication via NOT EXISTS
		// Deduplication check: article (trimmed) + mrc (numeric comparison)
		// Append-only: no DELETE, no UPDATE, only INSERT new records
		query := `
			WITH new_data AS (
				SELECT * FROM unnest(
					$1::text[], $2::text[], $3::text[], $4::text[],
					$5::text[], $6::numeric[], $7::timestamp[]
				) AS t(article, brand, type, size_model, nomenclature, mrc, email_date)
			)
			INSERT INTO mrc_etalon
			(article, brand, type, size_model, nomenclature, mrc, email_date, isimport)
			SELECT article, brand, type, size_model, nomenclature, mrc, email_date, 0
			FROM new_data nd
			WHERE NOT EXISTS (
				SELECT 1 FROM mrc_etalon en
				WHERE TRIM(en.article) = TRIM(nd.article)
				  AND en.mrc = nd.mrc
			)
		`

		_, err := tx.ExecContext(ctx, query,
			pq.Array(articles), pq.Array(brands), pq.Array(types),
			pq.Array(sizeModels), pq.Array(nomenclatures), pq.Array(mrcs),
			pq.Array(emailDates))
		if err != nil {
			return fmt.Errorf("failed to insert batch %d: %w", batchNum, err)
		}
	}

	return nil
}

// insertTyrePriceStockInTx inserts/updates tyre price stock data within an existing transaction
func (d *Database) insertTyrePriceStockInTx(ctx context.Context, tx *sql.Tx, rows []TyrePriceStockRow) error {
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

		// Build array parameters
		caes := make([]string, len(batch))
		prices := make([]float64, len(batch))
		stocks := make([]int, len(batch))
		warehouseNames := make([]string, len(batch))
		providers := make([]string, len(batch))
		emailDates := make([]time.Time, len(batch))

		for idx, row := range batch {
			caes[idx] = row.CAE
			prices[idx] = row.Price
			stocks[idx] = row.Stock
			warehouseNames[idx] = row.WarehouseName
			providers[idx] = row.Provider
			emailDates[idx] = row.EmailDate
		}

		// UPSERT with conditional UPDATE (same logic as InsertTyrePriceStockWithEmail)
		query := `
			WITH new_data AS (
				SELECT * FROM unnest(
					$1::text[], $2::numeric[], $3::integer[], $4::text[], $5::text[], $6::timestamp[]
				) AS t(cae, price, stock, warehouse_name, provider, email_date)
			)
			INSERT INTO tyres_prices_stock (cae, price, stock, warehouse_name, provider, email_date, isimport, updated_at)
			SELECT cae, price, stock, warehouse_name, provider, email_date, 0, now()
			FROM new_data nd
			ON CONFLICT (cae, warehouse_name, provider)
			DO UPDATE SET
				price = EXCLUDED.price,
				stock = EXCLUDED.stock,
				email_date = EXCLUDED.email_date,
				isimport = 0,
				updated_at = now()
			WHERE tyres_prices_stock.price != EXCLUDED.price
			   OR tyres_prices_stock.stock != EXCLUDED.stock
		`

		_, err := tx.ExecContext(ctx, query,
			pq.Array(caes), pq.Array(prices), pq.Array(stocks),
			pq.Array(warehouseNames), pq.Array(providers), pq.Array(emailDates))
		if err != nil {
			return fmt.Errorf("failed to upsert batch %d: %w", batchNum, err)
		}
	}

	return nil
}

// insertRimPriceStockInTx inserts/updates rim price stock data within an existing transaction
func (d *Database) insertRimPriceStockInTx(ctx context.Context, tx *sql.Tx, rows []RimPriceStockRow) error {
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

		// Build array parameters
		caes := make([]string, len(batch))
		prices := make([]float64, len(batch))
		stocks := make([]int, len(batch))
		warehouseNames := make([]string, len(batch))
		providers := make([]string, len(batch))
		emailDates := make([]time.Time, len(batch))

		for idx, row := range batch {
			caes[idx] = row.CAE
			prices[idx] = row.Price
			stocks[idx] = row.Stock
			warehouseNames[idx] = row.WarehouseName
			providers[idx] = row.Provider
			emailDates[idx] = row.EmailDate
		}

		// UPSERT with conditional UPDATE (same logic as tyres_prices_stock)
		query := `
			WITH new_data AS (
				SELECT * FROM unnest(
					$1::text[], $2::numeric[], $3::integer[], $4::text[], $5::text[], $6::timestamp[]
				) AS t(cae, price, stock, warehouse_name, provider, email_date)
			)
			INSERT INTO rims_prices_stock (cae, price, stock, warehouse_name, provider, email_date, isimport, updated_at)
			SELECT cae, price, stock, warehouse_name, provider, email_date, 0, now()
			FROM new_data nd
			ON CONFLICT (cae, warehouse_name, provider)
			DO UPDATE SET
				price = EXCLUDED.price,
				stock = EXCLUDED.stock,
				email_date = EXCLUDED.email_date,
				isimport = 0,
				updated_at = now()
			WHERE rims_prices_stock.price != EXCLUDED.price
			   OR rims_prices_stock.stock != EXCLUDED.stock
			   OR rims_prices_stock.warehouse_name != EXCLUDED.warehouse_name
			   OR rims_prices_stock.provider != EXCLUDED.provider
		`

		_, err := tx.ExecContext(ctx, query,
			pq.Array(caes), pq.Array(prices), pq.Array(stocks),
			pq.Array(warehouseNames), pq.Array(providers), pq.Array(emailDates))
		if err != nil {
			return fmt.Errorf("failed to upsert batch %d: %w", batchNum, err)
		}
	}

	return nil
}

// insertRimNomenclatureInTx inserts rim nomenclature data within an existing transaction
func (d *Database) insertRimNomenclatureInTx(ctx context.Context, tx *sql.Tx, rows []NomenclatureRimRow) error {
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

		// Build array parameters
		caes := make([]string, len(batch))
		names := make([]string, len(batch))
		widths := make([]float64, len(batch))
		diameters := make([]float64, len(batch))
		boltsCounts := make([]int, len(batch))
		boltsSpacings := make([]float64, len(batch))
		ets := make([]string, len(batch))
		dias := make([]string, len(batch))
		models := make([]string, len(batch))
		brands := make([]string, len(batch))
		colors := make([]string, len(batch))
		emailDates := make([]time.Time, len(batch))

		for idx, row := range batch {
			caes[idx] = row.CAE
			names[idx] = row.Name
			widths[idx] = row.Width
			diameters[idx] = row.Diameter
			boltsCounts[idx] = row.BoltsCount
			boltsSpacings[idx] = row.BoltsSpacing
			ets[idx] = row.ET
			dias[idx] = row.DIA
			models[idx] = row.Model
			brands[idx] = row.Brand
			colors[idx] = row.Color
			emailDates[idx] = row.EmailDate
		}

		// INSERT with ON CONFLICT DO NOTHING (skip if cae exists)
		query := `
			WITH new_data AS (
				SELECT * FROM unnest(
					$1::text[], $2::text[], $3::numeric[], $4::numeric[], $5::integer[],
					$6::numeric[], $7::text[], $8::text[], $9::text[], $10::text[], $11::text[], $12::timestamp[]
				) AS t(cae, name, width, diameter, bolts_count, bolts_spacing, et, dia, model, brand, color, email_date)
			)
			INSERT INTO nomenclature_rims
			(cae, name, width, diameter, bolts_count, bolts_spacing, et, dia, model, brand, color, email_date, isimport, created_at)
			SELECT cae, name, width, diameter, bolts_count, bolts_spacing, et, dia, model, brand, color, email_date, 0, now()
			FROM new_data nd
			ON CONFLICT (cae) DO NOTHING
		`

		_, err := tx.ExecContext(ctx, query,
			pq.Array(caes), pq.Array(names), pq.Array(widths), pq.Array(diameters),
			pq.Array(boltsCounts), pq.Array(boltsSpacings), pq.Array(ets), pq.Array(dias),
			pq.Array(models), pq.Array(brands), pq.Array(colors), pq.Array(emailDates))
		if err != nil {
			return fmt.Errorf("failed to insert batch %d: %w", batchNum, err)
		}
	}

	return nil
}

// InsertAllEmailDataWithTransaction inserts all email data (nomenclature, tyres, rims)
// in a SINGLE atomic transaction. Email is marked as processed ONLY if ALL data saves successfully.
func (d *Database) InsertAllEmailDataWithTransaction(
	ctx context.Context,
	nomenclatureRows []NomenclatureRow,
	tyreRows []TyrePriceStockRow,
	rimPriceRows []RimPriceStockRow,
	rimNomenclatureRows []NomenclatureRimRow,
	messageID string,
	emailDate time.Time,
) error {
	d.logger.Info("Starting atomic email data transaction",
		zap.String("message_id", messageID),
		zap.Int("nomenclature_rows", len(nomenclatureRows)),
		zap.Int("tyre_rows", len(tyreRows)),
		zap.Int("rim_price_rows", len(rimPriceRows)),
		zap.Int("rim_nomenclature_rows", len(rimNomenclatureRows)))

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

	// 2. Insert/update tyre price stock data if present
	if len(tyreRows) > 0 {
		d.logger.Info("Inserting/updating tyre price stock data",
			zap.Int("rows", len(tyreRows)))

		if err := d.insertTyrePriceStockInTx(ctx, tx, tyreRows); err != nil {
			d.logger.Error("Failed to insert/update tyre data",
				zap.Error(err))
			return fmt.Errorf("failed to insert/update tyres: %w", err)
		}

		d.logger.Info("Tyre price stock data inserted/updated successfully",
			zap.Int("rows", len(tyreRows)))
	}

	// 3. Insert/update rim price stock data if present
	if len(rimPriceRows) > 0 {
		d.logger.Info("Inserting/updating rim price stock data",
			zap.Int("rows", len(rimPriceRows)))

		if err := d.insertRimPriceStockInTx(ctx, tx, rimPriceRows); err != nil {
			d.logger.Error("Failed to insert/update rim price data",
				zap.Error(err))
			return fmt.Errorf("failed to insert/update rim prices: %w", err)
		}

		d.logger.Info("Rim price stock data inserted/updated successfully",
			zap.Int("rows", len(rimPriceRows)))
	}

	// 4. Insert rim nomenclature data if present
	if len(rimNomenclatureRows) > 0 {
		d.logger.Info("Inserting rim nomenclature data",
			zap.Int("rows", len(rimNomenclatureRows)))

		if err := d.insertRimNomenclatureInTx(ctx, tx, rimNomenclatureRows); err != nil {
			d.logger.Error("Failed to insert rim nomenclature data",
				zap.Error(err))
			return fmt.Errorf("failed to insert rim nomenclature: %w", err)
		}

		d.logger.Info("Rim nomenclature data inserted successfully",
			zap.Int("rows", len(rimNomenclatureRows)))
	}

	// 5. Mark email as processed - ONLY at the end after ALL data is saved!
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

	// 6. Commit entire transaction
	d.logger.Info("Committing atomic transaction",
		zap.String("message_id", messageID),
		zap.Int("total_nomenclature", len(nomenclatureRows)),
		zap.Int("total_tyres", len(tyreRows)),
		zap.Int("total_rim_prices", len(rimPriceRows)),
		zap.Int("total_rim_nomenclature", len(rimNomenclatureRows)))

	if err := tx.Commit(); err != nil {
		d.logger.Error("Failed to commit transaction",
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	d.logger.Info("✅ Atomic transaction committed successfully - ALL data saved",
		zap.String("message_id", messageID),
		zap.Int("nomenclature_rows", len(nomenclatureRows)),
		zap.Int("tyre_rows", len(tyreRows)),
		zap.Int("rim_price_rows", len(rimPriceRows)),
		zap.Int("rim_nomenclature_rows", len(rimNomenclatureRows)))

	return nil
}

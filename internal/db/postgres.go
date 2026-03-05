package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/prokoleso/etalon-nomenclature/config"
	_ "github.com/lib/pq"
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
    processed_at TIMESTAMP DEFAULT now()
);

-- Unique index on message_id to prevent duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_processed_emails_message_id
ON processed_emails(message_id);
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

	if exists {
		d.logger.Info("Database schema is up to date")
		return nil
	}

	d.logger.Info("Required tables not found, applying migrations...")
	if err := d.applyMigrations(ctx); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	d.logger.Info("Migrations applied successfully")
	return nil
}

// checkTablesExist verifies that required tables exist in the database
func (d *Database) checkTablesExist(ctx context.Context) (bool, error) {
	// Check for both required tables
	query := `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('etalon_nomenclature', 'processed_emails')
	`

	var count int
	if err := d.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to query tables: %w", err)
	}

	// Both tables should exist
	return count == 2, nil
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
func (d *Database) MarkEmailAsProcessed(ctx context.Context, messageID string) error {
	query := `INSERT INTO processed_emails (message_id) VALUES ($1) ON CONFLICT (message_id) DO NOTHING`

	_, err := d.db.ExecContext(ctx, query, messageID)
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
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert nomenclature data
	if len(rows) > 0 {
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
	}

	// Mark email as processed
	_, err = tx.ExecContext(ctx,
		`INSERT INTO processed_emails (message_id) VALUES ($1) ON CONFLICT (message_id) DO NOTHING`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	d.logger.Info("Inserted nomenclature data and marked email as processed",
		zap.Int("rows", len(rows)),
		zap.String("message_id", messageID))

	return nil
}

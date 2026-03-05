-- Migration: Initial schema
-- Description: Create tables for etalon_nomenclature and processed_emails

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

-- Add comment for documentation
COMMENT ON TABLE etalon_nomenclature IS 'Stores nomenclature data from Excel attachments';
COMMENT ON TABLE processed_emails IS 'Tracks processed email Message-IDs to prevent reprocessing';
COMMENT ON COLUMN etalon_nomenclature.isimport IS '0 = data not imported by system, 1 = imported';

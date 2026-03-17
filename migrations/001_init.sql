-- Migration: Initial schema
-- Description: Create tables for mrc_etalon and processed_emails

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
COMMENT ON TABLE mrc_etalon IS 'Stores nomenclature data from Excel attachments';
COMMENT ON TABLE processed_emails IS 'Tracks processed email Message-IDs to prevent reprocessing';
COMMENT ON COLUMN mrc_etalon.isimport IS '0 = data not imported by system, 1 = imported';

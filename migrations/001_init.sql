-- Migration: Initial schema
-- Description: Create tables for MRC_Etalon and processed_emails

-- Table: MRC_Etalon
-- Stores nomenclature data extracted from Excel files
CREATE TABLE IF NOT EXISTS MRC_Etalon (
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
CREATE INDEX IF NOT EXISTS idx_MRC_Etalon_article ON MRC_Etalon(article);
CREATE INDEX IF NOT EXISTS idx_MRC_Etalon_brand ON MRC_Etalon(brand);
CREATE INDEX IF NOT EXISTS idx_MRC_Etalon_isimport ON MRC_Etalon(isimport);
CREATE INDEX IF NOT EXISTS idx_MRC_Etalon_isimport_1С ON MRC_Etalon(isimport_1С);
CREATE INDEX IF NOT EXISTS idx_MRC_Etalon_created_at ON MRC_Etalon(created_at);

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
COMMENT ON TABLE MRC_Etalon IS 'Stores nomenclature data from Excel attachments';
COMMENT ON TABLE processed_emails IS 'Tracks processed email Message-IDs to prevent reprocessing';
COMMENT ON COLUMN MRC_Etalon.isimport IS '0 = data not imported by system, 1 = imported';

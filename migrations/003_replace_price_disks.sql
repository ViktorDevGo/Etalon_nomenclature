-- Migration: Replace price_disks with rims_prices_stock and nomenclature_rims
-- Description: Creates new rims_prices_stock and nomenclature_rims tables, drops old price_disks table

-- Create table: rims_prices_stock (for all providers)
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

-- Create indices for rims_prices_stock
CREATE INDEX IF NOT EXISTS idx_rims_prices_stock_cae ON rims_prices_stock(cae);
CREATE INDEX IF NOT EXISTS idx_rims_prices_stock_provider ON rims_prices_stock(provider);
CREATE INDEX IF NOT EXISTS idx_rims_prices_stock_created_at ON rims_prices_stock(created_at);

-- Create UNIQUE constraint for UPSERT logic: (cae, warehouse_name, provider)
CREATE UNIQUE INDEX IF NOT EXISTS idx_rims_prices_stock_unique ON rims_prices_stock(cae, warehouse_name, provider);

-- Create table: nomenclature_rims (only for ЗАПАСКА + specific manufacturers)
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

-- Create indices for nomenclature_rims
CREATE INDEX IF NOT EXISTS idx_nomenclature_rims_cae ON nomenclature_rims(cae);
CREATE INDEX IF NOT EXISTS idx_nomenclature_rims_brand ON nomenclature_rims(brand);
CREATE INDEX IF NOT EXISTS idx_nomenclature_rims_created_at ON nomenclature_rims(created_at);

-- Create UNIQUE constraint: (cae) - skip if CAE already exists
CREATE UNIQUE INDEX IF NOT EXISTS idx_nomenclature_rims_unique ON nomenclature_rims(cae);

-- Drop old table price_disks
DROP TABLE IF EXISTS price_disks CASCADE;

-- Add comments for documentation
COMMENT ON TABLE rims_prices_stock IS 'Stores rim/disk prices and stock from all suppliers with UPSERT logic on (cae, warehouse_name, provider)';
COMMENT ON COLUMN rims_prices_stock.isimport IS '0 = new or updated record, set to 0 on every change';
COMMENT ON COLUMN rims_prices_stock.cae IS 'Article code (formerly known as article)';
COMMENT ON COLUMN rims_prices_stock.stock IS 'Stock quantity (formerly known as balance)';
COMMENT ON COLUMN rims_prices_stock.warehouse_name IS 'Warehouse name (formerly known as store)';

COMMENT ON TABLE nomenclature_rims IS 'Stores rim nomenclature data ONLY for ЗАПАСКА provider with manufacturers: COX, FF, Koko, Sakura. Uses SKIP logic on duplicate CAE.';
COMMENT ON COLUMN nomenclature_rims.bolts_count IS 'Number of bolts (extracted from drilling field, e.g., "5" from "5*114.3")';
COMMENT ON COLUMN nomenclature_rims.bolts_spacing IS 'Bolts spacing in mm (extracted from drilling field, e.g., "114.3" from "5*114.3")';
COMMENT ON COLUMN nomenclature_rims.isimport IS '0 = new record, always 0 for nomenclature';

-- Migration: Replace price_tires with tyres_prices_stock
-- Description: Creates new tyres_prices_stock table and drops old price_tires table

-- Create new table: tyres_prices_stock
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

-- Create indices
CREATE INDEX IF NOT EXISTS idx_tyres_prices_stock_cae ON tyres_prices_stock(cae);
CREATE INDEX IF NOT EXISTS idx_tyres_prices_stock_provider ON tyres_prices_stock(provider);
CREATE INDEX IF NOT EXISTS idx_tyres_prices_stock_created_at ON tyres_prices_stock(created_at);

-- Create UNIQUE constraint for UPSERT logic: (cae, warehouse_name, provider)
-- This ensures one record per article+warehouse+provider combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_tyres_prices_stock_unique ON tyres_prices_stock(cae, warehouse_name, provider);

-- Drop old table price_tires
DROP TABLE IF EXISTS price_tires CASCADE;

-- Add comment for documentation
COMMENT ON TABLE tyres_prices_stock IS 'Stores tyre prices and stock from suppliers with UPSERT logic on (cae, warehouse_name, provider)';
COMMENT ON COLUMN tyres_prices_stock.isimport IS '0 = new or updated record, set to 0 on every change';
COMMENT ON COLUMN tyres_prices_stock.cae IS 'Article code (formerly known as article)';
COMMENT ON COLUMN tyres_prices_stock.stock IS 'Stock quantity (formerly known as balance)';
COMMENT ON COLUMN tyres_prices_stock.warehouse_name IS 'Warehouse name (formerly known as store)';

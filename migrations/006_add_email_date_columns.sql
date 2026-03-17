-- Migration: Add email_date columns to price tables
-- Description: Adds email_date timestamp column to tyres_prices_stock, rims_prices_stock, and nomenclature_rims

-- Add email_date to tyres_prices_stock
ALTER TABLE tyres_prices_stock ADD COLUMN IF NOT EXISTS email_date TIMESTAMP;

-- Add email_date to rims_prices_stock
ALTER TABLE rims_prices_stock ADD COLUMN IF NOT EXISTS email_date TIMESTAMP;

-- Add email_date to nomenclature_rims
ALTER TABLE nomenclature_rims ADD COLUMN IF NOT EXISTS email_date TIMESTAMP;

-- Add comments for documentation
COMMENT ON COLUMN tyres_prices_stock.email_date IS 'Date from the email when price data was received';
COMMENT ON COLUMN rims_prices_stock.email_date IS 'Date from the email when price data was received';
COMMENT ON COLUMN nomenclature_rims.email_date IS 'Date from the email when nomenclature data was received';

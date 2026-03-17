-- Migration: Rename etalon_nomenclature to mrc_etalon
-- Description: Renames the main nomenclature table and all its indices

-- Rename the table
ALTER TABLE IF EXISTS etalon_nomenclature RENAME TO mrc_etalon;

-- Rename all indices
ALTER INDEX IF EXISTS idx_etalon_nomenclature_article RENAME TO idx_mrc_etalon_article;
ALTER INDEX IF EXISTS idx_etalon_nomenclature_brand RENAME TO idx_mrc_etalon_brand;
ALTER INDEX IF EXISTS idx_etalon_nomenclature_isimport RENAME TO idx_mrc_etalon_isimport;
ALTER INDEX IF EXISTS idx_etalon_nomenclature_created_at RENAME TO idx_mrc_etalon_created_at;
ALTER INDEX IF EXISTS idx_etalon_nomenclature_dedup RENAME TO idx_mrc_etalon_dedup;

-- Update table comment
COMMENT ON TABLE mrc_etalon IS 'Stores nomenclature data with MRC (minimum retail price) from Excel attachments';

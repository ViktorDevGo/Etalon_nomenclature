-- Migration: Rename etalon_nomenclature to MRC_Etalon
-- Description: Renames the main nomenclature table and all its indices

-- Rename the table
ALTER TABLE IF EXISTS etalon_nomenclature RENAME TO MRC_Etalon;

-- Rename all indices
ALTER INDEX IF EXISTS idx_etalon_nomenclature_article RENAME TO idx_MRC_Etalon_article;
ALTER INDEX IF EXISTS idx_etalon_nomenclature_brand RENAME TO idx_MRC_Etalon_brand;
ALTER INDEX IF EXISTS idx_etalon_nomenclature_isimport RENAME TO idx_MRC_Etalon_isimport;
ALTER INDEX IF EXISTS idx_etalon_nomenclature_created_at RENAME TO idx_MRC_Etalon_created_at;
ALTER INDEX IF EXISTS idx_etalon_nomenclature_dedup RENAME TO idx_MRC_Etalon_dedup;

-- Update table comment
COMMENT ON TABLE MRC_Etalon IS 'Stores nomenclature data with MRC (minimum retail price) from Excel attachments';

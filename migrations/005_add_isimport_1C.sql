-- Migration: Add isimport_1С column to mrc_etalon
-- Description: Adds a flag to track if record has been imported to 1С system

-- Add isimport_1С column
ALTER TABLE mrc_etalon ADD COLUMN IF NOT EXISTS isimport_1С INTEGER DEFAULT 0;

-- Add index on isimport_1С column for filtering
CREATE INDEX IF NOT EXISTS idx_mrc_etalon_isimport_1С ON mrc_etalon(isimport_1С);

-- Add comment for documentation
COMMENT ON COLUMN mrc_etalon.isimport_1С IS '0 = not imported to 1С, 1 = imported to 1С';

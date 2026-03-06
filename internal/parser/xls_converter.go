package parser

import (
	"bytes"
	"fmt"
	"time"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

// ConvertXLStoXLSX converts .xls (binary) format to .xlsx (XML) format
func ConvertXLStoXLSX(xlsContent []byte, logger *zap.Logger) (result []byte, err error) {
	startTime := time.Now()

	// Recover from panic in xls library
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during XLS conversion: %v", r)
			if logger != nil {
				logger.Error("Panic during XLS conversion", zap.Any("panic", r))
			}
		}
	}()

	if logger != nil {
		logger.Debug("Starting XLS to XLSX conversion",
			zap.Int("input_size_bytes", len(xlsContent)))
	}

	// Open .xls file
	xlsFile, openErr := xls.OpenReader(bytes.NewReader(xlsContent), "utf-8")
	if openErr != nil {
		if logger != nil {
			logger.Error("Failed to open XLS file", zap.Error(openErr))
		}
		return nil, fmt.Errorf("failed to open xls file: %w", openErr)
	}

	if logger != nil {
		logger.Debug("XLS file opened successfully",
			zap.Int("num_sheets", xlsFile.NumSheets()))
	}

	// Create new .xlsx file
	xlsxFile := excelize.NewFile()

	// Process each sheet
	for sheetIdx := 0; sheetIdx < xlsFile.NumSheets(); sheetIdx++ {
		xlsSheet := xlsFile.GetSheet(sheetIdx)
		if xlsSheet == nil {
			continue
		}

		sheetName := xlsSheet.Name
		if logger != nil {
			logger.Debug("Processing sheet",
				zap.Int("sheet_index", sheetIdx),
				zap.String("sheet_name", sheetName),
				zap.Int("max_row", int(xlsSheet.MaxRow)))
		}

		// Create or use sheet in xlsx
		var xlsxSheetName string
		if sheetIdx == 0 {
			// Rename default sheet
			xlsxSheetName = "Sheet1"
			xlsxFile.SetSheetName(xlsxSheetName, sheetName)
		} else {
			// Create new sheet
			_, err := xlsxFile.NewSheet(sheetName)
			if err != nil {
				return nil, fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
			}
		}

		// Copy data row by row
		maxRow := int(xlsSheet.MaxRow)
		for rowIdx := 0; rowIdx <= maxRow; rowIdx++ {
			xlsRow := xlsSheet.Row(rowIdx)
			if xlsRow == nil {
				continue
			}

			// Get max column for this row
			maxCol := xlsRow.LastCol()

			for colIdx := 0; colIdx <= int(maxCol); colIdx++ {
				cellValue := xlsRow.Col(colIdx)

				// Convert to Excel cell address (A1, B1, etc.)
				cellName, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err != nil {
					continue
				}

				// Set cell value
				if err := xlsxFile.SetCellValue(sheetName, cellName, cellValue); err != nil {
					// Ignore individual cell errors
					continue
				}
			}
		}
	}

	// Write to buffer
	if logger != nil {
		logger.Debug("Writing XLSX to buffer")
	}

	buf := new(bytes.Buffer)
	if writeErr := xlsxFile.Write(buf); writeErr != nil {
		if logger != nil {
			logger.Error("Failed to write XLSX", zap.Error(writeErr))
		}
		return nil, fmt.Errorf("failed to write xlsx: %w", writeErr)
	}

	duration := time.Since(startTime)
	outputSize := buf.Len()

	if logger != nil {
		logger.Info("XLS to XLSX conversion completed",
			zap.Duration("duration", duration),
			zap.Int("input_size", len(xlsContent)),
			zap.Int("output_size", outputSize))
	}

	return buf.Bytes(), nil
}

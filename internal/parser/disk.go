package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prokoleso/etalon-nomenclature/internal/db"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

// DiskParser handles disk price list Excel file parsing
type DiskParser struct {
	logger *zap.Logger
}

// diskColumnMapping represents the mapping of column names to indices
type diskColumnMapping struct {
	article      int
	nomenclature int
	balance      map[int]string // index -> store name (for БИГМАШИН: multiple "Остаток*" columns)
	storeColumn  int            // For ЗАПАСКА/БРИНЕКС single "Склад" column
}

// NewDiskParser creates a new disk parser
func NewDiskParser(logger *zap.Logger) *DiskParser {
	return &DiskParser{
		logger: logger,
	}
}

// Parse parses an Excel disk file and returns disk rows
func (p *DiskParser) Parse(content []byte, filename string, provider string, emailDate time.Time) ([]db.PriceDiskRow, error) {
	// Convert .xls to .xlsx if needed
	if strings.HasSuffix(strings.ToLower(filename), ".xls") && !strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		p.logger.Info("Converting .xls to .xlsx with LibreOffice",
			zap.String("filename", filename),
			zap.Int("size_bytes", len(content)))
		convertedContent, err := ConvertXLStoXLSXWithLibreOffice(content, p.logger)
		if err != nil {
			p.logger.Error("XLS conversion failed",
				zap.String("filename", filename),
				zap.Error(err))
			return nil, fmt.Errorf("failed to convert xls to xlsx: %w", err)
		}
		p.logger.Info("XLS conversion successful with LibreOffice",
			zap.String("filename", filename))
		content = convertedContent
	}

	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	var allRows []db.PriceDiskRow

	// Get all sheet names
	sheets := f.GetSheetList()
	p.logger.Info("Processing disk Excel file",
		zap.String("filename", filename),
		zap.String("provider", provider),
		zap.Int("sheets", len(sheets)))

	for _, sheetName := range sheets {
		// Check if sheet should be processed based on provider
		if !p.shouldProcessSheet(sheetName, provider) {
			p.logger.Debug("Skipping sheet - not applicable for disks",
				zap.String("sheet", sheetName),
				zap.String("provider", provider))
			continue
		}

		rows, err := p.parseSheet(f, sheetName, provider, emailDate)
		if err != nil {
			p.logger.Warn("Failed to parse sheet",
				zap.String("sheet", sheetName),
				zap.Error(err))
			continue
		}

		if len(rows) > 0 {
			allRows = append(allRows, rows...)
			p.logger.Info("Parsed disk sheet",
				zap.String("sheet", sheetName),
				zap.Int("rows", len(rows)))
		}
	}

	if len(allRows) == 0 {
		p.logger.Error("No valid disk data found in any sheet",
			zap.String("filename", filename),
			zap.Int("total_sheets", len(sheets)))
		return nil, fmt.Errorf("no valid disk data found in any sheet")
	}

	return allRows, nil
}

// shouldProcessSheet checks if the sheet should be processed for disks
func (p *DiskParser) shouldProcessSheet(sheetName string, provider string) bool {
	normalized := strings.ToLower(strings.TrimSpace(sheetName))

	// For БРИНЕКС, only process "Автодиски" sheet
	if strings.Contains(provider, "БРИНЕКС") {
		return normalized == "автодиски"
	}

	// For ЗАПАСКА and БИГМАШИН, process main sheets
	allowedSheets := []string{
		"лист_1",
		"sheet1",
		"диски",
	}

	for _, allowed := range allowedSheets {
		if normalized == allowed {
			return true
		}
	}

	return false
}

// parseSheet parses a single sheet from the Excel file
func (p *DiskParser) parseSheet(f *excelize.File, sheetName string, provider string, emailDate time.Time) ([]db.PriceDiskRow, error) {
	rows, err := f.Rows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}
	defer rows.Close()

	var mapping *diskColumnMapping
	var result []db.PriceDiskRow
	rowNum := 0
	inDiskSection := false // Track if we're in the disk section (for ЗАПАСКА)

	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			p.logger.Warn("Failed to read row",
				zap.String("sheet", sheetName),
				zap.Int("row", rowNum),
				zap.Error(err))
			continue
		}

		if len(cols) == 0 {
			continue
		}

		// For ЗАПАСКА provider, check for section markers
		if strings.Contains(provider, "ЗАПАСКА") {
			// Check for "Диски" marker
			if p.containsDisksMarker(cols) {
				inDiskSection = true
				p.logger.Info("Found disks section marker",
					zap.String("sheet", sheetName),
					zap.Int("row", rowNum))
				continue
			}

			// Check for "Камеры" marker (end of disks section)
			if p.containsTubesMarker(cols) {
				if inDiskSection {
					p.logger.Info("Found tubes section marker - stopping disk parsing",
						zap.String("sheet", sheetName),
						zap.Int("row", rowNum),
						zap.Int("total_disks_parsed", len(result)))
					break
				}
				continue
			}
		}

		// Find header row
		if mapping == nil {
			mapping = p.findDiskColumns(cols)
			if mapping != nil {
				p.logger.Debug("Found header row",
					zap.String("sheet", sheetName),
					zap.Int("row", rowNum),
					zap.Int("balance_columns", len(mapping.balance)))
				continue
			}
			continue
		}

		// For ЗАПАСКА, only parse if we're in disk section
		if strings.Contains(provider, "ЗАПАСКА") && !inDiskSection {
			continue
		}

		// Parse data row
		parsedRows, err := p.parseDiskRow(cols, mapping, provider, emailDate)
		if err != nil {
			p.logger.Debug("Skipping invalid disk row",
				zap.String("sheet", sheetName),
				zap.Int("row", rowNum),
				zap.Error(err))
			continue
		}

		if len(parsedRows) > 0 {
			result = append(result, parsedRows...)
		}
	}

	return result, nil
}

// containsDisksMarker checks if any column contains the disks section marker
func (p *DiskParser) containsDisksMarker(cols []string) bool {
	for _, col := range cols {
		normalized := strings.TrimSpace(strings.ToLower(col))
		if strings.Contains(normalized, "диски") ||
			normalized == "02 диски" ||
			strings.HasPrefix(normalized, "02") && strings.Contains(normalized, "диск") {
			return true
		}
	}
	return false
}

// containsTubesMarker checks if any column contains the tubes section marker
func (p *DiskParser) containsTubesMarker(cols []string) bool {
	for _, col := range cols {
		normalized := strings.TrimSpace(strings.ToLower(col))
		if strings.Contains(normalized, "камеры") ||
			normalized == "03 камеры" ||
			strings.HasPrefix(normalized, "03") && strings.Contains(normalized, "камер") {
			return true
		}
	}
	return false
}

// findDiskColumns finds column indices based on header names
func (p *DiskParser) findDiskColumns(cols []string) *diskColumnMapping {
	mapping := &diskColumnMapping{
		article:      -1,
		nomenclature: -1,
		balance:      make(map[int]string),
		storeColumn:  -1,
	}

	for i, col := range cols {
		normalized := strings.TrimSpace(strings.ToLower(col))

		switch {
		case strings.Contains(normalized, "артикул"):
			mapping.article = i
		case strings.Contains(normalized, "номенклатура"):
			mapping.nomenclature = i
		case strings.HasPrefix(normalized, "остаток"):
			// Extract store name from "Остаток XXX"
			storeName := strings.TrimSpace(strings.TrimPrefix(normalized, "остаток"))
			if storeName == "" {
				storeName = "Основной"
			}
			mapping.balance[i] = storeName
		case strings.Contains(normalized, "склад") && mapping.storeColumn < 0:
			mapping.storeColumn = i
		}
	}

	// Required columns: article and nomenclature
	if mapping.article >= 0 && mapping.nomenclature >= 0 {
		p.logger.Info("Successfully found required disk columns",
			zap.Int("article", mapping.article),
			zap.Int("nomenclature", mapping.nomenclature),
			zap.Int("balance_columns", len(mapping.balance)),
			zap.Int("store_column", mapping.storeColumn))
		return mapping
	}

	return nil
}

// parseDiskRow parses a single row and returns disk data
func (p *DiskParser) parseDiskRow(cols []string, mapping *diskColumnMapping, provider string, emailDate time.Time) ([]db.PriceDiskRow, error) {
	if len(cols) == 0 {
		return nil, fmt.Errorf("empty row")
	}

	article := p.getColumn(cols, mapping.article)
	nomenclature := p.getColumn(cols, mapping.nomenclature)

	if article == "" || nomenclature == "" {
		return nil, fmt.Errorf("missing article or nomenclature")
	}

	// Parse disk specifications from nomenclature
	diskData, err := p.parseDiskSpecifications(nomenclature, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk specifications: %w", err)
	}

	// Validate parsed data
	if err := p.validateDiskData(diskData); err != nil {
		p.logger.Warn("Disk data validation failed",
			zap.String("nomenclature", nomenclature),
			zap.Error(err))
		return nil, err
	}

	diskData.Article = article
	diskData.Provider = provider
	diskData.EmailDate = emailDate

	var result []db.PriceDiskRow

	// Handle balance columns (for БИГМАШИН - multiple stores)
	if len(mapping.balance) > 0 {
		for balanceIdx, storeName := range mapping.balance {
			balanceStr := p.getColumn(cols, balanceIdx)
			balance, err := p.parseInt(balanceStr)
			if err != nil || balance == 0 {
				continue // Skip if balance is 0 or invalid
			}

			row := *diskData
			row.Store = storeName
			row.Balance = balance
			result = append(result, row)
		}
	} else if mapping.storeColumn >= 0 {
		// Single store column (for ЗАПАСКА/БРИНЕКС)
		store := p.getColumn(cols, mapping.storeColumn)
		if store != "" {
			diskData.Store = store
			diskData.Balance = 1 // Default balance if not specified
			result = append(result, *diskData)
		}
	} else {
		// No store/balance info
		diskData.Store = "Основной"
		diskData.Balance = 1
		result = append(result, *diskData)
	}

	return result, nil
}

// parseDiskSpecifications extracts disk specifications from nomenclature string
func (p *DiskParser) parseDiskSpecifications(nomenclature string, provider string) (*db.PriceDiskRow, error) {
	nomenclature = strings.TrimSpace(nomenclature)

	if strings.Contains(provider, "ЗАПАСКА") {
		return p.parseZapaskaDisk(nomenclature)
	} else if strings.Contains(provider, "БИГМАШИН") || strings.Contains(provider, "БРИНЕКС") {
		return p.parseBigMachineOrBrinexDisk(nomenclature)
	}

	return nil, fmt.Errorf("unknown provider: %s", provider)
}

// parseZapaskaDisk parses ЗАПАСКА format: "15 Alcasta M62 6.0*15 4*100 ET40 D60.1 BLACK"
func (p *DiskParser) parseZapaskaDisk(nomenclature string) (*db.PriceDiskRow, error) {
	parts := strings.Fields(nomenclature)
	if len(parts) < 5 {
		return nil, fmt.Errorf("not enough parts in nomenclature")
	}

	disk := &db.PriceDiskRow{}

	// Skip first part if it's a number (like "15")
	startIdx := 0
	if _, err := strconv.Atoi(parts[0]); err == nil {
		startIdx = 1
	}

	if startIdx >= len(parts) {
		return nil, fmt.Errorf("invalid nomenclature format")
	}

	// Manufacturer is the first word (after number if present)
	disk.Manufacturer = parts[startIdx]
	startIdx++

	// Parse remaining parts
	for i := startIdx; i < len(parts); i++ {
		part := parts[i]

		// Width x Diameter (e.g., "6.0*15" or "6.0х15")
		if strings.Contains(part, "*") || strings.Contains(part, "х") || strings.Contains(part, "x") {
			if err := p.parseWidthDiameter(part, disk); err == nil {
				continue
			}
		}

		// Drilling (e.g., "4*100" or "5x114.3")
		if regexp.MustCompile(`^\d+[*xх]\d+`).MatchString(part) {
			disk.Drilling = part
			continue
		}

		// Radius (e.g., "ET40")
		if strings.HasPrefix(strings.ToUpper(part), "ET") {
			disk.Radius = part
			continue
		}

		// Central hole (e.g., "D60.1" or "dia60.1")
		if strings.HasPrefix(strings.ToUpper(part), "D") || strings.Contains(strings.ToLower(part), "dia") {
			disk.CentralHole = part
			continue
		}

		// Model (if not yet set and looks like a model code)
		if disk.Model == "" && regexp.MustCompile(`^[A-Z0-9-]+$`).MatchString(strings.ToUpper(part)) {
			disk.Model = part
			continue
		}

		// Color (usually last text part)
		if !regexp.MustCompile(`\d`).MatchString(part) {
			disk.Color = part
		}
	}

	return disk, nil
}

// parseBigMachineOrBrinexDisk parses БИГМАШИН/БРИНЕКС format:
// "Диск литой 6.5х16 5х114.3 ЕТ40 dia 66.1 KHOMEN KHW1612 GRAY-FP"
func (p *DiskParser) parseBigMachineOrBrinexDisk(nomenclature string) (*db.PriceDiskRow, error) {
	disk := &db.PriceDiskRow{}

	// Split into words
	parts := strings.Fields(nomenclature)

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		// Width x Diameter (e.g., "6.5х16" or "6.5x16")
		if (strings.Contains(part, "х") || strings.Contains(part, "x")) &&
			regexp.MustCompile(`\d`).MatchString(part) {
			if err := p.parseWidthDiameter(part, disk); err == nil {
				continue
			}
		}

		// Drilling (e.g., "5х114.3" or "5x114.3")
		if regexp.MustCompile(`^\d+[xх]\d+\.?\d*$`).MatchString(part) && disk.Drilling == "" {
			disk.Drilling = part
			continue
		}

		// Radius (e.g., "ЕТ40" or "ET40")
		if strings.HasPrefix(strings.ToUpper(part), "ET") || strings.HasPrefix(strings.ToUpper(part), "ЕТ") {
			disk.Radius = part
			continue
		}

		// Central hole (e.g., "dia" followed by number)
		if strings.ToLower(part) == "dia" && i+1 < len(parts) {
			disk.CentralHole = "dia " + parts[i+1]
			i++ // Skip next part
			continue
		}

		// Central hole in one word (e.g., "D66.1")
		if strings.HasPrefix(strings.ToUpper(part), "D") && regexp.MustCompile(`\d`).MatchString(part) {
			disk.CentralHole = part
			continue
		}

		// Manufacturer - usually an uppercase word without numbers
		if disk.Manufacturer == "" && regexp.MustCompile(`^[A-Z][A-Z]+$`).MatchString(part) {
			disk.Manufacturer = part
			continue
		}

		// Model - alphanumeric code
		if disk.Model == "" && regexp.MustCompile(`[A-Z]\w*\d+`).MatchString(part) {
			disk.Model = part
			continue
		}

		// Color - last uppercase word or code
		if regexp.MustCompile(`^[A-Z-]+$`).MatchString(part) && !regexp.MustCompile(`\d`).MatchString(part) {
			disk.Color = part
		}
	}

	return disk, nil
}

// parseWidthDiameter parses width x diameter from string like "6.5х16"
func (p *DiskParser) parseWidthDiameter(s string, disk *db.PriceDiskRow) error {
	// Replace different "x" variations
	s = strings.ReplaceAll(s, "х", "x")
	s = strings.ReplaceAll(s, "*", "x")

	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return fmt.Errorf("invalid width x diameter format")
	}

	width, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return fmt.Errorf("invalid width: %w", err)
	}

	diameter, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return fmt.Errorf("invalid diameter: %w", err)
	}

	disk.Width = width
	disk.Diameter = diameter
	return nil
}

// validateDiskData validates the parsed disk data
func (p *DiskParser) validateDiskData(disk *db.PriceDiskRow) error {
	// Width should be reasonable (e.g., 4.0 to 15.0)
	if disk.Width < 4.0 || disk.Width > 15.0 {
		return fmt.Errorf("invalid width: %.1f (expected 4.0-15.0)", disk.Width)
	}

	// Diameter should be reasonable (e.g., 12 to 24 inches)
	if disk.Diameter < 12 || disk.Diameter > 24 {
		return fmt.Errorf("invalid diameter: %.1f (expected 12-24)", disk.Diameter)
	}

	// Manufacturer should not contain only numbers
	if disk.Manufacturer != "" && regexp.MustCompile(`^\d+$`).MatchString(disk.Manufacturer) {
		return fmt.Errorf("manufacturer contains only numbers: %s", disk.Manufacturer)
	}

	// Drilling should match pattern like "4x100" or "5x114.3"
	if disk.Drilling != "" && !regexp.MustCompile(`^\d+[*xх]\d+\.?\d*$`).MatchString(disk.Drilling) {
		return fmt.Errorf("invalid drilling format: %s", disk.Drilling)
	}

	return nil
}

// getColumn safely gets a column value by index
func (p *DiskParser) getColumn(cols []string, index int) string {
	if index < 0 || index >= len(cols) {
		return ""
	}
	return strings.TrimSpace(cols[index])
}

// parseInt parses an integer from a string
func (p *DiskParser) parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	// Remove spaces and commas
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", "")

	return strconv.Atoi(s)
}

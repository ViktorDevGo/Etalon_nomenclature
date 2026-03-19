package parser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/prokoleso/etalon-nomenclature/internal/db"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

// PriceParser handles price list Excel file parsing
type PriceParser struct {
	logger *zap.Logger
}

// priceColumnMapping represents the mapping of column names to indices
type priceColumnMapping struct {
	article     int
	price       int
	balance     map[int]string // index -> store name (for БИГМАШИН: multiple "Остаток*" columns)
	storeColumn int            // For ЗАПАСКА/БРИНЕКС single "Склад" column
}

// NewPriceParser creates a new price list parser
func NewPriceParser(logger *zap.Logger) *PriceParser {
	return &PriceParser{
		logger: logger,
	}
}

// Parse parses an Excel price list file and returns price tire rows
func (p *PriceParser) Parse(content []byte, filename string, provider string, emailDate time.Time) ([]db.TyrePriceStockRow, error) {
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

	var allRows []db.TyrePriceStockRow

	// Get all sheet names
	sheets := f.GetSheetList()
	p.logger.Info("Processing price Excel file",
		zap.String("filename", filename),
		zap.String("provider", provider),
		zap.Int("sheets", len(sheets)))

	for _, sheetName := range sheets {
		// Skip sheets that are not in the allowed list
		if !p.shouldProcessSheet(sheetName) {
			p.logger.Debug("Skipping sheet - not in allowed list",
				zap.String("sheet", sheetName))
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
			p.logger.Info("Parsed sheet",
				zap.String("sheet", sheetName),
				zap.Int("rows", len(rows)))
		}
	}

	if len(allRows) == 0 {
		p.logger.Error("No valid data found in any sheet",
			zap.String("filename", filename),
			zap.Int("total_sheets", len(sheets)))
		return nil, fmt.Errorf("no valid data found in any sheet")
	}

	return allRows, nil
}

// shouldProcessSheet checks if the sheet should be processed
func (p *PriceParser) shouldProcessSheet(sheetName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(sheetName))

	allowedSheets := []string{
		"автошины",
		"зимние",
		"летние",
		"легкогрузовые",
		"лист_1",
	}

	for _, allowed := range allowedSheets {
		if normalized == allowed {
			return true
		}
	}

	return false
}

// parseSheet parses a single sheet from the Excel file
func (p *PriceParser) parseSheet(f *excelize.File, sheetName string, provider string, emailDate time.Time) ([]db.TyrePriceStockRow, error) {
	// Get streaming reader for memory efficiency
	rows, err := f.Rows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}
	defer rows.Close()

	var mapping *priceColumnMapping
	var result []db.TyrePriceStockRow
	rowNum := 0
	headerRowsScanned := 0
	const maxHeaderRows = 20 // Scan first 20 rows for headers (БИГМАШИН has headers at row 11)

	// For ЗАПАСКА: track if we're in the tire section (before "Диски" marker)
	inTireSection := !strings.Contains(provider, "ЗАПАСКА") // true for non-ЗАПАСКА providers

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
			// Check for "Шины" marker (start of tire section)
			if p.containsTiresMarker(cols) {
				inTireSection = true
				p.logger.Info("Found tires section marker",
					zap.String("sheet", sheetName),
					zap.Int("row", rowNum))
				continue
			}

			// Check for "Диски" marker (end of tire section)
			if p.containsDisksMarker(cols) {
				if inTireSection {
					p.logger.Info("Found disks section marker - stopping tire parsing",
						zap.String("sheet", sheetName),
						zap.Int("row", rowNum),
						zap.Int("total_tires_parsed", len(result)))
					break
				}
				continue
			}
		}

		// Find header rows (may span multiple rows)
		if mapping == nil {
			if headerRowsScanned < maxHeaderRows {
				headerRowsScanned++
				mapping = p.findPriceColumns(cols, provider)
				if mapping != nil {
					p.logger.Debug("Found header row",
						zap.String("sheet", sheetName),
						zap.Int("row", rowNum),
						zap.Int("balance_columns", len(mapping.balance)))
					continue
				}
				// Try to update partial mapping
				if headerRowsScanned < maxHeaderRows {
					continue
				}
			}
			// If still no mapping after maxHeaderRows, skip this sheet
			if mapping == nil {
				continue
			}
		}

		// For ЗАПАСКА, only parse if we're in tire section
		if strings.Contains(provider, "ЗАПАСКА") && !inTireSection {
			continue
		}

		// Parse data row
		parsedRows, err := p.parseRow(cols, mapping, provider, emailDate)
		if err != nil {
			p.logger.Debug("Skipping invalid row",
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

// findPriceColumns finds column indices based on header names
func (p *PriceParser) findPriceColumns(cols []string, provider string) *priceColumnMapping {
	mapping := &priceColumnMapping{
		article:     -1,
		price:       -1,
		balance:     make(map[int]string),
		storeColumn: -1,
	}

	// Log found columns for debugging
	p.logger.Debug("Checking row for headers",
		zap.Strings("columns", cols),
		zap.String("provider", provider))

	for i, col := range cols {
		// Normalize: remove newlines and carriage returns, then trim and lowercase
		normalized := strings.ReplaceAll(col, "\n", " ")
		normalized = strings.ReplaceAll(normalized, "\r", " ")
		normalized = strings.TrimSpace(strings.ToLower(normalized))

		switch {
		case strings.Contains(normalized, "артикул"):
			mapping.article = i

		case strings.Contains(normalized, "оптовая") ||
		     (strings.Contains(normalized, "цена") && !strings.Contains(normalized, "розн")):
			// Match "Оптовая", "Оптовая цена", or just "Цена" (but not "Розница")
			if mapping.price < 0 {  // Take first matching price column
				mapping.price = i
			}

		case strings.Contains(normalized, "остаток"):
			// For БИГМАШИН: collect ALL "Остаток*" columns
			// For others: only the first one
			if provider == string(ProviderBigMachine) {
				// Extract store name from column header
				// e.g., "Остаток Нск Северный" -> "Нск Северный"
				// Normalize newlines in original column name too
				colNormalized := strings.ReplaceAll(col, "\n", " ")
				colNormalized = strings.ReplaceAll(colNormalized, "\r", " ")
				storeName := strings.TrimSpace(strings.TrimPrefix(colNormalized, "Остаток"))
				if storeName == "" {
					storeName = "Неизвестный склад"
				}
				mapping.balance[i] = storeName
			} else {
				// For ЗАПАСКА/БРИНЕКС - only first "Остаток" column (or "Остаток на складе")
				if len(mapping.balance) == 0 {
					mapping.balance[i] = ""
				}
			}

		case strings.Contains(normalized, "склад") && !strings.Contains(normalized, "остаток"):
			// Store column for ЗАПАСКА/БРИНЕКС (but not "Остаток на складе")
			if provider != string(ProviderBigMachine) && mapping.storeColumn < 0 {
				mapping.storeColumn = i
			}
		}
	}

	// Special handling for ЗАПАСКА format where "Артикул" is in col[1] and "Цена"+"Остаток" appear together
	// Example: ["", "", "", "Цена", "Остаток"] where price=3, balance=4
	if mapping.article < 0 && len(cols) > 1 {
		// Try col[1] as article for ЗАПАСКА (typical position)
		normalized1 := strings.TrimSpace(strings.ToLower(cols[1]))
		if normalized1 == "" || !strings.Contains(normalized1, "номенклатура") {
			// If col[1] is empty or not "Номенклатура", likely article is at position 1
			// We'll set it below if we find price+balance in the same row
		}
	}

	// If we found "Цена" and "Остаток" in the same row, this might be ЗАПАСКА/БРИНЕКС header row
	// Set article to column 1 (standard ЗАПАСКА/БРИНЕКС position)
	// DO NOT apply this logic to БИГМАШИН - they have proper "Артикул производителя" column
	if mapping.price >= 0 && len(mapping.balance) > 0 && mapping.article < 0 &&
		provider != string(ProviderBigMachine) {
		// ЗАПАСКА/БРИНЕКС typical format: Артикул at column 1
		mapping.article = 1
		p.logger.Debug("Using column 1 as article (ЗАПАСКА/БРИНЕКС format detected)")
	}

	// Validation: required columns are article, price, and at least one balance
	if mapping.article >= 0 && mapping.price >= 0 && len(mapping.balance) > 0 {
		p.logger.Info("Successfully found all required columns",
			zap.Int("article", mapping.article),
			zap.Int("price", mapping.price),
			zap.Int("balance_columns", len(mapping.balance)),
			zap.Int("store", mapping.storeColumn))
		return mapping
	}

	// Log which required columns are missing (only if at least one column was found)
	hasAnyColumn := mapping.article >= 0 || mapping.price >= 0 || len(mapping.balance) > 0
	if hasAnyColumn {
		missing := []string{}
		if mapping.article < 0 {
			missing = append(missing, "артикул")
		}
		if mapping.price < 0 {
			missing = append(missing, "оптовая цена")
		}
		if len(mapping.balance) == 0 {
			missing = append(missing, "остаток")
		}

		if len(missing) > 0 {
			p.logger.Debug("Partial match - missing columns",
				zap.Strings("missing", missing),
				zap.Strings("available", cols))
		}
	}

	return nil
}

// parseRow parses a single data row into price tire rows
// Returns an array because БИГМАШИН can create multiple rows for different warehouses
func (p *PriceParser) parseRow(cols []string, mapping *priceColumnMapping, provider string, emailDate time.Time) ([]db.TyrePriceStockRow, error) {
	if len(cols) == 0 {
		return nil, fmt.Errorf("empty row")
	}

	// Skip category marker rows (empty article and price columns)
	if isCategoryMarkerRow(cols, mapping) {
		return nil, fmt.Errorf("category marker row - skipping")
	}

	// Smart search for article (handles XLS conversion issues)
	article, foundArticle := smartFindArticle(cols, mapping)
	if !foundArticle {
		return nil, fmt.Errorf("article not found in row")
	}

	// Validate article format
	article = strings.TrimSpace(article)
	isValid := p.isValidArticle(article)
	if !isValid {
		return nil, fmt.Errorf("invalid article format: '%s'", article)
	}

	// Smart search for price (handles XLS conversion issues)
	price, foundPrice := smartFindPrice(cols, mapping, p)
	if !foundPrice {
		return nil, fmt.Errorf("price not found in row")
	}

	// Validate price (must be > 0)
	if price <= 0 {
		return nil, fmt.Errorf("invalid price: %.2f (must be > 0)", price)
	}

	var rows []db.TyrePriceStockRow

	if provider == string(ProviderBigMachine) {
		// БИГМАШИН: Create separate row for each warehouse with balance > 0
		for balanceIdx, storeName := range mapping.balance {
			balanceStr := p.getColumn(cols, balanceIdx)
			balance, err := p.parseInt(balanceStr)
			if err != nil {
				continue // Skip invalid balance values
			}

			// Only create row if balance > 0
			if balance > 0 {
				rows = append(rows, db.TyrePriceStockRow{
					CAE:           article,
					Price:         price,
					Stock:         balance,
					WarehouseName: storeName,
					Provider:      provider,
					EmailDate:     emailDate,
				})
			}
		}
	} else {
		// ЗАПАСКА/БРИНЕКС: Single row per article
		balanceIdx := -1
		for idx := range mapping.balance {
			balanceIdx = idx
			break // Get the first (and only) balance column
		}

		if balanceIdx < 0 {
			return nil, fmt.Errorf("no balance column found")
		}

		balanceStr := p.getColumn(cols, balanceIdx)
		balance, err := p.parseInt(balanceStr)
		if err != nil {
			return nil, fmt.Errorf("invalid balance value: %w", err)
		}

		// Only create row if balance > 0
		if balance > 0 {
			store := "Основной" // Default store for ЗАПАСКА/БРИНЕКС
			if mapping.storeColumn >= 0 {
				storeValue := p.getColumn(cols, mapping.storeColumn)
				if storeValue != "" {
					store = storeValue
				}
			}

			rows = append(rows, db.TyrePriceStockRow{
				CAE:           article,
				Price:         price,
				Stock:         balance,
				WarehouseName: store,
				Provider:      provider,
				EmailDate:     emailDate,
			})
		}
	}

	// If no rows created (all balances were 0), return error
	if len(rows) == 0 {
		return nil, fmt.Errorf("all balances are zero")
	}

	return rows, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isValidArticle checks if the article string has a valid format
// Valid articles should contain letters OR be long numeric codes (6+ digits)
// Invalid examples: "6 910" (price with space), "Автошина..." (nomenclature), "205/65" (size)
func (p *PriceParser) isValidArticle(article string) bool {
	if article == "" {
		return false
	}

	// Check if it starts with nomenclature prefixes or patterns - definitely nomenclature, not article
	lowerArticle := strings.ToLower(article)
	if strings.HasPrefix(lowerArticle, "автошина") ||
		strings.HasPrefix(lowerArticle, "а/ш") ||
		strings.Contains(lowerArticle, "б/у") ||
		strings.HasPrefix(lowerArticle, "на r") || // "на R19..." - truncated nomenclature
		strings.HasPrefix(lowerArticle, "шина") {
		return false
	}

	// Check if article contains too many spaces (typical for nomenclature)
	// Real articles usually have 0-2 spaces, nomenclature has many
	spaceCount := strings.Count(article, " ")
	if spaceCount > 3 {
		return false
	}

	// Remove spaces and common separators to check structure
	cleaned := strings.ReplaceAll(article, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "/", "")

	// Check if article contains at least one letter (latin or cyrillic)
	hasLetter := false
	for _, r := range article {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= 'А' && r <= 'Я') || (r >= 'а' && r <= 'я') {
			hasLetter = true
			break
		}
	}

	if hasLetter {
		return true
	}

	// If no letters, check if it's a long numeric code (>= 6 consecutive digits)
	// Examples: "1352917216", "4514700" - valid
	// Examples: "6 910", "5 040", "12 950" - invalid (prices with spaces)

	// First, reject numbers with spaces (typical for formatted prices)
	if strings.Contains(article, " ") {
		// Check if after removing spaces it's still a short number
		digitsOnly := strings.ReplaceAll(article, " ", "")
		allDigits := true
		for _, r := range digitsOnly {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		// If it's all digits with spaces (like "12 950"), it's a price
		if allDigits && len(digitsOnly) < 7 {
			return false
		}
	}

	digitCount := 0
	for _, r := range cleaned {
		if r >= '0' && r <= '9' {
			digitCount++
		}
	}

	// Must have at least 6 digits to be a valid numeric article
	if digitCount >= 6 {
		return true
	}

	// Short numeric strings (< 6 digits) are likely prices or sizes, not articles
	return false
}

// getColumn safely retrieves a column value
func (p *PriceParser) getColumn(cols []string, index int) string {
	if index < 0 || index >= len(cols) {
		return ""
	}
	return strings.TrimSpace(cols[index])
}

// parseFloat parses a float value from string
func (p *PriceParser) parseFloat(s string) (float64, error) {
	originalValue := s
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Remove spaces
	s = strings.ReplaceAll(s, " ", "")

	// Detect potential date formats (more than 3 dots/commas is suspicious)
	dotCount := strings.Count(s, ".")
	commaCount := strings.Count(s, ",")

	if dotCount > 3 || commaCount > 3 {
		p.logger.Warn("Suspicious value format - possibly a date",
			zap.String("original", originalValue),
			zap.Int("dots", dotCount),
			zap.Int("commas", commaCount))
		return 0, fmt.Errorf("suspicious format: too many separators (dots: %d, commas: %d)", dotCount, commaCount)
	}

	// Handle different decimal formats:
	// Russian: "1 234,56" or "1234,56" (comma as decimal separator)
	// European: "1.234,56" (dot for thousands, comma for decimal)
	// US/ЗАПАСКА: "1,234.56" or "4.124.00" (dot/comma for thousands, last separator is decimal)

	if dotCount > 1 {
		// Multiple dots: "4.124.00" - dots are thousand separators except the last one
		// Remove all dots except the last one
		parts := strings.Split(s, ".")
		if len(parts) > 0 {
			// Join all parts except last with empty string, then add last part with dot
			wholePart := strings.Join(parts[:len(parts)-1], "")
			decimalPart := parts[len(parts)-1]
			s = wholePart + "." + decimalPart
		}
	} else if commaCount > 1 {
		// Multiple commas: similar logic
		parts := strings.Split(s, ",")
		if len(parts) > 0 {
			wholePart := strings.Join(parts[:len(parts)-1], "")
			decimalPart := parts[len(parts)-1]
			s = wholePart + "." + decimalPart
		}
	} else if dotCount == 1 && commaCount == 1 {
		// Both dot and comma: determine which is decimal separator
		dotPos := strings.Index(s, ".")
		commaPos := strings.Index(s, ",")
		if dotPos < commaPos {
			// Dot comes first: "1.234,56" (European format)
			s = strings.ReplaceAll(s, ".", "")  // Remove thousand separator
			s = strings.ReplaceAll(s, ",", ".") // Replace decimal comma with dot
		} else {
			// Comma comes first: "1,234.56" (US format)
			s = strings.ReplaceAll(s, ",", "") // Remove thousand separator
		}
	} else if commaCount == 1 {
		// Only comma: "1234,56" (Russian format)
		s = strings.ReplaceAll(s, ",", ".")
	}
	// else: only dot or no separators - already in correct format

	result, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	// Validate: prices for tires should be reasonable (< 500,000 RUB)
	const maxReasonablePrice = 500000.0
	if result > maxReasonablePrice {
		p.logger.Warn("Price exceeds reasonable limit - skipping row",
			zap.String("original", originalValue),
			zap.Float64("parsed_value", result),
			zap.Float64("max_allowed", maxReasonablePrice))
		return 0, fmt.Errorf("price %.2f exceeds maximum reasonable price %.2f", result, maxReasonablePrice)
	}

	return result, nil
}

// parseInt parses an integer value from string
func (p *PriceParser) parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Handle ">N" notation (e.g., ">40" means more than 40)
	if strings.HasPrefix(s, ">") {
		s = strings.TrimPrefix(s, ">")
		s = strings.TrimSpace(s)
	}

	// Remove common separators
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", "")

	return strconv.Atoi(s)
}

// containsTiresMarker checks if any column contains the tires section marker
func (p *PriceParser) containsTiresMarker(cols []string) bool {
	for _, col := range cols {
		normalized := strings.TrimSpace(strings.ToLower(col))
		if strings.Contains(normalized, "шины") ||
			strings.Contains(normalized, "автошины") ||
			normalized == "01 шины" ||
			normalized == "01 автошины" ||
			(strings.HasPrefix(normalized, "01") && (strings.Contains(normalized, "шин") || strings.Contains(normalized, "tire"))) {
			return true
		}
	}
	return false
}

// containsDisksMarker checks if any column contains the disks section marker
func (p *PriceParser) containsDisksMarker(cols []string) bool {
	for _, col := range cols {
		normalized := strings.TrimSpace(strings.ToLower(col))
		if strings.Contains(normalized, "диски") ||
			normalized == "02 диски" ||
			normalized == "02 автодиски" ||
			(strings.HasPrefix(normalized, "02") && strings.Contains(normalized, "диск")) {
			return true
		}
	}
	return false
}

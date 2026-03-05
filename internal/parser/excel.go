package parser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/prokoleso/etalon-nomenclature/internal/db"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

// Parser handles Excel file parsing
type Parser struct {
	logger *zap.Logger
}

// columnMapping represents the mapping of column names to indices
type columnMapping struct {
	article      int
	brand        int
	typ          int
	sizeModel    int
	nomenclature int
	mrc          int
	hasType      bool
}

// New creates a new Excel parser
func New(logger *zap.Logger) *Parser {
	return &Parser{
		logger: logger,
	}
}

// Parse parses an Excel file and returns nomenclature rows
func (p *Parser) Parse(content []byte, filename string) ([]db.NomenclatureRow, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	var allRows []db.NomenclatureRow

	// Get all sheet names
	sheets := f.GetSheetList()
	p.logger.Info("Processing Excel file",
		zap.String("filename", filename),
		zap.Int("sheets", len(sheets)))

	for _, sheetName := range sheets {
		rows, err := p.parseSheet(f, sheetName)
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

func (p *Parser) parseSheet(f *excelize.File, sheetName string) ([]db.NomenclatureRow, error) {
	// Get streaming reader for memory efficiency
	rows, err := f.Rows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}
	defer rows.Close()

	var mapping *columnMapping
	var result []db.NomenclatureRow
	rowNum := 0

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

		// Find header row
		if mapping == nil {
			mapping = p.findColumns(cols)
			if mapping != nil {
				p.logger.Debug("Found header row",
					zap.String("sheet", sheetName),
					zap.Int("row", rowNum),
					zap.Bool("has_type", mapping.hasType))
				continue
			}
			// Skip rows until we find headers
			continue
		}

		// Parse data row
		row, err := p.parseRow(cols, mapping)
		if err != nil {
			p.logger.Debug("Skipping invalid row",
				zap.String("sheet", sheetName),
				zap.Int("row", rowNum),
				zap.Error(err))
			continue
		}

		if row != nil {
			result = append(result, *row)
		}
	}

	return result, nil
}

func (p *Parser) findColumns(cols []string) *columnMapping {
	mapping := &columnMapping{
		article:      -1,
		brand:        -1,
		typ:          -1,
		sizeModel:    -1,
		nomenclature: -1,
		mrc:          -1,
	}

	// Log found columns for debugging
	p.logger.Info("Checking row for headers",
		zap.Strings("columns", cols))

	for i, col := range cols {
		normalized := strings.TrimSpace(strings.ToLower(col))

		switch {
		case strings.Contains(normalized, "артикул") && !strings.Contains(normalized, "доп"):
			mapping.article = i
		case strings.Contains(normalized, "марка"):
			mapping.brand = i
		case normalized == "тип":
			mapping.typ = i
			mapping.hasType = true
		case strings.Contains(normalized, "размер") && strings.Contains(normalized, "модель"):
			mapping.sizeModel = i
			// Если есть отдельная колонка "номенклатура" - используем её
			// Иначе используем "Размер и Модель" как номенклатуру
			if strings.Contains(normalized, "номенклатура") {
				mapping.nomenclature = i
			} else if mapping.nomenclature < 0 {
				// Если номенклатуры нет, используем "Размер и Модель"
				mapping.nomenclature = i
			}
		case strings.Contains(normalized, "номенклатура"):
			mapping.nomenclature = i
		case strings.Contains(normalized, "мрц"):
			mapping.mrc = i
		}
	}

	// Required columns: article, brand, size_model, nomenclature, mrc
	// Type is optional
	if mapping.article >= 0 &&
		mapping.brand >= 0 &&
		mapping.sizeModel >= 0 &&
		mapping.nomenclature >= 0 &&
		mapping.mrc >= 0 {
		p.logger.Info("Successfully found all required columns",
			zap.Int("article", mapping.article),
			zap.Int("brand", mapping.brand),
			zap.Int("sizeModel", mapping.sizeModel),
			zap.Int("nomenclature", mapping.nomenclature),
			zap.Int("mrc", mapping.mrc),
			zap.Int("type", mapping.typ),
			zap.Bool("hasType", mapping.hasType))
		return mapping
	}

	// Log which required columns are missing
	missing := []string{}
	if mapping.article < 0 {
		missing = append(missing, "артикул")
	}
	if mapping.brand < 0 {
		missing = append(missing, "марка")
	}
	if mapping.sizeModel < 0 {
		missing = append(missing, "размер и модель")
	}
	if mapping.nomenclature < 0 {
		missing = append(missing, "номенклатура")
	}
	if mapping.mrc < 0 {
		missing = append(missing, "мрц")
	}

	if len(missing) > 0 {
		p.logger.Error("Required columns not found",
			zap.Strings("missing", missing),
			zap.Strings("available", cols))
	}

	return nil
}

func (p *Parser) parseRow(cols []string, mapping *columnMapping) (*db.NomenclatureRow, error) {
	if len(cols) == 0 {
		return nil, fmt.Errorf("empty row")
	}

	// Get values
	article := p.getColumn(cols, mapping.article)
	brand := p.getColumn(cols, mapping.brand)
	typ := ""
	if mapping.hasType {
		typ = p.getColumn(cols, mapping.typ)
	}
	sizeModel := p.getColumn(cols, mapping.sizeModel)
	mrcStr := p.getColumn(cols, mapping.mrc)

	// Build nomenclature from brand + type + size_model
	nomenclature := brand
	if typ != "" {
		nomenclature += " " + typ
	}
	if sizeModel != "" {
		nomenclature += " " + sizeModel
	}
	nomenclature = strings.TrimSpace(nomenclature)

	// Validate required fields
	if article == "" || brand == "" || sizeModel == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	// Parse MRC
	mrc, err := p.parseFloat(mrcStr)
	if err != nil {
		return nil, fmt.Errorf("invalid MRC value: %w", err)
	}

	return &db.NomenclatureRow{
		Article:      article,
		Brand:        brand,
		Type:         typ,
		SizeModel:    sizeModel,
		Nomenclature: nomenclature,
		MRC:          mrc,
	}, nil
}

func (p *Parser) getColumn(cols []string, index int) string {
	if index < 0 || index >= len(cols) {
		return ""
	}
	return strings.TrimSpace(cols[index])
}

func (p *Parser) parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Remove common separators
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")

	return strconv.ParseFloat(s, 64)
}

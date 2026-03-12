package parser

import (
	"strings"

	"go.uber.org/zap"
)

// FileType represents the type of Excel file
type FileType string

// Provider represents the tire supplier
type Provider string

const (
	// FileType constants
	FileTypeNomenclature FileType = "nomenclature"
	FileTypePrice        FileType = "price"
	FileTypeDisk         FileType = "disk"

	// Provider constants
	ProviderBigMachine Provider = "БИГМАШИН"
	ProviderZapaska    Provider = "ЗАПАСКА"
	ProviderBrinex     Provider = "ГРУППА БРИНЕКС"
	ProviderUnknown    Provider = "НЕИЗВЕСТНЫЙ"
)

// Detector detects file types and providers
type Detector struct {
	logger *zap.Logger
}

// NewDetector creates a new file type and provider detector
func NewDetector(logger *zap.Logger) *Detector {
	return &Detector{
		logger: logger,
	}
}

// DetectFileType determines if the file is a price list, disk file, or nomenclature based on filename
func (d *Detector) DetectFileType(filename string) FileType {
	normalized := strings.ToLower(filename)

	// Check if filename contains "мрц" - nomenclature files (МРЦ = Минимальная Розничная Цена)
	// These files contain recommended retail prices and nomenclature data
	if strings.Contains(normalized, "мрц") {
		d.logger.Info("Detected nomenclature file (МРЦ)",
			zap.String("filename", filename),
			zap.String("type", string(FileTypeNomenclature)))
		return FileTypeNomenclature
	}

	// Check if filename contains "диск" - disk files
	if strings.Contains(normalized, "диск") {
		d.logger.Info("Detected disk file",
			zap.String("filename", filename),
			zap.String("type", string(FileTypeDisk)))
		return FileTypeDisk
	}

	// Check if filename contains "прайс" or "прайс-лист" - price files
	// Note: Some providers use the same file for tires and disks (different sheets/sections)
	if strings.Contains(normalized, "прайс") {
		d.logger.Info("Detected price file",
			zap.String("filename", filename),
			zap.String("type", string(FileTypePrice)))
		return FileTypePrice
	}

	// Default to price file for backward compatibility (most files are price lists)
	d.logger.Info("Detected price file (default)",
		zap.String("filename", filename),
		zap.String("type", string(FileTypePrice)))
	return FileTypePrice
}

// DetectProvider determines the provider based on email sender address
func (d *Detector) DetectProvider(emailFrom string) Provider {
	emailLower := strings.ToLower(emailFrom)

	var provider Provider

	switch {
	case strings.Contains(emailLower, "bigm.pro"):
		provider = ProviderBigMachine
	case strings.Contains(emailLower, "sibzapaska.ru"):
		provider = ProviderZapaska
	case strings.Contains(emailLower, "brinex.ru"):
		provider = ProviderBrinex
	default:
		d.logger.Warn("Unknown provider email",
			zap.String("email", emailFrom))
		provider = ProviderUnknown
	}

	d.logger.Info("Detected provider",
		zap.String("email", emailFrom),
		zap.String("provider", string(provider)))

	return provider
}

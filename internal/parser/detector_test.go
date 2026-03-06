package parser

import (
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestDetector_DetectFileType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	detector := NewDetector(logger)

	tests := []struct {
		name     string
		filename string
		want     FileType
	}{
		{
			name:     "Price list with прайс",
			filename: "Прайс-лист.xlsx",
			want:     FileTypePrice,
		},
		{
			name:     "Price list lowercase",
			filename: "прайс.xlsx",
			want:     FileTypePrice,
		},
		{
			name:     "Price list uppercase",
			filename: "ПРАЙС_МАРТ_2026.xlsx",
			want:     FileTypePrice,
		},
		{
			name:     "Price list with hyphen",
			filename: "Прайс-лист шины.xlsx",
			want:     FileTypePrice,
		},
		{
			name:     "Nomenclature file",
			filename: "Номенклатура.xlsx",
			want:     FileTypeNomenclature,
		},
		{
			name:     "Etalon file",
			filename: "Эталон.xlsx",
			want:     FileTypeNomenclature,
		},
		{
			name:     "Generic Excel file",
			filename: "data.xlsx",
			want:     FileTypeNomenclature,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.DetectFileType(tt.filename)
			if got != tt.want {
				t.Errorf("DetectFileType(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestDetector_DetectProvider(t *testing.T) {
	logger := zaptest.NewLogger(t)
	detector := NewDetector(logger)

	tests := []struct {
		name      string
		emailFrom string
		want      Provider
	}{
		{
			name:      "БИГМАШИН",
			emailFrom: "m.timoshenkova@bigm.pro",
			want:      ProviderBigMachine,
		},
		{
			name:      "БИГМАШИН uppercase",
			emailFrom: "INFO@BIGM.PRO",
			want:      ProviderBigMachine,
		},
		{
			name:      "ЗАПАСКА",
			emailFrom: "pna@sibzapaska.ru",
			want:      ProviderZapaska,
		},
		{
			name:      "ЗАПАСКА with different username",
			emailFrom: "info@sibzapaska.ru",
			want:      ProviderZapaska,
		},
		{
			name:      "ГРУППА БРИНЕКС",
			emailFrom: "b2bportal@brinex.ru",
			want:      ProviderBrinex,
		},
		{
			name:      "ГРУППА БРИНЕКС different email",
			emailFrom: "sales@brinex.ru",
			want:      ProviderBrinex,
		},
		{
			name:      "Unknown provider",
			emailFrom: "test@example.com",
			want:      ProviderUnknown,
		},
		{
			name:      "Unknown provider - gmail",
			emailFrom: "user@gmail.com",
			want:      ProviderUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.DetectProvider(tt.emailFrom)
			if got != tt.want {
				t.Errorf("DetectProvider(%q) = %v, want %v", tt.emailFrom, got, tt.want)
			}
		})
	}
}

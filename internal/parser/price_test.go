package parser

import (
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestPriceParser_shouldProcessSheet(t *testing.T) {
	logger := zaptest.NewLogger(t)
	parser := NewPriceParser(logger)

	tests := []struct {
		name      string
		sheetName string
		want      bool
	}{
		{
			name:      "Автошины",
			sheetName: "Автошины",
			want:      true,
		},
		{
			name:      "автошины lowercase",
			sheetName: "автошины",
			want:      true,
		},
		{
			name:      "АВТОШИНЫ uppercase",
			sheetName: "АВТОШИНЫ",
			want:      true,
		},
		{
			name:      "Зимние",
			sheetName: "Зимние",
			want:      true,
		},
		{
			name:      "Летние",
			sheetName: "Летние",
			want:      true,
		},
		{
			name:      "Легкогрузовые",
			sheetName: "Легкогрузовые",
			want:      true,
		},
		{
			name:      "Лист_1",
			sheetName: "Лист_1",
			want:      true,
		},
		{
			name:      "лист_1 lowercase",
			sheetName: "лист_1",
			want:      true,
		},
		{
			name:      "Мотошины should not be processed",
			sheetName: "Мотошины",
			want:      false,
		},
		{
			name:      "Sheet1 should not be processed",
			sheetName: "Sheet1",
			want:      false,
		},
		{
			name:      "Random sheet",
			sheetName: "Прочее",
			want:      false,
		},
		{
			name:      "Empty sheet name",
			sheetName: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.shouldProcessSheet(tt.sheetName)
			if got != tt.want {
				t.Errorf("shouldProcessSheet(%q) = %v, want %v", tt.sheetName, got, tt.want)
			}
		})
	}
}

func TestPriceParser_findPriceColumns(t *testing.T) {
	logger := zaptest.NewLogger(t)
	parser := NewPriceParser(logger)

	tests := []struct {
		name     string
		cols     []string
		provider string
		wantNil  bool
	}{
		{
			name: "Standard columns for ЗАПАСКА",
			cols: []string{"Артикул", "Оптовая цена", "Остаток", "Склад"},
			provider: string(ProviderZapaska),
			wantNil:  false,
		},
		{
			name: "БИГМАШИН with multiple остаток columns",
			cols: []string{"Артикул", "Оптовая цена", "Остаток Нск Северный", "Остаток Нск Южный", "Остаток Москва"},
			provider: string(ProviderBigMachine),
			wantNil:  false,
		},
		{
			name: "Case insensitive",
			cols: []string{"артикул", "оптовая цена", "остаток", "склад"},
			provider: string(ProviderZapaska),
			wantNil:  false,
		},
		{
			name: "Missing артикул",
			cols: []string{"Оптовая цена", "Остаток", "Склад"},
			provider: string(ProviderZapaska),
			wantNil:  true,
		},
		{
			name: "Missing оптовая цена",
			cols: []string{"Артикул", "Остаток", "Склад"},
			provider: string(ProviderZapaska),
			wantNil:  true,
		},
		{
			name: "Missing остаток",
			cols: []string{"Артикул", "Оптовая цена", "Склад"},
			provider: string(ProviderZapaska),
			wantNil:  true,
		},
		{
			name: "Empty columns",
			cols: []string{},
			provider: string(ProviderZapaska),
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.findPriceColumns(tt.cols, tt.provider)
			if tt.wantNil {
				if got != nil {
					t.Errorf("findPriceColumns() should return nil for %v", tt.cols)
				}
			} else {
				if got == nil {
					t.Errorf("findPriceColumns() should not return nil for %v", tt.cols)
				}
			}
		})
	}
}

func TestPriceParser_parseFloat(t *testing.T) {
	logger := zaptest.NewLogger(t)
	parser := NewPriceParser(logger)

	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "Simple number",
			input:   "100",
			want:    100.0,
			wantErr: false,
		},
		{
			name:    "Number with comma",
			input:   "100,50",
			want:    100.50,
			wantErr: false,
		},
		{
			name:    "Number with spaces",
			input:   "1 000,50",
			want:    1000.50,
			wantErr: false,
		},
		{
			name:    "Number with point",
			input:   "100.50",
			want:    100.50,
			wantErr: false,
		},
		{
			name:    "Empty string",
			input:   "",
			want:    0,
			wantErr: false,
		},
		{
			name:    "Whitespace only",
			input:   "   ",
			want:    0,
			wantErr: false,
		},
		{
			name:    "Invalid value",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.parseFloat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFloat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseFloat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPriceParser_parseInt(t *testing.T) {
	logger := zaptest.NewLogger(t)
	parser := NewPriceParser(logger)

	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "Simple number",
			input:   "100",
			want:    100,
			wantErr: false,
		},
		{
			name:    "Number with spaces",
			input:   "1 000",
			want:    1000,
			wantErr: false,
		},
		{
			name:    "Zero",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "Empty string",
			input:   "",
			want:    0,
			wantErr: false,
		},
		{
			name:    "Negative number",
			input:   "-5",
			want:    -5,
			wantErr: false,
		},
		{
			name:    "Invalid value",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.parseInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseInt(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

package parser

import (
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestParser_findColumns(t *testing.T) {
	logger := zaptest.NewLogger(t)
	p := New(logger)

	tests := []struct {
		name    string
		cols    []string
		wantNil bool
	}{
		{
			name: "all required columns present",
			cols: []string{"Артикул", "Марка", "Размер и Модель", "Номенклатура", "МРЦ"},
			wantNil: false,
		},
		{
			name: "all columns including optional Type",
			cols: []string{"Артикул", "Марка", "Тип", "Размер и Модель", "Номенклатура", "МРЦ"},
			wantNil: false,
		},
		{
			name: "missing required column",
			cols: []string{"Артикул", "Марка"},
			wantNil: true,
		},
		{
			name: "empty columns",
			cols: []string{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.findColumns(tt.cols)
			if (result == nil) != tt.wantNil {
				t.Errorf("findColumns() returned nil = %v, wantNil %v", result == nil, tt.wantNil)
			}
		})
	}
}

func TestParser_parseFloat(t *testing.T) {
	logger := zaptest.NewLogger(t)
	p := New(logger)

	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "simple number",
			input:   "100",
			want:    100,
			wantErr: false,
		},
		{
			name:    "number with comma",
			input:   "100,50",
			want:    100.50,
			wantErr: false,
		},
		{
			name:    "number with spaces",
			input:   "1 000,50",
			want:    1000.50,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: false,
		},
		{
			name:    "invalid number",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.parseFloat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

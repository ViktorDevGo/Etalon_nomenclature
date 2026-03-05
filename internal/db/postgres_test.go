package db

import (
	"testing"

	"github.com/prokoleso/etalon-nomenclature/config"
	"go.uber.org/zap/zaptest"
)

func TestNew(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name    string
		cfg     config.DatabaseConfig
		wantErr bool
	}{
		{
			name: "empty DSN",
			cfg: config.DatabaseConfig{
				DSN: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

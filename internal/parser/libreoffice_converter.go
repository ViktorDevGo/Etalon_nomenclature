package parser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// ConvertXLStoXLSXWithLibreOffice converts XLS to XLSX using LibreOffice
func ConvertXLStoXLSXWithLibreOffice(xlsContent []byte, logger *zap.Logger) ([]byte, error) {
	startTime := time.Now()

	if logger != nil {
		logger.Info("Starting XLS to XLSX conversion with LibreOffice",
			zap.Int("input_size_bytes", len(xlsContent)))
	}

	// Create temporary directory for conversion
	tempDir, err := os.MkdirTemp("", "xls-convert-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write XLS content to temporary file
	xlsPath := filepath.Join(tempDir, "input.xls")
	if err := os.WriteFile(xlsPath, xlsContent, 0644); err != nil {
		return nil, fmt.Errorf("failed to write XLS file: %w", err)
	}

	if logger != nil {
		logger.Debug("Temporary XLS file created", zap.String("path", xlsPath))
	}

	// Run LibreOffice conversion with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"libreoffice",
		"--headless",
		"--convert-to", "xlsx",
		"--outdir", tempDir,
		xlsPath,
	)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			if logger != nil {
				logger.Error("LibreOffice conversion timeout",
					zap.Duration("timeout", 60*time.Second))
			}
			return nil, fmt.Errorf("LibreOffice conversion timeout after 60 seconds")
		}
		if logger != nil {
			logger.Error("LibreOffice conversion failed",
				zap.Error(err),
				zap.String("output", string(output)))
		}
		return nil, fmt.Errorf("LibreOffice conversion failed: %w, output: %s", err, string(output))
	}

	if logger != nil {
		logger.Debug("LibreOffice conversion completed",
			zap.String("output", string(output)))
	}

	// Read converted XLSX file
	xlsxPath := filepath.Join(tempDir, "input.xlsx")
	xlsxContent, err := os.ReadFile(xlsxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read converted XLSX file: %w", err)
	}

	duration := time.Since(startTime)
	if logger != nil {
		logger.Info("XLS to XLSX conversion completed with LibreOffice",
			zap.Duration("duration", duration),
			zap.Int("input_size", len(xlsContent)),
			zap.Int("output_size", len(xlsxContent)))
	}

	return xlsxContent, nil
}

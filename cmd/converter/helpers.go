package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func humanSize(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := unit, 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func writeFile(outDir, name string, data []byte) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	outPath := filepath.Join(outDir, name)
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	fmt.Printf("Extracted: %s (%s)\n", outPath, humanSize(len(data)))
	return nil
}

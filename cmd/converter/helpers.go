// helpers.go provides shared utility functions for the CLI commands.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// humanSize formats a byte count as a human-readable string (e.g. "1.2 KB").
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

// writeFile writes data to outDir/name, ensuring the resulting path stays
// within outDir to prevent directory traversal attacks.
func writeFile(outDir, name string, data []byte) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	outPath := filepath.Join(outDir, name)

	// Verify the resolved path is still inside outDir.
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("resolving output directory: %w", err)
	}
	absPath, err := filepath.Abs(outPath)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}
	if !strings.HasPrefix(absPath, absOut+string(filepath.Separator)) && absPath != absOut {
		return fmt.Errorf("path traversal blocked: %s", name)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	fmt.Printf("Extracted: %s (%s)\n", outPath, humanSize(len(data)))
	return nil
}

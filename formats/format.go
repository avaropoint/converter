// Package formats defines the Converter interface and a registry for
// pluggable file format converters. To add a new format, create a
// package that implements Converter and calls Register from its init
// function. The registry auto-detects formats by content (magic bytes)
// first and falls back to file extension matching.
package formats

import (
	"path/filepath"
	"strings"
)

// ConvertedFile is a single output file produced by a conversion.
type ConvertedFile struct {
	Name     string
	Data     []byte
	Category string // "body" or "attachment"
}

// Converter handles detection and conversion of a specific file format.
type Converter interface {
	// Name returns a human-readable format name.
	Name() string

	// Extensions returns file extensions this converter handles,
	// including the leading dot (e.g. ".dat", ".tnef").
	Extensions() []string

	// Match returns true if data begins with recognized magic bytes.
	Match(data []byte) bool

	// Convert processes raw file data and returns the extracted files.
	Convert(data []byte) ([]ConvertedFile, error)
}

var registry []Converter

// Register adds a converter to the global registry. Call this from
// an init function in your format package.
func Register(c Converter) {
	registry = append(registry, c)
}

// Detect identifies the correct converter for a file. It checks content
// (magic bytes) first, then falls back to extension matching.
func Detect(filename string, data []byte) Converter {
	for _, c := range registry {
		if c.Match(data) {
			return c
		}
	}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, c := range registry {
		for _, e := range c.Extensions() {
			if ext == e {
				return c
			}
		}
	}
	return nil
}

// All returns every registered converter.
func All() []Converter {
	return registry
}

// SanitizeFilename replaces characters that are unsafe in file paths
// and strips control characters to prevent header injection.
func SanitizeFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1 // drop control characters
		}
		return r
	}, name)
	for _, c := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
		name = strings.ReplaceAll(name, c, "_")
	}
	if name == "" {
		name = "unnamed"
	}
	return name
}

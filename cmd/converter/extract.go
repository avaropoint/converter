// extract.go provides the CLI conversion commands (extract, body, dump)
// that detect the input format, convert the file, and write results.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/avaropoint/converter/formats"
)

// convertFile reads a file, auto-detects its format, and returns the
// converted output files. Exits on error.
func convertFile(path string) []formats.ConvertedFile {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
		os.Exit(1)
	}
	conv := formats.Detect(filepath.Base(path), data)
	if conv == nil {
		fmt.Fprintf(os.Stderr, "Unsupported file format: %s\n", filepath.Base(path))
		os.Exit(1)
	}
	files, err := conv.Convert(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting %s: %v\n", path, err)
		os.Exit(1)
	}
	return files
}

// writeConvertedFiles writes each converted file to the given output directory.
func writeConvertedFiles(files []formats.ConvertedFile, outDir string) {
	if len(files) == 0 {
		fmt.Println("No content to extract.")
		return
	}
	for _, f := range files {
		if err := writeFile(outDir, f.Name, f.Data); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
}

// cmdExtract converts a file and writes only the attachment outputs to outDir.
func cmdExtract(path, outDir string) {
	files := convertFile(path)
	var filtered []formats.ConvertedFile
	for _, f := range files {
		if f.Category == "attachment" {
			filtered = append(filtered, f)
		}
	}
	writeConvertedFiles(filtered, outDir)
}

// cmdBody converts a file and writes only the message body outputs to outDir.
func cmdBody(path, outDir string) {
	files := convertFile(path)
	var filtered []formats.ConvertedFile
	for _, f := range files {
		if f.Category == "body" {
			filtered = append(filtered, f)
		}
	}
	writeConvertedFiles(filtered, outDir)
}

// cmdDump converts a file and writes all extracted outputs to outDir.
func cmdDump(path, outDir string) {
	files := convertFile(path)
	writeConvertedFiles(files, outDir)
}

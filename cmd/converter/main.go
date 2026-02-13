// Converter is a CLI tool and HTTP server for converting and extracting
// the contents of TNEF (winmail.dat) email attachments.
package main

import (
	"fmt"
	"os"
	"strings"

	_ "github.com/avaropoint/converter/formats/tnef"
)

// version is the application version, embedded in API responses and used
// for static asset cache-busting.
const version = "1.0.0"

// usage prints command-line help to stderr.
func usage() {
	fmt.Fprintf(os.Stderr, `converter v%s
File converter and extractor

Usage:
  converter view    <file>              Show file summary
  converter extract <file> [output_dir] Extract attachments
  converter body    <file> [output_dir] Extract message body
  converter dump    <file> [output_dir] Extract everything
  converter serve   [port] [options]    Start web interface (default port 8080)
  converter help                        Show this help message

Serve options:
  --base-path <path>  Serve under a URL prefix (e.g. /converter)

Examples:
  converter view winmail.dat
  converter extract winmail.dat ./output
  converter dump winmail.dat ./output
  converter serve 9090
  converter serve 8080 --base-path /converter
`, version)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := strings.ToLower(os.Args[1])
	args := os.Args[2:]

	switch cmd {
	case "help", "-h", "--help":
		usage()
	case "version", "-v", "--version":
		fmt.Println(version)
	case "healthcheck":
		cmdHealthcheck(args)
	case "view":
		requireFile(args)
		cmdView(args[0])
	case "extract":
		requireFile(args)
		cmdExtract(args[0], outputDir(args))
	case "body":
		requireFile(args)
		cmdBody(args[0], outputDir(args))
	case "dump":
		requireFile(args)
		cmdDump(args[0], outputDir(args))
	case "serve", "server", "web":
		port := "8080"
		basePath := ""
		for i := 0; i < len(args); i++ {
			if args[i] == "--base-path" && i+1 < len(args) {
				basePath = args[i+1]
				i++
			} else {
				port = args[i]
			}
		}
		cmdServe(port, basePath)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}
}

// requireFile exits with an error if no file argument was provided.
func requireFile(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: file path required")
		usage()
		os.Exit(1)
	}
}

// outputDir returns the output directory from args, defaulting to ".".
func outputDir(args []string) string {
	if len(args) >= 2 {
		return args[1]
	}
	return "."
}

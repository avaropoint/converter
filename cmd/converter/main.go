package main

import (
	"fmt"
	"os"
	"strings"

	_ "github.com/avaropoint/converter/formats/tnef"
)

const version = "1.0.0"

func usage() {
	fmt.Fprintf(os.Stderr, `converter v%s
File converter and extractor

Usage:
  converter view    <file>              Show file summary
  converter extract <file> [output_dir] Extract attachments
  converter body    <file> [output_dir] Extract message body
  converter dump    <file> [output_dir] Extract everything
  converter serve   [port]              Start web interface (default port 8080)
  converter help                        Show this help message

Examples:
  converter view winmail.dat
  converter extract winmail.dat ./output
  converter dump winmail.dat ./output
  converter serve 9090
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
		if len(args) > 0 {
			port = args[0]
		}
		cmdServe(port)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}
}

func requireFile(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: file path required")
		usage()
		os.Exit(1)
	}
}

func outputDir(args []string) string {
	if len(args) >= 2 {
		return args[1]
	}
	return "."
}

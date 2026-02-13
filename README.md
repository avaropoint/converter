# Converter

A fast, zero-dependency file converter and extractor. Drop in a file, get back
its contents — as a CLI tool or a self-contained web interface.

Currently supports **TNEF** (`winmail.dat`) files with a pluggable architecture
for adding new formats.

[![CI](https://github.com/avaropoint/converter/actions/workflows/ci.yml/badge.svg)](https://github.com/avaropoint/converter/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/avaropoint/converter)](https://goreportcard.com/report/github.com/avaropoint/converter)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Features

- **TNEF / winmail.dat extraction** — attachments, HTML bodies, embedded messages
- **LZFu RTF decompression** and HTML de-encapsulation from RTF
- **CID image resolution** — inline images converted to self-contained data URIs
- **External image embedding** — remote `<img>` sources fetched and inlined
- **Pluggable format architecture** — add new formats without touching core code
- **Modern web interface** — drag-and-drop upload, file preview, bulk download
- **CLI with multiple commands** — view, extract, body, dump, serve
- **Zero external dependencies** — Go standard library only
- **Single binary** — cross-platform, no runtime requirements
- **Security hardened** — strict CSP, SSRF protection with DNS rebinding defense, rate limiting, filename sanitization
- **Embedded web assets** — HTML, CSS, JS compiled into the binary via `go:embed`

## Quick Start

### Install from Source

```bash
go install github.com/avaropoint/converter/cmd/converter@latest
```

### Download Binary

Pre-built binaries for Linux, macOS, and Windows are available on the
[Releases](https://github.com/avaropoint/converter/releases) page.

### Docker

```bash
docker pull ghcr.io/avaropoint/converter:latest
docker run -p 8080:8080 ghcr.io/avaropoint/converter:latest
```

Or with Docker Compose:

```bash
docker compose up
```

## Usage

### Web Interface

```bash
converter serve [port]    # Default: 8080
```

Open `http://localhost:8080` in your browser, drop a file, and view or download
the extracted contents.

### CLI

```bash
converter view    <file>              # Show file summary
converter extract <file> [output_dir] # Extract attachments only
converter body    <file> [output_dir] # Extract message body only
converter dump    <file> [output_dir] # Extract everything
```

### Examples

```bash
# View what's inside a winmail.dat
converter view winmail.dat

# Extract all attachments to a folder
converter extract winmail.dat ./output

# Dump everything (body + attachments + embedded messages)
converter dump winmail.dat ./output

# Start the web interface on port 9090
converter serve 9090
```

## Architecture

```
converter
├── cmd/converter/       CLI + web server
├── cmd/inspect/         Low-level TNEF diagnostic tool
├── formats/             Converter interface + registry
│   └── tnef/            TNEF format implementation
├── tnefparser/          TNEF binary stream parser
└── web/                 Embedded static assets (go:embed)
    └── static/          HTML, CSS, JS served by the web UI
```

### Pluggable Format System

Converter uses a registry pattern for format auto-detection:

1. **Magic bytes** — each format checks file headers first
2. **Extension fallback** — matches by file extension if magic bytes don't match
3. **Auto-registration** — formats register themselves via `init()`

### Adding a New Format

Create a package under `formats/` implementing the `Converter` interface:

```go
package myformat

import "github.com/avaropoint/converter/formats"

func init() {
    formats.Register(&conv{})
}

type conv struct{}

func (c *conv) Name() string           { return "My Format" }
func (c *conv) Extensions() []string   { return []string{".myf"} }
func (c *conv) Match(data []byte) bool { return len(data) > 4 && data[0] == 0xAB }
func (c *conv) Convert(data []byte) ([]formats.ConvertedFile, error) {
    // Parse the format and return extracted files
    return nil, nil
}
```

Then add a blank import in `cmd/converter/main.go`:

```go
import _ "github.com/avaropoint/converter/formats/myformat"
```

## Development

### Prerequisites

- Go 1.25 or later

### Build & Test

```bash
make build    # Build the binary
make test     # Run tests with race detection
make vet      # Run go vet
make lint     # Run staticcheck
make check    # All of the above
make run      # Build and start web server
```

### Project Principles

- **Zero external dependencies** — standard library only
- **Single binary deployment** — no config files, no runtime dependencies
- **Security by default** — CSP headers, SSRF blocks, input sanitization
- **Pluggable architecture** — new formats require zero changes to existing code

## Security

See [SECURITY.md](SECURITY.md) for the full security policy.

Key protections:

| Threat | Mitigation |
|--------|-----------|
| XSS in extracted HTML | Strict CSP: `'self'` for main page, `default-src 'none'` for extracted files |
| SSRF via image URLs | DNS rebinding-safe custom dialer, redirect validation, private IP blocks |
| Header injection | Control characters stripped from filenames |
| Upload abuse | 50 MB limit via `MaxBytesReader` + rate limiting |
| Session enumeration | 128-bit `crypto/rand` session IDs |
| Slowloris / connection exhaustion | Read/Write/Idle timeouts + graceful shutdown |
| Clickjacking | `X-Frame-Options: DENY` + `frame-ancestors 'none'` |
| MIME sniffing | `X-Content-Type-Options: nosniff` |

### Production Deployment

For internet-facing deployments, we recommend placing Converter behind a reverse
proxy (nginx, Caddy) that provides:

- TLS termination
- Authentication
- Rate limiting
- Access logging

## Free & Open Source

This project is released under the [MIT License](LICENSE) and is **completely
free to use**. Monetization of this software or derivative works is **strictly
prohibited**. This tool is built for the community and must remain freely
available to everyone.

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE) — Copyright (c) 2026 Avaropoint

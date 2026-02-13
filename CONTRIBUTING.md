# Contributing to Converter

Thank you for your interest in contributing! This project is free and open source
— contributions of all kinds are welcome.

## How to Contribute

### Reporting Bugs

Open a [GitHub Issue](https://github.com/avaropoint/converter/issues) with:
- A clear title and description
- Steps to reproduce the problem
- The file format you were converting (if applicable)
- Your OS and Go version (`go version`)

### Suggesting New Formats

We use a pluggable architecture — adding a new format converter is straightforward.
Open an issue describing the format and we can discuss the approach.

### Submitting Code

1. **Fork** the repository
2. **Create a branch** from `main`: `git checkout -b my-feature`
3. **Make your changes** — keep commits focused and atomic
4. **Test** your changes: `make test`
5. **Vet and build**: `make build`
6. **Submit a Pull Request** against `main`

### Adding a New Format Converter

Create a new package under `formats/` that implements the `formats.Converter` interface:

```go
package myformat

import "github.com/avaropoint/converter/formats"

func init() {
    formats.Register(&converter{})
}

type converter struct{}

func (c *converter) Name() string           { return "My Format" }
func (c *converter) Extensions() []string   { return []string{".myf"} }
func (c *converter) Match(data []byte) bool { return len(data) > 2 && data[0] == 0xAB }
func (c *converter) Convert(data []byte) ([]formats.ConvertedFile, error) {
    // Parse and extract files...
    return nil, nil
}
```

Then add a blank import in `cmd/converter/main.go`:

```go
import _ "github.com/avaropoint/converter/formats/myformat"
```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Zero external dependencies — use the standard library only
- Keep functions focused and well-documented
- Handle errors explicitly

## Development Setup

```bash
git clone https://github.com/avaropoint/converter.git
cd converter
make build     # Build the binary
make test      # Run tests
make vet       # Run go vet
```

## Monetization Policy

This project is and will remain **completely free**. Monetization of this software
or derivative works is **strictly prohibited**. See the README for details.

## License

By contributing, you agree that your contributions will be licensed under the
MIT License.

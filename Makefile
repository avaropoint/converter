.PHONY: build test vet lint clean run docker

# Default binary name
BINARY := bin/converter
VERSION := $(shell grep 'const version' cmd/converter/main.go | cut -d'"' -f2)

## build: Compile the binary
build:
	@mkdir -p bin
	go build -ldflags="-s -w" -o $(BINARY) ./cmd/converter

## test: Run all tests with race detection
test:
	go test -v -race -count=1 ./...

## vet: Run go vet
vet:
	go vet ./...

## lint: Run staticcheck (install with: go install honnef.co/go/tools/cmd/staticcheck@latest)
lint:
	staticcheck ./...

## vuln: Run govulncheck (install with: go install golang.org/x/vuln/cmd/govulncheck@latest)
vuln:
	govulncheck ./...

## check: Run all checks (vet + test + lint)
check: vet test lint

## run: Build and start the web server on port 8080
run: build
	./$(BINARY) serve

## clean: Remove build artifacts
clean:
	rm -rf bin/
	rm -rf out/

## docker: Build Docker image
docker:
	docker build -t converter:$(VERSION) .

## install: Install to $GOPATH/bin
install:
	go install ./cmd/converter

## help: Show this help
help:
	@echo "converter v$(VERSION)"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'

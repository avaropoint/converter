# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /converter ./cmd/converter

# Runtime stage — scratch: zero OS, zero shell, zero attack surface.
# The only thing in this image is the statically-linked converter binary.
FROM scratch

# Import CA certificates so HTTPS (external image inlining) works.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Run as non-root (UID 65534 = nobody).
USER 65534:65534

COPY --from=builder /converter /converter

EXPOSE 8080

# Healthcheck uses the binary itself — no wget/curl needed.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/converter", "healthcheck"]

ENTRYPOINT ["/converter"]
CMD ["serve", "8080"]

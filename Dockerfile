# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /converter ./cmd/converter

# Runtime stage
FROM alpine:3.20

RUN addgroup -S converter && adduser -S converter -G converter

COPY --from=builder /converter /usr/local/bin/converter

USER converter

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/info || exit 1

ENTRYPOINT ["converter"]
CMD ["serve", "8080"]

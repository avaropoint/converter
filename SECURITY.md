# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.0.x   | Yes       |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do NOT open a public issue**
2. Email security concerns to the repository maintainers via GitHub's
   [private vulnerability reporting](https://github.com/avaropoint/converter/security/advisories/new)
3. Include a clear description of the vulnerability and steps to reproduce

We will acknowledge receipt within 48 hours and aim to provide a fix within 7 days
for critical issues.

## Security Model

Converter is a file parsing and extraction tool. Its security posture:

### What We Protect Against

- **XSS in extracted HTML**: Extracted HTML files are served with a strict
  Content-Security-Policy (`default-src 'none'; style-src 'unsafe-inline'; img-src data:`)
  that blocks all script execution. The main web UI page uses
  `default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:`
  — no inline scripts or styles are permitted.
- **Static asset integrity**: Web UI assets (HTML, CSS, JS) are compiled into
  the binary via `go:embed` — no filesystem access is needed at runtime, and
  the assets cannot be tampered with after build.
- **SSRF via image inlining**: External image fetching uses a custom dialer that
  validates resolved IP addresses before connecting, preventing DNS rebinding
  attacks. Private, loopback, link-local, and cloud metadata IP ranges are
  blocked. Redirects are validated at each hop.
- **Rate limiting**: Upload endpoint (`/api/convert`) is rate-limited to prevent
  resource exhaustion from concurrent CPU/memory-intensive conversions.
- **Header injection**: Filenames from converted files are sanitized to remove
  control characters and path separators before use in HTTP headers.
- **Upload abuse**: 50 MB upload limit enforced via `MaxBytesReader`.
- **Session enumeration**: 128-bit cryptographically random session IDs.
- **Clickjacking**: `X-Frame-Options: DENY` and `frame-ancestors 'none'`.
- **MIME sniffing**: `X-Content-Type-Options: nosniff` on all responses.
- **Referrer leakage**: `Referrer-Policy: no-referrer` on all responses.
- **Device access**: `Permissions-Policy: camera=(), microphone=(), geolocation=()`
  prevents access to sensitive device APIs.
- **Path traversal**: Filenames from untrusted sources are sanitized and validated
  to ensure writes stay within the intended output directory.
- **Memory safety**: Parser allocations are bounded to prevent crafted files from
  causing out-of-memory crashes.
- **Graceful shutdown**: The server handles SIGINT/SIGTERM for clean connection
  draining.

### What Is Out of Scope

- **Malware scanning**: Converter parses and extracts files but does not scan for
  viruses or malware. Extracted attachments (executables, documents, etc.) should
  be treated with the same caution as any email attachment.
- **Authentication**: The web interface has no login system. If exposed to the
  internet, place it behind a reverse proxy with authentication.
- **TLS**: The built-in server uses plain HTTP. Use a reverse proxy (nginx, Caddy)
  to terminate TLS in production.

### Production Deployment Recommendations

1. Run behind a reverse proxy with TLS termination
2. Add authentication if the server is internet-facing
3. Use the Docker image for isolation
4. Set appropriate firewall rules to restrict access
5. Monitor logs for unusual activity

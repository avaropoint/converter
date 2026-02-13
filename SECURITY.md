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

- **XSS in extracted HTML**: All extracted HTML is served with a strict
  Content-Security-Policy (`default-src 'none'`) that blocks script execution
- **SSRF via image inlining**: External image fetching blocks private, loopback,
  link-local, and cloud metadata IP ranges
- **Header injection**: Filenames from converted files are sanitized to remove
  control characters before use in HTTP headers
- **Upload abuse**: 50 MB upload limit enforced via `MaxBytesReader`
- **Session enumeration**: 128-bit cryptographically random session IDs
- **Clickjacking**: `X-Frame-Options: DENY` and `frame-ancestors 'none'`
- **MIME sniffing**: `X-Content-Type-Options: nosniff` on all responses

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

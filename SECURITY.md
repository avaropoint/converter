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

### Container Sandbox (Defense in Depth)

The Docker deployment provides a hardened, multi-layer sandbox:

| Layer | Control | What It Prevents |
|-------|---------|-------------------|
| **Base image** | `scratch` (zero OS) | No shell, no utilities, no package manager — nothing to exploit |
| **User** | UID 65534 (nobody) | No root access even if code is compromised |
| **Filesystem** | `read_only: true` | Cannot write malware, modify binaries, or create files |
| **Capabilities** | `cap_drop: ALL` | No raw sockets, no mount, no ptrace, no kernel interaction |
| **Privilege** | `no-new-privileges` | Blocks SUID/SGID escalation |
| **Syscalls** | Custom seccomp profile | Only ~40 Go-required syscalls whitelisted; all others denied |
| **Memory** | 256 MB limit | OOM/memory exhaustion cannot consume host RAM |
| **CPU** | 1.0 core limit | CPU exhaustion cannot slow the host |
| **PIDs** | 64 process limit | Fork bombs are impossible |
| **File descriptors** | 1024 soft / 4096 hard | Descriptor exhaustion blocked |
| **tmpfs** | 16 MB, noexec, nosuid, nodev | Ephemeral, non-executable, vanishes on restart |
| **Logs** | Capped at 30 MB (3 × 10 MB) | Log flooding cannot fill host disk |
| **Health** | Built-in `healthcheck` command | No wget/curl needed in the image |

Even a theoretical RCE through a parser bug would give an attacker:
- A read-only filesystem with zero capabilities
- No shell, no package manager, no scripting runtime
- An ephemeral 16 MB tmpfs that vanishes on restart
- No access to the host filesystem or other containers
- Only ~40 permitted syscalls — cannot spawn processes, mount filesystems, or use raw sockets

### Production Deployment Recommendations

1. Run via `docker compose up` (inherits all sandbox controls)
2. Place behind a reverse proxy with TLS termination (nginx, Caddy)
3. Add authentication if the server is internet-facing
4. Set appropriate firewall rules to restrict access
5. Monitor logs for unusual activity
6. For maximum isolation, add `network_mode: internal` to restrict egress

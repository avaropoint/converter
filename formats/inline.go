// inline.go fetches external images referenced in HTML and converts them
// to inline data URIs. All network access goes through an SSRF-safe HTTP
// client that blocks private, loopback, and link-local addresses.

package formats

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// imgSrcRe matches <img src="..."> attributes for URL replacement.
var imgSrcRe = regexp.MustCompile(`(<img\b[^>]*?\bsrc=")([^"]+)(")`)

// ssrfSafeDialer returns a DialContext that resolves DNS and checks every
// resolved IP against the private/loopback/link-local blocklist BEFORE
// connecting.  This eliminates the DNS rebinding TOCTOU race that exists
// when isPrivateHost() and the actual connection resolve independently.
func ssrfSafeDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	resolver := &net.Resolver{}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		// Block obviously dangerous hostnames before any DNS lookup.
		if isBlockedHostname(host) {
			return nil, errors.New("blocked host")
		}

		// Resolve with a tight timeout to prevent hanging on unresolvable hosts.
		resolveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		ips, err := resolver.LookupIPAddr(resolveCtx, host)
		if err != nil {
			return nil, err
		}

		// Check ALL resolved IPs -- block if any are private.
		for _, ip := range ips {
			if isBlockedIP(ip.IP) {
				return nil, errors.New("blocked IP")
			}
		}

		// Connect to the resolved IP directly (bypasses further DNS).
		// Try each resolved address until one succeeds.
		for _, ip := range ips {
			target := net.JoinHostPort(ip.IP.String(), port)
			conn, err := dialer.DialContext(ctx, network, target)
			if err == nil {
				return conn, nil
			}
		}
		return nil, errors.New("all addresses failed")
	}
}

// inlineClient uses a custom transport with an SSRF-safe dialer that
// validates resolved IPs before connecting.
var inlineClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		DialContext:           ssrfSafeDialer(),
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
	},
	// Validate each redirect target against the SSRF blocklist.
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		host := req.URL.Hostname()
		if isBlockedHostname(host) {
			return errors.New("redirect to blocked host")
		}
		// The custom dialer will also check the resolved IP, so even if
		// the hostname looks benign, a private IP will still be blocked.
		return nil
	},
}

// InlineExternalImages finds all <img src="https://..."> references in html,
// fetches the image data, and replaces src with a data URI. Images that
// fail to download are left as-is. Only http/https URLs are fetched.
// Each unique URL is fetched only once to avoid rate limiting. Pass a
// non-nil cache map to share results across multiple calls.
func InlineExternalImages(html []byte, cache map[string]string) []byte {
	if cache == nil {
		cache = make(map[string]string)
	}

	return imgSrcRe.ReplaceAllFunc(html, func(match []byte) []byte {
		parts := imgSrcRe.FindSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		prefix := parts[1] // <img ... src="
		rawURL := string(parts[2])
		suffix := parts[3] // "

		if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
			return match
		}

		if strings.HasPrefix(rawURL, "data:") {
			return match
		}

		dataURI, seen := cache[rawURL]
		if !seen {
			data, contentType, err := fetchImage(rawURL)
			if err != nil || len(data) == 0 {
				cache[rawURL] = ""
			} else {
				mime := imageContentType(contentType)
				b64 := base64.StdEncoding.EncodeToString(data)
				dataURI = "data:" + mime + ";base64," + b64
				cache[rawURL] = dataURI
			}
		}

		if dataURI == "" {
			return match
		}

		var result []byte
		result = append(result, prefix...)
		result = append(result, []byte(dataURI)...)
		result = append(result, suffix...)
		return result
	})
}

// fetchImage downloads an image from rawURL and returns the bytes and content type.
// Returns empty results (without error) for non-image or blocked URLs.
func fetchImage(rawURL string) ([]byte, string, error) {
	// Basic URL validation.
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, "", nil
	}

	// Hostname pre-check (the dialer also checks resolved IPs).
	if isBlockedHostname(parsed.Hostname()) {
		return nil, "", nil
	}

	resp, err := inlineClient.Get(rawURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", nil
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return nil, "", nil
	}

	// Limit to 5 MB per image.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, "", err
	}
	return data, ct, nil
}

// isBlockedHostname returns true if the hostname should be blocked
// without needing DNS resolution.
func isBlockedHostname(host string) bool {
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".local") ||
		lower == "metadata.google.internal" ||
		strings.HasSuffix(lower, ".internal") {
		return true
	}
	// Also block if the host is a raw IP that is private.
	if ip := net.ParseIP(host); ip != nil {
		return isBlockedIP(ip)
	}
	return false
}

// isBlockedIP returns true if the IP address is in a range that should
// not be fetched (private, loopback, link-local, unspecified).
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// imageContentType normalises a Content-Type header to a standard image MIME type.
func imageContentType(ct string) string {
	ct = strings.ToLower(ct)
	switch {
	case strings.Contains(ct, "png"):
		return "image/png"
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return "image/jpeg"
	case strings.Contains(ct, "gif"):
		return "image/gif"
	case strings.Contains(ct, "webp"):
		return "image/webp"
	case strings.Contains(ct, "svg"):
		return "image/svg+xml"
	case strings.Contains(ct, "bmp"):
		return "image/bmp"
	default:
		return "image/png"
	}
}

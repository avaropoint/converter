package formats

import (
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var imgSrcRe = regexp.MustCompile(`(<img\b[^>]*?\bsrc=")([^"]+)(")`)

var inlineClient = &http.Client{
	Timeout: 5 * time.Second,
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
		url := string(parts[2])
		suffix := parts[3] // "

		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return match
		}

		// Already a data URI
		if strings.HasPrefix(url, "data:") {
			return match
		}

		// Check cache first
		dataURI, seen := cache[url]
		if !seen {
			data, contentType, err := fetchImage(url)
			if err != nil || len(data) == 0 {
				cache[url] = ""
			} else {
				mime := imageContentType(contentType)
				b64 := base64.StdEncoding.EncodeToString(data)
				dataURI = "data:" + mime + ";base64," + b64
				cache[url] = dataURI
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

func fetchImage(rawURL string) ([]byte, string, error) {
	// SSRF protection: block private, loopback, and link-local IPs.
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", err
	}
	host := parsed.Hostname()
	if isPrivateHost(host) {
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

	// Limit to 5 MB per image
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, "", err
	}
	return data, ct, nil
}

// isPrivateHost returns true if the host resolves to a private, loopback,
// or link-local address that should not be fetched (SSRF protection).
func isPrivateHost(host string) bool {
	// Block obvious names first.
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".local") ||
		lower == "metadata.google.internal" ||
		strings.HasSuffix(lower, ".internal") {
		return true
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		// Can't resolve — block to be safe.
		return true
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return true
		}
	}
	return false
}

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

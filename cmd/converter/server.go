package main

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/avaropoint/converter/formats"
	"github.com/avaropoint/converter/web"
)

// ---------------------------------------------------------------------------
// Session management
// ---------------------------------------------------------------------------

// session holds the extracted files for a single conversion.
type session struct {
	files   []extractedFile
	created time.Time
}

// extractedFile is a single file produced by conversion.
type extractedFile struct {
	Name string `json:"name"`
	Size int    `json:"size"`
	Type string `json:"type"`
	data []byte
}

// sessionStore manages in-memory conversion results.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
	done     chan struct{} // closed on shutdown to stop cleanup goroutine
}

func newSessionStore() *sessionStore {
	s := &sessionStore{
		sessions: make(map[string]*session),
		done:     make(chan struct{}),
	}
	go s.cleanup()
	return s
}

func (s *sessionStore) create(files []extractedFile) string {
	id := randomID()
	s.mu.Lock()
	s.sessions[id] = &session{files: files, created: time.Now()}
	s.mu.Unlock()
	return id
}

func (s *sessionStore) get(id string) *session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

// cleanup removes sessions older than 10 minutes.
func (s *sessionStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			for id, sess := range s.sessions {
				if time.Since(sess.created) > 10*time.Minute {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		case <-s.done:
			return
		}
	}
}

// stop signals the cleanup goroutine to exit.
func (s *sessionStore) stop() { close(s.done) }

// ---------------------------------------------------------------------------
// Rate limiter (stdlib-only token bucket)
// ---------------------------------------------------------------------------

// rateLimiter implements a simple token-bucket rate limiter.
type rateLimiter struct {
	tokens     int64 // current tokens (atomic)
	maxTokens  int64
	refillRate int64 // tokens added per second
	done       chan struct{}
}

func newRateLimiter(maxTokens, refillRate int64) *rateLimiter {
	rl := &rateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		done:       make(chan struct{}),
	}
	go rl.refill()
	return rl
}

func (rl *rateLimiter) refill() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cur := atomic.LoadInt64(&rl.tokens)
			next := cur + rl.refillRate
			if next > rl.maxTokens {
				next = rl.maxTokens
			}
			atomic.StoreInt64(&rl.tokens, next)
		case <-rl.done:
			return
		}
	}
}

// allow returns true if a token is available, consuming one.
func (rl *rateLimiter) allow() bool {
	for {
		cur := atomic.LoadInt64(&rl.tokens)
		if cur <= 0 {
			return false
		}
		if atomic.CompareAndSwapInt64(&rl.tokens, cur, cur-1) {
			return true
		}
	}
}

func (rl *rateLimiter) stop() { close(rl.done) }

// ---------------------------------------------------------------------------
// Crypto
// ---------------------------------------------------------------------------

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is unrecoverable -- log and exit cleanly
		// rather than panicking inside an HTTP handler.
		slog.Error("crypto/rand failed", "error", err)
		os.Exit(1)
	}
	return hex.EncodeToString(b)
}

// isHexString returns true if s contains only hex characters.
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return len(s) > 0
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

// cmdServe starts the web interface on the given port.
func cmdServe(port string) {
	// Structured JSON logger for machine-readable, searchable logs.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	store := newSessionStore()
	limiter := newRateLimiter(10, 2) // 10 burst, 2/sec refill

	mux := http.NewServeMux()

	// Serve the main page from embedded static files.
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/convert", handleConvert(store, limiter))
	mux.HandleFunc("/api/files/", handleFile(store))
	mux.HandleFunc("/api/zip/", handleZip(store))

	// Serve embedded static assets (CSS, JS) under /static/ with cache headers.
	staticContent, _ := fs.Sub(web.StaticFS, "static")
	mux.Handle("/static/", cacheHeaders(
		http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))),
	))

	addr := ":" + port
	srv := &http.Server{
		Addr:              addr,
		Handler:           requestLogger(securityHeaders(mux)),
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("server starting",
			"version", version,
			"addr", addr,
			"url", "http://localhost"+addr,
		)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutdown initiated", "timeout", "10s")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	store.stop()
	limiter.stop()
	slog.Info("server stopped")
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// responseCapture wraps http.ResponseWriter to capture the status code.
type responseCapture struct {
	http.ResponseWriter
	status int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.status = code
	rc.ResponseWriter.WriteHeader(code)
}

// requestLogger logs every HTTP request with method, path, status, and duration.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rc := &responseCapture{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rc, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rc.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

// securityHeaders wraps a handler with common security headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("X-DNS-Prefetch-Control", "off")
		next.ServeHTTP(w, r)
	})
}

// cacheHeaders adds long-lived cache headers for immutable embedded assets.
func cacheHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		next.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// handleIndex serves the embedded HTML page.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self'; style-src 'self'; "+
			"img-src 'self' data:; base-uri 'self'; form-action 'self'; "+
			"object-src 'none'; frame-ancestors 'none'")
	data, err := web.StaticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// handleInfo returns the server version as JSON.
func handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]string{"version": version})
}

// convertResponse is the JSON returned after a successful conversion.
type convertResponse struct {
	SessionID string          `json:"sessionId"`
	Files     []extractedFile `json:"files"`
}

// handleConvert processes an uploaded file, auto-detecting its format.
func handleConvert(store *sessionStore, limiter *rateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		// Rate limit conversion requests.
		if !limiter.allow() {
			w.Header().Set("Retry-After", "1")
			slog.Warn("rate limit exceeded", "remote", r.RemoteAddr)
			jsonError(w, "Too many requests -- try again shortly", http.StatusTooManyRequests)
			return
		}

		// Limit upload to 50 MB.
		r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

		file, header, err := r.FormFile("file")
		if err != nil {
			jsonError(w, "No file uploaded", http.StatusBadRequest)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			jsonError(w, "Failed to read file", http.StatusBadRequest)
			return
		}

		conv := formats.Detect(header.Filename, data)
		if conv == nil {
			jsonError(w, "Unsupported file format", http.StatusBadRequest)
			return
		}

		items, err := conv.Convert(data)
		if err != nil {
			jsonError(w, "Conversion failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		if len(items) == 0 {
			jsonError(w, "No content found in file", http.StatusUnprocessableEntity)
			return
		}

		files := make([]extractedFile, len(items))
		for i, item := range items {
			files[i] = extractedFile{
				Name: item.Name,
				Size: len(item.Data),
				Type: guessType(item.Name),
				data: item.Data,
			}
		}

		sid := store.create(files)

		slog.Info("conversion complete",
			"session", sid,
			"filename", header.Filename,
			"input_bytes", len(data),
			"output_files", len(files),
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(convertResponse{
			SessionID: sid,
			Files:     files,
		})
	}
}

// handleFile serves a single extracted file by session ID and filename.
func handleFile(store *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Path: /api/files/{sessionID}/{filename}
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/files/"), "/", 2)
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		sid, name := parts[0], parts[1]

		// Validate session ID format (must be 32 hex chars).
		if len(sid) != 32 || !isHexString(sid) {
			http.NotFound(w, r)
			return
		}

		sess := store.get(sid)
		if sess == nil {
			jsonError(w, "Session expired or not found", http.StatusNotFound)
			return
		}

		for _, f := range sess.files {
			if f.Name == name {
				ct := contentType(f.Name, f.Type)
				w.Header().Set("Content-Type", ct)
				w.Header().Set("Content-Disposition", safeDisposition(f.Name))
				w.Header().Set("Cache-Control", "private, no-store")
				// Extracted HTML may contain malicious scripts;
				// block execution with a strict CSP.
				if f.Type == "html" {
					w.Header().Set("Content-Security-Policy",
						"default-src 'none'; style-src 'unsafe-inline'; img-src data:; frame-ancestors 'none'")
				}
				w.Write(f.data)
				return
			}
		}
		http.NotFound(w, r)
	}
}

// handleZip streams all extracted files as a zip archive directly to the client.
func handleZip(store *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid := strings.TrimPrefix(r.URL.Path, "/api/zip/")

		// Validate session ID format.
		if len(sid) != 32 || !isHexString(sid) {
			http.NotFound(w, r)
			return
		}

		sess := store.get(sid)
		if sess == nil {
			jsonError(w, "Session expired or not found", http.StatusNotFound)
			return
		}

		// Set headers before streaming -- cannot change after first write.
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="converted_output.zip"`)

		// Stream zip directly to the response writer (no buffering).
		zw := zip.NewWriter(w)
		for _, f := range sess.files {
			fw, err := zw.Create(f.Name)
			if err != nil {
				break
			}
			if _, err := fw.Write(f.data); err != nil {
				break
			}
		}
		zw.Close()
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func guessType(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm"):
		return "html"
	case strings.HasSuffix(lower, ".txt"):
		return "text"
	case strings.HasSuffix(lower, ".rtf"):
		return "rtf"
	case strings.HasSuffix(lower, ".png"):
		return "image"
	case strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg"):
		return "image"
	case strings.HasSuffix(lower, ".gif"):
		return "image"
	case strings.HasSuffix(lower, ".pdf"):
		return "pdf"
	case strings.HasSuffix(lower, ".doc") || strings.HasSuffix(lower, ".docx"):
		return "document"
	case strings.HasSuffix(lower, ".xls") || strings.HasSuffix(lower, ".xlsx"):
		return "spreadsheet"
	default:
		return "file"
	}
}

func contentType(name, fileType string) string {
	switch fileType {
	case "html":
		return "text/html; charset=utf-8"
	case "text":
		return "text/plain; charset=utf-8"
	case "rtf":
		return "application/rtf"
	case "image":
		return imageMIME(name)
	case "pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func imageMIME(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".svg"):
		return "image/svg+xml"
	default:
		return "image/png"
	}
}

// safeDisposition returns a Content-Disposition header value with the
// filename sanitized to prevent header injection.
func safeDisposition(name string) string {
	safe := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)
	return fmt.Sprintf(`inline; filename="%s"`, safe)
}

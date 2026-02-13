package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/avaropoint/converter/formats"
	"github.com/avaropoint/converter/web"
)

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
}

func newSessionStore() *sessionStore {
	s := &sessionStore{sessions: make(map[string]*session)}
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
	for {
		time.Sleep(time.Minute)
		s.mu.Lock()
		for id, sess := range s.sessions {
			if time.Since(sess.created) > 10*time.Minute {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// cmdServe starts the web interface on the given port.
func cmdServe(port string) {
	store := newSessionStore()
	mux := http.NewServeMux()

	// Serve the main page from embedded static files.
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/convert", handleConvert(store))
	mux.HandleFunc("/api/files/", handleFile(store))
	mux.HandleFunc("/api/zip/", handleZip(store))

	// Serve embedded static assets (CSS, JS) under /static/.
	staticContent, _ := fs.Sub(web.StaticFS, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	addr := ":" + port
	fmt.Printf("converter v%s web interface\n", version)
	fmt.Printf("Listening on http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, securityHeaders(mux)))
}

// securityHeaders wraps a handler with common security headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// handleIndex serves the embedded HTML page.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
	data, _ := web.StaticFS.ReadFile("static/index.html")
	w.Write(data)
}

// handleInfo returns the server version as JSON.
func handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": version})
}

// convertResponse is the JSON returned after a successful conversion.
type convertResponse struct {
	SessionID string          `json:"sessionId"`
	Files     []extractedFile `json:"files"`
}

// handleConvert processes an uploaded file, auto-detecting its format.
func handleConvert(store *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
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

// handleZip returns all extracted files as a zip archive.
func handleZip(store *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid := strings.TrimPrefix(r.URL.Path, "/api/zip/")
		sess := store.get(sid)
		if sess == nil {
			jsonError(w, "Session expired or not found", http.StatusNotFound)
			return
		}

		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for _, f := range sess.files {
			fw, err := zw.Create(f.Name)
			if err != nil {
				continue
			}
			fw.Write(f.data)
		}
		zw.Close()

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="converted_output.zip"`)
		w.Write(buf.Bytes())
	}
}

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
	// Remove control characters, quotes, and backslashes
	safe := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)
	return fmt.Sprintf(`inline; filename="%s"`, safe)
}

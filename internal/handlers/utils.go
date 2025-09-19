package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// --- Request Validation Helpers ---

// validateGETRequest checks if request is GET method and matches expected path.
func (h *Handler) validateGETRequest(w http.ResponseWriter, r *http.Request, expectedPath string) bool {
	if r.Method != http.MethodGet {
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}

	if r.URL.Path != expectedPath {
		h.Error(w, r, http.StatusNotFound, "Page not found")
		return false
	}

	return true
}

// --- Template Helpers ---

// render renders a template with the given data and status code (if provided) managing all errors.
func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, data any, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}

	tmpl, ok := h.templates[name]
	if !ok {
		// If we are already rendering an error, don't call h.Error again.
		if name == "error.tmpl" {
			log.Printf("FATAL: error.tmpl is missing")
			http.Error(w, "500 Internal Server Error - Error template not found", http.StatusInternalServerError)
			return
		}
		h.Error(w, r, http.StatusInternalServerError, fmt.Sprintf("Template %s not found", name))
		return
	}

	buf := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(buf, "base", data); err != nil {
		// If we get an error while executing the error template, we need to fallback.
		if name == "error.tmpl" {
			log.Printf("Error executing error template: %v", err)
			http.Error(w, "500 Internal Server Error - Failed to execute error template", http.StatusInternalServerError)
			return
		}
		h.Error(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	buf.WriteTo(w)
}

// loadTemplates loads and parses all templates from the templates directory.
func (h *Handler) loadTemplates() {
	h.templates = make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"join":  func(items []string, sep string) string { return strings.Join(items, sep) },
		"upper": func(s string) string { return strings.ToUpper(s) },
		"title": func(s string) string {
			words := strings.Fields(strings.ReplaceAll(s, "-", " "))
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},
	}

	const templateDir = "templates"
	baseTmplPath := filepath.Join(templateDir, "base.tmpl")

	if _, err := os.Stat(baseTmplPath); err != nil {
		log.Fatalf("Failed to find base template at %s: %v", baseTmplPath, err)
	}

	pages, err := filepath.Glob(filepath.Join(templateDir, "*.tmpl"))
	if err != nil {
		log.Fatalf("Failed to glob templates: %v", err)
	}

	for _, page := range pages {
		name := filepath.Base(page)
		if name == "base.tmpl" {
			continue
		}

		ts, err := template.New(name).Funcs(funcMap).ParseFiles(baseTmplPath, page)
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", name, err)
		}

		h.templates[name] = ts
	}
}

// --- Static File Helpers ---

// generateETag creates a simple ETag based on file size and modification time
func (h *Handler) generateETag(fi os.FileInfo) string {
	return fmt.Sprintf(`"%x-%x"`, fi.Size(), fi.ModTime().Unix())
}

// getContentType returns the appropriate content type for file extensions
func (h *Handler) getContentType(ext string) string {
	switch ext {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}

// getCacheControl returns appropriate cache control headers based on file type
func (h *Handler) getCacheControl(ext string) string {
	switch ext {
	case ".css", ".js", ".woff", ".woff2", ".ttf", ".eot":
		// Assets like CSS, JS, and fonts are fingerprinted and can be cached "forever".
		return "public, max-age=31536000" // 1 year
	case ".png", ".jpg", ".jpeg", ".gif", ".svg":
		// Images can be cached for a long time, but not indefinitely.
		return "public, max-age=2592000" // 30 days
	case ".ico":
		// Favicon can be cached for a day.
		return "public, max-age=86400" // 24 hours
	default:
		// Other files (like JSON manifests, etc.) - cache for 1 hour.
		return "public, max-age=3600"
	}
}

// handleConditionalRequest handles If-None-Match and If-Modified-Since headers
func (h *Handler) handleConditionalRequest(w http.ResponseWriter, r *http.Request, fi os.FileInfo) bool {
	modTime := fi.ModTime()
	etag := h.generateETag(fi)

	// Set ETag and Last-Modified headers
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))

	// Check If-None-Match (ETag)
	if inm := r.Header.Get("If-None-Match"); inm != "" {
		if inm == etag || inm == "*" {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	// Check If-Modified-Since
	if ims := r.Header.Get("If-Modified-Since"); ims != "" {
		if t, err := http.ParseTime(ims); err == nil {
			// Compare with 1-second precision (HTTP time format limitation)
			if modTime.Unix() <= t.Unix() {
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}

	return false
}

// setStaticFileHeaders sets appropriate headers for static files
func (h *Handler) setStaticFileHeaders(w http.ResponseWriter, target string) {
	// Set content type based on file extension
	ext := strings.ToLower(filepath.Ext(target))
	contentType := h.getContentType(ext)
	w.Header().Set("Content-Type", contentType)

	// Set caching headers based on file type
	cacheControl := h.getCacheControl(ext)
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("Vary", "Accept-Encoding")

	// Set security headers for static files
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// isValidStaticPath validates the relative path to prevent directory traversal
func (h *Handler) isValidStaticPath(rel string) bool {
	// Clean the path and normalize path separators
	rel = filepath.Clean(rel)
	rel = filepath.ToSlash(rel) // Convert Windows backslashes to forward slashes

	// Reject empty, current directory, or paths ending with slash
	if rel == "." || rel == "" || strings.HasSuffix(rel, "/") {
		return false
	}

	// Reject paths that try to go up directories
	if strings.Contains(rel, "..") {
		return false
	}

	// Reject paths starting with slash
	if strings.HasPrefix(rel, "/") {
		return false
	}

	return true
}

// isPathSafe ensures the resolved path is within the static directory
func (h *Handler) isPathSafe(staticDir, target string) bool {
	absStatic, err := filepath.Abs(staticDir)
	if err != nil {
		return false
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}

	// Ensure target is within static directory
	staticPrefix := absStatic + string(filepath.Separator)
	return strings.HasPrefix(absTarget, staticPrefix) || absTarget == absStatic
}

// serveFavicon handles favicon.ico requests with appropriate caching
func (h *Handler) serveFavicon(w http.ResponseWriter, r *http.Request, staticDir string) {
	faviconPath := filepath.Join(staticDir, "favicon.ico")
	fi, err := os.Stat(faviconPath)
	if err != nil || fi.IsDir() {
		h.Error(w, r, http.StatusNotFound, "Favicon not found")
		return
	}

	// Handle conditional requests for favicon
	if h.handleConditionalRequest(w, r, fi) {
		return
	}

	// Set favicon-specific headers
	w.Header().Set("Cache-Control", "public, max-age=86400") // 24 hours for favicon
	w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	http.ServeFile(w, r, faviconPath)
}

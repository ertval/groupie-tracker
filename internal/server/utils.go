package server

import (
	"bytes"
	"fmt"
	"groupie-tracker/internal/config"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// --- Request Validation Helpers ---

// validateRequestGETPath checks if request is GET method and matches expected path.
func (a *App) validateRequestGETPath(w http.ResponseWriter, r *http.Request, expectedPath string) bool {
	if r.Method != http.MethodGet {
		a.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}

	if r.URL.Path != expectedPath {
		a.Error(w, r, http.StatusNotFound, "Page not found")
		return false
	}

	return true
}

// --- Template Helpers ---

// render renders a template with the given data and status code (if provided) managing all errors.
func (a *App) render(w http.ResponseWriter, r *http.Request, name string, data any, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}

	tmpl, ok := a.templates[name]
	if !ok {
		// If we are already rendering an error, don't call h.Error again.
		if name == "error.tmpl" {
			log.Printf("FATAL: error.tmpl is missing")
			http.Error(w, "500 Internal Server Error - Error template not found", http.StatusInternalServerError)
			return
		}
		a.Error(w, r, http.StatusInternalServerError, fmt.Sprintf("Template %s not found", name))
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
		a.Error(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	buf.WriteTo(w)
}

// loadTemplates loads and parses all templates from the templates directory.
func (a *App) loadTemplates() {
	a.templates = make(map[string]*template.Template)

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

		a.templates[name] = ts
	}
}

// getPort returns the port to run the server on.
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return config.DefaultPort
	}

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return port
}

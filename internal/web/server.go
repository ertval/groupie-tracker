package web

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
)

// Server encapsulates HTTP server dependencies.
type Server struct {
	store     *data.Store
	templates map[string]*template.Template
	server    *http.Server
}

// NewServer creates and initializes a new web server.
func NewServer(store *data.Store) (*Server, error) {
	s := &Server{
		store: store,
	}

	// Load templates
	if err := s.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Setup HTTP server
	s.server = &http.Server{
		Addr:         config.DefaultPort,
		Handler:      s.routes(),
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return s, nil
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	log.Printf("Server starting on %s", config.DefaultPort)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Handler returns the HTTP handler for testing purposes.
func (s *Server) Handler() http.Handler {
	return s.routes()
}

// loadTemplates loads and compiles all templates.
func (s *Server) loadTemplates() error {
	s.templates = make(map[string]*template.Template)

	// Define template functions
	funcMap := template.FuncMap{
		"join":       strings.Join,
		"pluralize":  s.pluralize,
		"formatYear": s.formatYear,
		"contains":   s.contains,
		"ne":         s.notEqual,
		"sub":        s.subtract,
		"add":        s.add,
		"lt":         s.lessThan,
		"title":      s.titleCase,
	}

	// Template files to load
	templateFiles := []string{
		"home.tmpl",
		"artists.tmpl",
		"artist_detail.tmpl",
		"locations.tmpl",
		"location_detail.tmpl",
		"search.tmpl",
		"error.tmpl",
	}

	for _, file := range templateFiles {
		tmpl, err := template.New("base").Funcs(funcMap).ParseFiles(
			"templates/base.tmpl",
			"templates/"+file,
		)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", file, err)
		}
		s.templates[file] = tmpl
	}

	return nil
}

// Helper functions for templates
func (s *Server) pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func (s *Server) formatYear(year int) string {
	if year == 0 {
		return "Unknown"
	}
	return fmt.Sprintf("%d", year)
}

func (s *Server) contains(slice interface{}, item interface{}) bool {
	switch s := slice.(type) {
	case []int:
		if i, ok := item.(int); ok {
			for _, v := range s {
				if v == i {
					return true
				}
			}
		}
	case []string:
		if str, ok := item.(string); ok {
			for _, v := range s {
				if v == str {
					return true
				}
			}
		}
	}
	return false
}

func (s *Server) notEqual(a, b interface{}) bool {
	return a != b
}

func (s *Server) subtract(a, b int) int {
	return a - b
}

func (s *Server) add(a, b int) int {
	return a + b
}

func (s *Server) lessThan(a, b int) bool {
	return a < b
}

func (s *Server) titleCase(str string) string {
	words := strings.Fields(strings.ReplaceAll(str, "-", " "))
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

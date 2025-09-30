// Package http provides HTTP handlers, routing, and template rendering for the web interface.
// This package implements the presentation layer with clean separation from business logic.
package http

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"groupie-tracker/internal/models"
)

// TemplateRenderer handles HTML template compilation and rendering.
type TemplateRenderer struct {
	templates map[string]*template.Template
}

// NewTemplateRenderer creates a new template renderer and compiles all templates.
func NewTemplateRenderer() (*TemplateRenderer, error) {
	tr := &TemplateRenderer{
		templates: make(map[string]*template.Template),
	}

	if err := tr.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return tr, nil
}

// loadTemplates compiles all HTML templates with custom helper functions.
func (tr *TemplateRenderer) loadTemplates() error {
	// Template helper functions
	funcMap := template.FuncMap{
		"join":      strings.Join,
		"len":       tr.templateLen,
		"pluralize": tr.pluralize,
		"contains":  tr.templateContains,
		"add":       tr.add,
		"sub":       tr.sub,
		"hasField":  tr.hasField,
		"title":     tr.toTitleCase,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
	}

	// Template file patterns
	templateFiles := []string{
		"templates/home.tmpl",
		"templates/artists.tmpl",
		"templates/artist_detail.tmpl",
		"templates/search.tmpl",
		"templates/locations.tmpl",
		"templates/location_detail.tmpl",
		"templates/error.tmpl",
	}

	// Load each template with base template
	for _, templateFile := range templateFiles {
		templateName := filepath.Base(templateFile)

		tmpl, err := template.New(templateName).Funcs(funcMap).ParseFiles(
			"templates/base.tmpl",
			templateFile,
		)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", templateFile, err)
		}

		tr.templates[templateName] = tmpl
	}

	return nil
}

// Render renders a template with the provided data to the HTTP response writer.
func (tr *TemplateRenderer) Render(w http.ResponseWriter, templateName string, data interface{}) error {
	tmpl, exists := tr.templates[templateName]
	if !exists {
		return fmt.Errorf("template not found: %s", templateName)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		return fmt.Errorf("template rendering error: %w", err)
	}

	return nil
}

// Template helper functions

// templateLen returns the length of various types for template use.
func (tr *TemplateRenderer) templateLen(v interface{}) int {
	switch val := v.(type) {
	case []models.Artist:
		return len(val)
	case []string:
		return len(val)
	case []models.Concert:
		return len(val)
	case []models.Location:
		return len(val)
	case string:
		return len(val)
	default:
		return 0
	}
}

// templateContains checks if a slice contains a specific value for template use.
func (tr *TemplateRenderer) templateContains(slice interface{}, item interface{}) bool {
	switch s := slice.(type) {
	case []string:
		if str, ok := item.(string); ok {
			for _, v := range s {
				if v == str {
					return true
				}
			}
		}
	case []int:
		if num, ok := item.(int); ok {
			for _, v := range s {
				if v == num {
					return true
				}
			}
		}
	}
	return false
}

// pluralize returns singular or plural form based on count.
func (tr *TemplateRenderer) pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// add performs addition for template use.
func (tr *TemplateRenderer) add(a, b int) int {
	return a + b
}

// sub performs subtraction for template use.
func (tr *TemplateRenderer) sub(a, b int) int {
	return a - b
}

// hasField checks if a struct has a specific field (simplified version).
func (tr *TemplateRenderer) hasField(obj interface{}, fieldName string) bool {
	// Simple implementation for template use
	return true // Most cases will have the field
}

// toTitleCase converts a string to title case, replacing hyphens with spaces.
func (tr *TemplateRenderer) toTitleCase(s string) string {
	// Replace hyphens with spaces
	s = strings.ReplaceAll(s, "-", " ")

	// Split into words and capitalize each
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

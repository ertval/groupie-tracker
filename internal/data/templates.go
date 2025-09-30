package data

import (
	"fmt"
	"html/template"
	"path/filepath"
	"slices"
	"strings"
)

// --- Template Helper Functions ---

// toTitleCase converts a string to title case, replacing hyphens with spaces.
func toTitleCase(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// getTemplateFuncMap returns the custom template functions available in all templates.
func getTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"hasField": func(obj any, field string) bool {
			// Simplified version - for basic template functionality
			return true
		},
		"contains": func(slice []string, item string) bool {
			return slices.Contains(slice, item)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"join": func(slice []string, sep string) string {
			return strings.Join(slice, sep)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"title": func(s string) string {
			return toTitleCase(s)
		},
		"formatPlural": func(count int, singular, plural string) string {
			if count == 1 {
				return fmt.Sprintf("%d %s", count, singular)
			}
			return fmt.Sprintf("%d %s", count, plural)
		},
	}
}

// loadAllTemplates compiles all HTML templates with template inheritance.
func loadAllTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)

	// Template files to compile
	templateFiles := []string{
		"base.tmpl", "home.tmpl", "artists.tmpl", "artist_detail.tmpl",
		"locations.tmpl", "location_detail.tmpl", "search.tmpl",
		"error.tmpl", "dev.tmpl",
	}

	funcMap := getTemplateFuncMap()

	for _, filename := range templateFiles {
		tmpl := template.New(filename).Funcs(funcMap)
		tmpl = template.Must(tmpl.ParseFiles(
			filepath.Join("templates", "base.tmpl"),
			filepath.Join("templates", filename),
		))
		templates[filename] = tmpl
	}

	return templates
}

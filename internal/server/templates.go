package server

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// TemplateManager manages parsed templates and exposes a safe render helper.
type TemplateManager struct {
	templates map[string]*template.Template
}

// LoadTemplates parses the base template plus all page templates into a manager instance.
func LoadTemplates(patterns ...string) (*TemplateManager, error) {
	root, err := locateProjectRoot()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(root, "templates")
	baseTemplate := filepath.Join(baseDir, "base.tmpl")

	if len(patterns) == 0 {
		patterns = []string{
			"home.tmpl",
			"artists.tmpl",
			"artist_detail.tmpl",
			"search.tmpl",
			"locations.tmpl",
			"location_detail.tmpl",
			"error.tmpl",
		}
	}

	funcMap := template.FuncMap{
		"join":      strings.Join,
		"len":       templateLen,
		"pluralize": pluralize,
		"contains":  templateContains,
		"add":       func(a, b int) int { return a + b },
		"sub":       func(a, b int) int { return a - b },
		"title":     toTitleCase,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
	}

	cache := make(map[string]*template.Template, len(patterns))
	for _, page := range patterns {
		name := filepath.Base(page)
		pagePath := filepath.Join(baseDir, name)
		parsed, err := template.New(name).Funcs(funcMap).ParseFiles(baseTemplate, pagePath)
		if err != nil {
			return nil, fmt.Errorf("templates: parse %s: %w", pagePath, err)
		}
		cache[name] = parsed
	}

	return &TemplateManager{templates: cache}, nil
}

// Render executes a template using the shared base layout.
func (m *TemplateManager) Render(w http.ResponseWriter, templateName string, data any) {
	tmpl, ok := m.templates[templateName]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "template rendering error", http.StatusInternalServerError)
	}
}

func templateLen(v any) int {
	switch value := v.(type) {
	case []string:
		return len(value)
	case []int:
		return len(value)
	case []interface{}:
		return len(value)
	case string:
		return len(value)
	default:
		return 0
	}
}

func templateContains(collection any, item any) bool {
	switch typed := collection.(type) {
	case []string:
		other, ok := item.(string)
		if !ok {
			return false
		}
		for _, candidate := range typed {
			if candidate == other {
				return true
			}
		}
	case []int:
		n, ok := item.(int)
		if !ok {
			return false
		}
		for _, candidate := range typed {
			if candidate == n {
				return true
			}
		}
	}
	return false
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func toTitleCase(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}
	return strings.Join(words, " ")
}

func locateProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("templates: could not locate project root from working directory")
}

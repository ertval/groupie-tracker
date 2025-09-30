package web

import (
	"log"
	"net/http"
)

// render executes a template with the given data.
func (s *Server) render(w http.ResponseWriter, templateName string, data interface{}) {
	tmpl, exists := s.templates[templateName]
	if !exists {
		log.Printf("Template %s not found", templateName)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("Failed to execute template %s: %v", templateName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

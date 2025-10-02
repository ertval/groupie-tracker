package web

import (
	"fmt"
	"net/http"
	"time"

	"groupie-tracker/internal/data"
)

// Error handles all errors (4xx and 5xx) in a centralized way.
func (s *Server) Error(w http.ResponseWriter, r *http.Request, status int, message string) {
	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		Suggestions  []data.SearchSuggestion
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		ExtraCSS:     "errors.css",
		ExtraJS:      "",
		Suggestions:  nil, // Error pages don't need search suggestions
		ErrorCode:    status,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	s.render(w, r, "error.tmpl", data, status)
}

// NotFoundError sends a 404 error response.
func (s *Server) NotFoundError(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "Page not found"
	}
	s.Error(w, r, http.StatusNotFound, message)
}

// BadRequestError sends a standardized 400 error response.
func (s *Server) BadRequestError(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "Bad request"
	}
	s.Error(w, r, http.StatusBadRequest, message)
}

// validateExactPath checks if request path matches expected path.
func (s *Server) validateExactPath(w http.ResponseWriter, r *http.Request, expectedPath string) bool {
	if r.URL.Path != expectedPath {
		s.NotFoundError(w, r, "")
		return false
	}
	return true
}

// parseFormOrError parses form data and handles errors.
func (s *Server) parseFormOrError(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		s.BadRequestError(w, r, "Failed to parse form data")
		return false
	}
	return true
}

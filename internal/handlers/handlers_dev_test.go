package handlers

import (
	"context"
	"groupie-tracker/internal/repository"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupHandler(t *testing.T) *Handler {
	repo := repository.NewRepository("https://groupietrackers.herokuapp.com", 10*time.Second)
	if err := repo.LoadData(context.Background()); err != nil {
		t.Fatalf("failed to load data for tests: %v", err)
	}

	// Create minimal in-memory templates to avoid filesystem dependencies.
	tplText := `{{define "error.tmpl"}}<html><body><h1>{{.Title}}</h1><p>{{.Message}}</p></body></html>{{end}}` +
		`{{define "artists.tmpl"}}<html><body><h1>Artists</h1></body></html>{{end}}` +
		`{{define "home.tmpl"}}<html><body><h1>Home</h1></body></html>{{end}}`

	tpl := template.New("")
	if _, err := tpl.Parse(strings.TrimSpace(tplText)); err != nil {
		t.Fatalf("failed to parse in-memory templates: %v", err)
	}

	return &Handler{repo: repo, templates: tpl}
}

func TestDev404(t *testing.T) {
	h := setupHandler(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dev/404", nil)
	h.Dev404(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if body := rr.Body.String(); body == "" {
		t.Fatalf("expected non-empty body for 404")
	}
}

func TestDev500(t *testing.T) {
	h := setupHandler(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dev/500", nil)
	h.Dev500(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestDevTemplateError(t *testing.T) {
	h := setupHandler(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dev/template-error", nil)
	h.DevTemplateError(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if got := rr.Body.String(); got == "" {
		t.Fatalf("expected non-empty body for template error")
	}
}

func TestDevPanicRecoveredByMiddleware(t *testing.T) {
	h := setupHandler(t)

	// Wrap the DevPanic handler with the recovery middleware used in server.go
	recovered := withRecoveryLocal(http.HandlerFunc(h.DevPanic))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dev/panic", nil)

	recovered.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected recovered panic to produce 500, got %d", rr.Code)
	}
}

// withRecoveryLocal mirrors the panic recovery middleware from the server
// to allow unit testing without importing the server package.
func withRecoveryLocal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Return a simple 500 response like server middleware
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 Internal Server Error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

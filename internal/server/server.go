package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/search"
)

// Server wires the data store, search service and template system into an HTTP handler.
type Server struct {
	store     *data.Store
	search    *search.Service
	templates *TemplateManager
	cfg       config.Config
	router    *http.ServeMux
	handler   http.Handler
}

// New constructs a Server instance ready to serve requests.
func New(store *data.Store, searchService *search.Service, cfg config.Config) (*Server, error) {
	if store == nil {
		return nil, fmt.Errorf("server: store is required")
	}

	if searchService == nil {
		searchService = search.NewService(store)
	}

	templateManager, err := LoadTemplates()
	if err != nil {
		return nil, err
	}

	if cfg.Port == "" {
		cfg.Port = ":8082"
	}

	mux := http.NewServeMux()

	s := &Server{
		store:     store,
		search:    searchService,
		templates: templateManager,
		cfg:       cfg,
		router:    mux,
	}

	s.registerRoutes(mux)
	s.handler = s.applyMiddleware(mux)

	return s, nil
}

// Handler exposes the fully wrapped HTTP handler for integration tests.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// ServeHTTP allows Server to satisfy the http.Handler interface directly.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// HTTPServer assembles a configured *http.Server for the CLI entrypoint to run.
func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:         s.cfg.Port,
		Handler:      s.Handler(),
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		IdleTimeout:  s.cfg.IdleTimeout,
	}
}

// ListenAndServe starts the HTTP server using the constructed configuration.
func (s *Server) ListenAndServe() error {
	httpServer := s.HTTPServer()
	log.Printf("Server starting on %s", httpServer.Addr)
	return httpServer.ListenAndServe()
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/favicon.ico", s.handleFavicon)

	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/artists", s.handleArtists)
	mux.HandleFunc("/artists/", s.handleArtistDetail)
	mux.HandleFunc("/locations", s.handleLocations)
	mux.HandleFunc("/locations/", s.handleLocationDetail)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/api/suggestions", s.handleSuggestions)
	mux.HandleFunc("/health", s.handleHealth)
}

func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	faviconPath := filepath.Join("static", "favicon.ico")
	if _, err := os.Stat(faviconPath); err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, faviconPath)
}

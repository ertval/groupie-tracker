package web

import (
	"log"
	"net/http"
)

// routes configures and returns the HTTP router.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	
	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	
	// Page routes
	mux.HandleFunc("/", s.Home)
	mux.HandleFunc("/artists", s.Artists)
	mux.HandleFunc("/artist/", s.ArtistDetail)
	mux.HandleFunc("/locations", s.Locations)
	mux.HandleFunc("/location/", s.LocationDetail)
	mux.HandleFunc("/search", s.Search)
	mux.HandleFunc("/health", s.Health)
	
	// Apply middleware
	return s.withMiddleware(mux)
}

// withMiddleware applies common middleware to the handler.
func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return s.loggingMiddleware(s.corsMiddleware(next))
}

// loggingMiddleware logs HTTP requests.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}
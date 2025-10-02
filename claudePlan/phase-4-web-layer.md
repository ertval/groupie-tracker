# Phase 4: Web Layer & Template Simplification

## Overview
This phase focuses on cleaning up the web layer by creating shared view models, slimming down handlers, clarifying middleware, and optimizing template handling for better maintainability and performance.

## Step 4.1: Create Shared View Models
**Goal:** Eliminate repetitive anonymous structs.

### Sub-steps:
1. **Create view package:**
   ```bash
   mkdir internal/view
   touch internal/view/models.go
   ```

2. **Define base Page struct:**
   ```go
   package view

   type Page struct {
       Title       string
       Description string
       Data        interface{}
       Assets      Assets
       Error       *ErrorInfo
   }

   type Assets struct {
       CSS []string
       JS  []string
   }

   type ErrorInfo struct {
       Code    int
       Message string
   }
   ```

3. **Define specific page types:**
   ```go
   type HomePage struct {
       Page
       Stats      SiteStats
       Featured   []Artist
   }

   type ArtistListPage struct {
       Page
       Artists       []Artist
       FilterOptions ArtistFilterOptions
       ActiveFilters ArtistFilters
   }

   type ArtistDetailPage struct {
       Page
       Artist    Artist
       Prev      *Artist
       Next      *Artist
       Locations []Location
   }

   // ... etc
   ```

4. **Create view builder functions:**
   ```go
   func NewHomePage(store *data.Store) HomePage
   func NewArtistListPage(artists []Artist, options ArtistFilterOptions, filters ArtistFilters) ArtistListPage
   func NewArtistDetailPage(artist Artist, store *data.Store) ArtistDetailPage
   ```

5. **Update handlers to use view models:**
   ```go
   func (h *Handlers) handleHome(w http.ResponseWriter, r *http.Request) {
       page := view.NewHomePage(h.store)
       h.render(w, "home.tmpl", page)
   }
   ```

6. **Run tests:** `go test ./internal/web/...`

## Step 4.2: Slim Down Handlers
**Goal:** Push business logic to data layer.

### Sub-steps:
1. **Create reusable request helpers:**
   ```go
   // In internal/web/helpers.go

   func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
       if r.Method != method {
           w.Header().Set("Allow", method)
           http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
           return false
       }
       return true
   }

   func parseFilters(r *http.Request) (data.ArtistFilters, error) {
       // Consolidated filter parsing
   }

   func respondJSON(w http.ResponseWriter, data interface{}) error {
       w.Header().Set("Content-Type", "application/json")
       return json.NewEncoder(w).Encode(data)
   }

   func respondError(w http.ResponseWriter, code int, message string) {
       // Standardized error response
   }
   ```

2. **Move sorting logic to Store:**
   ```go
   // In internal/data/store.go

   func (s *Store) ArtistsSortedBy(field string, ascending bool) []Artist {
       artists := s.catalog.AllArtists()
       // Sorting logic
       return artists
   }
   ```

3. **Refactor handler pattern:**
   ```go
   func (h *Handlers) handleArtists(w http.ResponseWriter, r *http.Request) {
       if !requireMethod(w, r, http.MethodGet) {
           return
       }

       filters, err := parseFilters(r)
       if err != nil {
           respondError(w, http.StatusBadRequest, err.Error())
           return
       }

       artists := h.store.FilterArtists(filters)
       options := h.store.GetArtistFilterOptions()

       page := view.NewArtistListPage(artists, options, filters)
       h.render(w, "artists.tmpl", page)
   }
   ```

4. **Update all handlers:**
   - Apply consistent pattern
   - Remove embedded business logic
   - Use helper functions

5. **Run tests:** `go test ./internal/web/...`

## Step 4.3: Clarify Middleware
**Goal:** Clean and reusable middleware chain.

### Sub-steps:
1. **Create method restriction middleware:**
   ```go
   func methodOnly(method string) func(http.Handler) http.Handler {
       return func(next http.Handler) http.Handler {
           return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
               if r.Method != method {
                   w.Header().Set("Allow", method)
                   http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
                   return
               }
               next.ServeHTTP(w, r)
           })
       }
   }
   ```

2. **Review existing middleware:**
   - Logging middleware
   - Recovery middleware
   - Security headers middleware

3. **Establish clear middleware ordering:**
   ```go
   func (h *Handlers) setupMiddleware() http.Handler {
       mux := http.NewServeMux()
       // Register routes

       // Apply middleware in order
       var handler http.Handler = mux
       handler = h.logging(handler)
       handler = h.recovery(handler)
       handler = h.securityHeaders(handler)

       return handler
   }
   ```

4. **Document middleware purpose:**
   - Add clear comments
   - Document ordering requirements

5. **Run tests:** `go test ./internal/web/...`

## Step 4.4: Optimize Template Handling
**Goal:** Compile once, render efficiently.

### Sub-steps:
1. **Update template compilation:**
   ```go
   type Templates struct {
       templates map[string]*template.Template
       funcMap   template.FuncMap
   }

   func LoadTemplates(dir string) (*Templates, error) {
       t := &Templates{
           templates: make(map[string]*template.Template),
           funcMap:   makeFuncMap(),
       }

       // Parse base template
       base := template.Must(template.New("base").Funcs(t.funcMap).ParseFiles(
           filepath.Join(dir, "base.tmpl"),
       ))

       // Parse all page templates
       files, err := filepath.Glob(filepath.Join(dir, "*.tmpl"))
       if err != nil {
           return nil, err
       }

       for _, file := range files {
           name := filepath.Base(file)
           if name == "base.tmpl" {
               continue
           }

           tmpl, err := base.Clone()
           if err != nil {
               return nil, err
           }

           tmpl, err = tmpl.ParseFiles(file)
           if err != nil {
               return nil, err
           }

           t.templates[name] = tmpl
       }

       return t, nil
   }
   ```

2. **Update render helper:**
   ```go
   func (h *Handlers) render(w http.ResponseWriter, name string, data interface{}) {
       tmpl, ok := h.templates.templates[name]
       if !ok {
           log.Printf("Template not found: %s", name)
           http.Error(w, "Internal Server Error", http.StatusInternalServerError)
           return
       }

       w.Header().Set("Content-Type", "text/html; charset=utf-8")

       if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
           log.Printf("Template execution error: %v", err)
           http.Error(w, "Internal Server Error", http.StatusInternalServerError)
       }
   }
   ```

3. **Add template functions:**
   ```go
   func makeFuncMap() template.FuncMap {
       return template.FuncMap{
           "add":      func(a, b int) int { return a + b },
           "sub":      func(a, b int) int { return a - b },
           "join":     strings.Join,
           "lower":    strings.ToLower,
           "title":    strings.Title,
           "slugify":  slugify,
           "contains": strings.Contains,
       }
   }
   ```

4. **Initialize templates once:**
   ```go
   func main() {
       templates, err := LoadTemplates("templates")
       if err != nil {
           log.Fatal(err)
       }

       handlers := web.NewHandlers(store, templates)
       // ...
   }
   ```

5. **Run tests:** `go test ./internal/web/...`
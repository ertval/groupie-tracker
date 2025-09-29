#!/bin/bash
# Script to systematically convert handler functions to Server methods

cd "d:\\Ertval One\\_Software\\zone-modules\\Modules\\groupie-tracker"

# Functions that need to be converted (excluding already converted ones)
functions=("Search" "Locations" "LocationDetail" "DevIndex" "Health" "SuggestionsAPI" "DevPanic" "Dev404" "Dev500" "Dev500Tmpl" "StaticFiles")

for func in "${functions[@]}"; do
    echo "Converting $func..."
    
    # Use sed to replace function signature
    sed -i "s/^func $func(/func (s *Server) $func(/g" internal/server/handlers.go
    
    # Replace repo calls with s.repo
    sed -i "s/repo\./${s.repo.}/g" internal/server/handlers.go
    
    # Replace Error calls with s.Error
    sed -i "s/Error(w, r,/s.Error(w, r,/g" internal/server/handlers.go
    
    # Replace render calls with s.render 
    sed -i "s/render(w, r,/s.render(w, r,/g" internal/server/handlers.go
    
    # Replace NewBaseTemplateData with s.NewBaseTemplateData
    sed -i "s/NewBaseTemplateData(/s.NewBaseTemplateData(/g" internal/server/handlers.go
done

echo "Conversion completed!"
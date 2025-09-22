// Filter functionality for artists page
class ArtistFilter {
    constructor() {
        this.form = document.getElementById('filter-form');
        this.artistsGrid = document.getElementById('artists-grid');
        this.noResults = document.getElementById('no-results');
        this.loading = document.getElementById('loading');
        this.clearButton = document.getElementById('clear-filters');
        this.resetButton = document.getElementById('reset-filters');
        
        this.originalArtists = Array.from(this.artistsGrid.children);
        this.isFiltering = false;
        
        this.init();
    }
    
    init() {
        if (this.form) {
            this.form.addEventListener('submit', this.handleFormSubmit.bind(this));
        }
        
        if (this.clearButton) {
            this.clearButton.addEventListener('click', this.clearAllFilters.bind(this));
        }
        
        if (this.resetButton) {
            this.resetButton.addEventListener('click', this.resetFilters.bind(this));
        }
        
        // Add real-time filtering on input change (debounced)
        this.addRealTimeFiltering();
    }
    
    addRealTimeFiltering() {
        if (!this.form) return;
        
        const inputs = this.form.querySelectorAll('input');
        let debounceTimer;
        
        inputs.forEach(input => {
            input.addEventListener('input', () => {
                clearTimeout(debounceTimer);
                debounceTimer = setTimeout(() => {
                    this.filterArtists();
                }, 300);
            });
            
            if (input.type === 'checkbox') {
                input.addEventListener('change', () => {
                    this.filterArtists();
                });
            }
        });
    }
    
    handleFormSubmit(e) {
        e.preventDefault();
        this.filterArtists();
    }
    
    async filterArtists() {
        if (this.isFiltering) return;
        
        this.isFiltering = true;
        this.showLoading(true);
        
        try {
            const formData = new FormData(this.form);
            const filterParams = this.buildFilterParams(formData);
            
            // Use client-side filtering for better performance
            this.clientSideFilter(filterParams);
            
        } catch (error) {
            console.error('Filter error:', error);
            this.showError('Failed to filter artists. Please try again.');
        } finally {
            this.isFiltering = false;
            this.showLoading(false);
        }
    }
    
    buildFilterParams(formData) {
        const params = {};
        
        // Creation year range
        const creationYearFrom = formData.get('creationYearFrom');
        const creationYearTo = formData.get('creationYearTo');
        if (creationYearFrom || creationYearTo) {
            params.creationYear = {
                from: creationYearFrom ? parseInt(creationYearFrom) : null,
                to: creationYearTo ? parseInt(creationYearTo) : null
            };
        }
        
        // First album date range
        const firstAlbumFrom = formData.get('firstAlbumFrom');
        const firstAlbumTo = formData.get('firstAlbumTo');
        if (firstAlbumFrom || firstAlbumTo) {
            params.firstAlbum = {
                from: firstAlbumFrom || null,
                to: firstAlbumTo || null
            };
        }
        
        // Member count range
        const membersFrom = formData.get('membersFrom');
        const membersTo = formData.get('membersTo');
        if (membersFrom || membersTo) {
            params.members = {
                from: membersFrom ? parseInt(membersFrom) : null,
                to: membersTo ? parseInt(membersTo) : null
            };
        }
        
        // Selected locations
        const selectedLocations = formData.getAll('locations');
        if (selectedLocations.length > 0) {
            params.locations = selectedLocations;
        }
        
        return params;
    }
    
    clientSideFilter(filterParams) {
        let visibleCount = 0;
        
        this.originalArtists.forEach(artistCard => {
            const matches = this.artistMatchesFilters(artistCard, filterParams);
            
            if (matches) {
                artistCard.style.display = 'block';
                visibleCount++;
            } else {
                artistCard.style.display = 'none';
            }
        });
        
        // Show/hide no results message
        this.noResults.style.display = visibleCount === 0 ? 'block' : 'none';
        
        // Update results count if there's a counter
        this.updateResultsCount(visibleCount);
    }
    
    artistMatchesFilters(artistCard, filterParams) {
        // Get artist data from data attributes
        const creationYear = parseInt(artistCard.dataset.year);
        const memberCount = parseInt(artistCard.dataset.members);
        const firstAlbum = artistCard.dataset.firstAlbum;
        const locations = artistCard.dataset.locations || '';
        
        // Check creation year filter
        if (filterParams.creationYear) {
            const { from, to } = filterParams.creationYear;
            if (from !== null && creationYear < from) return false;
            if (to !== null && creationYear > to) return false;
        }
        
        // Check member count filter
        if (filterParams.members) {
            const { from, to } = filterParams.members;
            if (from !== null && memberCount < from) return false;
            if (to !== null && memberCount > to) return false;
        }
        
        // Check first album date filter
        if (filterParams.firstAlbum) {
            const { from, to } = filterParams.firstAlbum;
            const albumYear = this.extractYearFromDate(firstAlbum);
            
            if (from && albumYear) {
                const fromYear = this.extractYearFromDate(from);
                if (fromYear && albumYear < fromYear) return false;
            }
            
            if (to && albumYear) {
                const toYear = this.extractYearFromDate(to);
                if (toYear && albumYear > toYear) return false;
            }
        }
        
        // Check location filter
        if (filterParams.locations && filterParams.locations.length > 0) {
            const hasMatchingLocation = filterParams.locations.some(location => 
                locations.toLowerCase().includes(location.toLowerCase())
            );
            if (!hasMatchingLocation) return false;
        }
        
        return true;
    }
    
    extractYearFromDate(dateString) {
        if (!dateString) return null;
        
        // Try different date formats
        const formats = [
            /(\d{4})/,                    // Just year: 1995
            /\d{2}-\d{2}-(\d{4})/,       // DD-MM-YYYY: 26-03-2001
            /\d{2}\/\d{2}\/(\d{4})/,     // DD/MM/YYYY: 26/03/2001
            /(\d{4})-\d{2}-\d{2}/,       // YYYY-MM-DD: 2001-03-26
        ];
        
        for (const format of formats) {
            const match = dateString.match(format);
            if (match) {
                return parseInt(match[1]);
            }
        }
        
        return null;
    }
    
    clearAllFilters() {
        if (!this.form) return;
        
        // Clear all form inputs
        const inputs = this.form.querySelectorAll('input');
        inputs.forEach(input => {
            if (input.type === 'checkbox') {
                input.checked = false;
            } else {
                input.value = '';
            }
        });
        
        // Reset display
        this.resetDisplay();
    }
    
    resetFilters() {
        this.clearAllFilters();
    }
    
    resetDisplay() {
        // Show all artists
        this.originalArtists.forEach(artistCard => {
            artistCard.style.display = 'block';
        });
        
        // Hide no results message
        this.noResults.style.display = 'none';
        
        // Update results count
        this.updateResultsCount(this.originalArtists.length);
    }
    
    updateResultsCount(count) {
        // Update any results counter if it exists
        const counter = document.querySelector('.results-count');
        if (counter) {
            const total = this.originalArtists.length;
            counter.textContent = count === total ? 
                `Showing all ${total} artists` : 
                `Showing ${count} of ${total} artists`;
        }
    }
    
    showLoading(show) {
        if (this.loading) {
            this.loading.style.display = show ? 'block' : 'none';
        }
    }
    
    showError(message) {
        // Simple error display - could be enhanced with a proper error component
        const errorDiv = document.createElement('div');
        errorDiv.className = 'filter-error';
        errorDiv.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: #ef4444;
            color: white;
            padding: 1rem;
            border-radius: 8px;
            z-index: 1000;
            box-shadow: 0 4px 12px rgba(0,0,0,0.2);
        `;
        errorDiv.textContent = message;
        
        document.body.appendChild(errorDiv);
        
        // Auto remove after 5 seconds
        setTimeout(() => {
            if (errorDiv.parentNode) {
                errorDiv.parentNode.removeChild(errorDiv);
            }
        }, 5000);
    }
}

// Initialize filter functionality when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    // Initialize filter toggle (already exists in template)
    const toggleButton = document.getElementById('toggle-filters');
    const filtersPanel = document.getElementById('filters-panel');
    
    if (toggleButton && filtersPanel) {
        toggleButton.addEventListener('click', function() {
            const isExpanded = toggleButton.getAttribute('aria-expanded') === 'true';
            const newExpanded = !isExpanded;
            
            toggleButton.setAttribute('aria-expanded', newExpanded);
            filtersPanel.style.display = newExpanded ? 'block' : 'none';
            toggleButton.querySelector('.filter-text').textContent = 
                newExpanded ? 'Hide Filters' : 'Show Filters';
        });
    }
    
    // Initialize artist filter
    if (document.getElementById('filter-form')) {
        new ArtistFilter();
    }
});
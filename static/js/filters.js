// Filter functionality for artists page
class ArtistFilter {
    constructor() {
        this.form = document.getElementById('filter-form');
        this.artistsGrid = document.getElementById('artists-grid');
        this.noResults = document.getElementById('no-results');
        this.loading = document.getElementById('loading');
        this.clearButton = document.getElementById('clear-filters');
        this.resetButton = document.getElementById('reset-filters');
        
        // Range sliders
        this.creationYearFromSlider = document.getElementById('creation-year-from');
        this.creationYearToSlider = document.getElementById('creation-year-to');
        this.firstAlbumYearFromSlider = document.getElementById('first-album-year-from');
        this.firstAlbumYearToSlider = document.getElementById('first-album-year-to');
        
        // Value display elements
        this.creationYearFromValue = document.getElementById('creation-year-from-value');
        this.creationYearToValue = document.getElementById('creation-year-to-value');
        this.firstAlbumYearFromValue = document.getElementById('first-album-year-from-value');
        this.firstAlbumYearToValue = document.getElementById('first-album-year-to-value');
        
        this.originalArtists = Array.from(this.artistsGrid.children);
        this.isFiltering = false;
        
        this.init();
    }
    
    init() {
        this.initRangeSliders();
        this.initEventListeners();
        this.initToggleFilters();
    }
    
    initRangeSliders() {
        // Creation Year Sliders
        if (this.creationYearFromSlider && this.creationYearToSlider) {
            this.creationYearFromSlider.addEventListener('input', this.updateCreationYearValues.bind(this));
            this.creationYearToSlider.addEventListener('input', this.updateCreationYearValues.bind(this));
        }
        
        // First Album Year Sliders
        if (this.firstAlbumYearFromSlider && this.firstAlbumYearToSlider) {
            this.firstAlbumYearFromSlider.addEventListener('input', this.updateFirstAlbumYearValues.bind(this));
            this.firstAlbumYearToSlider.addEventListener('input', this.updateFirstAlbumYearValues.bind(this));
        }
    }
    
    updateCreationYearValues() {
        let fromValue = parseInt(this.creationYearFromSlider.value);
        let toValue = parseInt(this.creationYearToSlider.value);
        
        // Ensure from is not greater than to
        if (fromValue > toValue) {
            if (this.creationYearFromSlider === document.activeElement) {
                this.creationYearToSlider.value = fromValue;
                toValue = fromValue;
            } else {
                this.creationYearFromSlider.value = toValue;
                fromValue = toValue;
            }
        }
        
        this.creationYearFromValue.textContent = fromValue;
        this.creationYearToValue.textContent = toValue;
        
        // Trigger filtering
        this.filterArtists();
    }
    
    updateFirstAlbumYearValues() {
        let fromValue = parseInt(this.firstAlbumYearFromSlider.value);
        let toValue = parseInt(this.firstAlbumYearToSlider.value);
        
        // Ensure from is not greater than to
        if (fromValue > toValue) {
            if (this.firstAlbumYearFromSlider === document.activeElement) {
                this.firstAlbumYearToSlider.value = fromValue;
                toValue = fromValue;
            } else {
                this.firstAlbumYearFromSlider.value = toValue;
                fromValue = toValue;
            }
        }
        
        this.firstAlbumYearFromValue.textContent = fromValue;
        this.firstAlbumYearToValue.textContent = toValue;
        
        // Trigger filtering
        this.filterArtists();
    }
    
    initEventListeners() {
        if (this.form) {
            this.form.addEventListener('submit', this.handleFormSubmit.bind(this));
            
            // Add event listeners to checkboxes
            const checkboxes = this.form.querySelectorAll('input[type="checkbox"]');
            checkboxes.forEach(checkbox => {
                checkbox.addEventListener('change', () => {
                    this.filterArtists();
                });
            });
        }
        
        if (this.clearButton) {
            this.clearButton.addEventListener('click', this.clearAllFilters.bind(this));
        }
        
        if (this.resetButton) {
            this.resetButton.addEventListener('click', this.resetFilters.bind(this));
        }
    }
    
    initToggleFilters() {
        const toggleButton = document.getElementById('toggle-filters');
        const filtersPanel = document.getElementById('filters-panel');
        
        if (toggleButton && filtersPanel) {
            toggleButton.addEventListener('click', () => {
                const isVisible = filtersPanel.style.display !== 'none';
                
                if (isVisible) {
                    filtersPanel.style.display = 'none';
                    toggleButton.setAttribute('aria-expanded', 'false');
                    toggleButton.querySelector('.filter-text').textContent = 'Show Filters';
                } else {
                    filtersPanel.style.display = 'block';
                    toggleButton.setAttribute('aria-expanded', 'true');
                    toggleButton.querySelector('.filter-text').textContent = 'Hide Filters';
                }
            });
        }
    }
    
    handleFormSubmit(e) {
        e.preventDefault();
        this.filterArtists();
    }
    
    async filterArtists() {
        if (this.isFiltering) return;
        
        this.isFiltering = true;
        this.showLoading();
        
        try {
            const filterParams = this.getFilterParams();
            
            // Send filter request to server
            const response = await fetch('/api/filter-artists', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(filterParams)
            });
            
            if (!response.ok) {
                throw new Error('Filter request failed');
            }
            
            const filteredArtists = await response.json();
            this.updateArtistsDisplay(filteredArtists);
            
        } catch (error) {
            console.error('Filtering error:', error);
            this.showError('Failed to filter artists. Please try again.');
        } finally {
            this.hideLoading();
            this.isFiltering = false;
        }
    }
    
    getFilterParams() {
        const params = {};
        
        // Creation year range
        if (this.creationYearFromSlider && this.creationYearToSlider) {
            const fromValue = parseInt(this.creationYearFromSlider.value);
            const toValue = parseInt(this.creationYearToSlider.value);
            const minValue = parseInt(this.creationYearFromSlider.min);
            const maxValue = parseInt(this.creationYearFromSlider.max);
            
            if (fromValue > minValue) params.creationYearFrom = fromValue;
            if (toValue < maxValue) params.creationYearTo = toValue;
        }
        
        // First album year range
        if (this.firstAlbumYearFromSlider && this.firstAlbumYearToSlider) {
            const fromValue = parseInt(this.firstAlbumYearFromSlider.value);
            const toValue = parseInt(this.firstAlbumYearToSlider.value);
            const minValue = parseInt(this.firstAlbumYearFromSlider.min);
            const maxValue = parseInt(this.firstAlbumYearFromSlider.max);
            
            if (fromValue > minValue) params.firstAlbumYearFrom = fromValue;
            if (toValue < maxValue) params.firstAlbumYearTo = toValue;
        }
        
        // Member counts (checkbox)
        const memberCheckboxes = this.form.querySelectorAll('input[name="memberCounts"]:checked');
        if (memberCheckboxes.length > 0) {
            params.memberCounts = Array.from(memberCheckboxes).map(cb => parseInt(cb.value));
        }
        
        // Countries (checkbox)
        const countryCheckboxes = this.form.querySelectorAll('input[name="countries"]:checked');
        if (countryCheckboxes.length > 0) {
            params.countries = Array.from(countryCheckboxes).map(cb => cb.value);
        }
        
        return params;
    }
    
    updateArtistsDisplay(filteredArtists) {
        // Clear current display
        this.artistsGrid.innerHTML = '';
        
        if (filteredArtists.length === 0) {
            this.showNoResults();
            return;
        }
        
        // Create artist cards from filtered data
        filteredArtists.forEach(artist => {
            const artistCard = this.createArtistCard(artist);
            this.artistsGrid.appendChild(artistCard);
        });
        
        this.hideNoResults();
        this.updateResultsCount(filteredArtists.length);
    }
    
    createArtistCard(artist) {
        const card = document.createElement('div');
        card.className = 'artist-card';
        card.setAttribute('data-year', artist.CreationYear || artist.creationYear);
        card.setAttribute('data-name', artist.Name || artist.name);
        card.setAttribute('data-members', artist.Members ? artist.Members.length : 0);
        card.setAttribute('data-first-album', artist.FirstAlbum || artist.firstAlbum);
        
        const membersText = artist.Members && artist.Members.length === 1 ? 'member' : 'members';
        const membersCount = artist.Members ? artist.Members.length : 0;
        
        card.innerHTML = `
            <a href="/artists/${artist.Slug || artist.slug}">
                <div class="artist-image-container">
                    <img src="${artist.Image || artist.image}" alt="${artist.Name || artist.name}" class="artist-image">
                </div>
                <div class="artist-info">
                    <h3>${artist.Name || artist.name}</h3>
                    <p class="creation-year">${artist.CreationYear || artist.creationYear}</p>
                    <p class="first-album">First Album: ${artist.FirstAlbum || artist.firstAlbum}</p>
                    <p class="members-count">${membersCount} ${membersText}</p>
                    <div class="members-preview">
                        ${this.createMembersPreview(artist.Members || [])}
                    </div>
                </div>
            </a>
        `;
        
        return card;
    }
    
    createMembersPreview(members) {
        if (!members || members.length === 0) return '';
        
        let preview = '';
        const maxShow = 3;
        
        for (let i = 0; i < Math.min(members.length, maxShow); i++) {
            preview += `<span class="member">${members[i]}</span>`;
        }
        
        if (members.length > maxShow) {
            preview += `<span class="more-members">+${members.length - maxShow} more</span>`;
        }
        
        return preview;
    }
    
    clearAllFilters() {
        // Reset sliders to their min/max values
        if (this.creationYearFromSlider && this.creationYearToSlider) {
            this.creationYearFromSlider.value = this.creationYearFromSlider.min;
            this.creationYearToSlider.value = this.creationYearToSlider.max;
            this.updateCreationYearValues();
        }
        
        if (this.firstAlbumYearFromSlider && this.firstAlbumYearToSlider) {
            this.firstAlbumYearFromSlider.value = this.firstAlbumYearFromSlider.min;
            this.firstAlbumYearToSlider.value = this.firstAlbumYearToSlider.max;
            this.updateFirstAlbumYearValues();
        }
        
        // Uncheck all checkboxes
        const checkboxes = this.form.querySelectorAll('input[type="checkbox"]');
        checkboxes.forEach(checkbox => {
            checkbox.checked = false;
        });
        
        // Show all artists
        this.showAllArtists();
    }
    
    resetFilters() {
        this.clearAllFilters();
    }
    
    showAllArtists() {
        this.artistsGrid.innerHTML = '';
        this.originalArtists.forEach(artist => {
            this.artistsGrid.appendChild(artist.cloneNode(true));
        });
        this.hideNoResults();
        this.updateResultsCount(this.originalArtists.length);
    }
    
    showLoading() {
        if (this.loading) {
            this.loading.style.display = 'flex';
        }
    }
    
    hideLoading() {
        if (this.loading) {
            this.loading.style.display = 'none';
        }
    }
    
    showNoResults() {
        if (this.noResults) {
            this.noResults.style.display = 'block';
        }
    }
    
    hideNoResults() {
        if (this.noResults) {
            this.noResults.style.display = 'none';
        }
    }
    
    updateResultsCount(count) {
        const resultsCount = document.querySelector('.results-count');
        if (resultsCount) {
            const text = count === 1 ? 'artist' : 'artists';
            resultsCount.textContent = `Browse through our collection of ${count} ${text}`;
        }
    }
    
    showError(message) {
        // Simple error display - you can enhance this
        console.error(message);
        alert(message);
    }
}

// Initialize the filter when the DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new ArtistFilter();
});
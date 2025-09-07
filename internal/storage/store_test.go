package storage

import (
	"context"
	"sync"
	"testing"
	"time"

	"groupie-tracker/internal/models"
)

// MockAPIClient implements APIClient for testing
type MockAPIClient struct {
	data          *APIData
	err           error
	callCount     int
	mu            sync.Mutex
	responseDelay time.Duration
}

func (m *MockAPIClient) FetchAllData(ctx context.Context) (*APIData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.responseDelay > 0 {
		time.Sleep(m.responseDelay)
	}

	m.callCount++
	return m.data, m.err
}

func (m *MockAPIClient) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func (m *MockAPIClient) ResetCallCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
}

func (m *MockAPIClient) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *MockAPIClient) SetResponseDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responseDelay = delay
}

func createMockAPIData() *APIData {
	return &APIData{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May"}, CreationYear: 1970},
			{ID: 2, Name: "Gorillaz", Members: []string{"Damon Albarn"}, CreationYear: 1998},
		},
		Locations: []models.Location{
			{ID: 1, Locations: []string{"london-uk", "manchester-uk"}},
			{ID: 2, Locations: []string{"london-uk", "new_york-usa"}},
		},
		Dates: []models.Date{
			{ID: 1, Dates: []string{"23-08-2019", "24-08-2019"}},
			{ID: 2, Dates: []string{"25-08-2019"}},
		},
		Relations: []models.Relation{
			{ID: 1, DatesLocations: map[string][]string{"london-uk": {"23-08-2019", "24-08-2019"}}},
			{ID: 2, DatesLocations: map[string][]string{"new_york-usa": {"25-08-2019"}}},
		},
	}
}

func TestStore_CacheUpdateInterval(t *testing.T) {
	if CacheUpdateInterval != 30*time.Second {
		t.Errorf("Expected CacheUpdateInterval to be 30 seconds, got %v", CacheUpdateInterval)
	}
}

func TestNewStore(t *testing.T) {
	store := NewStore()

	if store == nil {
		t.Fatal("NewStore() returned nil")
	}

	if store.artists == nil || store.locations == nil || store.dates == nil || store.relations == nil {
		t.Error("Store maps are not initialized")
	}

	if store.IsRunning() {
		t.Error("Expected cache to not be running initially")
	}

	if store.stopCache == nil {
		t.Error("stopCache channel is not initialized")
	}

	if store.updateInterval != CacheUpdateInterval {
		t.Errorf("Expected updateInterval to be %v, got %v", CacheUpdateInterval, store.updateInterval)
	}
}

func TestNewStoreWithCache(t *testing.T) {
	mockClient := &MockAPIClient{data: createMockAPIData()}
	store := NewStoreWithCache(mockClient)

	if store == nil {
		t.Fatal("NewStoreWithCache() returned nil")
	}

	if store.apiClient != mockClient {
		t.Error("API client not set correctly")
	}

	if store.IsRunning() {
		t.Error("Expected cache to not be running initially")
	}
}

func TestStore_StartAndStopCache(t *testing.T) {
	mockClient := &MockAPIClient{data: createMockAPIData()}
	store := NewStoreWithCache(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test starting cache
	store.StartCache(ctx)

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	if !store.IsRunning() {
		t.Error("Expected cache to be running after StartCache()")
	}

	// Check that initial data was loaded
	if mockClient.GetCallCount() == 0 {
		t.Error("Expected at least one API call during cache start")
	}

	// Test stopping cache
	store.StopCache()

	// Give it a moment to stop
	time.Sleep(10 * time.Millisecond)

	if store.IsRunning() {
		t.Error("Expected cache to not be running after StopCache()")
	}
}

func TestStore_StartCacheWithoutAPIClient(t *testing.T) {
	store := NewStore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Starting cache without API client should not panic
	store.StartCache(ctx)

	if store.IsRunning() {
		t.Error("Expected cache to not be running when no API client is set")
	}
}

func TestStore_StartCacheMultipleTimes(t *testing.T) {
	mockClient := &MockAPIClient{data: createMockAPIData()}
	store := NewStoreWithCache(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cache multiple times - should not cause issues
	store.StartCache(ctx)
	store.StartCache(ctx)
	store.StartCache(ctx)

	time.Sleep(10 * time.Millisecond)

	if !store.IsRunning() {
		t.Error("Expected cache to be running")
	}

	store.StopCache()
}

func TestStore_CachePeriodicUpdate(t *testing.T) {
	mockClient := &MockAPIClient{data: createMockAPIData()}
	store := NewStoreWithCache(mockClient)

	// Set a very short update interval for testing
	store.updateInterval = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockClient.ResetCallCount()
	store.StartCache(ctx)

	// Wait for multiple update cycles
	time.Sleep(150 * time.Millisecond)

	store.StopCache()

	callCount := mockClient.GetCallCount()
	if callCount < 2 {
		t.Errorf("Expected at least 2 API calls (initial + periodic), got %d", callCount)
	}
}

func TestStore_GetLastUpdate(t *testing.T) {
	mockClient := &MockAPIClient{data: createMockAPIData()}
	store := NewStoreWithCache(mockClient)

	// Initially should be zero time
	lastUpdate := store.GetLastUpdate()
	if !lastUpdate.IsZero() {
		t.Error("Expected initial last update to be zero time")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store.StartCache(ctx)
	time.Sleep(50 * time.Millisecond)
	store.StopCache()

	// After cache runs, should have a recent timestamp
	lastUpdate = store.GetLastUpdate()
	if lastUpdate.IsZero() {
		t.Error("Expected last update to be set after cache runs")
	}

	// Should be recent (within last second)
	if time.Since(lastUpdate) > time.Second {
		t.Error("Expected last update to be recent")
	}
}

func TestStore_CacheErrorHandling(t *testing.T) {
	mockClient := &MockAPIClient{
		data: createMockAPIData(),
		err:  nil,
	}
	store := NewStoreWithCache(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store.StartCache(ctx)
	time.Sleep(10 * time.Millisecond)

	// Introduce an error
	mockClient.SetError(context.DeadlineExceeded)

	// Cache should continue running despite errors
	time.Sleep(10 * time.Millisecond)

	if !store.IsRunning() {
		t.Error("Expected cache to continue running despite API errors")
	}

	store.StopCache()
}

func TestStore_ContextCancellation(t *testing.T) {
	mockClient := &MockAPIClient{data: createMockAPIData()}
	store := NewStoreWithCache(mockClient)

	ctx, cancel := context.WithCancel(context.Background())

	store.StartCache(ctx)
	time.Sleep(50 * time.Millisecond)

	if !store.IsRunning() {
		t.Error("Expected cache to be running")
	}

	// Cancel context
	cancel()
	time.Sleep(100 * time.Millisecond) // Longer wait time for context cancellation

	// Cache should stop due to context cancellation
	if store.IsRunning() {
		t.Error("Expected cache to stop after context cancellation")
	}
}

func TestStore_ComputeDerivedData(t *testing.T) {
	store := NewStore()

	// Add test data
	store.AddLocation(models.Location{ID: 1, Locations: []string{"london-uk", "manchester-uk"}})
	store.AddLocation(models.Location{ID: 2, Locations: []string{"london-uk", "new_york-usa"}})
	store.AddDate(models.Date{ID: 1, Dates: []string{"23-08-2019", "24-08-2019"}})
	store.AddDate(models.Date{ID: 2, Dates: []string{"24-08-2019", "25-08-2019"}})

	// Manually trigger computation
	store.mu.Lock()
	store.computeDerivedData()
	store.mu.Unlock()

	// Check unique locations
	uniqueLocations := store.GetUniqueLocations()
	expectedLocations := 3 // london-uk, manchester-uk, new_york-usa
	if len(uniqueLocations) != expectedLocations {
		t.Errorf("Expected %d unique locations, got %d", expectedLocations, len(uniqueLocations))
	}

	// Check unique dates
	uniqueDates := store.GetUniqueDates()
	expectedDates := 3 // 23-08-2019, 24-08-2019, 25-08-2019
	if len(uniqueDates) != expectedDates {
		t.Errorf("Expected %d unique dates, got %d", expectedDates, len(uniqueDates))
	}
}

func TestStore_LoadDataComputesDerivatives(t *testing.T) {
	store := NewStore()

	testData := StoreData{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970},
		},
		Locations: []models.Location{
			{ID: 1, Locations: []string{"london-uk", "manchester-uk"}},
		},
		Dates: []models.Date{
			{ID: 1, Dates: []string{"23-08-2019", "24-08-2019"}},
		},
		Relations: []models.Relation{
			{ID: 1, DatesLocations: map[string][]string{"london-uk": {"23-08-2019"}}},
		},
	}

	store.LoadData(testData)

	// Check that derived data was computed
	uniqueLocations := store.GetUniqueLocations()
	if len(uniqueLocations) != 2 {
		t.Errorf("Expected 2 unique locations after LoadData, got %d", len(uniqueLocations))
	}

	uniqueDates := store.GetUniqueDates()
	if len(uniqueDates) != 2 {
		t.Errorf("Expected 2 unique dates after LoadData, got %d", len(uniqueDates))
	}

	stats := store.GetStats()
	if stats["locations"] != 2 {
		t.Errorf("Expected locations stat to be 2, got %d", stats["locations"])
	}
	if stats["dates"] != 2 {
		t.Errorf("Expected dates stat to be 2, got %d", stats["dates"])
	}
}

func TestStore_CacheWithSlowAPI(t *testing.T) {
	mockClient := &MockAPIClient{
		data:          createMockAPIData(),
		responseDelay: 100 * time.Millisecond,
	}
	store := NewStoreWithCache(mockClient)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	store.StartCache(ctx)

	// Wait for initial load
	time.Sleep(150 * time.Millisecond)

	elapsed := time.Since(start)
	if elapsed < 100*time.Millisecond {
		t.Error("Expected initial cache load to take at least 100ms due to API delay")
	}

	store.StopCache()

	// Verify data was loaded despite delay
	artists := store.GetAllArtists()
	if len(artists) == 0 {
		t.Error("Expected artists to be loaded despite API delay")
	}
}

func TestStore_AddAndGetArtist(t *testing.T) {
	store := NewStore()

	artist := models.Artist{
		ID:           1,
		Name:         "Queen",
		Members:      []string{"Freddie Mercury", "Brian May"},
		CreationYear: 1970,
		FirstAlbum:   "14-12-1973",
	}

	// Test adding artist
	store.AddArtist(artist)

	// Test getting artist by ID
	retrievedArtist, exists := store.GetArtist(1)
	if !exists {
		t.Error("Expected artist to exist, but it doesn't")
	}

	if retrievedArtist.Name != "Queen" {
		t.Errorf("Expected artist name to be Queen, got %s", retrievedArtist.Name)
	}

	// Test getting non-existent artist
	_, exists = store.GetArtist(999)
	if exists {
		t.Error("Expected artist to not exist, but it does")
	}
}

func TestStore_GetAllArtists(t *testing.T) {
	store := NewStore()

	artists := []models.Artist{
		{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970},
		{ID: 2, Name: "Gorillaz", Members: []string{"Damon Albarn"}, CreationYear: 1998},
	}

	for _, artist := range artists {
		store.AddArtist(artist)
	}

	allArtists := store.GetAllArtists()
	if len(allArtists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(allArtists))
	}
}

func TestStore_SearchArtists(t *testing.T) {
	store := NewStore()

	artists := []models.Artist{
		{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May"}, CreationYear: 1970},
		{ID: 2, Name: "Gorillaz", Members: []string{"Damon Albarn"}, CreationYear: 1998},
		{ID: 3, Name: "Queen Bee", Members: []string{"Someone"}, CreationYear: 2000},
	}

	for _, artist := range artists {
		store.AddArtist(artist)
	}

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"exact match", "Queen", 2},
		{"case insensitive", "queen", 2},
		{"partial match", "Que", 2},
		{"member search", "Freddie", 1},
		{"no match", "Beatles", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := store.SearchArtists(tt.query)
			if len(results) != tt.expected {
				t.Errorf("Expected %d results for query '%s', got %d", tt.expected, tt.query, len(results))
			}
		})
	}
}

func TestStore_FilterArtistsByYear(t *testing.T) {
	store := NewStore()

	artists := []models.Artist{
		{ID: 1, Name: "Queen", CreationYear: 1970},
		{ID: 2, Name: "Gorillaz", CreationYear: 1998},
		{ID: 3, Name: "Modern Band", CreationYear: 2010},
	}

	for _, artist := range artists {
		store.AddArtist(artist)
	}

	// Test filtering by year range
	results := store.FilterArtistsByYear(1990, 2000)
	if len(results) != 1 {
		t.Errorf("Expected 1 artist between 1990-2000, got %d", len(results))
	}

	if results[0].Name != "Gorillaz" {
		t.Errorf("Expected Gorillaz, got %s", results[0].Name)
	}

	// Test with no year restrictions
	results = store.FilterArtistsByYear(0, 0)
	if len(results) != 3 {
		t.Errorf("Expected all 3 artists with no year filter, got %d", len(results))
	}
}

func TestStore_LocationsAndDates(t *testing.T) {
	store := NewStore()

	// Test locations
	location := models.Location{
		ID:        1,
		Locations: []string{"london-uk", "manchester-uk"},
	}
	store.AddLocation(location)

	retrievedLocation, exists := store.GetLocation(1)
	if !exists {
		t.Error("Expected location to exist")
	}

	if len(retrievedLocation.Locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(retrievedLocation.Locations))
	}

	// Test dates
	date := models.Date{
		ID:    1,
		Dates: []string{"23-08-2019", "24-08-2019"},
	}
	store.AddDate(date)

	retrievedDate, exists := store.GetDate(1)
	if !exists {
		t.Error("Expected date to exist")
	}

	if len(retrievedDate.Dates) != 2 {
		t.Errorf("Expected 2 dates, got %d", len(retrievedDate.Dates))
	}

	// Test relations
	relation := models.Relation{
		ID: 1,
		DatesLocations: map[string][]string{
			"london-uk": {"23-08-2019", "24-08-2019"},
		},
	}
	store.AddRelation(relation)

	retrievedRelation, exists := store.GetRelation(1)
	if !exists {
		t.Error("Expected relation to exist")
	}

	if len(retrievedRelation.DatesLocations) != 1 {
		t.Errorf("Expected 1 dates-location mapping, got %d", len(retrievedRelation.DatesLocations))
	}
}

func TestStore_GetUniqueLocations(t *testing.T) {
	store := NewStore()

	locations := []models.Location{
		{ID: 1, Locations: []string{"london-uk", "manchester-uk"}},
		{ID: 2, Locations: []string{"london-uk", "new_york-usa"}},
	}

	for _, location := range locations {
		store.AddLocation(location)
	}

	// Manually compute since cache won't run in this test
	store.mu.Lock()
	store.computeDerivedData()
	store.mu.Unlock()

	uniqueLocations := store.GetUniqueLocations()

	expected := 3 // london-uk, manchester-uk, new_york-usa
	if len(uniqueLocations) != expected {
		t.Errorf("Expected %d unique locations, got %d", expected, len(uniqueLocations))
	}

	// Check if london-uk appears only once
	count := 0
	for _, loc := range uniqueLocations {
		if loc == "london-uk" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected london-uk to appear once, appeared %d times", count)
	}
}

func TestStore_LoadData(t *testing.T) {
	store := NewStore()

	testData := StoreData{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970},
		},
		Locations: []models.Location{
			{ID: 1, Locations: []string{"london-uk"}},
		},
		Dates: []models.Date{
			{ID: 1, Dates: []string{"23-08-2019"}},
		},
		Relations: []models.Relation{
			{ID: 1, DatesLocations: map[string][]string{"london-uk": {"23-08-2019"}}},
		},
	}

	store.LoadData(testData)

	// Verify data was loaded
	if len(store.GetAllArtists()) != 1 {
		t.Error("Expected 1 artist after loading data")
	}

	_, exists := store.GetLocation(1)
	if !exists {
		t.Error("Expected location to exist after loading data")
	}

	_, exists = store.GetDate(1)
	if !exists {
		t.Error("Expected date to exist after loading data")
	}

	_, exists = store.GetRelation(1)
	if !exists {
		t.Error("Expected relation to exist after loading data")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewStore()

	// Test concurrent writes and reads
	done := make(chan bool, 2)

	// Goroutine 1: Add artists
	go func() {
		for i := 1; i <= 100; i++ {
			artist := models.Artist{
				ID:           i,
				Name:         "Artist",
				CreationYear: 2000 + i,
			}
			store.AddArtist(artist)
		}
		done <- true
	}()

	// Goroutine 2: Read artists
	go func() {
		for i := 0; i < 100; i++ {
			store.GetAllArtists()
		}
		done <- true
	}()

	// Wait for both goroutines to finish
	<-done
	<-done

	// Verify all artists were added
	artists := store.GetAllArtists()
	if len(artists) != 100 {
		t.Errorf("Expected 100 artists, got %d", len(artists))
	}
}

func TestStore_ConcurrentCacheOperations(t *testing.T) {
	mockClient := &MockAPIClient{data: createMockAPIData()}
	store := NewStoreWithCache(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cache
	store.StartCache(ctx)
	time.Sleep(10 * time.Millisecond)

	// Test concurrent operations while cache is running
	done := make(chan bool, 3)

	// Goroutine 1: Read operations
	go func() {
		for i := 0; i < 50; i++ {
			store.GetAllArtists()
			store.GetUniqueLocations()
			store.GetUniqueDates()
			store.GetStats()
		}
		done <- true
	}()

	// Goroutine 2: More read operations
	go func() {
		for i := 0; i < 50; i++ {
			store.SearchArtists("Queen")
			store.FilterArtistsByYear(1900, 2000)
		}
		done <- true
	}()

	// Goroutine 3: Cache management operations
	go func() {
		for i := 0; i < 10; i++ {
			store.GetLastUpdate()
			store.IsRunning()
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	store.StopCache()

	// Verify store state is consistent
	artists := store.GetAllArtists()
	if len(artists) == 0 {
		t.Error("Expected artists to be present after concurrent operations")
	}
}

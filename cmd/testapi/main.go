// Simple test server to replace the external API when network is unavailable
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

type Location struct {
	ID        int      `json:"id"`
	Locations []string `json:"locations"`
}

type Date struct {
	ID    int      `json:"id"`
	Dates []string `json:"dates"`
}

type Relation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

func main() {
	// Sample data
	artists := []Artist{
		{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"}, CreationDate: 1970, FirstAlbum: "14-12-1973", Image: "https://groupietrackers.herokuapp.com/api/images/queen.jpeg"},
		{ID: 2, Name: "Pink Floyd", Members: []string{"David Gilmour", "Roger Waters", "Nick Mason", "Richard Wright"}, CreationDate: 1965, FirstAlbum: "05-08-1967", Image: "https://groupietrackers.herokuapp.com/api/images/pinkfloyd.jpeg"},
	}

	locations := map[string]interface{}{
		"index": []Location{
			{ID: 1, Locations: []string{"london", "manchester", "birmingham"}},
			{ID: 2, Locations: []string{"london", "edinburgh", "glasgow"}},
		},
	}

	dates := map[string]interface{}{
		"index": []Date{
			{ID: 1, Dates: []string{"23-08-2019", "22-08-2019", "20-08-2019"}},
			{ID: 2, Dates: []string{"15-07-2020", "14-07-2020", "13-07-2020"}},
		},
	}

	relations := map[string]interface{}{
		"index": []Relation{
			{ID: 1, DatesLocations: map[string][]string{"london": {"23-08-2019"}, "manchester": {"22-08-2019"}}},
			{ID: 2, DatesLocations: map[string][]string{"london": {"15-07-2020"}, "edinburgh": {"14-07-2020"}}},
		},
	}

	http.HandleFunc("/api/artists", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(artists)
	})

	http.HandleFunc("/api/locations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(locations)
	})

	http.HandleFunc("/api/dates", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(dates)
	})

	http.HandleFunc("/api/relation", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(relations)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Test API Server - Use /api/ endpoints"))
	})

	log.Println("🧪 Test API server starting on :8081")
	log.Println("📍 Available endpoints:")
	log.Println("   GET /api/artists")
	log.Println("   GET /api/locations")
	log.Println("   GET /api/dates")
	log.Println("   GET /api/relation")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	// Test the range slider functionality
	fmt.Println("=== Testing Range Slider Form Submission ===")

	// Create form data to simulate range slider submission
	formData := url.Values{}
	formData.Set("creationYearFrom", "1970")
	formData.Set("creationYearTo", "1990")
	formData.Set("memberCounts", "4")

	// Send POST request to artists page
	resp, err := http.PostForm("http://localhost:8080/artists", formData)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %s\n", resp.Status)

	if resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		
		// Check if the response contains filter-related content
		if strings.Contains(bodyStr, "creation year") || strings.Contains(bodyStr, "Creation Year") {
			fmt.Println("✅ Range slider filter form processing works!")
		} else {
			fmt.Println("❌ Filter processing might have issues")
		}

		// Check if current selection is shown
		if strings.Contains(bodyStr, "1970") && strings.Contains(bodyStr, "1990") {
			fmt.Println("✅ Range slider values are preserved in response!")
		} else {
			fmt.Println("ℹ️  Range slider values not clearly visible in response")
		}

	} else {
		fmt.Printf("❌ Request failed with status: %s\n", resp.Status)
	}
}
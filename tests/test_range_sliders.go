package tests

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

func isServerUp() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func main() {
	fmt.Println("=== Testing Range Slider Form Submission ===")

	// Ensure server is running; if not, try to start it with `go run ./cmd/cli/`.
	var serverCmd *exec.Cmd
	if !isServerUp() {
		fmt.Println("Server not reachable on :8080 — attempting to start it (this requires Go in PATH)...")
		serverCmd = exec.Command("go", "run", "./cmd/cli/")
		serverCmd.Stdout = os.Stdout
		serverCmd.Stderr = os.Stderr
		if err := serverCmd.Start(); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
			return
		}

		// Ensure we clean up the process on exit
		defer func() {
			if serverCmd != nil && serverCmd.Process != nil {
				_ = serverCmd.Process.Kill()
			}
		}()

		// Wait for health endpoint
		ready := false
		for i := 0; i < 30; i++ {
			if isServerUp() {
				ready = true
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if !ready {
			fmt.Println("Server did not become ready in time — aborting test")
			return
		}
		fmt.Println("Server started and ready")
	}

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

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check if the response contains filter-related content
	if strings.Contains(bodyStr, "creation year") || strings.Contains(bodyStr, "Creation Year") {
		fmt.Println("✅ Range slider filter form processing works!")
	} else {
		fmt.Println("❌ Filter processing might have issues: 'Creation Year' text not found in response")
	}

	// Check if current selection is shown
	if strings.Contains(bodyStr, "1970") && strings.Contains(bodyStr, "1990") {
		fmt.Println("✅ Range slider values are preserved in response!")
	} else {
		fmt.Println("ℹ️  Range slider values not clearly visible in response")
	}

	// If we started the server, allow it a moment to flush logs before killing
	if serverCmd != nil && serverCmd.Process != nil {
		time.Sleep(200 * time.Millisecond)
	}
}

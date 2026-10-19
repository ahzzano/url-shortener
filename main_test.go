package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"url-shortener/api"
	"url-shortener/services/shortener"

	"github.com/joho/godotenv"
)

const targetURL = "https://www.youtube.com/"

func getDatabaseAndApi(t *testing.T) (*sql.DB, *httptest.Server) {
	if err := godotenv.Load(".env"); err != nil {
		t.Fatal("Unable to load .env file")
	}

	db, err := setupDatabase()
	if err != nil {
		t.Fatalf("Unable to setup database: %v", err.Error())
	}

	httpServer := httptest.NewServer(api.New(db))

	return db, httpServer

}

func Test_HappyPath(t *testing.T) {
	db, httpServer := getDatabaseAndApi(t)
	defer db.Close()
	defer httpServer.Close()

	formData := url.Values{}
	formData.Set("url", targetURL)

	resp, err := http.PostForm(httpServer.URL+"/shorten", formData)
	if err != nil {
		t.Fatalf("Failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected Status Code: %v, Found: %v", http.StatusCreated, resp.StatusCode)
	}

	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Expected JSON content type, got %q", got)
	}

	var result shortener.ShortenURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	t.Log("Successfully created a shortened URL")
	t.Logf("URL ID is %v", result.URL)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err = client.Get(httpServer.URL + "/" + result.URL)
	if err != nil {
		t.Fatalf("Failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("Expected status %d, got %d", http.StatusFound, resp.StatusCode)
	}

	expectedLocation := targetURL
	actualLocation := resp.Header.Get("Location")

	if actualLocation != expectedLocation {
		t.Fatalf("Expected redirect to %v, did not redirect, stayed at %v", expectedLocation, actualLocation)
	}
	t.Log("Successfully created redirects a URL")

	id, err := strconv.ParseInt(result.URL, 36, 64)
	if err != nil {
		t.Fatalf("Failed to parse shortened ID: %v", err)
	}
	if _, err := db.Exec("DELETE FROM urls WHERE id = $1", id); err != nil {
		t.Fatalf("Failed to remove test URL for missing-link check: %v", err)
	}

	for _, tc := range []struct {
		id     string
		status int
		body   string
	}{
		{"!", http.StatusBadRequest, "Invalid URL ID\n"},
		{"zzzzzzzzzzzzzzzzzzzz", http.StatusBadRequest, "Invalid URL ID\n"},
		{result.URL, http.StatusNotFound, "URL not found\n"},
		{result.URL, http.StatusInternalServerError, "Unable to find url ID\n"},
	} {
		if tc.status == http.StatusInternalServerError {
			if err := db.Close(); err != nil {
				t.Fatalf("Failed to close test database: %v", err)
			}
		}

		resp, err := client.Get(httpServer.URL + "/" + tc.id)
		if err != nil {
			t.Fatalf("Failed to request URL ID %q: %v", tc.id, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("Failed to read error response: %v", err)
		}
		if resp.StatusCode != tc.status {
			t.Errorf("URL ID %q: expected status %d, got %d", tc.id, tc.status, resp.StatusCode)
		}
		if location := resp.Header.Get("Location"); location != "" {
			t.Errorf("URL ID %q: unexpected redirect to %q", tc.id, location)
		}
		if string(body) != tc.body {
			t.Errorf("URL ID %q: expected body %q, got %q", tc.id, tc.body, body)
		}
	}
}

func Test_InvalidURL(t *testing.T) {
	db, httpServer := getDatabaseAndApi(t)
	defer db.Close()
	defer httpServer.Close()

	invalidRequests := []string{
		"why have you forsaken me",
		"https://in my heart",
		"http://this-ffire/ woah",
		"icommendmyspirit.io",
		"/path",
		"https://code:pass@example.com",
		"https://@self.com",
		"https://example.com/\t",
		"https://example.com/\n",
	}

	for _, badURL := range invalidRequests {
		formData := url.Values{}
		formData.Set("url", badURL)

		resp, err := http.PostForm(httpServer.URL+"/shorten", formData)
		if err != nil {
			t.Fatalf("Failed to send POST request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Shortened request went through")
		}
	}
}

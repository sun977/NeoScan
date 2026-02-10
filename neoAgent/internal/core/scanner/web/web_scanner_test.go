package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"neoagent/internal/core/model"
)

func TestWebScanner_Fingerprint(t *testing.T) {
	// 1. Mock Server (Nginx + PHP)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "nginx/1.18.0")
		w.Header().Set("X-Powered-By", "PHP/7.4.3")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, "<html><head><title>Test Page</title></head><body>Hello World</body></html>")
	}))
	defer ts.Close()

	// 2. Initialize Scanner
	scanner := NewWebScanner()

	// 3. Force Init (to load rules from relative path)
	// ensureInit is private, but we are in package web
	scanner.ensureInit()

	// 4. Create Task
	task := &model.Task{
		ID:     "test-task",
		Target: "127.0.0.1",
	}

	// 5. Run Fallback Scan directly
	// We use fallbackScan to avoid dependency on Chrome/Rod in unit tests
	ctx := context.Background()
	results, err := scanner.fallbackScan(ctx, task, ts.URL, time.Now())

	if err != nil {
		t.Fatalf("fallbackScan failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	res, ok := results[0].Result.(*model.WebResult)
	if !ok {
		t.Fatal("Result is not *model.WebResult")
	}

	// 6. Verify Fingerprints
	t.Logf("TechStack: %v", res.TechStack)

	foundNginx := false
	foundPHP := false

	for _, tech := range res.TechStack {
		if tech == "Nginx" {
			foundNginx = true
		}
		if tech == "PHP" {
			foundPHP = true
		}
	}

	if !foundNginx {
		t.Error("Failed to identify Nginx (Check if rules/fingerprint/web/web_fingerprints.json is loaded)")
	}
	if !foundPHP {
		t.Error("Failed to identify PHP")
	}
	if res.Title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got '%s'", res.Title)
	}
}

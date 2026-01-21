package fingerprint_test

import (
	"context"
	"testing"

	"neoagent/internal/pkg/fingerprint"
	"neoagent/internal/pkg/fingerprint/engines/http"
	"neoagent/internal/pkg/fingerprint/engines/service"
	"neoagent/internal/pkg/fingerprint/model"
)

func TestFingerprintService(t *testing.T) {
	// 1. Setup Engines
	httpEngine := http.NewHTTPEngine(nil)
	serviceEngine := service.NewServiceEngine(nil)
	fpService := fingerprint.NewFingerprintService(httpEngine, serviceEngine)

	// 2. Load Rules
	fingerRules := []model.FingerRule{
		{
			Name:       "TestApp",
			Body:       "welcome to test app",
			StatusCode: "200",
			Enabled:    true,
		},
		{
			Name:    "Nginx",
			Header:  "Server: nginx",
			Enabled: true,
		},
	}
	cpeRules := []model.CPERule{
		{
			MatchStr: `(?i)Apache/([\d\.]+)`,
			Vendor:   "apache",
			Product:  "http_server",
			CPE:      "cpe:2.3:a:apache:http_server:$1:*:*:*:*:*:*:*",
			Enabled:  true,
		},
	}

	fpService.LoadRules(fingerRules, cpeRules)

	// 3. Test HTTP Matching
	t.Run("HTTP Match Body", func(t *testing.T) {
		input := &fingerprint.Input{
			Body:       "<html>welcome to test app</html>",
			StatusCode: 200,
			Headers:    map[string]string{},
		}
		result, err := fpService.Identify(context.Background(), input)
		if err != nil {
			t.Fatalf("Identify failed: %v", err)
		}
		if result == nil || len(result.Matches) == 0 {
			t.Fatal("Expected match, got none")
		}
		found := false
		for _, m := range result.Matches {
			if m.Product == "TestApp" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected match for TestApp")
		}
	})

	t.Run("HTTP Match Header", func(t *testing.T) {
		input := &fingerprint.Input{
			Body:       "nothing",
			StatusCode: 404,
			Headers: map[string]string{
				"Server": "nginx/1.18.0",
			},
		}
		result, err := fpService.Identify(context.Background(), input)
		if err != nil {
			t.Fatalf("Identify failed: %v", err)
		}
		if result == nil || len(result.Matches) == 0 {
			t.Fatal("Expected match, got none")
		}
		found := false
		for _, m := range result.Matches {
			if m.Product == "Nginx" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected match for Nginx")
		}
	})

	// 4. Test Service Matching
	t.Run("Service Match Regex", func(t *testing.T) {
		input := &fingerprint.Input{
			Banner: "Apache/2.4.41 (Ubuntu)",
		}
		result, err := fpService.Identify(context.Background(), input)
		if err != nil {
			t.Fatalf("Identify failed: %v", err)
		}
		if result == nil || len(result.Matches) == 0 {
			t.Fatal("Expected match, got none")
		}
		found := false
		for _, m := range result.Matches {
			if m.Product == "http_server" && m.Version == "2.4.41" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected match for Apache http_server 2.4.41")
		}
	})
}

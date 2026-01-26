package port_service

import (
	"context"
	"testing"
	"time"

	"neoagent/internal/core/model"
)

func TestPortServiceScanner_Compile(t *testing.T) {
	// Simple compilation check
	s := NewPortServiceScanner()
	_ = s.Name()
}

func TestPortServiceScanner_Run_Mock(t *testing.T) {
	// This test just ensures the Run method doesn't panic and handles empty inputs gracefully
	scanner := NewPortServiceScanner()

	task := &model.Task{
		ID:        "test-task",
		Target:    "127.0.0.1",
		PortRange: "80,443",
		Params: map[string]interface{}{
			"service_detect": false,
			"rate":           10,
		},
	}

	// We don't actually expect it to find anything on localhost without a server running,
	// but we want to ensure it runs without error.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	results, err := scanner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Results might be empty if ports are closed
	t.Logf("Scan finished with %d results", len(results))
}

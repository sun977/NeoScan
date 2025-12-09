package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"neomaster/internal/service/orchestrator/policy"

	"github.com/stretchr/testify/assert"
)

func TestTargetProvider_ResolveTargets(t *testing.T) {
	provider := policy.NewTargetProvider()
	ctx := context.Background()
	seedTargets := []string{"192.168.1.1", "192.168.1.2"}

	// Helper function to extract values
	getValues := func(targets []policy.Target) []string {
		values := make([]string, len(targets))
		for i, t := range targets {
			values[i] = t.Value
		}
		return values
	}

	t.Run("Empty Policy", func(t *testing.T) {
		targets, err := provider.ResolveTargets(ctx, "", seedTargets)
		assert.NoError(t, err)
		assert.Equal(t, seedTargets, getValues(targets))
	})

	t.Run("Empty JSON Policy", func(t *testing.T) {
		targets, err := provider.ResolveTargets(ctx, "{}", seedTargets)
		assert.NoError(t, err)
		assert.Equal(t, seedTargets, getValues(targets))
	})

	t.Run("Manual Source", func(t *testing.T) {
		config := policy.TargetPolicyConfig{
			TargetSources: []policy.TargetSourceConfig{
				{
					SourceType:  "manual",
					TargetType:  "ip",
					SourceValue: "10.0.0.1, 10.0.0.2",
				},
			},
		}
		jsonBytes, _ := json.Marshal(config)

		targets, err := provider.ResolveTargets(ctx, string(jsonBytes), seedTargets)
		assert.NoError(t, err)
		vals := getValues(targets)
		assert.Contains(t, vals, "10.0.0.1")
		assert.Contains(t, vals, "10.0.0.2")
		assert.Len(t, targets, 2)
	})

	t.Run("Project Target Source", func(t *testing.T) {
		config := policy.TargetPolicyConfig{
			TargetSources: []policy.TargetSourceConfig{
				{
					SourceType: "project_target",
					TargetType: "ip",
				},
			},
		}
		jsonBytes, _ := json.Marshal(config)

		targets, err := provider.ResolveTargets(ctx, string(jsonBytes), seedTargets)
		assert.NoError(t, err)
		assert.ElementsMatch(t, seedTargets, getValues(targets))
	})

	t.Run("Mixed Sources (Manual + Project)", func(t *testing.T) {
		config := policy.TargetPolicyConfig{
			TargetSources: []policy.TargetSourceConfig{
				{
					SourceType:  "manual",
					TargetType:  "ip",
					SourceValue: "10.0.0.3",
				},
				{
					SourceType: "project_target",
					TargetType: "ip",
				},
			},
		}
		jsonBytes, _ := json.Marshal(config)

		targets, err := provider.ResolveTargets(ctx, string(jsonBytes), seedTargets)
		assert.NoError(t, err)
		assert.Len(t, targets, 3)
		vals := getValues(targets)
		assert.Contains(t, vals, "10.0.0.3")
		assert.Contains(t, vals, "192.168.1.1")
	})

	t.Run("Unsupported Source", func(t *testing.T) {
		config := policy.TargetPolicyConfig{
			TargetSources: []policy.TargetSourceConfig{
				{
					SourceType: "unknown_source",
				},
				{
					SourceType:  "manual",
					SourceValue: "10.0.0.4",
				},
			},
		}
		jsonBytes, _ := json.Marshal(config)

		// Should skip unknown and return manual
		targets, err := provider.ResolveTargets(ctx, string(jsonBytes), seedTargets)
		assert.NoError(t, err)
		assert.Len(t, targets, 1)
		vals := getValues(targets)
		assert.Contains(t, vals, "10.0.0.4")
	})

	t.Run("File Source (Line)", func(t *testing.T) {
		// Create temp file
		f, err := os.CreateTemp("", "targets_line_*.txt")
		assert.NoError(t, err)
		defer os.Remove(f.Name())
		f.WriteString("1.1.1.1\n2.2.2.2\n")
		f.Close()

		config := policy.TargetPolicyConfig{
			TargetSources: []policy.TargetSourceConfig{
				{
					SourceType:   "file",
					TargetType:   "ip",
					SourceValue:  f.Name(),
					ParserConfig: json.RawMessage(`{"format":"line"}`),
				},
			},
		}
		jsonBytes, _ := json.Marshal(config)
		targets, err := provider.ResolveTargets(ctx, string(jsonBytes), seedTargets)
		assert.NoError(t, err)
		vals := getValues(targets)
		assert.Contains(t, vals, "1.1.1.1")
		assert.Contains(t, vals, "2.2.2.2")
	})

	t.Run("File Source (CSV)", func(t *testing.T) {
		// Create temp file
		f, err := os.CreateTemp("", "targets_csv_*.csv")
		assert.NoError(t, err)
		defer os.Remove(f.Name())
		f.WriteString("id,ip,desc\n1,3.3.3.3,test1\n2,4.4.4.4,test2\n")
		f.Close()

		config := policy.TargetPolicyConfig{
			TargetSources: []policy.TargetSourceConfig{
				{
					SourceType:   "file",
					TargetType:   "ip",
					SourceValue:  f.Name(),
					ParserConfig: json.RawMessage(`{"format":"csv", "csv_column":"ip"}`),
				},
			},
		}
		jsonBytes, _ := json.Marshal(config)
		targets, err := provider.ResolveTargets(ctx, string(jsonBytes), seedTargets)
		assert.NoError(t, err)
		vals := getValues(targets)
		assert.Contains(t, vals, "3.3.3.3")
		assert.Contains(t, vals, "4.4.4.4")
	})

	t.Run("File Source (JSON Array)", func(t *testing.T) {
		// Create temp file
		f, err := os.CreateTemp("", "targets_json_*.json")
		assert.NoError(t, err)
		defer os.Remove(f.Name())
		f.WriteString(`[{"host":"5.5.5.5"},{"host":"6.6.6.6"}]`)
		f.Close()

		config := policy.TargetPolicyConfig{
			TargetSources: []policy.TargetSourceConfig{
				{
					SourceType:   "file",
					TargetType:   "ip",
					SourceValue:  f.Name(),
					ParserConfig: json.RawMessage(`{"format":"json_array", "json_path":"host"}`),
				},
			},
		}
		jsonBytes, _ := json.Marshal(config)
		targets, err := provider.ResolveTargets(ctx, string(jsonBytes), seedTargets)
		assert.NoError(t, err)
		vals := getValues(targets)
		assert.Contains(t, vals, "5.5.5.5")
		assert.Contains(t, vals, "6.6.6.6")
	})

	t.Run("Health Check", func(t *testing.T) {
		health := provider.CheckHealth(ctx)
		assert.NotNil(t, health)
		assert.Contains(t, health, "manual")
		assert.Contains(t, health, "project_target")
		assert.Nil(t, health["manual"])
		assert.Nil(t, health["project_target"])
	})
}

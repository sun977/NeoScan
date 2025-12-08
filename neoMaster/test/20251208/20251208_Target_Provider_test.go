package test

import (
	"context"
	"encoding/json"
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

	t.Run("Stub File Source", func(t *testing.T) {
		config := policy.TargetPolicyConfig{
			TargetSources: []policy.TargetSourceConfig{
				{
					SourceType:  "file",
					SourceValue: "/tmp/targets.txt",
				},
			},
		}
		jsonBytes, _ := json.Marshal(config)

		targets, err := provider.ResolveTargets(ctx, string(jsonBytes), seedTargets)
		assert.NoError(t, err)
		assert.Empty(t, targets) // Stub returns nil/empty
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

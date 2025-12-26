package main

import (
	"context"
	"fmt"
	"neomaster/internal/model/orchestrator"
	"neomaster/internal/service/orchestrator/policy"
)

func main() {
	// 1. Setup
	provider := policy.NewTargetProvider(nil)

	// 2. Define policy
	targetPolicy := orchestrator.TargetPolicy{
		WhitelistEnabled: true,
		WhitelistSources: []orchestrator.WhitelistSource{
			{
				SourceType:  "manual",
				SourceValue: "192.168.1.1,192.168.1.2",
			},
		},
		TargetSources: []orchestrator.TargetSource{},
	}
	seedTargets := []string{"192.168.1.1", "192.168.1.2", "8.8.8.8"}

	// 3. Run
	fmt.Println("Starting ResolveTargets...")
	targets, err := provider.ResolveTargets(context.Background(), targetPolicy, seedTargets)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result targets count: %d\n", len(targets))
	for _, t := range targets {
		fmt.Printf(" - %s\n", t.Value)
	}
}

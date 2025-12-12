package test_20251212

import (
	"context"
	"testing"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/orchestrator"
	"neomaster/internal/service/orchestrator/allocator"

	"github.com/stretchr/testify/assert"
)

func TestAllocator_CanExecute(t *testing.T) {
	alloc := allocator.NewResourceAllocator()
	ctx := context.Background()

	tests := []struct {
		name         string
		agent        *agentModel.Agent
		task         *orchestrator.AgentTask
		expected     bool
		description  string
	}{
		{
			name: "Basic Success",
			agent: &agentModel.Agent{
				Status:       agentModel.AgentStatusOnline,
				Capabilities: agentModel.StringSlice{"nmap"},
				Tags:         agentModel.StringSlice{"dmz"},
			},
			task: &orchestrator.AgentTask{
				ToolName:     "nmap",
				RequiredTags: `["dmz"]`,
			},
			expected:    true,
			description: "Agent online, has tool, has tag",
		},
		{
			name: "Agent Offline",
			agent: &agentModel.Agent{
				Status:       agentModel.AgentStatusOffline,
				Capabilities: agentModel.StringSlice{"nmap"},
			},
			task: &orchestrator.AgentTask{ToolName: "nmap"},
			expected:    false,
			description: "Agent offline should fail",
		},
		{
			name: "Missing Capability",
			agent: &agentModel.Agent{
				Status:       agentModel.AgentStatusOnline,
				Capabilities: agentModel.StringSlice{"masscan"},
			},
			task: &orchestrator.AgentTask{ToolName: "nmap"},
			expected:    false,
			description: "Agent missing tool should fail",
		},
		{
			name: "Missing Tag",
			agent: &agentModel.Agent{
				Status:       agentModel.AgentStatusOnline,
				Capabilities: agentModel.StringSlice{"nmap"},
				Tags:         agentModel.StringSlice{"internal"},
			},
			task: &orchestrator.AgentTask{
				ToolName:     "nmap",
				RequiredTags: `["dmz"]`,
			},
			expected:    false,
			description: "Agent missing required tag should fail",
		},
		{
			name: "Capability Case Insensitive",
			agent: &agentModel.Agent{
				Status:       agentModel.AgentStatusOnline,
				Capabilities: agentModel.StringSlice{"NMAP"},
			},
			task: &orchestrator.AgentTask{ToolName: "nmap"},
			expected:    true,
			description: "Capability check should be case insensitive",
		},
		{
			name: "Tag Case Sensitive (Current Behavior)",
			agent: &agentModel.Agent{
				Status:       agentModel.AgentStatusOnline,
				Capabilities: agentModel.StringSlice{"nmap"},
				Tags:         agentModel.StringSlice{"DMZ"},
			},
			task: &orchestrator.AgentTask{
				ToolName:     "nmap",
				RequiredTags: `["dmz"]`,
			},
			expected:    false,
			description: "Tag matching is currently case sensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := alloc.CanExecute(ctx, tt.agent, tt.task)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

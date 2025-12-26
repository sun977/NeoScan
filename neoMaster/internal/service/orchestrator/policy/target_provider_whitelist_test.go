package policy

import (
	"context"
	"os"
	"testing"

	orcmodel "neomaster/internal/model/orchestrator"

	"github.com/stretchr/testify/assert"
)

// TestTargetProvider_Whitelist 测试白名单功能
// 重点验证：
// 1. 白名单命中目标是否被正确移除（黑名单行为）
// 2. 支持 manual 和 file 两种来源
// 3. 不同格式的 manual 输入（字符串 vs 数组）
func TestTargetProvider_Whitelist(t *testing.T) {
	// 1. 准备测试数据
	// 创建临时文件作为文件白名单源
	whitelistContent := "192.168.1.100\n10.0.0.5"
	tmpFile, err := os.CreateTemp("", "whitelist_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name()) // 清理临时文件
	_, err = tmpFile.WriteString(whitelistContent)
	assert.NoError(t, err)
	tmpFile.Close()

	// 2. 初始化 TargetProvider
	provider := NewTargetProvider(nil)

	// 3. 定义测试用例
	tests := []struct {
		name           string
		seedTargets    []string               // 种子目标（作为输入）
		policy         orcmodel.TargetPolicy  // 策略配置
		expectedCount  int                    // 期望剩余目标数量
		expectedValues []string               // 期望剩余的具体目标值
	}{
		{
			name:        "Case 1: 只有 Manual 白名单 (字符串格式)",
			seedTargets: []string{"192.168.1.1", "192.168.1.2", "8.8.8.8"},
			policy: orcmodel.TargetPolicy{
				WhitelistEnabled: true,
				WhitelistSources: []orcmodel.WhitelistSource{
					{
						SourceType:  "manual",
						SourceValue: "192.168.1.1,192.168.1.2", // 字符串格式
					},
				},
				TargetSources: []orcmodel.TargetSource{}, // 为空则回退使用 seedTargets
			},
			expectedCount:  1,
			expectedValues: []string{"8.8.8.8"},
		},
		{
			name:        "Case 2: 只有 Manual 白名单 (数组格式)",
			seedTargets: []string{"192.168.1.1", "192.168.1.2", "8.8.8.8"},
			policy: orcmodel.TargetPolicy{
				WhitelistEnabled: true,
				WhitelistSources: []orcmodel.WhitelistSource{
					{
						SourceType:  "manual",
						SourceValue: []string{"192.168.1.1", "192.168.1.2"}, // 数组格式
					},
				},
				TargetSources: []orcmodel.TargetSource{},
			},
			expectedCount:  1,
			expectedValues: []string{"8.8.8.8"},
		},
		{
			name:        "Case 3: File 白名单",
			seedTargets: []string{"192.168.1.100", "10.0.0.5", "1.1.1.1"},
			policy: orcmodel.TargetPolicy{
				WhitelistEnabled: true,
				WhitelistSources: []orcmodel.WhitelistSource{
					{
						SourceType:  "file",
						SourceValue: tmpFile.Name(), // 使用临时文件路径
					},
				},
				TargetSources: []orcmodel.TargetSource{},
			},
			expectedCount:  1,
			expectedValues: []string{"1.1.1.1"},
		},
		{
			name:        "Case 4: 混合白名单 (Manual + File)",
			seedTargets: []string{"192.168.1.1", "192.168.1.100", "8.8.8.8"},
			policy: orcmodel.TargetPolicy{
				WhitelistEnabled: true,
				WhitelistSources: []orcmodel.WhitelistSource{
					{
						SourceType:  "manual",
						SourceValue: "192.168.1.1",
					},
					{
						SourceType:  "file",
						SourceValue: tmpFile.Name(), // 包含 192.168.1.100
					},
				},
				TargetSources: []orcmodel.TargetSource{},
			},
			expectedCount:  1,
			expectedValues: []string{"8.8.8.8"},
		},
		{
			name:        "Case 5: 白名单未启用",
			seedTargets: []string{"192.168.1.1", "8.8.8.8"},
			policy: orcmodel.TargetPolicy{
				WhitelistEnabled: false, // 未启用
				WhitelistSources: []orcmodel.WhitelistSource{
					{
						SourceType:  "manual",
						SourceValue: "192.168.1.1",
					},
				},
				TargetSources: []orcmodel.TargetSource{},
			},
			expectedCount:  2,
			expectedValues: []string{"192.168.1.1", "8.8.8.8"},
		},
		{
			name:        "Case 6: 白名单为空",
			seedTargets: []string{"192.168.1.1", "8.8.8.8"},
			policy: orcmodel.TargetPolicy{
				WhitelistEnabled: true,
				WhitelistSources: []orcmodel.WhitelistSource{}, // 空列表
				TargetSources:    []orcmodel.TargetSource{},
			},
			expectedCount:  2,
			expectedValues: []string{"192.168.1.1", "8.8.8.8"},
		},
	}

	// 4. 执行测试循环
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 调用 ResolveTargets
			// 注意：因为 TargetSources 为空，ResolveTargets 内部会回退使用 seedTargets
			// 并在回退后应用 Whitelist 逻辑
			targets, err := provider.ResolveTargets(context.Background(), tt.policy, tt.seedTargets)
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(targets), "剩余目标数量不符合预期")

			// 验证具体值
			if tt.expectedCount > 0 {
				actualValues := make([]string, 0, len(targets))
				for _, t := range targets {
					actualValues = append(actualValues, t.Value)
				}
				assert.ElementsMatch(t, tt.expectedValues, actualValues, "剩余目标值不符合预期")
			}
		})
	}
}

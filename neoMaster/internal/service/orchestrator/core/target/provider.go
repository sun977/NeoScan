// 目标提供者
// - 这个组件负责解析 ScanStage.TargetPolicy ，并将其“翻译”为具体的 IP/URL 列表。
package target

import (
	"context"
	"encoding/json"
	"neomaster/internal/pkg/logger"
	"strings"
)

// TargetProvider 目标提供者接口
// 负责解析 TargetPolicy 并返回具体的扫描目标列表
type TargetProvider interface {
	ResolveTargets(ctx context.Context, policyJSON string, seedTargets []string) ([]string, error)
}

type targetProvider struct {
	// 可以添加 database repo 依赖，用于查询 database 类型的目标
}

// NewTargetProvider 创建目标提供者实例
func NewTargetProvider() TargetProvider {
	return &targetProvider{}
}

// TargetPolicyConfig 目标策略配置结构
type TargetPolicyConfig struct {
	TargetSources []TargetSource `json:"target_sources"`
}

// TargetSource 目标源定义
type TargetSource struct {
	SourceType  string `json:"source_type"`  // manual, project_target, file, etc.
	SourceValue string `json:"source_value"` // 具体值，如IP列表或文件路径
}

// ResolveTargets 解析目标
func (p *targetProvider) ResolveTargets(ctx context.Context, policyJSON string, seedTargets []string) ([]string, error) {
	// 1. 如果策略为空，默认使用种子目标
	if policyJSON == "" || policyJSON == "{}" {
		return seedTargets, nil
	}

	var config TargetPolicyConfig
	if err := json.Unmarshal([]byte(policyJSON), &config); err != nil {
		logger.LogWarn("Failed to parse target policy, using seed targets", "", 0, "", "ResolveTargets", "", map[string]interface{}{
			"error":  err.Error(),
			"policy": policyJSON,
		})
		return seedTargets, nil
	}

	// 2. 如果没有配置源，也使用种子目标
	if len(config.TargetSources) == 0 {
		return seedTargets, nil
	}

	// 3. 收集所有目标
	targetSet := make(map[string]struct{})

	for _, source := range config.TargetSources {
		switch source.SourceType {
		case "manual":
			// 手动指定：支持逗号或换行分隔
			parts := strings.FieldsFunc(source.SourceValue, func(r rune) bool {
				return r == ',' || r == '\n' || r == ';'
			})
			for _, part := range parts {
				t := strings.TrimSpace(part)
				if t != "" {
					targetSet[t] = struct{}{}
				}
			}
		case "project_target":
			// 项目种子目标
			for _, t := range seedTargets {
				targetSet[t] = struct{}{}
			}
		case "file":
			// TODO: 实现文件读取逻辑
			logger.LogWarn("File source not supported yet", "", 0, "", "ResolveTargets", "", nil)
		default:
			logger.LogWarn("Unknown source type", "", 0, "", "ResolveTargets", "", map[string]interface{}{
				"type": source.SourceType,
			})
		}
	}

	// 4. 转换为切片
	targets := make([]string, 0, len(targetSet))
	for t := range targetSet {
		targets = append(targets, t)
	}

	// 如果解析后为空（例如配置了错误的源），回退到种子目标？
	// 策略决定：如果显式配置了策略但没解析出目标，应该返回空，而不是回退，避免意外扫描
	return targets, nil
}

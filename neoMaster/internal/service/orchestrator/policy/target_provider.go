// 目标提供者
// 处理多元异构数据，将其转换为统一的 Target 格式作为扫描目标

package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"neomaster/internal/pkg/logger"
	"strings"
	"sync"
)

// TargetProvider 目标提供者服务接口
// 负责解析 TargetPolicy 并返回具体的扫描目标列表
type TargetProvider interface {
	// ResolveTargets 解析策略并返回目标列表
	ResolveTargets(ctx context.Context, policyJSON string, seedTargets []string) ([]Target, error)
	// RegisterProvider 注册新的目标源提供者
	RegisterProvider(name string, provider SourceProvider)
	// CheckHealth 检查所有已注册 Provider 的健康状态
	CheckHealth(ctx context.Context) map[string]error
}

// SourceProvider 单个目标源提供者接口
// 对应设计文档中的 TargetProvider Interface
type SourceProvider interface {
	// Name 返回 Provider 名称
	Name() string

	// Provide 执行获取逻辑
	// ctx: 上下文
	// config: 目标源配置
	// seedTargets: 种子目标(用于 project_target)
	Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error)

	// HealthCheck 检查 Provider 健康状态 (如数据库连接、文件访问权限)
	HealthCheck(ctx context.Context) error
}

// Target 标准目标对象
// 对应设计文档: 运行时对象 (Target Object)
type Target struct {
	Type   string            `json:"type"`   // 目标类型: ip, domain, url
	Value  string            `json:"value"`  // 目标值
	Source string            `json:"source"` // 来源标识
	Meta   map[string]string `json:"meta"`   // 元数据
}

// TargetPolicyConfig 目标策略配置结构
// 对应 ScanStage.target_policy
type TargetPolicyConfig struct {
	TargetSources []TargetSourceConfig `json:"target_sources"`
}

// TargetSourceConfig 目标源配置详细结构
// 对应设计文档: 配置结构 (TargetSource Config)
type TargetSourceConfig struct {
	SourceType   string          `json:"source_type"`             // manual, project_target, file, database, api, previous_stage
	QueryMode    string          `json:"query_mode,omitempty"`    // table, view, sql (database only)
	TargetType   string          `json:"target_type"`             // ip, ip_range, domain, url
	SourceValue  string          `json:"source_value,omitempty"`  // 具体值
	CustomSQL    string          `json:"custom_sql,omitempty"`    // custom_sql
	FilterRules  json.RawMessage `json:"filter_rules,omitempty"`  // 过滤规则
	AuthConfig   json.RawMessage `json:"auth_config,omitempty"`   // 认证配置
	ParserConfig json.RawMessage `json:"parser_config,omitempty"` // 解析配置
}

type targetProviderService struct {
	providers map[string]SourceProvider
	mu        sync.RWMutex
}

// NewTargetProvider 创建目标提供者实例
func NewTargetProvider() TargetProvider {
	svc := &targetProviderService{
		providers: make(map[string]SourceProvider),
	}
	// 注册内置提供者
	svc.RegisterProvider("manual", &ManualProvider{})
	svc.RegisterProvider("project_target", &ProjectTargetProvider{})

	// 注册待实现的提供者 (占位符)
	svc.RegisterProvider("file", &StubProvider{name: "file"})
	svc.RegisterProvider("database", &StubProvider{name: "database"})
	svc.RegisterProvider("api", &StubProvider{name: "api"})
	svc.RegisterProvider("previous_stage", &StubProvider{name: "previous_stage"})

	return svc
}

func (p *targetProviderService) RegisterProvider(name string, provider SourceProvider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.providers[name] = provider
}

// CheckHealth 检查所有已注册 Provider 的健康状态
func (p *targetProviderService) CheckHealth(ctx context.Context) map[string]error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	results := make(map[string]error)
	for name, provider := range p.providers {
		results[name] = provider.HealthCheck(ctx)
	}
	return results
}

// ResolveTargets 解析目标
func (p *targetProviderService) ResolveTargets(ctx context.Context, policyJSON string, seedTargets []string) ([]Target, error) {
	// 1. 如果策略为空，默认使用种子目标
	if policyJSON == "" || policyJSON == "{}" {
		return p.fallbackToSeed(seedTargets), nil
	}

	var config TargetPolicyConfig
	if err := json.Unmarshal([]byte(policyJSON), &config); err != nil {
		logger.LogWarn("Failed to parse target policy, using seed targets", "", 0, "", "ResolveTargets", "", map[string]interface{}{
			"error":  err.Error(),
			"policy": policyJSON,
		})
		return p.fallbackToSeed(seedTargets), nil
	}

	// 2. 如果没有配置源，也使用种子目标
	if len(config.TargetSources) == 0 {
		return p.fallbackToSeed(seedTargets), nil
	}

	// 3. 并发/顺序获取所有目标
	allTargets := make([]Target, 0)
	for _, sourceConfig := range config.TargetSources {
		p.mu.RLock()
		provider, exists := p.providers[sourceConfig.SourceType]
		p.mu.RUnlock()

		if !exists {
			logger.LogWarn("Unknown source type", "", 0, "", "ResolveTargets", "", map[string]interface{}{
				"type": sourceConfig.SourceType,
			})
			continue // 跳过未知的源类型
		}

		targets, err := provider.Provide(ctx, sourceConfig, seedTargets)
		if err != nil {
			logger.LogError(err, "Provider failed to get targets", 0, "", "ResolveTargets", "", map[string]interface{}{
				"type": sourceConfig.SourceType,
			})
			continue // 跳过失败的源
		}
		allTargets = append(allTargets, targets...)
	}

	// 4. 去重 (基于 Value)
	targetSet := make(map[string]struct{})
	result := make([]Target, 0, len(allTargets))
	for _, t := range allTargets {
		if _, ok := targetSet[t.Value]; !ok {
			targetSet[t.Value] = struct{}{}
			result = append(result, t)
		}
	}

	return result, nil
}

// fallbackToSeed 辅助方法：将种子目标转换为 Target 对象
func (p *targetProviderService) fallbackToSeed(seedTargets []string) []Target {
	targets := make([]Target, 0, len(seedTargets))
	for _, t := range seedTargets {
		targets = append(targets, Target{
			Type:   "unknown",
			Value:  t,
			Source: "seed",
			Meta:   nil,
		})
	}
	return targets
}

// --- 具体 Provider 实现 ---

// ManualProvider 人工输入
type ManualProvider struct{}

func (m *ManualProvider) Name() string { return "manual" }

func (m *ManualProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	parts := strings.FieldsFunc(config.SourceValue, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	targets := make([]Target, 0, len(parts))
	for _, part := range parts {
		t := strings.TrimSpace(part)
		if t != "" {
			targets = append(targets, Target{
				Type:   config.TargetType, // 使用配置中的类型
				Value:  t,
				Source: "manual",
				Meta:   nil,
			})
		}
	}
	return targets, nil
}

func (m *ManualProvider) HealthCheck(ctx context.Context) error {
	return nil
}

// ProjectTargetProvider 项目种子目标
type ProjectTargetProvider struct{}

func (p *ProjectTargetProvider) Name() string { return "project_target" }

func (p *ProjectTargetProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	targets := make([]Target, 0, len(seedTargets))
	for _, t := range seedTargets {
		targets = append(targets, Target{
			Type:   config.TargetType, // 假设种子目标类型与配置一致，或者需要自动检测
			Value:  t,
			Source: "project_target",
			Meta:   nil,
		})
	}
	return targets, nil
}

func (p *ProjectTargetProvider) HealthCheck(ctx context.Context) error {
	return nil
}

// StubProvider 桩实现
type StubProvider struct {
	name string
}

func (s *StubProvider) Name() string { return s.name }

func (s *StubProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	logger.LogWarn(fmt.Sprintf("%s source not supported yet", s.name), "", 0, "", "Provide", "", nil)
	return nil, nil
}

func (s *StubProvider) HealthCheck(ctx context.Context) error {
	return nil
}

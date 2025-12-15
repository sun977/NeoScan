// 目标提供者 --- 负责目标数据的生成和清洗(包含目标过滤)
// 处理多元异构数据，将其转换为统一的 Target 格式作为扫描目标
// 工作流程如下：
// 1.初始化：创建TargetProvider实例并注册各种SourceProvider（包括实际实现和桩实现）
// 2.策略解析：通过ResolveTargets方法解析目标策略：
// - 如果策略为空，使用种子目标作为回退
// - 解析JSON策略配置
// - 遍历所有目标源配置
// 3.目标获取：针对每个目标源配置：
// - 根据源类型查找对应的Provider
// - 调用Provider的Provide方法获取目标
// - 处理异常情况（未知源类型或获取失败）
// 4.结果处理：
// - 合并所有获取到的目标
// - 基于目标值进行去重
// - 返回统一格式的目标列表
// 这种设计使得系统可以灵活地从多种来源获取扫描目标，并将其统一为标准格式，便于后续处理。
// 注意：provider仅负责搬运数据
// - 所有提供者都遵循愚蠢哲学，即所有输入都会原样封装给 Target 对象
// - 提供者不负责校验输入的有效性，也不负责解析输入的格式
// targetProvider 和 policyProvider 模块都具有白名单和跳过策略的功能，但是区别如下：
// - targetProvider 模块的白名单和跳过策略来源有多重，可以是file,API,DB,用户指定或者是上一个阶段的输出,更灵活
// - policyProvider 模块的白名单和跳过策略来源只有一种，即DB,而且他是强制全局阻断的，更像合规最后一道检测

package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/matcher"

	"gorm.io/gorm"
)

// 定义 Context Key 类型以避免冲突
// 这些 Key 用于在 Context 中传递 Workflow 和 Stage 信息，供 Provider 使用
type ContextKey string

const (
	// CtxKeyProjectID ProjectID 上下文键
	CtxKeyProjectID ContextKey = "project_id"
	// CtxKeyWorkflowID WorkflowID 上下文键
	CtxKeyWorkflowID ContextKey = "workflow_id"
	// CtxKeyStageID StageID 上下文键 (当前 Stage 的 ID)
	CtxKeyStageID ContextKey = "current_stage_id"
	// CtxKeyStageOrder StageOrder 上下文键 (Deprecated: Use CtxKeyStageID instead)
	CtxKeyStageOrder ContextKey = "current_stage_order"
)

// TargetProvider 目标提供者服务接口
type TargetProvider interface {
	ResolveTargets(ctx context.Context, policyJSON string, seedTargets []string) ([]Target, error) // ResolveTargets 解析策略并返回目标列表
	RegisterProvider(name string, provider SourceProvider)                                         // RegisterProvider 注册新的目标源提供者
	CheckHealth(ctx context.Context) map[string]error                                              // CheckHealth 检查所有已注册 Provider 的健康状态
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
	TargetSources    []TargetSourceConfig `json:"target_sources"`
	WhitelistEnabled bool                 `json:"whitelist_enabled"` // 是否启用白名单
	WhitelistSources []TargetSourceConfig `json:"whitelist_sources"` // 白名单来源 (值匹配)
	WhitelistRule    matcher.MatchRule    `json:"whitelist_rule"`    // 白名单规则 (逻辑匹配)
	SkipEnabled      bool                 `json:"skip_enabled"`      // 是否启用跳过
	SkipRule         matcher.MatchRule    `json:"skip_rule"`         // 跳过规则 (逻辑匹配)
}

// TargetSourceConfig 目标源配置详细结构
// 对应设计文档: 配置结构 (TargetSource Config)
type TargetSourceConfig struct {
	SourceType   string          `json:"source_type"`             // manual, project_target, file, database, api, previous_stage
	QueryMode    string          `json:"query_mode,omitempty"`    // table, view, sql (仅用于数据库)
	TargetType   string          `json:"target_type"`             // ip, ip_range, domain, url
	SourceValue  string          `json:"source_value,omitempty"`  // 具体值
	CustomSQL    string          `json:"custom_sql,omitempty"`    // custom_sql
	FilterRules  json.RawMessage `json:"filter_rules,omitempty"`  // 过滤规则
	AuthConfig   json.RawMessage `json:"auth_config,omitempty"`   // 认证配置
	ParserConfig json.RawMessage `json:"parser_config,omitempty"` // 解析配置
}

// 定义目标提供者服务
type targetProviderService struct {
	providers map[string]SourceProvider // 储存已注册的目标源提供者
	mu        sync.RWMutex              // 读写互斥锁, 保护 providers  map 并发访问
}

// NewTargetProvider 创建目标提供者实例
func NewTargetProvider(db *gorm.DB) TargetProvider {
	svc := &targetProviderService{
		providers: make(map[string]SourceProvider),
	}
	// 注册内置提供者
	svc.RegisterProvider("manual", &ManualProvider{})                    // 手动输入来源提供(前端用户手动输入目标)
	svc.RegisterProvider("project_target", &ProjectTargetProvider{})     // 项目种子目标提供(直接用项目目标覆盖当前目标)
	svc.RegisterProvider("file", &FileProvider{})                        // 注册文件提供者
	svc.RegisterProvider("database", NewDatabaseProvider(db))            // 注册数据库提供者
	svc.RegisterProvider("previous_stage", NewPreviousStageProvider(db)) // 注册上一阶段结果提供者
	// 注册待实现的提供者
	svc.RegisterProvider("api", &ApiProvider{})

	return svc
}

// RegisterProvider 注册新的目标源提供者
func (p *targetProviderService) RegisterProvider(name string, provider SourceProvider) {
	p.mu.Lock() // 使用写锁保证并发安全
	defer p.mu.Unlock()
	p.providers[name] = provider // 注册新的目标源提供者
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

// ResolveTargets 解析策略并返回目标列表 --- 核心实现逻辑
// 策略解析器：对应 ScanStage.target_policy
// 1. 如果策略为空，默认使用种子目标
// 2. 不为空解析 json 策略
// 3. 并发/顺序获取所有目标
// 4. 白名单过滤 (如果启用)
// 5. 跳过条件过滤 (如果启用)
// 6. 去重 (基于 Value)
// 策略样例：
//
//	{
//	  "target_sources": [
//	    {
//	      "source_type": "file",           // 来源类型：file/db/view/sql/manual/api/previous_stage
//	      "source_value": "/path/to/targets.txt",
//	      "target_type": "ip_range"
//	    }
//	  ],
//	  "whitelist_enabled": true,           // 是否启用白名单
//	  "whitelist_sources": [               // 1. 基于来源的白名单 (值匹配)
//	    {
//	      "source_type": "file",
//	      "source_value": "/path/to/whitelist.txt"
//	    }
//	  ],
//	  "whitelist_rule": {                  // 2. 基于规则的白名单 (逻辑匹配)
//	      "field": "value",
//	      "operator": "cidr",
//	      "value": "10.0.0.0/8",
//	      "ignore_case": true // 可选：忽略大小写
//	  },
//	  "skip_enabled": true,                // 是否启用跳过条件
//	  "skip_rule": {                       // 跳过规则 (逻辑匹配，支持 AND/OR/NOT 嵌套)
//	      "or": [
//	          { "field": "device_type", "operator": "equals", "value": "honeypot", "ignore_case": true }, // 可选：忽略大小写
//	          { "field": "type", "operator": "equals", "value": "url" }
//	      ]
//	  }
//	}
func (p *targetProviderService) ResolveTargets(ctx context.Context, policyJSON string, seedTargets []string) ([]Target, error) {
	logger.LogInfo("[TargetProvider] ResolveTargets called", "", 0, "", "ResolveTargets", "", map[string]interface{}{"policy": policyJSON})

	// 1. 如果策略为空，默认使用种子目标
	if policyJSON == "" || policyJSON == "{}" {
		return p.fallbackToSeed(seedTargets), nil
	}

	// 2. 解析策略配置
	var config TargetPolicyConfig
	if err := json.Unmarshal([]byte(policyJSON), &config); err != nil {
		logger.LogError(err, "", 0, "", "ResolveTargets", "", nil)
		return nil, fmt.Errorf("invalid policy json: %w", err)
	}
	logger.LogInfo("[TargetProvider] Sources count", "", 0, "", "ResolveTargets", "", map[string]interface{}{"count": len(config.TargetSources)})

	// 3. 如果没有配置源，默认使用种子目标
	if len(config.TargetSources) == 0 {
		return p.fallbackToSeed(seedTargets), nil
	}

	// 3. 获取初始目标列表
	allTargets := p.fetchTargetsFromSources(ctx, config.TargetSources, seedTargets)

	// 4. 白名单过滤 (如果启用)
	if config.WhitelistEnabled {
		// 4.1 基于来源的白名单 (值匹配)
		if len(config.WhitelistSources) > 0 {
			whitelistTargets := p.fetchTargetsFromSources(ctx, config.WhitelistSources, seedTargets)
			whitelistMap := make(map[string]struct{})
			for _, t := range whitelistTargets {
				whitelistMap[t.Value] = struct{}{} // 构建一个白名单列表
			}

			filtered := make([]Target, 0)
			for _, t := range allTargets { // 基于值的白名单过滤
				if _, ok := whitelistMap[t.Value]; ok { // 如果目标值在白名单中
					filtered = append(filtered, t) // 加入过滤后的列表
				}
			}
			allTargets = filtered
		}

		// 4.2 基于规则的白名单 (逻辑匹配)
		// 如果定义了规则，只保留匹配规则的目标
		if !matcher.IsEmptyRule(config.WhitelistRule) { // 如果规则不为空
			filtered := make([]Target, 0)
			for _, t := range allTargets {
				// 转换为 Map 以便 matcher 匹配 (支持 type, value 等小写字段)
				targetMap := p.targetToMap(t)
				matched, err := matcher.Match(targetMap, config.WhitelistRule) // 评估目标是否符合白名单规则
				if err != nil {
					logger.LogWarn("Failed to match whitelist rule", "", 0, "", "ResolveTargets", "", map[string]interface{}{
						"error":  err.Error(),
						"target": t.Value,
					})
					// 策略：出错时默认为不匹配（安全起见）
					continue
				}
				if matched {
					filtered = append(filtered, t)
				}
			}
			allTargets = filtered
		}
	}

	// 5. 跳过条件过滤 (如果启用)
	if config.SkipEnabled && !matcher.IsEmptyRule(config.SkipRule) {
		filtered := make([]Target, 0)
		for _, t := range allTargets {
			targetMap := p.targetToMap(t)
			matched, err := matcher.Match(targetMap, config.SkipRule)
			if err != nil {
				logger.LogWarn("Failed to match skip rule", "", 0, "", "ResolveTargets", "", map[string]interface{}{
					"error":  err.Error(),
					"target": t.Value,
				})
				// 策略：出错时不跳过（保留目标）
				filtered = append(filtered, t)
				continue
			}

			// 如果匹配规则，则跳过 (不加入 filtered)
			if !matched {
				filtered = append(filtered, t)
			}
		}
		allTargets = filtered
	}

	// 6. 去重 (基于 Value)
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

// fetchTargetsFromSources 从指定源列表获取目标
func (p *targetProviderService) fetchTargetsFromSources(ctx context.Context, sources []TargetSourceConfig, seedTargets []string) []Target {
	allTargets := make([]Target, 0)
	for _, sourceConfig := range sources {
		fmt.Printf("[TargetProvider] Processing source type: %s\n", sourceConfig.SourceType)
		p.mu.RLock()
		provider, exists := p.providers[sourceConfig.SourceType]
		p.mu.RUnlock()

		if !exists {
			fmt.Printf("[TargetProvider] Provider not found: %s\n", sourceConfig.SourceType)
			logger.LogWarn("Unknown target source type", "", 0, "", "fetchTargetsFromSources", "", map[string]interface{}{
				"type": sourceConfig.SourceType,
			})
			continue
		}

		targets, err := provider.Provide(ctx, sourceConfig, seedTargets)
		if err != nil {
			fmt.Printf("[TargetProvider] Provider error: %v\n", err)
			logger.LogError(err, "", 0, "", "fetchTargetsFromSources", "PROVIDER_ERROR", map[string]interface{}{
				"type": sourceConfig.SourceType,
			})
			continue
		}
		fmt.Printf("[TargetProvider] Provider returned %d targets\n", len(targets))
		allTargets = append(allTargets, targets...)
	}
	return allTargets
}

// targetToMap 将 Target 转换为 Map 以供 matcher 使用
func (p *targetProviderService) targetToMap(t Target) map[string]interface{} {
	m := map[string]interface{}{
		"type":   t.Type,
		"value":  t.Value,
		"source": t.Source,
		"meta":   t.Meta,
	}
	// 将 meta 字段提升到顶层，方便规则直接访问 (如 device_type 而不是 meta.device_type)
	// 如果有冲突，优先使用 meta 中的值 (或者反过来，这里选择优先保留 type/value/source)
	for k, v := range t.Meta {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return m
}

// fallbackToSeed 辅助方法：将种子目标转换为 Target 对象
func (p *targetProviderService) fallbackToSeed(seedTargets []string) []Target {
	targets := make([]Target, 0, len(seedTargets))
	for _, t := range seedTargets {
		targetType := "unknown"
		if net.ParseIP(t) != nil {
			targetType = "ip"
		} else if _, _, err := net.ParseCIDR(t); err == nil {
			targetType = "ip_range"
		} else if strings.Contains(t, "://") {
			targetType = "url"
		} else if strings.Contains(t, ".") && !strings.Contains(t, " ") { // Simple domain check
			targetType = "domain"
		}

		targets = append(targets, Target{
			Type:   targetType,
			Value:  t,
			Source: "seed",
			Meta:   make(map[string]string),
		})
	}
	return targets
}

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
	"fmt"
	"net"
	"strings"
	"sync"

	orcmodel "neomaster/internal/model/orchestrator"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/matcher"

	"gorm.io/gorm"
	// 定义 Context Key 类型以避免冲突
	// 这些 Key 用于在 Context 中传递 Workflow 和 Stage 信息，供 Provider 使用
)

// CtxKey ContextKey 类型
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
// 对应设计文档中的 TargetProvider Interface
type TargetProvider interface {
	ResolveTargets(ctx context.Context, targetPolicy orcmodel.TargetPolicy, seedTargets []string) ([]Target, error) // ResolveTargets 解析策略并返回目标列表
	RegisterProvider(name string, provider SourceProvider)                                                          // RegisterProvider 注册新的目标源提供者
	CheckHealth(ctx context.Context) map[string]error                                                               // CheckHealth 检查所有已注册 Provider 的健康状态
}

// SourceProvider 单个目标源提供者接口
// 对应设计文档中的 TargetProvider Interface
type SourceProvider interface {
	// Name 返回 Provider 名称
	Name() string

	// Provide 执行获取逻辑
	// ctx: 上下文
	// config: 目标源配置 TargetSource 包含源类型、源值、元数据等信息
	// seedTargets: 种子目标(用于 project_target)
	Provide(ctx context.Context, config orcmodel.TargetSource, seedTargets []string) ([]Target, error)

	// HealthCheck 检查 Provider 健康状态 (如数据库连接、文件访问权限)
	HealthCheck(ctx context.Context) error
}

// Target 标准目标对象
// 对应设计文档: 运行时对象 (Target Object)
type Target = orcmodel.Target

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
func (p *targetProviderService) ResolveTargets(ctx context.Context, targetPolicy orcmodel.TargetPolicy, seedTargets []string) ([]Target, error) {
	logger.LogInfo("[TargetProvider] ResolveTargets called", "", 0, "", "ResolveTargets", "", map[string]interface{}{"targetPolicy_sources_count": len(targetPolicy.TargetSources)})

	// 1. 如果没有配置源，默认使用种子目标
	if len(targetPolicy.TargetSources) == 0 {
		return p.fallbackToSeed(seedTargets), nil
	}

	// 3. 获取初始目标列表 (并发)
	allTargets := p.fetchTargetsFromSources(ctx, targetPolicy.TargetSources, seedTargets)

	// 4. 白名单过滤 (如果启用)
	if targetPolicy.WhitelistEnabled {
		allTargets = p.applyWhitelist(ctx, allTargets, targetPolicy, seedTargets)
	}

	// 5. 跳过条件过滤 (如果启用)
	if targetPolicy.SkipEnabled && !matcher.IsEmptyRule(targetPolicy.SkipRule) {
		allTargets = p.applySkipRule(allTargets, targetPolicy.SkipRule)
	}

	// 6. 去重 (基于 Value)
	result := p.deduplicateTargets(allTargets)

	return result, nil
}

// applyWhitelist 应用白名单过滤
// 白名单中的资产不会进入扫描流程，因此在白名单中的资产应该被忽略或者从目标列表中移除。
// 白名单有两种类型：
// 1. SourceType = file 文件白名单：从文件中读取资产列表，格式为每行一个资产值。要转换成["192.168.1.1","192.168.1.2"]
// 2. SourceType =manual 手动白名单：用户手动输入资产列表，格式为["192.168.1.1","192.168.1.2"]。
// target.Value 是资产值，需要与白名单中的值进行比较。
func (p *targetProviderService) applyWhitelist(ctx context.Context, targets []Target, targetPolicy orcmodel.TargetPolicy, seedTargets []string) []Target {
	if len(targetPolicy.WhitelistSources) == 0 {
		return targets
	}

	// 转换 WhitelistSource 为 TargetSource 以复用 Provider
	whitelistTargetSources := make([]orcmodel.TargetSource, len(targetPolicy.WhitelistSources))
	for i, ws := range targetPolicy.WhitelistSources {
		whitelistTargetSources[i] = orcmodel.TargetSource{
			SourceType:  ws.SourceType,
			SourceValue: ws.SourceValue,
			TargetType:  "whitelist", // 标记类型，Provider 可按需使用
		}
	}

	// 复用 fetchTargetsFromSources 获取白名单目标列表
	whitelistTargets := p.fetchTargetsFromSources(ctx, whitelistTargetSources, seedTargets)

	// 构建白名单 Map
	whitelistMap := make(map[string]struct{})
	for _, t := range whitelistTargets {
		whitelistMap[t.Value] = struct{}{}
	}

	// 基于值的白名单过滤 (黑名单行为：命中则移除)
	filtered := make([]Target, 0)
	for _, t := range targets {
		if _, ok := whitelistMap[t.Value]; !ok { // 只有不在白名单中的目标才保留
			filtered = append(filtered, t)
		}
	}

	// 注意：根据用户要求，移除了基于规则的白名单逻辑 (targetPolicy.WhitelistRule)

	return filtered
}

// applySkipRule 应用跳过规则过滤
func (p *targetProviderService) applySkipRule(targets []Target, rule matcher.MatchRule) []Target {
	filtered := make([]Target, 0)
	for _, t := range targets {
		targetMap := p.targetToMap(t)
		matched, err := matcher.Match(targetMap, rule)
		if err != nil {
			logger.LogWarn("Failed to match skip rule", "", 0, "", "applySkipRule", "", map[string]interface{}{
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
	return filtered
}

// deduplicateTargets 去重 (基于 Value)
func (p *targetProviderService) deduplicateTargets(targets []Target) []Target {
	targetSet := make(map[string]struct{})
	result := make([]Target, 0, len(targets))
	for _, t := range targets {
		if _, ok := targetSet[t.Value]; !ok {
			targetSet[t.Value] = struct{}{}
			result = append(result, t)
		}
	}
	return result
}

// fetchTargetsFromSources 从指定源列表获取目标 (并发优化)
func (p *targetProviderService) fetchTargetsFromSources(ctx context.Context, sources []orcmodel.TargetSource, seedTargets []string) []Target {
	var wg sync.WaitGroup
	// 避免过大的缓冲，但足够容纳所有 sources
	targetChan := make(chan []Target, len(sources))

	for _, sourceConfig := range sources {
		wg.Add(1)
		go func(cfg orcmodel.TargetSource) {
			defer wg.Done()
			fmt.Printf("[TargetProvider] Processing source type: %s\n", cfg.SourceType)
			p.mu.RLock()
			provider, exists := p.providers[cfg.SourceType]
			p.mu.RUnlock()

			if !exists {
				fmt.Printf("[TargetProvider] Provider not found: %s\n", cfg.SourceType)
				logger.LogWarn("Unknown target source type", "", 0, "", "fetchTargetsFromSources", "", map[string]interface{}{
					"type": cfg.SourceType,
				})
				return
			}

			targets, err := provider.Provide(ctx, cfg, seedTargets)
			if err != nil {
				fmt.Printf("[TargetProvider] Provider error: %v\n", err)
				logger.LogError(err, "", 0, "", "fetchTargetsFromSources", "PROVIDER_ERROR", map[string]interface{}{
					"type": cfg.SourceType,
				})
				return
			}
			fmt.Printf("[TargetProvider] Provider returned %d targets\n", len(targets))
			targetChan <- targets
		}(sourceConfig)
	}

	// 等待所有 goroutine 完成后关闭 channel
	go func() {
		wg.Wait()
		close(targetChan)
	}()

	allTargets := make([]Target, 0)
	for targets := range targetChan {
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
	// 将 meta.Custom 字段提升到顶层，方便规则直接访问 (如 device_type 而不是 meta.custom.device_type)
	// 如果有冲突，优先使用 meta 中的值 (或者反过来，这里选择优先保留 type/value/source)
	for k, v := range t.Meta.Custom {
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
			Meta:   orcmodel.TargetMeta{},
		})
	}
	return targets
}

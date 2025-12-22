// PolicyEnforcer 策略执行器接口
// 职责: 在任务下发前的最后一道防线，负责"安检"与合规。
// 对应文档: 1.2 Policy Enforcer (策略执行器)
package policy

import (
	"context"
	"encoding/json"
	"fmt"
	agentModel "neomaster/internal/model/orchestrator"
	"net"
	"net/url"
	"strings"
	"time"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/matcher"
	"neomaster/internal/pkg/utils"
	assetrepo "neomaster/internal/repo/mysql/asset"
	orcrepo "neomaster/internal/repo/mysql/orchestrator"
)

// PolicyEnforcer 策略执行器接口
// 职责: 在任务下发前的最后一道防线，负责"安检"与合规。
// 对应文档: 1.2 Policy Enforcer (策略执行器)
type PolicyEnforcer interface {
	// Enforce 检查任务是否符合策略 (Whitelist, Scope, SkipLogic)
	// 如果返回 error，则任务不应下发
	Enforce(ctx context.Context, task *agentModel.AgentTask) error // 检查任务是否符合策略 (Whitelist, Scope, SkipLogic)
}

// 策略执行器实现结构体
type policyEnforcer struct {
	projectRepo *orcrepo.ProjectRepository       // 项目仓库
	policyRepo  *assetrepo.AssetPolicyRepository // 资产策略仓库
}

// NewPolicyEnforcer 创建策略执行器
// 问题：策略执行器执行对象是 ScanStage 生成的还没下发的 agenttask 不应该是 project
// ScanStage.target_policy JSON 中定义了目标策略，同时数据库中也存储了一套策略，这两套策略都需要满足
// 白名单检验：scanStage.target_policy.whitelist_enabled 为 true 时，需要检查任务目标是否在白名单中(whitelist_sources + 数据库中存储的白名单)
// - ScanStage.target_policy.whitelist_sources 中的默认只执行跳过动作，如果是db类型，使用白名单的时候需要注意白名单的作用域Scope，白名单中的目标只能在当前作用域中生效
// 跳过条件检验：scanStage.target_policy.skip_enabled 为 true 时，需要检查任务目标是否符合跳过条件(skip_conditions + 数据库中存储的跳过条件)
// - ScanStage.target_policy.skip_conditions 中的跳过策略默认只执行跳过扫描动作(skip)，数据库中的跳过策略支持命中后执行动作配置(如：skip,tag,log,alert 动作)
//
//	{
//	  "target_sources": [
//	    {
//	      "source_type": "file",           // 来源类型：file/db/view/sql/manual/api/previous_stage【上一个阶段结果】
//	      "source_value": "/path/to/targets.txt",  // 根据类型的具体值
//	      "target_type": "ip_range"        // 目标类型：ip/ip_range/domain/url
//	    }
//	  ],
//	  "whitelist_enabled": true,           // 是否启用白名单
//	  "whitelist_sources": [               // 白名单来源/数据库/文件/手动输入
//	    {
//	      "source_type": "file",   	// 白名单来源类型：file/db/manual
//	      "source_value": "/path/to/whitelist.txt" // file 对应文件路径, db 对应默认白名单表(默认写死), manual 对应手动输入内容
//	    }
//	  ],
//	  "skip_enabled": true,                // 是否启用跳过条件
//	  "skip_conditions": [                 // 跳过条件,列表中可添加多个条件，这里写的条件直接执行跳过动作
//	    {
//	      "condition_field": "device_type",
//	      "operator": "equals",
//	      "value": "honeypot"
//	    }
//	  ]
//	}
func NewPolicyEnforcer(projectRepo *orcrepo.ProjectRepository, policyRepo *assetrepo.AssetPolicyRepository) PolicyEnforcer {
	return &policyEnforcer{
		projectRepo: projectRepo,
		policyRepo:  policyRepo,
	}
}

// Enforce 执行策略检查 --- 核心实现逻辑
// 职责: 校验任务是否符合项目策略 (Scope, Whitelist, SkipLogic)
// 参数:
//   - ctx: 上下文，用于日志记录与取消操作
//   - task: 待校验的任务
//
// 返回值:
//   - error: 如果任务不符合策略，返回具体错误信息
func (p *policyEnforcer) Enforce(ctx context.Context, task *agentModel.AgentTask) error {
	// 1. ScopeValidator: 范围校验
	// 确保扫描目标严格限制在 Project 定义的 TargetScope 内
	if task.InputTarget == "" {
		return fmt.Errorf("policy violation: target is empty")
	}

	// 获取项目 Scope 配置
	// 注意: 这是一个数据库查询，为了性能，可以使用缓存
	// 获取项目信息,判断项目是否存在
	project, err := p.projectRepo.GetProjectByID(ctx, task.ProjectID)
	if err != nil {
		logger.LogError(err, "failed to get project for policy check", 0, "", "service.orchestrator.policy.Enforce", "REPO", map[string]interface{}{
			"project_id": task.ProjectID,
		})
		return fmt.Errorf("policy check failed: cannot get project")
	}
	if project == nil {
		return fmt.Errorf("policy violation: project not found")
	}

	// 解析 InputTarget (可能是 JSON 列表或单个字符串)
	targets, err := parseTargets(task.InputTarget)
	if err != nil {
		// 尝试作为单个字符串处理
		targets = []string{task.InputTarget}
	}

	// 校验所有目标是否在 Scope 内
	if err1 := validateScope(targets, project.TargetScope); err1 != nil {
		return err1
	}

	// 2. WhitelistChecker: 白名单检查 (强制阻断)
	// 检查目标是否命中 AssetWhitelist
	for _, target := range targets {
		isBlocked, ruleName, err2 := p.checkWhitelist(ctx, target)
		if err2 != nil {
			logger.LogError(err2, "whitelist check error", 0, "", "service.orchestrator.policy.Enforce", "REPO", nil)
			return fmt.Errorf("policy check error: %v", err2)
		}
		if isBlocked {
			logger.LogInfo("Task blocked by whitelist", "", 0, "", "service.orchestrator.policy.Enforce", "", map[string]interface{}{
				"task_id":   task.TaskID,
				"target":    target,
				"rule_name": ruleName,
			})
			return fmt.Errorf("policy violation: target %s is whitelisted by rule: %s", target, ruleName)
		}
	}

	// 3. SkipLogicEvaluator: 动态跳过
	// 执行 AssetSkipPolicy 逻辑 (e.g. 生产环境限制)
	shouldSkip, reason, err := p.checkSkipPolicy(ctx, project)
	if err != nil {
		logger.LogError(err, "skip policy check error", 0, "", "service.orchestrator.policy.Enforce", "REPO", nil)
		return fmt.Errorf("policy check error: %v", err)
	}
	if shouldSkip {
		logger.LogInfo("Task skipped by policy", "", 0, "", "service.orchestrator.policy.Enforce", "", map[string]interface{}{
			"task_id": task.TaskID,
			"reason":  reason,
		})
		return fmt.Errorf("policy violation: task skipped due to policy: %s", reason)
	}

	return nil
}

// checkWhitelist 检查目标是否在白名单中
func (p *policyEnforcer) checkWhitelist(ctx context.Context, target string) (bool, string, error) {
	whitelists, err := p.policyRepo.GetEnabledWhitelists(ctx)
	if err != nil {
		return false, "", err
	}

	// 预处理目标：尝试提取 Host (IP 或 域名)
	// 如果 target 是 URL，提取 Host；否则 target 本身就是 Host
	targetHost := target
	// 只有当看起来像 URL 时才尝试解析 Host
	if strings.Contains(target, "://") {
		if u, err := url.Parse(target); err == nil && u.Host != "" {
			targetHost = u.Host
		}
	}
	// 去掉端口号 (无论是从 URL 解析出来的，还是原始的 "IP:Port")
	if h, _, err := net.SplitHostPort(targetHost); err == nil {
		targetHost = h
	}

	for _, w := range whitelists {
		match := false
		switch w.TargetType {
		case "ip", "ip_range", "cidr":
			// 支持单个IP, CIDR(10.0.0.0/24), IP范围(192.168.1.0-192.168.1.255)
			// 使用统一的 CheckIPInRange 函数检查
			// 注意：这里用 targetHost，因为 target 可能是 URL
			isMatch, err := utils.CheckIPInRange(targetHost, w.TargetValue)
			if err == nil && isMatch {
				match = true
			}
		case "domain", "host", "domain_pattern":
			// 域名匹配：精确匹配或后缀匹配
			// 使用 targetHost
			if targetHost == w.TargetValue {
				match = true
			} else if strings.HasPrefix(w.TargetValue, ".") && strings.HasSuffix(targetHost, w.TargetValue) {
				match = true
			} else if strings.HasPrefix(w.TargetValue, "*.") && strings.HasSuffix(targetHost, w.TargetValue[1:]) {
				// 支持 *.example.com 格式
				match = true
			}
		case "url":
			// URL 匹配：前缀匹配
			// 使用原始 target
			if strings.HasPrefix(target, w.TargetValue) {
				match = true
			}
		case "keyword":
			// 关键字包含
			// 使用原始 target
			if strings.Contains(target, w.TargetValue) {
				match = true
			}
		default:
			// 忽略不支持的类型
			continue
		}

		if match {
			return true, w.WhitelistName, nil
		}
	}

	return false, "", nil
}

// SkipConditionRules 跳过条件规则
// 定义在 AssetSkipPolicy 中的条件，用于动态判断是否跳过任务
type SkipConditionRules struct {
	BlockTimeWindows []string          `json:"block_time_windows"` // e.g. ["00:00-06:00", "22:00-23:59"]
	BlockEnvTags     []string          `json:"block_env_tags"`     // e.g. ["production", "sensitive"]
	MatchRule        matcher.MatchRule `json:"match_rule"`         // 复杂匹配规则 (New)
}

// checkSkipPolicy 检查是否应该跳过任务
// 1. 支持传统的时间窗口和环境标签检查
// 2. 支持基于 Matcher 的复杂规则检查
func (p *policyEnforcer) checkSkipPolicy(ctx context.Context, project *agentModel.Project) (bool, string, error) {
	policies, err := p.policyRepo.GetEnabledSkipPolicies(ctx)
	if err != nil {
		return false, "", err
	}

	now := time.Now()
	currentTimeStr := now.Format("15:04") // HH:MM
	projectTags := parseTags(project.Tags)

	// 构建用于 Matcher 的上下文数据
	// 包含项目信息、时间和环境标签
	matchContext := map[string]interface{}{
		"project_id":   project.ID,
		"project_name": project.Name,
		"tags":         projectTags, // []string
		"time":         currentTimeStr,
		"weekday":      now.Weekday().String(),
		"hour":         now.Hour(),
	}

	for _, policy := range policies {
		var rules SkipConditionRules
		if err := json.Unmarshal([]byte(policy.ConditionRules), &rules); err != nil {
			logger.LogWarn("Failed to parse skip policy rules", "", 0, "", "checkSkipPolicy", "", map[string]interface{}{
				"policy_id": policy.ID,
				"error":     err.Error(),
			})
			continue // 忽略解析失败的规则
		}

		// 构建聚合规则 (Root Rule)
		// 逻辑: LegacyTags OR LegacyTime OR NewMatchRule
		// 只要命中其中任何一个，就视为跳过
		var rootRule matcher.MatchRule
		rootRule.Or = make([]matcher.MatchRule, 0)

		// 1. 转换 Legacy Environment Tags -> MatchRule
		if len(rules.BlockEnvTags) > 0 {
			// 逻辑: 只要项目包含任意一个 BlockEnvTag，即命中
			// tags list_contains tag1 OR tags list_contains tag2 ...
			for _, blockTag := range rules.BlockEnvTags {
				rootRule.Or = append(rootRule.Or, matcher.MatchRule{
					Field:      "tags",
					Operator:   "list_contains",
					Value:      blockTag,
					IgnoreCase: true, // 忽略大小写
				})
			}
		}

		// 2. 转换 Legacy Time Windows -> MatchRule
		if len(rules.BlockTimeWindows) > 0 {
			// 逻辑: 当前时间在任意一个时间窗内
			// (time >= start AND time <= end) OR ...
			for _, window := range rules.BlockTimeWindows {
				parts := strings.Split(window, "-")
				if len(parts) != 2 {
					continue
				}
				start := strings.TrimSpace(parts[0])
				end := strings.TrimSpace(parts[1])

				timeWindowRule := matcher.MatchRule{
					And: []matcher.MatchRule{
						{Field: "time", Operator: "greater_than_or_equal", Value: start},
						{Field: "time", Operator: "less_than_or_equal", Value: end},
					},
				}
				rootRule.Or = append(rootRule.Or, timeWindowRule)
			}
		}

		// 3. 合并新的 MatchRule
		if !matcher.IsEmptyRule(rules.MatchRule) {
			rootRule.Or = append(rootRule.Or, rules.MatchRule)
		}

		// 如果没有任何规则，跳过
		if len(rootRule.Or) == 0 {
			continue
		}

		// 执行匹配
		matched, err := matcher.Match(matchContext, rootRule)
		if err != nil {
			logger.LogWarn("Failed to execute skip match rule", "", 0, "", "checkSkipPolicy", "", map[string]interface{}{
				"policy": policy.PolicyName,
				"error":  err.Error(),
			})
			continue
		}

		if matched {
			return true, fmt.Sprintf("Matched skip policy: %s", policy.PolicyName), nil
		}
	}

	return false, "", nil
}

func parseTags(tagsStr string) []string {
	// 尝试解析 JSON
	var tags []string
	if err := json.Unmarshal([]byte(tagsStr), &tags); err == nil {
		return tags
	}
	// 尝试逗号分隔
	return strings.Split(tagsStr, ",")
}

// parseTargets 解析目标
func parseTargets(input string) ([]string, error) {
	// 1. 尝试解析为字符串数组 (Legacy format: ["1.1.1.1", "2.2.2.2"])
	var strTargets []string
	if err := json.Unmarshal([]byte(input), &strTargets); err == nil {
		return strTargets, nil
	}

	// 2. 尝试解析为 Target 对象数组 (New format: [{"type":"ip", "value":"1.1.1.1", ...}])
	var objTargets []Target
	if err := json.Unmarshal([]byte(input), &objTargets); err == nil {
		targets := make([]string, len(objTargets))
		for i, t := range objTargets {
			targets[i] = t.Value
		}
		return targets, nil
	} else {
		logger.LogWarn("Failed to parse targets as objects", "", 0, "", "parseTargets", "", map[string]interface{}{
			"input": input,
			"error": err.Error(),
		})
	}

	return nil, fmt.Errorf("failed to parse targets")
}

// validateScope 校验目标是否在范围内
func validateScope(targets []string, scope string) error {
	if scope == "" {
		// 如果 Scope 为空，默认允许所有 (或者默认拒绝，取决于安全策略)
		// 这里为了安全，建议如果 Scope 为空则仅允许特定类型，或者默认拒绝
		// 但为了易用性，暂时默认允许
		return nil
	}

	allowedScopes := strings.Split(scope, ",")
	// 简单的包含检查或 CIDR 检查
	// TODO: 实现更强大的 CIDR / Domain 匹配
	for _, target := range targets {
		inScope := false
		for _, s := range allowedScopes {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			// 1. 精确匹配
			if target == s {
				inScope = true
				break
			}
			// 2. 简单的后缀匹配 (Domain)
			if strings.HasPrefix(s, ".") && strings.HasSuffix(target, s) {
				inScope = true
				break
			}
			// 3. CIDR 匹配 (IP)
			if _, ipNet, err := net.ParseCIDR(s); err == nil {
				// 尝试分离 Host 和 Port (处理 ip:port 情况)
				targetIPStr := target
				if h, _, err := net.SplitHostPort(target); err == nil {
					targetIPStr = h
				}

				if ip := net.ParseIP(targetIPStr); ip != nil {
					if ipNet.Contains(ip) {
						inScope = true
						break
					}
				}
			} else {
				// 4. 单个 IP 匹配 (支持 ip:port)
				// 如果 scope 是单个 IP (例如 192.168.1.1)，ParseCIDR 会失败
				// 我们需要处理这种情况
				scopeIP := net.ParseIP(s)
				if scopeIP != nil {
					targetIPStr := target
					if h, _, err := net.SplitHostPort(target); err == nil {
						targetIPStr = h
					}
					targetIP := net.ParseIP(targetIPStr)
					if targetIP != nil && targetIP.Equal(scopeIP) {
						inScope = true
						break
					}
				}
			}
		}

		if !inScope {
			return fmt.Errorf("policy violation: target %s is not in project scope", target)
		}
	}
	return nil
}

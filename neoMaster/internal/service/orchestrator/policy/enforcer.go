// PolicyEnforcer 策略执行器接口
// 职责: 在任务下发前的最后一道防线，负责"安检"与合规。
// 对应文档: 1.2 Policy Enforcer (策略执行器)
// 检查输入目标 InputTarget 是否符合策略配置 PolicySnapshot
package policy

import (
	"context"
	"encoding/json"
	"fmt"
	agentModel "neomaster/internal/model/orchestrator"
	"net"
	"net/url"
	"strings"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/matcher"
	"neomaster/internal/pkg/utils"
	assetrepo "neomaster/internal/repo/mysql/asset"
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
	// projectRepo *orcrepo.ProjectRepository // 移除 ProjectRepository 依赖，不再查 Project 表
	policyRepo *assetrepo.AssetPolicyRepository // 资产策略仓库
}

// NewPolicyEnforcer 创建策略执行器
// 策略执行器执行对象是 ScanStage 生成的还没下发的 agentTask
// 上游的 TargetProvider 会根据 ScanStage.target_policy 的配置进行白名单和条件逻辑跳过过滤，这是局部策略 --- 处理结果:剔除不合格的target
// 下游的 PolicyEnforcer 不用再根据局部策略再来一遍，仅需要检查全局策略，即从数据库加载全局白名单和全局跳过策略处理就好 --- 处理结果:阻止不合格的AgentTask下发
// 作用域检验: 确保扫描目标严格限制在 Project.TargetScope 内，只不过这里的 Project.TargetScope 已经镜像到了 AgentTask.PolicySnapshot 中
// - 等于校验：agentTask.InputTarget 必须在 AgentTask.PolicySnapshot["target_scope"] (Project.TargetScope) 中
// ScanStage.target_policy JSON 中定义了目标策略，同时数据库中也存储了一套策略，这两套策略都需要满足
// 白名单检验：从数据库中加载白名单，使用白名单的时候需要注意白名单的作用域Scope，白名单中的目标只能在当前作用域中生效
// 跳过条件检验：从数据库中的跳过策略支持命中后执行动作配置(如：skip,tag,log,alert 动作)
// 核心变更: 移除 ProjectRepository 依赖，Enforcer 只对 AgentTask 和 ScanStage 快照负责
// PolicyEnforcer 不应该知道 Project 的存在，它只关心 Scope 和 PolicySnapshot
func NewPolicyEnforcer(policyRepo *assetrepo.AssetPolicyRepository) PolicyEnforcer {
	return &policyEnforcer{
		policyRepo: policyRepo,
	}
}

// Enforce 执行策略检查 --- 核心实现逻辑
// 职责: 校验任务是否符合项目策略 (Scope, Whitelist, SkipLogic)
// 变更说明:
// 1. 优先使用 PolicySnapshot (任务生成时的快照)，保证原子性
// 2. 移除 DB Project 查询，改为从 Snapshot 获取 Scope
// 3. 实现 Local (ScanStage) 和 Global (DB) 策略的逻辑 OR
func (p *policyEnforcer) Enforce(ctx context.Context, task *agentModel.AgentTask) error {
	// 1. 解析 PolicySnapshot (使用结构体)
	// AgentTask.PolicySnapshot 已经是结构体类型，直接使用
	policySnapshot := task.PolicySnapshot

	// task.PolicySnapshot 包含 project.TargetScope(list) 和 scanStage.target_policy（Struct）
	// 项目 Scope 是一个 CIDR 列表，可以是一个网段，或者域名（如：["192.168.1.0/24", "10.0.0.0/16"]）
	// 扫描阶段的 target_policy 是一个 JSON 字符串，包含白名单、跳过条件等策略配置
	// task.PolicySnapshot 样例:
	// {
	//   "target_scope": ["192.168.1.0/24", "10.0.0.0/16"], // 项目 TargetScope 可以为空，为空时表示不限制范围
	//   "target_policy": { // 扫描阶段的 target_policy 是一个 JSON 字符串，包含白名单、跳过条件等策略配置
	//     "target_sources": [
	//       {
	//         "source_type": "file", // 来源类型：file/db/view/sql/manual/api/previous_stage【上一个阶段结果】
	//         "source_value": "/path/to/targets.txt",
	//         "target_type": "ip_range" // 目标类型：ip/ip_range/domain/url
	//       }
	//     ],
	//     "whitelist_enabled": true,
	//     "whitelist_sources": [ // 白名单来源/文件/手动输入
	//       {
	//         "source_type": "file",
	//         "source_value": "/path/to/whitelist.txt" // file 对应文件路径, manual 对应手动输入内容["192.168.1.0/24","10.0.0.0/16"]
	//       }
	//     ],
	//     "skip_enabled": true,
	//     "skip_conditions": [  // 跳过条件,列表中可添加多个条件，这里写的条件直接执行跳过动作
	//       {
	//         "condition_field": "device_type",
	//         "operator": "equals",
	//         "value": "honeypot"
	//       },
	//       {
	//         "condition_field": "os",
	//         "operator": "contains",
	//         "value": "linux"
	//       }
	//     ]
	//   }
	// }

	// 2. ScopeValidator: 范围校验 (基于 Snapshot.TargetScope)
	// 确保扫描目标严格限制在 Project 定义的 TargetScope 内
	if task.InputTarget == "" {
		return fmt.Errorf("policy violation: target is empty")
	}

	// 解析 InputTarget (可能是 JSON 列表或单个字符串)
	// 支持格式：["192.168.1.0/24", "10.0.0.0/16"] 或
	// [{"type": "ip", "value": "192.168.1.1", "source": "file", "meta": {"device_type": "honeypot"}}, {"type": "ip", "value": "192.168.1.2", "source": "file", "meta": {"device_type": "honeypot"}}]
	targets, err := parseTargets(task.InputTarget)
	if err != nil {
		// 尝试作为单个字符串处理
		targets = []string{task.InputTarget}
	}

	// 校验所有目标是否在 Scope 内
	// 注意: TargetScope 现在是 []string 类型
	if err1 := validateScope(targets, policySnapshot.TargetScope); err1 != nil {
		return err1
	}

	// 3. WhitelistChecker: 白名单检查 (Global only)
	// 检查目标是否命中白名单
	for _, target := range targets {
		// 3.1 检查全局白名单 (DB) - 强制执行
		isBlocked, ruleName, err2 := p.checkWhitelist(ctx, target)
		if err2 != nil {
			logger.LogError(err2, "whitelist check error", 0, "", "service.orchestrator.policy.Enforce", "REPO", nil)
			return fmt.Errorf("policy check error: %v", err2)
		}
		if isBlocked {
			logger.LogInfo("Task blocked by global whitelist", "", 0, "", "service.orchestrator.policy.Enforce", "", map[string]interface{}{
				"task_id":   task.TaskID,
				"target":    target,
				"rule_name": ruleName,
			})
			return fmt.Errorf("policy violation: target %s is whitelisted by global rule: %s", target, ruleName)
		}
	}

	// 4. SkipLogicEvaluator: 动态跳过 (Global only)
	// 执行 AssetSkipPolicy 逻辑
	// 全局跳过策略从数据库中获取condition_rules跳过规则和targets匹配
	// 命中就跳过，否则不跳过

	for _, target := range targets {
		// 4.1 检查全局跳过策略 (DB)
		// 从数据库中获取 condition_rules 跳过规则和 targets 匹配
		shouldSkip, ruleName, err := p.checkGlobalSkipPolicy(ctx, target)
		if err != nil {
			logger.LogError(err, "global skip policy check error", 0, "", "service.orchestrator.policy.Enforce", "REPO", nil)
			return fmt.Errorf("policy check error: %v", err)
		}
		if shouldSkip {
			logger.LogInfo("Task skipped by global policy", "", 0, "", "service.orchestrator.policy.Enforce", "", map[string]interface{}{
				"task_id":   task.TaskID,
				"target":    target,
				"rule_name": ruleName,
			})
			return fmt.Errorf("policy violation: target %s skipped by global policy: %s", target, ruleName)
		}
	}

	return nil
}

// checkWhitelist 检查目标是否在白名单中(全局策略白名单)
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
			// 匹配成功，返回 true, 白名单名称, nil
			return true, w.WhitelistName, nil
		}
	}
	// 没有匹配的白名单，返回 false, "", nil
	return false, "", nil
}

// SkipConditionRules 跳过条件规则
// 定义在 AssetSkipPolicy 中的条件，用于动态判断是否跳过任务
type SkipConditionRules struct {
	BlockTimeWindows []string          `json:"block_time_windows"` // e.g. ["00:00-06:00", "22:00-23:59"]
	BlockEnvTags     []string          `json:"block_env_tags"`     // e.g. ["production", "sensitive"]
	MatchRule        matcher.MatchRule `json:"match_rule"`         // 复杂匹配规则 (New)
}

// checkGlobalSkipPolicy 检查是否应该跳过任务 (全局策略跳过)
// 1. 仅支持基于 Matcher 的复杂规则检查
// 2. 忽略 BlockTimeWindows 和 BlockEnvTags (已按需求移除)
func (p *policyEnforcer) checkGlobalSkipPolicy(ctx context.Context, target string) (bool, string, error) {
	policies, err := p.policyRepo.GetEnabledSkipPolicies(ctx)
	if err != nil {
		return false, "", err
	}

	// 构建上下文数据
	matchContext := map[string]interface{}{
		"target": target,
		"ip":     target,
		"domain": target,
		"host":   target,
	}

	for _, policy := range policies {
		var rules SkipConditionRules
		if err := json.Unmarshal([]byte(policy.ConditionRules), &rules); err != nil {
			logger.LogWarn("Failed to parse global skip policy rules", "", 0, "", "checkGlobalSkipPolicy", "", map[string]interface{}{
				"policy_id": policy.ID,
				"error":     err.Error(),
			})
			continue // 忽略解析失败的规则
		}

		// 仅检查 MatchRule
		if !matcher.IsEmptyRule(rules.MatchRule) {
			matched, err := matcher.Match(matchContext, rules.MatchRule)
			if err != nil {
				logger.LogWarn("Failed to execute global skip match rule", "", 0, "", "checkGlobalSkipPolicy", "", map[string]interface{}{
					"policy": policy.PolicyName,
					"error":  err.Error(),
				})
				continue
			}

			if matched {
				return true, fmt.Sprintf("GlobalPolicy:%s", policy.PolicyName), nil
			}
		}
	}

	return false, "", nil
}

// parseTargets 解析目标
func parseTargets(input string) ([]string, error) {
	// 1. 尝试解析为字符串数组 (Legacy format: ["1.1.1.1", "2.2.2.2"])
	var strTargets []string
	if err := json.Unmarshal([]byte(input), &strTargets); err == nil {
		return strTargets, nil
	}

	// 2. 尝试解析为 Target 对象数组 (New format: [{"type":"ip", "value":"1.1.1.1", ...}])
	var objTargets []agentModel.Target
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

// validateScope 校验目标是否在范围内 --- 基于 Project.TargetScope 检测
func validateScope(targets []string, allowedScopes []string) error {
	if len(allowedScopes) == 0 {
		// 如果 Scope 为空，默认允许所有 (或者默认拒绝，取决于安全策略)
		// 这里为了安全，建议如果 Scope 为空则仅允许特定类型，或者默认拒绝
		// 但为了易用性，暂时默认允许
		return nil
	}

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

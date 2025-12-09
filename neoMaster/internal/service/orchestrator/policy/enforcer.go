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
	"strings"
	"time"

	"neomaster/internal/pkg/logger"
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
	Enforce(ctx context.Context, task *agentModel.AgentTask) error
}

type policyEnforcer struct {
	projectRepo *orcrepo.ProjectRepository
	policyRepo  *assetrepo.AssetPolicyRepository
}

// NewPolicyEnforcer 创建策略执行器
func NewPolicyEnforcer(projectRepo *orcrepo.ProjectRepository, policyRepo *assetrepo.AssetPolicyRepository) PolicyEnforcer {
	return &policyEnforcer{
		projectRepo: projectRepo,
		policyRepo:  policyRepo,
	}
}

// Enforce 执行策略检查
func (p *policyEnforcer) Enforce(ctx context.Context, task *agentModel.AgentTask) error {
	// 1. ScopeValidator: 范围校验
	// 确保扫描目标严格限制在 Project 定义的 TargetScope 内
	if task.InputTarget == "" {
		return fmt.Errorf("policy violation: target is empty")
	}

	// 获取项目 Scope 配置
	// 注意: 这是一个数据库查询，为了性能，可以使用缓存
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

	for _, w := range whitelists {
		match := false
		switch w.TargetType {
		case "ip":
			// 支持单个IP, CIDR, IP范围
			if utils.IsIPRange(w.TargetValue) {
				// 这是一个 IP 范围 (e.g. 192.168.1.1-192.168.1.5)
				// 或者是 CIDR
				// 这里的 IsIPRange 只是判断格式，我们需要判断 target 是否在范围内
				// 我们需要实际的检查逻辑。
				// 让我们用 utils.ParseIPPairs 来解析范围，然后看 target 是否在其中。
				// 但 ParseIPPairs 返回所有 IP，性能太差。
				// 应该使用 net 库解析 CIDR 或者手动解析 Range。

				// 简单起见，如果 utils 支持 CheckIPInRange 最好。
				// 目前 utils 似乎没有直接的 CheckIPInRange。
				// 既然是实时查库，我们在这里做一下解析。

				// 1. CIDR
				if strings.Contains(w.TargetValue, "/") {
					_, ipNet, err := net.ParseCIDR(w.TargetValue)
					if err == nil {
						if ip := net.ParseIP(target); ip != nil && ipNet.Contains(ip) {
							match = true
						}
					}
				} else if strings.Contains(w.TargetValue, "-") {
					// 2. Range (1.1.1.1-1.1.1.5)
					// 简单的字符串比较在这里行不通，需要转换成数字。
					// 暂时为了性能和简单，先假设 target 必须是 IP。
					targetIP := net.ParseIP(target)
					if targetIP != nil {
						// 解析 Range
						parts := strings.Split(w.TargetValue, "-")
						if len(parts) == 2 {
							startIP := net.ParseIP(strings.TrimSpace(parts[0]))
							endIP := net.ParseIP(strings.TrimSpace(parts[1]))
							if startIP != nil && endIP != nil {
								if utils.IPCompare(targetIP, startIP) >= 0 && utils.IPCompare(targetIP, endIP) <= 0 {
									match = true
								}
							}
						}
					}
				} else {
					// 3. Single IP (Although IsIPRange check should mostly prevent this path if strict,
					// but let's keep it safe or handle it in else block.
					// Wait, IsIPRange only returns true if "-" or "/" exists.
					// So this "Single IP" block inside here is unreachable if IsIPRange definition is strict about "-" or "/".
					// Let's double check IsIPRange definition: strings.Contains(s, "-") || strings.Contains(s, "/")
					// So Single IP "1.1.1.1" will return false.
					// So this block inside IsIPRange is purely for Ranges/CIDRs.
					// The Single IP case is handled in the else block of the outer if.
				}
			} else {

				// 普通 IP 字符串
				if target == w.TargetValue {
					match = true
				}
			}
		case "domain":
			// 域名匹配：精确匹配或后缀匹配
			if target == w.TargetValue {
				match = true
			} else if strings.HasPrefix(w.TargetValue, ".") && strings.HasSuffix(target, w.TargetValue) {
				match = true
			}
		case "keyword":
			// 关键字包含
			if strings.Contains(target, w.TargetValue) {
				match = true
			}
		}

		if match {
			return true, w.WhitelistName, nil
		}
	}

	return false, "", nil
}

type SkipConditionRules struct {
	BlockTimeWindows []string `json:"block_time_windows"` // e.g. ["00:00-06:00", "22:00-23:59"]
	BlockEnvTags     []string `json:"block_env_tags"`     // e.g. ["production", "sensitive"]
}

// checkSkipPolicy 检查是否应该跳过任务
func (p *policyEnforcer) checkSkipPolicy(ctx context.Context, project *agentModel.Project) (bool, string, error) {
	policies, err := p.policyRepo.GetEnabledSkipPolicies(ctx)
	if err != nil {
		return false, "", err
	}

	now := time.Now()
	currentTimeStr := now.Format("15:04") // HH:MM

	for _, policy := range policies {
		var rules SkipConditionRules
		if err := json.Unmarshal([]byte(policy.ConditionRules), &rules); err != nil {
			logger.LogWarn("Failed to parse skip policy rules", "", 0, "", "checkSkipPolicy", "", map[string]interface{}{
				"policy_id": policy.ID,
				"error":     err.Error(),
			})
			continue // 忽略解析失败的规则
		}

		// 1. 检查环境标签
		if len(rules.BlockEnvTags) > 0 {
			projectTags := parseTags(project.Tags) // 假设 Project.Tags 是 JSON 字符串或逗号分隔
			for _, blockTag := range rules.BlockEnvTags {
				for _, projTag := range projectTags {
					if strings.EqualFold(blockTag, projTag) {
						return true, fmt.Sprintf("Environment tag '%s' is blocked by policy '%s'", blockTag, policy.PolicyName), nil
					}
				}
			}
		}

		// 2. 检查时间窗
		if len(rules.BlockTimeWindows) > 0 {
			for _, window := range rules.BlockTimeWindows {
				parts := strings.Split(window, "-")
				if len(parts) != 2 {
					continue
				}
				start := strings.TrimSpace(parts[0])
				end := strings.TrimSpace(parts[1])

				// 简单的字符串比较对于 HH:MM 格式是有效的
				if currentTimeStr >= start && currentTimeStr <= end {
					return true, fmt.Sprintf("Current time %s is within blocked window %s of policy '%s'", currentTimeStr, window, policy.PolicyName), nil
				}
			}
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
	var targets []string
	if err := json.Unmarshal([]byte(input), &targets); err != nil {
		return nil, err
	}
	return targets, nil
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
				if ip := net.ParseIP(target); ip != nil {
					if ipNet.Contains(ip) {
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

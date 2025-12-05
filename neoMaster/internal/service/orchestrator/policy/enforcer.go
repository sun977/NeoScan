package policy

import (
	"context"
	"encoding/json"
	"fmt"
	agentModel "neomaster/internal/model/orchestrator"
	"net"
	"strings"

	"neomaster/internal/pkg/logger"
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
}

// NewPolicyEnforcer 创建策略执行器
func NewPolicyEnforcer(projectRepo *orcrepo.ProjectRepository) PolicyEnforcer {
	return &policyEnforcer{
		projectRepo: projectRepo,
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
	if err := validateScope(targets, project.TargetScope); err != nil {
		return err
	}

	// 2. WhitelistChecker: 白名单检查 (强制阻断)
	// 检查目标是否命中 AssetWhitelist
	for _, target := range targets {
		if isWhitelisted(target) {
			logger.LogInfo("Task blocked by whitelist", "", 0, "", "service.orchestrator.policy.Enforce", "", map[string]interface{}{
				"task_id": task.TaskID,
				"target":  target,
			})
			return fmt.Errorf("policy violation: target %s is whitelisted", target)
		}
	}

	// 3. SkipLogicEvaluator: 动态跳过
	// 执行 AssetSkipPolicy 逻辑 (e.g. 生产环境限制)
	// TODO: 实现动态跳过逻辑

	return nil
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

// isWhitelisted 简单的白名单检查模拟
func isWhitelisted(target string) bool {
	// TODO: 连接真实的白名单数据库
	// 示例：禁止扫描政府和教育网站，或者特定敏感IP
	sensitiveKeywords := []string{".gov.cn", ".edu.cn", "127.0.0.1", "localhost"}
	for _, kw := range sensitiveKeywords {
		if strings.Contains(target, kw) {
			return true
		}
	}
	return false
}

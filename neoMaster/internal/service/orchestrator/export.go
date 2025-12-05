package orchestrator

import (
	"neomaster/internal/service/orchestrator/allocator"
	"neomaster/internal/service/orchestrator/policy"
	"neomaster/internal/service/orchestrator/task_dispatcher"
)

// 核心组件接口导出 (Facade Pattern)
// 允许外部模块 (如 setup) 通过 orchestratorService 统一访问这些核心组件接口，
// 而无需直接导入各个子包。

// TaskDispatcher 任务分发器接口别名
type TaskDispatcher = task_dispatcher.TaskDispatcher

// PolicyEnforcer 策略执行器接口别名
type PolicyEnforcer = policy.PolicyEnforcer

// ResourceAllocator 资源调度器接口别名
type ResourceAllocator = allocator.ResourceAllocator

// AgentTaskService Agent任务服务接口别名
type AgentTaskService = task_dispatcher.AgentTaskService

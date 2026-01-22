package runner

import (
	"context"
	"neoagent/internal/core/model"
)

// Runner 定义了扫描执行器的通用接口
type Runner interface {
	// Name 返回 Runner 的名称 (对应 TaskType)
	Name() model.TaskType

	// Run 执行具体的扫描任务
	// ctx: 用于控制超时和取消
	// task: 任务参数
	// 返回: 扫描结果列表 (可能包含多个结果，如多个端口)
	Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error)
}

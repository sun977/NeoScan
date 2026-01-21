package options

import (
	"neoagent/internal/core/model"
)

// TaskOption 定义所有指令参数结构体必须实现的接口
type TaskOption interface {
	// Validate 验证参数合法性
	Validate() error

	// ToTask 将参数转换为核心任务模型
	ToTask() *model.Task
}

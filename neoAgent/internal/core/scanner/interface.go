/**
 * 核心扫描器接口定义
 * @author: Sun977
 * @date: 2026.01.21
 * @description: 定义所有扫描能力的通用接口。
 */

package scanner

import (
	"context"
	"neoagent/internal/core/model"
)

// Scanner 是所有扫描能力的基类接口
// 无论是 Native Port Scan 还是 Wrapper Nmap，都必须实现此接口
type Scanner interface {
	// Name 返回扫描器的唯一标识 (e.g., "native_port_scanner")
	Name() string

	// Type 返回支持的任务类型
	Type() model.TaskType

	// Scan 执行扫描任务
	// ctx: 用于超时控制和取消
	// task: 任务详情
	Scan(ctx context.Context, task *model.Task) (*model.TaskResult, error)
}

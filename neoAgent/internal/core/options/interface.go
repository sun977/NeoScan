/**
 * 参数选项接口定义
 * @author: Sun977
 * @date: 2026-08-14
 * @description: 定义所有指令参数结构体必须实现的契约接口。
 *               每种扫描任务（web/api/port/...）各有一个 Options 结构体，
 *               均须实现 TaskOption，由 CLI 层调用 ToTask() 转换为核心任务模型。
 */

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

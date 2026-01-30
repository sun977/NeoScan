/**
 * 通用响应模型
 * @author: sun977
 * @date: 2025.10.21
 * @description: 通用响应数据模型，遵循"好品味"原则
 * @func: 定义通用的同步和上报响应结构
 */
package client

import "time"

// ==================== 通用响应 ====================

// SyncResponse 同步响应
// 遵循Linus原则：通用响应结构简洁，包含必要的状态信息
type SyncResponse struct {
	Success   bool      `json:"success"`   // 同步是否成功
	Message   string    `json:"message"`   // 响应消息
	Timestamp time.Time `json:"timestamp"` // 响应时间戳
}

// ReportResponse 上报响应
// 遵循"好品味"原则：上报响应与同步响应结构一致，避免特殊情况
type ReportResponse struct {
	Success   bool      `json:"success"`   // 上报是否成功
	Message   string    `json:"message"`   // 响应消息
	Timestamp time.Time `json:"timestamp"` // 响应时间戳
}
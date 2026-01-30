/**
 * 通信错误定义
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端与Master端通信的错误定义，遵循"好品味"原则
 * @func: 定义所有通信相关的错误类型
 */
package client

import "errors"

// ==================== 通信错误定义 ====================
// 遵循Linus原则：错误定义要简洁明了，消除特殊情况

var (
	// 连接相关错误
	ErrNotConnected     = errors.New("not connected to master")     // 未连接到Master
	ErrConnectionFailed = errors.New("failed to connect to master") // 连接Master失败
	
	// 认证相关错误
	ErrAuthFailed = errors.New("authentication failed") // 认证失败
	
	// 操作相关错误
	ErrTimeout         = errors.New("operation timeout")        // 操作超时
	ErrInvalidResponse = errors.New("invalid response from master") // Master响应无效
)
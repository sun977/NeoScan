/**
 * 通用响应结构体
 * @author: sun977
 * @date: 2025.10.21
 * @description: 通用API响应结构
 * @func: 定义了API响应的通用结构，包含状态码、状态、消息、数据、错误信息
 */

package base

// APIResponse 通用API响应结构
type APIResponse struct {
	Code    int         `json:"code"`            // 响应状态码
	Status  string      `json:"status"`          // 响应状态："success" 或 "failed"
	Message string      `json:"message"`         // 响应消息
	Data    interface{} `json:"data,omitempty"`  // 响应数据，可选
	Error   string      `json:"error,omitempty"` // 错误信息，可选
}

// PaginationResponse 分页响应结构
type PaginationResponse struct {
	Total       int64       `json:"total"`        // 总记录数
	Page        int         `json:"page"`         // 当前页码
	PageSize    int         `json:"page_size"`    // 每页大小
	TotalPages  int         `json:"total_pages"`  // 总页数
	HasNext     bool        `json:"has_next"`     // 是否有下一页
	HasPrevious bool        `json:"has_previous"` // 是否有上一页
	Data        interface{} `json:"data"`         // 分页数据
}

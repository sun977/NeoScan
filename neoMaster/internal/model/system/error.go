/**
 * 模型:错误定义
 * @author: sun977
 * @date: 2025.08.29
 * @description: 系统错误常量和错误类型定义
 * @func: 各种错误常量和ValidationError结构体
 */
package system

import "errors"

// 用户相关错误
var (
	// 验证错误
	ErrInvalidUsername = errors.New("用户名格式无效")
	ErrInvalidEmail    = errors.New("邮箱格式无效")
	ErrInvalidPassword = errors.New("密码格式无效")
	ErrInvalidPhone    = errors.New("手机号格式无效")

	// 业务逻辑错误
	ErrUserAlreadyExists        = errors.New("用户已存在")
	ErrUserNotFound             = errors.New("用户不存在")
	ErrEmailAlreadyExists       = errors.New("邮箱已存在")
	ErrUsernameAlreadyExists    = errors.New("用户名已存在")
	ErrUserOrEmailAlreadyExists = errors.New("用户名或邮箱已存在")

	// 认证错误
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserDisabled       = errors.New("用户已被禁用")
	ErrTokenExpired       = errors.New("令牌已过期")
	ErrTokenInvalid       = errors.New("令牌无效")

	// 权限错误
	ErrPermissionDenied = errors.New("权限不足")
	ErrUnauthorized     = errors.New("未授权访问")
)

// ValidationError 验证错误结构体
type ValidationError struct {
	Field   string `json:"field"`   // 字段名
	Message string `json:"message"` // 错误消息
}

// NewValidationError 创建验证错误
func NewValidationError(message string) *ValidationError {
	return &ValidationError{
		Message: message,
	}
}

// Error 实现error接口
func (e *ValidationError) Error() string {
	return e.Message
}

// IsValidationError 检查是否为验证错误
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

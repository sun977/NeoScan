package brute

import (
	"context"
	"errors"
)

// AuthMode 定义爆破模式
type AuthMode int

const (
	AuthModeUserPass AuthMode = iota // 需要用户名和密码 (SSH, MySQL)
	AuthModeOnlyPass                 // 仅需要密码 (Redis, VNC)
	AuthModeNone                     // 无需认证/默认凭据 (Telnet, MongoDB)
)

// Auth 认证凭据 (数据传输对象)
type Auth struct {
	Username string
	Password string
	Other    map[string]string // 扩展字段 (e.g. Oracle SID)
}

// Cracker 协议适配器接口
type Cracker interface {
	// Name 返回协议名称 (e.g. "ssh", "mysql")
	Name() string

	// Mode 返回该协议的爆破模式
	Mode() AuthMode

	// Check 验证单个凭据
	// context: 用于控制超时 (通常 3-5秒)
	// host, port: 目标地址
	// auth: 待验证凭据
	// 返回:
	// - bool: true 表示认证成功
	// - error: 见下方错误定义
	Check(ctx context.Context, host string, port int, auth Auth) (bool, error)
}

var (
	// ErrAuthFailed 认证失败 (账号密码错误) -> 继续尝试下一个
	// 实现 Check 时，如果确定是密码错误，返回 (false, nil) 即可，也可以返回此 error 用于日志
	ErrAuthFailed = errors.New("auth failed")

	// ErrConnectionFailed 连接失败 (超时/拒绝/重置) -> 触发重试或熔断
	ErrConnectionFailed = errors.New("connection failed")

	// ErrProtocolError 协议交互错误 (如非预期响应) -> 视为该协议不支持，跳过
	ErrProtocolError = errors.New("protocol error")
)

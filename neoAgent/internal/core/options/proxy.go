package options

import (
	"fmt"
	"strings"

	"neoagent/internal/core/model"
)

// ProxyOptions 定义代理指令的参数
type ProxyOptions struct {
	Mode    string // 模式: socks5, http, port_forward
	Listen  string // 监听地址
	Auth    string // 认证信息 user:pass
	Forward string // 转发目标 (仅 port_forward)
}

// NewProxyOptions 创建默认参数
func NewProxyOptions() *ProxyOptions {
	return &ProxyOptions{
		Mode:   "socks5",
		Listen: ":1080",
	}
}

// Validate 验证参数
func (o *ProxyOptions) Validate() error {
	validModes := map[string]bool{
		"socks5":       true,
		"http":         true,
		"port_forward": true,
	}

	if !validModes[o.Mode] {
		return fmt.Errorf("invalid mode: %s (allowed: socks5, http, port_forward)", o.Mode)
	}

	if o.Listen == "" {
		return fmt.Errorf("listen address is required")
	}

	if o.Mode == "port_forward" && o.Forward == "" {
		return fmt.Errorf("forward address is required for port_forward mode")
	}

	if o.Auth != "" && !strings.Contains(o.Auth, ":") {
		return fmt.Errorf("invalid auth format, expected user:pass")
	}

	return nil
}

// ToTask 转换为 Task 模型
func (o *ProxyOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeProxy, o.Listen)

	// Proxy 任务默认无超时 (params 中设置 timeout=0 明确标识)
	task.Timeout = 0

	task.Params["mode"] = o.Mode
	task.Params["auth"] = o.Auth
	task.Params["forward"] = o.Forward

	return task
}

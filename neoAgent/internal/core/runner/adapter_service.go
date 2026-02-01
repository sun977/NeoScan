package runner

import (
	"context"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/port_service"
)

// ServiceRunner 服务扫描器适配器
// 实际上复用了 PortServiceScanner 的逻辑，但注册为 TaskTypeServiceScan
type ServiceRunner struct {
	scanner *port_service.PortServiceScanner
}

// NewServiceRunner 创建新的服务扫描器适配器
func NewServiceRunner(scanner *port_service.PortServiceScanner) *ServiceRunner {
	return &ServiceRunner{scanner: scanner}
}

// Name 返回Runner名称
func (r *ServiceRunner) Name() model.TaskType {
	return model.TaskTypeServiceScan
}

// Run 执行服务扫描
func (r *ServiceRunner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	// Service Scan 隐含开启服务探测
	// PortServiceScanner 通过 Params["service_detect"] = true 来控制
	if task.Params == nil {
		task.Params = make(map[string]interface{})
	}
	task.Params["service_detect"] = true
	
	return r.scanner.Run(ctx, task)
}

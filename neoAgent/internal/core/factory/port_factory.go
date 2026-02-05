package factory

import (
	"neoagent/internal/core/scanner/port_service"
)

// NewPortScanner 创建端口服务扫描器
// 返回的 PortServiceScanner 实现了 Runner 接口 (TaskTypePortScan)
func NewPortScanner() *port_service.PortServiceScanner {
	return port_service.NewPortServiceScanner()
}

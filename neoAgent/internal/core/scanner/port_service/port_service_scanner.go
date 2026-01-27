package port_service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/dialer"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/port_service/gonmap"
)

const (
	ScannerName    = "port_service_scanner"
	DefaultTimeout = 2 * time.Second
)

// PortServiceScanner 端口服务扫描器
// 实现了 Scanner 接口，整合了 TCP Connect 扫描与 Nmap 服务识别逻辑
type PortServiceScanner struct {
	gonmapEngine *gonmap.Engine

	initOnce sync.Once
	initErr  error
}

func NewPortServiceScanner() *PortServiceScanner {
	return &PortServiceScanner{
		gonmapEngine: gonmap.NewEngine(),
	}
}

func (s *PortServiceScanner) Name() model.TaskType {
	return model.TaskTypePortScan
}

// ensureInit 确保规则已加载
func (s *PortServiceScanner) ensureInit() error {
	s.initOnce.Do(func() {
		// 优先使用 Embed 规则 (Zero Dependency)
		if len(gonmap.NmapServiceProbes) > 0 {
			s.gonmapEngine.LoadRules(gonmap.NmapServiceProbes)
			return
		}

		// Fallback: 尝试加载外部规则文件 (仅开发环境或特殊配置)
		paths := []string{
			"rules/fingerprint/nmap-service-probes",
			"../rules/fingerprint/nmap-service-probes",
			"../../rules/fingerprint/nmap-service-probes", // 针对 test 目录
		}

		for _, path := range paths {
			content, err := os.ReadFile(path)
			if err == nil && len(content) > 0 {
				s.gonmapEngine.LoadRules(string(content))
				break
			}
		}
	})
	return nil
}

func (s *PortServiceScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	if err := s.ensureInit(); err != nil {
		return nil, err
	}

	target := task.Target
	portRange := task.PortRange
	if portRange == "" {
		// 默认扫描 Top 1000? 或者报错
		// 这里假设调用方已处理好
		return nil, fmt.Errorf("port range is required")
	}

	// 解析参数
	serviceDetect := false
	if val, ok := task.Params["service_detect"]; ok {
		if v, ok := val.(bool); ok {
			serviceDetect = v
		}
	}

	// 解析端口列表
	// 注意：由于 ParsePortList 迁移到了 nmap 包，但它可能不是公开的？
	// 最好把 ParsePortList 放到 utils 或 nmap 包公开
	// 这里假设 nmap.ParsePortList 是公开的
	ports := gonmap.ParsePortList(portRange)

	// 并发控制 (使用 Runner 或简单的 WaitGroup)
	concurrency := 100
	if val, ok := task.Params["rate"]; ok {
		// handle float64 or int
		if v, ok := val.(float64); ok {
			concurrency = int(v)
		} else if v, ok := val.(int); ok {
			concurrency = v
		}
	}

	if concurrency <= 0 {
		concurrency = 100
	}

	results := make([]*model.TaskResult, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for _, port := range ports {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// 1. 基础端口连通性检查 (TCP Connect)
			// 如果只是为了探测服务，这一步可以快速过滤关闭的端口
			if !s.isPortOpen(ctx, target, p, DefaultTimeout) {
				return
			}

			// 端口开放，构建基础结果
			portResult := &model.PortServiceResult{
				IP:       target,
				Port:     p,
				Protocol: "tcp",
				Status:   "open",
				Service:  "unknown",
			}

			// 2. 服务识别 (如果启用)
			if serviceDetect {
				// 给服务识别更多时间
				scanTimeout := DefaultTimeout * 3
				fp, err := s.gonmapEngine.Scan(ctx, target, p, scanTimeout)
				if err == nil && fp != nil {
					portResult.Service = fp.Service
					portResult.Product = fp.ProductName
					portResult.Version = fp.Version
					portResult.Info = fp.Info
					portResult.Hostname = fp.Hostname
					portResult.OS = fp.OperatingSystem
					portResult.DeviceType = fp.DeviceType
					portResult.CPE = fp.CPE
				}
			}

			result := &model.TaskResult{
				TaskID:      task.ID,
				Status:      model.TaskStatusSuccess,
				Result:      portResult,
				ExecutedAt:  time.Now(), // approximate
				CompletedAt: time.Now(),
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(port)
	}

	wg.Wait()
	return results, nil
}

// isPortOpen 检查端口是否开放 (TCP Connect)
func (s *PortServiceScanner) isPortOpen(ctx context.Context, ip string, port int, timeout time.Duration) bool {
	address := fmt.Sprintf("%s:%d", ip, port)
	d := dialer.Get()
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

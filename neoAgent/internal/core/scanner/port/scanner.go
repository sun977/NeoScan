package port

import (
	"context"
	"fmt"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/dialer"
	"neoagent/internal/core/model"
	"neoagent/internal/pkg/fingerprint/engines/nmap"
)

const (
	ScannerName    = "port_service_scanner"
	DefaultTimeout = 2 * time.Second
)

// PortServiceScanner 端口服务扫描器
// 实现了 Scanner 接口，整合了 TCP Connect 扫描与 Nmap 服务识别逻辑
type PortServiceScanner struct {
	nmapEngine *nmap.NmapEngine

	initOnce sync.Once
	initErr  error
}

func NewPortServiceScanner() *PortServiceScanner {
	return &PortServiceScanner{
		nmapEngine: nmap.NewNmapEngine(),
	}
}

func (s *PortServiceScanner) Name() model.TaskType {
	return model.TaskTypePortScan
}

// ensureInit 确保规则已加载
func (s *PortServiceScanner) ensureInit() error {
	// Nmap Engine 现在负责懒加载，这里我们只需要确保没报错
	// 但实际上 NmapEngine 的 ensureInit 是私有的，
	// 我们在 Scan 时会自动触发
	return nil
}

func (s *PortServiceScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
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
	ports := nmap.ParsePortList(portRange)

	// 并发控制 (使用 Runner 或简单的 WaitGroup)
	// 这里为了简单演示，使用 Semaphore
	concurrency := 100
	if val, ok := task.Params["rate"]; ok {
		if v, ok := val.(int); ok {
			concurrency = v
		}
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []interface{}

	for _, port := range ports {
		wg.Add(1)
		sem <- struct{}{}
		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()

			// 执行扫描
			res := s.scanPort(ctx, target, p, serviceDetect)
			if res != nil {
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()

	return []*model.TaskResult{{
		TaskID: task.ID,
		Status: model.TaskStatusSuccess,
		Result: results,
	}}, nil
}

func (s *PortServiceScanner) scanPort(ctx context.Context, ip string, port int, serviceDetect bool) *model.PortServiceResult {
	address := fmt.Sprintf("%s:%d", ip, port)
	timeout := DefaultTimeout

	// 1. TCP Connect 探测端口开放
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	conn, err := dialer.Get().DialContext(dialCtx, "tcp", address)
	cancel()
	if err != nil {
		return nil // 端口关闭 (或被过滤)
	}
	defer conn.Close()

	result := &model.PortServiceResult{
		IP:       ip,
		Port:     port,
		Protocol: "tcp",
		Status:   "open",
	}

	// 如果不进行服务识别，直接返回 Open
	if !serviceDetect {
		return result
	}

	// 2. 服务识别 (Service Discovery) - 使用 Nmap Engine
	// 这里使用独立的超时时间，或者总超时？
	// 假设每个探针有自己的超时，这里传递上下文
	fp, err := s.nmapEngine.Scan(ctx, ip, port, timeout)
	if err != nil {
		// 记录错误?
	}

	if fp != nil {
		// 匹配成功!
		result.Service = fp.Service
		result.Product = fp.ProductName
		result.Version = fp.Version
		result.Info = fp.Info
		result.Hostname = fp.Hostname
		result.OS = fp.OperatingSystem
		result.DeviceType = fp.DeviceType
		result.CPE = fp.CPE
		result.Status = "matched"
		return result
	}

	// 如果所有探针都失败，标记为 unknown service
	result.Service = "unknown"
	return result
}

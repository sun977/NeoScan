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
		// 尝试加载外部规则文件
		// 优先级:
		// 1. 当前目录 rules/fingerprint/nmap-service-probes
		// 2. 上级目录 (开发环境)

		paths := []string{
			"rules/fingerprint/nmap-service-probes",
			"../rules/fingerprint/nmap-service-probes",
			"../../rules/fingerprint/nmap-service-probes", // 针对 test 目录
			"internal/core/scanner/port_service/gonmap/nmap-service-probes",
		}

		for _, path := range paths {
			content, err := os.ReadFile(path)
			if err == nil && len(content) > 0 {
				s.gonmapEngine.LoadRules(string(content))
				break
			}
		}
		// 如果都没找到，需要加载默认 embed 规则 (后续从 nmap-service-probes 文件读取)
		// 这里假设 nmap-service-probes 已经迁移到 gonmap 目录并 embed
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
		// 端口关闭 (或被过滤)
		// 调试日志：fmt.Printf("Port %d closed: %v\n", port, err)
		return nil
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

	// 2. 服务识别 (Service Discovery) - 使用 Gonmap Engine
	// 这里使用独立的超时时间，或者总超时？
	// 假设每个探针有自己的超时，这里传递上下文
	fp, err := s.gonmapEngine.Scan(ctx, ip, port, timeout)
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
		result.Status = "open" // 只要连接成功就是 open，不应使用 matched
		return result
	}

	// 如果所有探针都失败，标记为 unknown service
	result.Service = "unknown"
	return result
}

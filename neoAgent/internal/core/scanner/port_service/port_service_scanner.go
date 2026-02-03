package port_service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/dialer"
	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/port_service/nmap_service"
)

const (
	ScannerName    = "port_service_scanner"
	DefaultTimeout = 2 * time.Second
)

// PortServiceScanner 端口服务扫描器
// 实现了 Scanner 接口，整合了 TCP Connect 扫描与 Nmap 服务识别逻辑
type PortServiceScanner struct {
	gonmapEngine *nmap_service.Engine
	rttEstimator *qos.RttEstimator
	limiter      *qos.AdaptiveLimiter

	initOnce sync.Once
	initErr  error
}

func NewPortServiceScanner() *PortServiceScanner {
	return &PortServiceScanner{
		gonmapEngine: nmap_service.NewEngine(),
		rttEstimator: qos.NewRttEstimator(),
		// 初始并发 100，最小 10，最大 2000
		limiter: qos.NewAdaptiveLimiter(100, 10, 2000),
	}
}

func (s *PortServiceScanner) Name() model.TaskType {
	return model.TaskTypePortScan
}

// ensureInit 确保规则已加载
func (s *PortServiceScanner) ensureInit() error {
	s.initOnce.Do(func() {
		// 优先使用 Embed 规则 (Zero Dependency)
		if len(nmap_service.NmapServiceProbes) > 0 {
			s.gonmapEngine.LoadRules(nmap_service.NmapServiceProbes)
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

	// 解析端口列表(使用专门的解析函数,没使用utils.ParseIntList,因为有 -p top100 这种定制情况)
	ports := nmap_service.ParsePortList(portRange)
	// ports := utils.ParseIntList(portRange)

	// 并发控制参数 (覆盖默认值)
	// 如果用户指定了 rate，我们将其作为 Initial 和 Max
	if val, ok := task.Params["rate"]; ok {
		var rate int
		if v, ok := val.(float64); ok {
			rate = int(v)
		} else if v, ok := val.(int); ok {
			rate = v
		}
		if rate > 0 {
			// 重置 limiter
			s.limiter = qos.NewAdaptiveLimiter(rate, 10, rate*2)
		}
	}

	results := make([]*model.TaskResult, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)

		// 获取并发令牌 (带上下文超时)
		if err := s.limiter.Acquire(ctx); err != nil {
			wg.Done()
			return nil, err // 上下文取消
		}

		go func(p int) {
			defer wg.Done()
			defer s.limiter.Release()

			// 动态获取当前 RTO
			timeout := s.rttEstimator.Timeout()

			// 1. 基础端口连通性检查 (TCP Connect)
			// 测量 RTT
			start := time.Now()
			isOpen := s.isPortOpen(ctx, target, p, timeout)
			duration := time.Since(start)

			if isOpen {
				// 成功连接：更新 RTT，增加并发
				s.rttEstimator.Update(duration)
				s.limiter.OnSuccess()
			} else {
				// 连接失败
				// 如果是因为超时失败的，才应该惩罚
				// 这里简化逻辑：如果是网络不可达，其实也会很快返回，不算超时
				// 只有当 duration 接近 timeout 时，才认为是拥塞导致的丢包
				if duration >= timeout {
					s.limiter.OnFailure()
				}
				// 端口关闭，直接返回
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
				// 给服务识别更多时间，通常是 RTO 的 3-5 倍
				scanTimeout := timeout * 3
				// 确保不低于默认的 2s，因为服务响应需要时间
				if scanTimeout < DefaultTimeout {
					scanTimeout = DefaultTimeout
				}

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

	// 创建带超时的上下文
	connCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := d.DialContext(connCtx, "tcp", address)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

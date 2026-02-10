package pipeline

import (
	"context"
	"fmt"
	"sync"

	"neoagent/internal/core/model"
	"neoagent/internal/core/options"
	"neoagent/internal/core/scanner/brute"
	"neoagent/internal/core/scanner/web"
	"neoagent/internal/pkg/logger"
)

// DispatchStrategy 分发策略
type DispatchStrategy int

const (
	StrategyDefault DispatchStrategy = iota
	StrategyQuick                    // 快速模式：跳过深度扫描和耗时爆破
	StrategyFull                     // 全量模式：执行所有可能的扫描
)

// ServiceDispatcher 服务分发器
// 负责 Phase 2 的任务分发与协调
type ServiceDispatcher struct {
	strategy     DispatchStrategy
	bruteScanner *brute.BruteScanner
	webScanner   *web.WebScanner
	opts         *options.ScanRunOptions // 全局选项，包含 Brute 开关和字典
	// vulnScanner  *vuln.Scanner // Future
}

// NewServiceDispatcher 创建服务分发器
func NewServiceDispatcher(strategy DispatchStrategy, bruteScanner *brute.BruteScanner, webScanner *web.WebScanner, opts *options.ScanRunOptions) *ServiceDispatcher {
	return &ServiceDispatcher{
		strategy:     strategy,
		bruteScanner: bruteScanner,
		webScanner:   webScanner,
		opts:         opts,
	}
}

// Dispatch 执行分发逻辑
// 这是一个阻塞方法，直到所有派生的 Phase 2 任务完成
func (d *ServiceDispatcher) Dispatch(ctx context.Context, pCtx *PipelineContext) {
	// Phase 2a: High Priority (Web & Vuln)
	// "Vuln First"
	d.dispatchHighPriority(ctx, pCtx)

	// Phase 2b: Low Priority (Brute Force)
	// "Brute Last"
	d.dispatchLowPriority(ctx, pCtx)
}

// dispatchHighPriority 分发高优先级任务 (Web, Vuln)
func (d *ServiceDispatcher) dispatchHighPriority(ctx context.Context, pCtx *PipelineContext) {
	var wg sync.WaitGroup

	// 1. Web Scan
	if d.shouldScanWeb(pCtx) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.runWebScan(ctx, pCtx)
		}()
	}

	// 2. Vuln Scan (Nuclei)
	// Placeholder
	if d.strategy == StrategyFull {
		// 检查是否有服务可以扫描漏洞
		// 这里简化处理，只要有开放服务就尝试 Vuln Scan
		if len(pCtx.Services) > 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				logger.Infof("[%s] [Phase 2] Dispatching Vuln Scan...", pCtx.IP)
				// TODO: 调用真实的 Vuln Scanner
			}()
		}
	}

	// 等待高优先级任务完成，以确保 "Vuln First"
	wg.Wait()
}

// dispatchLowPriority 分发低优先级任务 (Brute)
func (d *ServiceDispatcher) dispatchLowPriority(ctx context.Context, pCtx *PipelineContext) {
	// 1. 检查全局开关
	if d.opts != nil && !d.opts.EnableBrute {
		logger.Debugf("[%s] [Phase 2] Brute Scan disabled by user.", pCtx.IP)
		return
	}

	if d.strategy == StrategyQuick {
		return // 快速模式跳过爆破
	}

	if d.bruteScanner == nil {
		return
	}

	var wg sync.WaitGroup

	// 支持的爆破协议列表
	// TODO: 从 BruteScanner 获取支持列表，而不是硬编码
	supportedProtos := []string{
		"ssh", "rdp", "mysql", "redis", "postgres", "mssql",
		"ftp", "telnet", "smb", "oracle", "elasticsearch", "mongodb",
	}

	for _, proto := range supportedProtos {
		ports := pCtx.GetPortsByService(proto)
		if len(ports) > 0 {
			for _, port := range ports {
				wg.Add(1)
				go func(p int, pr string) {
					defer wg.Done()
					d.runBruteTask(ctx, pCtx, p, pr)
				}(port, proto)
			}
		}
	}

	wg.Wait()
}

func (d *ServiceDispatcher) runBruteTask(ctx context.Context, pCtx *PipelineContext, port int, service string) {
	logger.Infof("[%s] [Phase 2] Dispatching Brute Scan for %s:%d...", pCtx.IP, service, port)

	// 构造 Task
	task := model.NewTask(model.TaskTypeBrute, pCtx.IP)
	task.PortRange = fmt.Sprintf("%d", port)
	task.Params["service"] = service

	// 传递自定义字典参数
	if d.opts != nil {
		if d.opts.BruteUsers != "" {
			task.Params["users"] = d.opts.BruteUsers
		}
		if d.opts.BrutePass != "" {
			task.Params["passwords"] = d.opts.BrutePass
		}
	}

	// 如果是全量模式，可能设置 stop_on_success = false ? 默认 true
	task.Params["stop_on_success"] = true

	// 执行
	results, err := d.bruteScanner.Run(ctx, task)
	if err != nil {
		logger.Warnf("[%s] Brute scan failed for %s:%d: %v", pCtx.IP, service, port, err)
		return
	}

	// 收集结果
	for _, res := range results {
		if res.Status == model.TaskStatusSuccess {
			if bruteResults, ok := res.Result.(model.BruteResults); ok {
				for _, r := range bruteResults {
					pCtx.AddBruteResult(&r)
				}
			}
		}
	}
}

func (d *ServiceDispatcher) shouldScanWeb(pCtx *PipelineContext) bool {
	// 简单的 Web 服务判定
	return pCtx.HasService("http") ||
		pCtx.HasService("https") ||
		pCtx.HasService("http-alt") ||
		pCtx.HasService("http-proxy")
}

func (d *ServiceDispatcher) runWebScan(ctx context.Context, pCtx *PipelineContext) {
	// 遍历所有 Web 端口
	// TODO: 优化逻辑，避免重复扫描 (http/https 可能在同一端口?)
	webPorts := pCtx.GetPortsByService("http")
	webPorts = append(webPorts, pCtx.GetPortsByService("https")...)
	webPorts = append(webPorts, pCtx.GetPortsByService("http-alt")...)
	webPorts = append(webPorts, pCtx.GetPortsByService("http-proxy")...)

	// 去重
	seenPorts := make(map[int]bool)
	var uniquePorts []int
	for _, p := range webPorts {
		if !seenPorts[p] {
			seenPorts[p] = true
			uniquePorts = append(uniquePorts, p)
		}
	}

	var wg sync.WaitGroup
	for _, port := range uniquePorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			logger.Infof("[%s] [Phase 2] Dispatching Web Scan for port %d...", pCtx.IP, p)

			// 构造 Task
			task := model.NewTask(model.TaskTypeWebScan, pCtx.IP)
			task.PortRange = fmt.Sprintf("%d", p)
			// WebScanner 需要 Target 是 URL 吗？不需要，它自己会 normalizeURL (ip + port -> http://ip:port)
			// 但我们需要传递一些配置，比如是否启用截图
			task.Params["screenshot"] = true // 默认开启截图

			results, err := d.webScanner.Run(ctx, task)
			if err != nil {
				logger.Warnf("[%s] Web scan failed for port %d: %v", pCtx.IP, p, err)
				return
			}

			// 收集结果
			for _, res := range results {
				if res.Status == model.TaskStatusSuccess {
					if webRes, ok := res.Result.(*model.WebResult); ok {
						pCtx.AddWebResult(webRes)
					}
				}
			}
		}(port)
	}
	wg.Wait()
}

package pipeline

import (
	"context"
	"fmt"
	"strings"
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
	// 检查用户是否显式禁用了 Web 扫描
	if d.opts != nil && d.opts.NoWeb {
		return false
	}

	// 遍历所有服务，检查是否有 Web 相关服务
	services := pCtx.GetAllServices()
	for _, svc := range services {
		// 规则 1: Service Name
		if svc.Service == "http" || svc.Service == "http-alt" || svc.Service == "http-proxy" || svc.Service == "wbem-http" ||
			svc.Service == "https" || svc.Service == "ssl/http" || svc.Service == "ssl/https" {
			return true
		}

		// 规则 2: Product Name (Common Web Servers)
		prod := svc.Product
		if prod == "" {
			prod = svc.Banner
		}
		if len(prod) > 0 {
			prodLower := strings.ToLower(prod)
			keywords := []string{"html", "http", "apache", "nginx", "iis", "jetty", "tomcat", "node.js", "express", "php", "jsp"}
			for _, kw := range keywords {
				if strings.Contains(prodLower, kw) {
					return true
				}
			}
		}
	}

	return false
}

func (d *ServiceDispatcher) runWebScan(ctx context.Context, pCtx *PipelineContext) {
	// 1. 获取所有服务 (Thread-Safe)
	services := pCtx.GetAllServices()

	// 2. 筛选 Web 目标 (Port -> Protocol Hint)
	// 使用 map 去重，Key 为 Port
	targets := make(map[int]string)

	for port, svc := range services {
		isWeb := false
		protocol := ""

		// 规则 1: Service Name
		if svc.Service == "http" || svc.Service == "http-alt" || svc.Service == "http-proxy" || svc.Service == "wbem-http" {
			isWeb = true
			protocol = "http"
		} else if svc.Service == "https" || svc.Service == "ssl/http" || svc.Service == "ssl/https" {
			isWeb = true
			protocol = "https"
		}

		// 规则 2: Product Name (Common Web Servers)
		if !isWeb {
			// 转换为小写进行匹配
			prod := ""
			if svc.Product != "" {
				prod = svc.Product
			} else if svc.Banner != "" {
				prod = svc.Banner // 如果没有 Product，尝试看 Banner
			}

			// 简单的关键词匹配
			// 注意: 这种匹配可能误报，但 WebScanner 本身有容错，如果不是 Web 会失败
			if len(prod) > 0 {
				prodLower := strings.ToLower(prod)
				// 常见 Web 服务器特征
				keywords := []string{"html", "http", "apache", "nginx", "iis", "jetty", "tomcat", "node.js", "express", "php", "jsp"}
				for _, kw := range keywords {
					if strings.Contains(prodLower, kw) {
						isWeb = true
						break
					}
				}
			}
		}

		if isWeb {
			// 如果端口是 443/8443，即使 Service 没说是 https，我们也倾向于 https
			// 除非 Service 明确说是 http
			if (port == 443 || port == 8443) && protocol == "http" {
				// 保持 http，因为 Nmap 可能已经探测过了
				// 但如果 protocol 为空，则设为 https
			}
			if (port == 443 || port == 8443) && protocol == "" {
				protocol = "https"
			}
			targets[port] = protocol
		}
	}

	// 3. 执行扫描
	var wg sync.WaitGroup
	for port, protocol := range targets {
		wg.Add(1)
		go func(p int, proto string) {
			defer wg.Done()
			logger.Infof("[%s] [Phase 2] Dispatching Web Scan for port %d (Proto: %s)...", pCtx.IP, p, proto)

			// 构造 Task
			task := model.NewTask(model.TaskTypeWebScan, pCtx.IP)
			task.PortRange = fmt.Sprintf("%d", p)
			// 传递 Protocol Hint
			if proto != "" {
				task.Params["protocol"] = proto
			}

			// 截图配置
			if d.opts != nil {
				task.Params["screenshot"] = d.opts.WebScreenshot
			} else {
				task.Params["screenshot"] = false // 默认关闭
			}

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
		}(port, protocol)
	}
	wg.Wait()
}

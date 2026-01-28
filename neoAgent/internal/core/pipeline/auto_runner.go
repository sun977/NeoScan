package pipeline

import (
	"context"
	"fmt"
	"sync"

	"neoagent/internal/core/model"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/scanner/alive"
	"neoagent/internal/core/scanner/os"
	"neoagent/internal/core/scanner/port_service"
	"neoagent/internal/pkg/logger"
)

// AutoRunner 自动编排运行器
// 负责串联 Alive -> Port -> Service -> OS 的全流程
type AutoRunner struct {
	targetGenerator <-chan string
	concurrency     int
	portRange       string

	// Scanners
	aliveScanner *alive.IpAliveScanner
	portScanner  *port_service.PortServiceScanner
	osScanner    *os.Scanner // 注意：这是 os 包的 Scanner struct，不是接口

	// 结果收集器 (线程安全)
	summaryMu sync.Mutex
	summaries []*PipelineContext
}

func NewAutoRunner(targetInput string, concurrency int, portRange string) *AutoRunner {
	if portRange == "" {
		portRange = "top1000"
	}
	return &AutoRunner{
		targetGenerator: GenerateTargets(targetInput),
		concurrency:     concurrency,
		portRange:       portRange,
		aliveScanner:    alive.NewIpAliveScanner(),
		portScanner:     port_service.NewPortServiceScanner(),
		osScanner:       os.NewScanner(),
		summaries:       make([]*PipelineContext, 0),
	}
}

func (r *AutoRunner) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	sem := make(chan struct{}, r.concurrency)

	// 结果收集器 (用于最终输出)
	// 在实际流式处理中，可能每完成一个 IP 就输出一次
	// 这里为了简单，先收集所有 Context，或者直接在 Worker 中输出
	// 为了体验更好，我们采用 Worker 内实时输出的方式

	for ip := range r.targetGenerator {
		wg.Add(1)
		sem <- struct{}{}

		go func(targetIP string) {
			defer wg.Done()
			defer func() { <-sem }()

			// 创建 Pipeline Context
			pCtx := NewPipelineContext(targetIP)

			// 执行流水线
			r.executePipeline(ctx, pCtx)

			// 收集结果
			r.summaryMu.Lock()
			r.summaries = append(r.summaries, pCtx)
			r.summaryMu.Unlock()

		}(ip)
	}

	wg.Wait()

	// 输出最终总结报告
	r.printFinalReport()

	return nil
}

// printFinalReport 输出最终的扫描总结
func (r *AutoRunner) printFinalReport() {
	if len(r.summaries) == 0 {
		return
	}

	fmt.Println("\n" + "============================================================")
	fmt.Println("[+] Scan Summary Report")
	fmt.Println("============================================================")

	aliveCount := 0
	totalOpenPorts := 0

	for _, pCtx := range r.summaries {
		if pCtx.Alive {
			aliveCount++
			// 直接访问字段（因为此时所有Goroutine已结束，没有并发冲突）
			// 或者添加 GetOpenPorts 方法
			ports := pCtx.OpenPorts
			totalOpenPorts += len(ports)

			fmt.Printf("\n[Target: %s]\n", pCtx.IP)
			if pCtx.Hostname != "" {
				fmt.Printf("  Hostname : %s\n", pCtx.Hostname)
			}
			if pCtx.OSGuess != "" {
				fmt.Printf("  OS Guess : %s\n", pCtx.OSGuess)
			} else if pCtx.OSInfo != nil {
				fmt.Printf("  OS Info  : %s (Accuracy: %d%%)\n", pCtx.OSInfo.Name, pCtx.OSInfo.Accuracy)
			}

			if len(ports) > 0 {
				fmt.Printf("  Open Ports (%d):\n", len(ports))
				for _, p := range ports {
					serviceInfo := fmt.Sprintf("%d/tcp", p)
					if svc, ok := pCtx.Services[p]; ok {
						serviceInfo += fmt.Sprintf("\t%s\t%s", svc.Service, svc.Product+" "+svc.Version)
					}
					fmt.Printf("    - %s\n", serviceInfo)
				}
			} else {
				fmt.Println("  No open ports found.")
			}
		}
	}

	fmt.Println("------------------------------------------------------------")
	fmt.Printf("Total Targets Scanned : %d\n", len(r.summaries))
	fmt.Printf("Alive Targets         : %d\n", aliveCount)
	fmt.Printf("Total Open Ports      : %d\n", totalOpenPorts)
	fmt.Println("============================================================")
}

// executePipeline 执行单个 IP 的流水线逻辑
// Linear Flow: Alive -> Port -> Service -> OS -> Report
func (r *AutoRunner) executePipeline(ctx context.Context, pCtx *PipelineContext) {
	// 1. Alive Scan
	// 构造一个临时的 Alive Task
	logger.Debugf("[%s] Starting Alive Scan...", pCtx.IP)
	aliveTask := model.NewTask(model.TaskTypeIpAliveScan, pCtx.IP)
	// 使用默认参数 (Auto Mode)
	// TODO: 允许外部传入参数
	aliveResults, err := r.aliveScanner.Run(ctx, aliveTask)
	if err != nil || len(aliveResults) == 0 {
		logger.Debugf("[%s] Alive scan failed or no result.", pCtx.IP)
		return // 扫描失败或无结果
	}

	// 解析 Alive 结果
	aliveRes := aliveResults[0].Result.(model.IpAliveResult)
	if !aliveRes.Alive {
		logger.Debugf("[%s] Target is not alive.", pCtx.IP)
		return // 目标不存活，直接跳过后续步骤
	}
	logger.Infof("[%s] Target is alive (RTT: %v).", pCtx.IP, aliveRes.RTT)

	pCtx.SetAlive(true)
	// 如果 Alive 扫描带回了 OS/Hostname 信息，先存起来
	if aliveRes.OS != "" {
		pCtx.OSGuess = aliveRes.OS
	}
	if aliveRes.Hostname != "" {
		pCtx.Hostname = aliveRes.Hostname
	}

	// 2. Port Scan
	// 构造 Port Task
	logger.Debugf("[%s] Starting Port Scan (Range: %s)...", pCtx.IP, r.portRange)
	portTask := model.NewTask(model.TaskTypePortScan, pCtx.IP)
	// 使用配置的端口范围 (默认 top1000)
	portTask.PortRange = r.portRange
	// 禁用 Service Detect (为了速度，Service Detect 在下一阶段做，或者合并)
	// 但目前的 PortServiceScanner 把 Port 和 Service 耦合在一起了
	// 如果 params["service_detect"] = false，它只做 TCP Connect
	portTask.Params["service_detect"] = false
	portTask.Params["rate"] = 1000 // 单 IP 内部并发

	portResults, err := r.portScanner.Run(ctx, portTask)
	if err != nil {
		logger.Warnf("[%s] Port scan failed: %v", pCtx.IP, err)
		return
	}

	if len(portResults) == 0 {
		logger.Infof("[%s] No open ports found.", pCtx.IP)
		return
	}
	logger.Infof("[%s] Found %d open ports.", pCtx.IP, len(portResults))

	// 提取开放端口
	var openPorts []int
	for _, res := range portResults {
		if pr, ok := res.Result.(*model.PortServiceResult); ok {
			pCtx.AddOpenPort(pr.Port)
			openPorts = append(openPorts, pr.Port)
		}
	}

	// 3. Service Scan (Version Detection)
	// 针对开放的端口进行服务识别
	// PortServiceScanner 其实支持一次性做完 Port+Service
	// 但为了架构清晰，我们这里演示分步，或者直接复用 PortServiceScanner 的能力
	// 优化：其实 step 2 和 step 3 可以合并，直接让 PortServiceScanner 开启 service_detect
	// 但为了展示 Pipeline 的灵活性，我们假设这里需要针对特定端口做深度扫描

	logger.Debugf("[%s] Starting Service Detection on %d ports...", pCtx.IP, len(openPorts))

	// 这里我们选择：重新对 Open Ports 进行一次带 Service Detect 的扫描
	// 这种做法虽然多了一次连接，但逻辑解耦。
	// 更优解：PortServiceScanner 支持 "Probe List" 模式，跳过 Discovery。
	// 目前 PortServiceScanner 总是先 isPortOpen 再 Scan。
	// 让我们简化：直接在 Step 2 开启 service_detect?
	// 不，Step 2 应该快速。Step 3 精细。

	serviceTask := model.NewTask(model.TaskTypePortScan, pCtx.IP)
	// 将 int slice 转为 string (e.g. "80,443")
	serviceTask.PortRange = portsToString(openPorts)
	serviceTask.Params["service_detect"] = true
	serviceTask.Params["rate"] = 100 // 服务识别并发低一点

	serviceResults, err := r.portScanner.Run(ctx, serviceTask)
	if err == nil {
		for _, res := range serviceResults {
			if pr, ok := res.Result.(*model.PortServiceResult); ok {
				pCtx.SetService(pr.Port, pr)
				logger.Debugf("[%s] Port %d service identified: %s %s", pCtx.IP, pr.Port, pr.Service, pr.Version)
			}
		}
	}
	logger.Infof("[%s] Service Detection completed.", pCtx.IP)

	// 4. OS Scan
	// 结合 TTL (Stage 1) 和 Service Banner (Stage 3) 和 Nmap Stack
	// OS Scanner 支持 "auto" 模式
	logger.Debugf("[%s] Starting OS Detection...", pCtx.IP)
	osInfo, err := r.osScanner.Scan(ctx, pCtx.IP, "auto")
	if err == nil && osInfo != nil {
		pCtx.SetOS(osInfo)
		logger.Infof("[%s] OS Identified: %s (Accuracy: %d%%)", pCtx.IP, osInfo.Name, osInfo.Accuracy)
	} else {
		logger.Debugf("[%s] OS Detection failed or inconclusive.", pCtx.IP)
	}

	// 5. Report / Output
	r.report(pCtx)
}

func (r *AutoRunner) report(pCtx *PipelineContext) {
	// 简单的控制台输出
	// 实际应该调用 Reporter 接口
	console := reporter.NewConsoleReporter()

	fmt.Printf("\n=== Scan Report for %s ===\n", pCtx.IP)
	fmt.Printf("[Alive] Status: UP | OS Guess: %s | Hostname: %s\n", pCtx.OSGuess, pCtx.Hostname)

	if pCtx.OSInfo != nil {
		fmt.Printf("[OS Fingerprint] %s (Accuracy: %d%%)\n", pCtx.OSInfo.Name, pCtx.OSInfo.Accuracy)
	}

	// 转换 Service 结果用于表格输出
	var taskResults []*model.TaskResult
	for _, svc := range pCtx.Services {
		taskResults = append(taskResults, &model.TaskResult{
			Result: svc,
		})
	}
	if len(taskResults) > 0 {
		console.PrintResults(taskResults)
	} else {
		fmt.Println("[Ports] No services identified.")
	}
	fmt.Println("========================================")
}

func portsToString(ports []int) string {
	if len(ports) == 0 {
		return ""
	}
	// 简单实现
	var s string
	for i, p := range ports {
		if i > 0 {
			s += fmt.Sprintf(",%d", p)
		} else {
			s += fmt.Sprintf("%d", p)
		}
	}
	return s
}

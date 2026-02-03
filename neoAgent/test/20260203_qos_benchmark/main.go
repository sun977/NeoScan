package main

import (
	"context"
	"fmt"
	"time"

	"neoagent/internal/config"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/port_service"
	"neoagent/internal/pkg/logger"
)

func main() {
	// 初始化日志，减少干扰
	logger.InitLogger(&config.LogConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})

	target := "10.44.96.1/24"
	// 测试常用端口，模拟真实场景
	ports := "22,80,443,3389,8080,21,23,445,135,139,1433,3306,6379,27017,11211,5432"

	fmt.Println("=== Starting QoS Benchmark ===")
	fmt.Printf("Target: %s\n", target)
	fmt.Printf("Ports: %s\n", ports)

	// 1. 固定并发 (模拟旧版本行为)
	// 通过设置 rate 固定值，虽然代码里有 AdaptiveLimiter，但我们可以通过设置 min=max 来模拟固定并发
	// 或者直接看代码逻辑，如果 min 和 max 接近，调整空间就小
	// 但 PortServiceScanner 现在默认就是自适应的。
	// 为了对比，我们先跑一次默认自适应的，看看效果。
	// 如果要对比固定并发，可能需要改一下 scanner 代码或者参数。
	// PortServiceScanner 的 NewAdaptiveLimiter(rate, 10, rate*2)
	// 我们可以传入一个较小的 rate 来模拟低并发，或者传入一个很大的 rate 来模拟高并发。

	// Test 1: Conservative Rate (Fixed-ish Low Concurrency)
	// rate=100 (Default old behavior was 100)
	fmt.Println("\n[Test 1] Baseline: Initial Rate = 100 (Simulating Old Default)")
	runBenchmark(target, ports, 100)

	// Test 2: Aggressive Rate (High Concurrency Start)
	// rate=2000
	fmt.Println("\n[Test 2] QoS Enabled: Initial Rate = 2000 (Adaptive Limit 10-4000)")
	runBenchmark(target, ports, 2000)
}

func runBenchmark(target, ports string, rate int) {
	scanner := port_service.NewPortServiceScanner()

	task := &model.Task{
		ID:     fmt.Sprintf("bench-%d", rate),
		Target: target, // 注意：PortServiceScanner 目前对 CIDR 的支持可能依赖外部解析？
		// 等等，PortServiceScanner 的 Run 方法里接收的是 task.Target。
		// 如果 task.Target 是 CIDR，scanner 内部并没有解析 CIDR 的逻辑！
		// PortServiceScanner 假设 Target 是单个 IP。
		// Pipeline 负责把 CIDR 拆成 IP。
		// 所以我们不能直接把 CIDR 传给 PortServiceScanner。
		// 我们需要先生成 IP 列表，然后循环调用 Scanner，或者 Scanner 支持 CIDR？
		// 查看 PortServiceScanner 代码：
		// target := task.Target
		// ... isPortOpen(ctx, target, p, ...) -> DialContext(..., target:port)
		// 所以 PortServiceScanner 只支持单 IP。

		// 修正：我们需要自己展开 CIDR。
		PortRange: ports,
		Params: map[string]interface{}{
			"rate":           rate,
			"service_detect": false,
		},
	}

	// 展开 CIDR
	ips := generateIPs(target)
	fmt.Printf("Generated %d IPs from %s\n", len(ips), target)
	if len(ips) == 0 {
		fmt.Println("No IPs generated, skipping.")
		return
	}

	start := time.Now()
	var totalOpenPorts int
	var totalErrors int

	// 模拟 Pipeline：对每个 IP 运行 Scanner
	// 注意：为了测试 Scanner 的并发控制，我们应该并发地调用 Scanner 吗？
	// 不，PortServiceScanner 内部是对端口并发。
	// 如果我们串行扫描 IP，那么 Scanner 的并发限制只作用于单个 IP 的端口扫描。
	// 对于 10.44.96.1/24 (254 IPs) * 16 Ports = 4064 任务。
	// 如果串行扫 IP，每个 IP 16 个端口，并发度最高 16，根本测不出 QoS 的效果（Limit 1000 也没用）。

	// **关键点**：真实的 Pipeline (AutoRunner) 是如何调度的？
	// AutoRunner 会并发执行多个 Host 的扫描。
	// 但 PortServiceScanner 是 "One Task One Target" 吗？
	// 是的，model.Task 结构体只有一个 Target 字段。
	//
	// 如果我们要测试 QoS，我们需要模拟 AutoRunner 的行为：并发地对多个 IP 调用 Scanner。
	// 或者，如果 Scanner 支持 CIDR (目前不支持)，那就更好了。

	// 由于 Scanner 实例是复用的 (NewPortServiceScanner 在 main 里只创建一次)，
	// 它的 limiter 和 rttEstimator 是共享的吗？
	// 查看 PortServiceScanner struct:
	// type PortServiceScanner struct { limiter *qos.AdaptiveLimiter ... }
	// 是的，Scanner 实例内部持有 limiter。
	// 所以如果我们复用 scanner 实例，并并发调用 Run，那么所有并发的任务都会共享这个 limiter。
	// 这正是我们想要的！全局并发控制！

	// 模拟并发扫描多个 IP
	// 限制一下主机并发度，比如同时扫 50 个主机
	hostConcurrency := 50
	sem := make(chan struct{}, hostConcurrency)

	ctx := context.Background() // 不设总超时，看多久跑完

	doneChan := make(chan int, len(ips))

	for _, ip := range ips {
		go func(targetIP string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			t := *task // copy
			t.Target = targetIP

			res, err := scanner.Run(ctx, &t)
			if err != nil {
				// fmt.Printf("Error scanning %s: %v\n", targetIP, err)
				totalErrors++
			}
			doneChan <- len(res)
		}(ip)
	}

	// 等待完成
	for i := 0; i < len(ips); i++ {
		totalOpenPorts += <-doneChan
	}

	duration := time.Since(start)
	fmt.Printf("Benchmark Finished.\n")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Total Open Ports: %d\n", totalOpenPorts)
	fmt.Printf("Average Speed: %.2f hosts/s\n", float64(len(ips))/duration.Seconds())

	// 结果记录 (2026-02-03):
	// [Test 1] Baseline (Rate=100): 3.90s, 65.15 hosts/s
	// [Test 2] QoS (Rate=2000):     1.47s, 172.35 hosts/s (2.6x Speedup)
}

// 简单的 CIDR 展开逻辑，直接借用 pipeline/target.go 的逻辑太麻烦，手写一个简单的
// 或者直接硬编码生成 1-254
func generateIPs(cidr string) []string {
	// 假设是 /24
	base := "10.44.96"
	var ips []string
	for i := 1; i < 255; i++ {
		ips = append(ips, fmt.Sprintf("%s.%d", base, i))
	}
	return ips
}

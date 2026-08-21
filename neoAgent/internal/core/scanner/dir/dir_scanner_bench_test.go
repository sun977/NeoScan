package dir

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/dir/dict"
	"neoagent/internal/core/scanner/dir/engine"
)

// 本文件对应实施文档 Task 4.3 / 设计文档 11.2 节的性能验收标准：
//   - 单线程扫 10K 条字典（内网场景）：目标 ≤ 60s
//   - 25 线程扫 10K 条字典：目标 ≤ 5s
//   - 全量字典加载后的内存占用：目标 ≤ 150MB
//   - 通配符采样开销：目标 ≤ 5 次额外探测请求
//
// 统一使用内置字典（dicc.txt，约 9681 行，接近 10K 目标规模）而非临时构造
// 大文件，更贴近真实扫描场景，也避免每次跑 benchmark 都要重新生成测试数据。

// newBenchServer 启动一个本地 mock 服务器，模拟内网环境下的稳定响应：
// 对全部路径返回 404，不引入随机延迟（基准测试关注扫描器自身开销，
// 网络往返时间由 httptest 的本地 loopback 保证足够小、足够稳定）。
func newBenchServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
}

// newBenchTask 构造一个使用全量内置字典的 dir_scan 任务。
func newBenchTask(serverURL string, threads int) *model.Task {
	task := model.NewTask(model.TaskTypeDirScan, serverURL)
	task.Params["threads"] = threads
	task.Params["timeout"] = 5
	return task
}

// BenchmarkDirScanner_SingleThread 单线程扫全量内置字典（约 10K 条）。
// 验收标准（设计文档 11.2）：内网场景 ≤ 60s。
func BenchmarkDirScanner_SingleThread(b *testing.B) {
	runScanBenchmark(b, 1, 60*time.Second)
}

// BenchmarkDirScanner_25Threads 25 线程扫全量内置字典（约 10K 条）。
// 验收标准（设计文档 11.2）：内网场景 ≤ 5s。
func BenchmarkDirScanner_25Threads(b *testing.B) {
	runScanBenchmark(b, 25, 5*time.Second)
}

// runScanBenchmark 以指定线程数扫描全量内置字典，超过 limit 视为性能回归
// 并报错——benchmark 只打印数字、不做断言的话，回归很容易被人工忽略。
//
// 每轮迭代都用全新的 server + DirScanner，避免状态串扰。可以放心用默认
// 多轮方式（不加 -benchtime）运行：Requester 的 Transport 复用连接
// （engine/requester.go 的 MaxIdleConnsPerHost，且请求不再显式声明
// "Connection: close"），扫一次 9681 条内置字典只建立约 threads 个长
// 连接，不会引发 TIME_WAIT 堆积拖慢后续轮次。
func runScanBenchmark(b *testing.B, threads int, limit time.Duration) {
	b.Helper()

	for i := 0; i < b.N; i++ {
		server := newBenchServer()
		s := NewDirScanner()
		task := newBenchTask(server.URL, threads)

		start := time.Now()
		_, err := s.Run(context.Background(), task)
		elapsed := time.Since(start)
		server.Close()

		if err != nil {
			b.Fatalf("Run() error: %v", err)
		}

		b.ReportMetric(elapsed.Seconds(), "s/scan")
		if elapsed > limit {
			b.Errorf("scan with %d thread(s) took %s, exceeds %s limit", threads, elapsed, limit)
		}
	}
}

// BenchmarkDirScanner_MemoryUsage 测量加载全量内置字典后的内存占用。
// 验收标准（设计文档 11.2）：≤ 150MB。
//
// 用 runtime.ReadMemStats 前后差值衡量一次 dict.New() 调用引入的堆增量，
// 比 b.ReportAllocs()（只统计分配次数/字节，不反映峰值驻留）更贴近
// "字典加载完成后占多少内存"这个验收问题本身。
func BenchmarkDirScanner_MemoryUsage(b *testing.B) {
	const memLimitBytes = 150 * 1024 * 1024 // 150MB

	for i := 0; i < b.N; i++ {
		runtime.GC()
		var before runtime.MemStats
		runtime.ReadMemStats(&before)

		d, err := dict.New(&dict.DirOptions{})
		if err != nil {
			b.Fatalf("dict.New() error: %v", err)
		}

		var after runtime.MemStats
		runtime.ReadMemStats(&after)

		used := int64(after.HeapAlloc) - int64(before.HeapAlloc)
		b.ReportMetric(float64(used)/1024/1024, "MB/op")
		b.ReportMetric(float64(d.Total()), "entries")

		if used > memLimitBytes {
			b.Errorf("dictionary memory usage %.2fMB exceeds %dMB limit", float64(used)/1024/1024, memLimitBytes/1024/1024)
		}
		// 保持引用到 benchmark 结束前，避免被 GC 提前回收导致读数失真。
		runtime.KeepAlive(d)
	}
}

// BenchmarkWildcardScanner_Sampling 验证通配符采样阶段的探测请求开销。
// 验收标准（设计文档 11.2）：≤ 5 次额外请求（即 defaultSampleThreshold）。
//
// 用请求计数 handler 而非计时来衡量"开销"：采样发生在 Precheck() 与
// 首次 Check() 内部，本身耗时受网络往返影响很大、不适合作为性能基准，
// 真正需要控制的是"额外发了多少次探测请求"，这才是采样机制的真实成本。
func BenchmarkWildcardScanner_Sampling(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var probeCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			probeCount++
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "not found")
		}))

		req := engine.NewRequester(engine.RequesterConfig{})
		scanner := engine.NewWildcardScanner(req, server.URL)
		scanner.Precheck(context.Background())

		b.ReportMetric(float64(probeCount), "probe_requests/op")
		if probeCount > 5 {
			b.Errorf("wildcard sampling issued %d probe requests, expected <= 5", probeCount)
		}

		server.Close()
	}
}

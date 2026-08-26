// Package dir 实现目录/路径扫描原子能力（DirScanner）。
// 依赖本包下的 dict/engine/result 子包，不 import 其他扫描器的内部包，
// 也不被其他扫描器 import（原子扫描器隔离原则）。
package dir

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/dir/dict"
	"neoagent/internal/core/scanner/dir/engine"
	"neoagent/internal/core/scanner/dir/result"
	"neoagent/internal/pkg/logger"
)

// DirOptions 是 DirScanner 运行所需的完整参数集合，字段对应设计文档 8.2 节。
// 与 dict.DirOptions（字典构建子集）是两个独立类型，调用 dict.New() 时按需转换，
// 不使用 struct embedding（保持封装边界清晰）。
type DirOptions struct {
	// 字典配置
	Wordlists   []string
	Categories  []string // 技术栈分类字典名称（如 wordpress、php/laravel），见 dict.LoadCategoryWordlist
	Extensions  []string
	ForceExt    bool
	Prefixes    []string
	Suffixes    []string
	Uppercase   bool
	Lowercase   bool
	Capital     bool
	SkipBuiltin bool // 跳过内置字典，仅测试场景使用，见 dict.DirOptions 注释

	// 扫描控制
	Threads           int
	Timeout           time.Duration
	MaxRetries        int
	HTTPMethod        string
	Recursive         bool
	DeepRecursive     bool
	ForceRecursive    bool
	MaxRecursionDepth int
	RateLimit         int
	Delay             time.Duration
	MaxTime           int
	TargetMaxTime     int
	Progress          bool // 实时进度输出（CLI 经 task.Params 开启，server 模式缺省关闭）
	CircuitBreaker    int  // 连续超时熔断阈值（默认 20，0=禁用）

	// 过滤配置
	IncludeStatus   []int
	ExcludeStatus   []int
	ExcludeSize     []int64
	ExcludeKeywords []string
	ExcludeRegex    []*regexp.Regexp

	// 请求配置
	Headers          map[string]string
	FollowRedirects  bool
	UserAgents       []string
	Proxy            string
	IP               string
	NetworkInterface string
}

// defaultDirOptions 返回带默认值的 DirOptions（实施文档 Task 2.2 要求的默认值）。
func defaultDirOptions() *DirOptions {
	return &DirOptions{
		Threads:           25,
		Timeout:           10 * time.Second,
		MaxRetries:        2,
		HTTPMethod:        "GET",
		ExcludeStatus:     engine.DefaultExcludeStatus(),
		MaxRecursionDepth: 3,
		CircuitBreaker:    20,
	}
}

// toDictOptions 转换出 Dictionary 构建所需的选项子集。
func (o *DirOptions) toDictOptions() *dict.DirOptions {
	mode := dict.ExtensionModeClassic
	if o.ForceExt {
		mode = dict.ExtensionModeForce
	}
	return &dict.DirOptions{
		Wordlists:   o.Wordlists,
		Categories:  o.Categories,
		Extensions:  o.Extensions,
		ExtMode:     mode,
		Uppercase:   o.Uppercase,
		Lowercase:   o.Lowercase,
		Capital:     o.Capital,
		Prefixes:    o.Prefixes,
		Suffixes:    o.Suffixes,
		SkipBuiltin: o.SkipBuiltin,
	}
}

// DirScanner 是目录扫描原子扫描器，实现 runner.Runner 接口。
// limiter 以普通字段持有，不使用 Go struct embedding。
type DirScanner struct {
	limiter *qos.AdaptiveLimiter
}

// NewDirScanner 创建 DirScanner，独立实例化自己的限流器（不与其他扫描器共享）。
func NewDirScanner() *DirScanner {
	return &DirScanner{
		limiter: qos.NewAdaptiveLimiter(25, 5, 500),
	}
}

// Name 返回扫描器对应的任务类型。
func (s *DirScanner) Name() model.TaskType {
	return model.TaskTypeDirScan
}

// Run 执行一次完整的目录扫描：初始化字典/请求器/过滤器/通配符检测引擎，
// 启动 Worker 池并发扫描，汇总命中结果后返回。
func (s *DirScanner) Run(ctx context.Context, task *model.Task) (results []*model.TaskResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("DirScanner panic recovered: %v", r)
			logger.Errorf("[DirScanner] panic recovered: %v", r)
		}
	}()

	opts := parseDirOptions(task.Params)
	baseURL := normalizeTarget(task.Target, task.PortRange)

	// 熔断器 context：必须在所有下游（含 Precheck 探测与 worker）之前建立，
	// 熔断触发时取消整个扫描。后续 MaxTime 的 WithTimeout 包装在其外层，
	// 两种取消互不干扰（取消沿 context 链向下传播）。
	ctx, breakerCancel := context.WithCancel(ctx)
	defer breakerCancel()
	breaker := newCircuitBreaker(opts.CircuitBreaker, breakerCancel)

	if opts.Threads > 0 {
		s.limiter = qos.NewAdaptiveLimiter(opts.Threads, 5, opts.Threads*4)
	}

	dictionary, dictErr := dict.New(opts.toDictOptions())
	if dictErr != nil {
		return nil, fmt.Errorf("init dictionary: %w", dictErr)
	}

	requester := engine.NewRequester(engine.RequesterConfig{
		Timeout:          opts.Timeout,
		MaxRetries:       opts.MaxRetries,
		Proxy:            opts.Proxy,
		Method:           opts.HTTPMethod,
		Headers:          opts.Headers,
		UserAgents:       opts.UserAgents,
		RateLimit:        opts.RateLimit,
		Delay:            opts.Delay,
		FollowRedirects:  opts.FollowRedirects,
		IP:               opts.IP,
		NetworkInterface: opts.NetworkInterface,
		MaxConns:         opts.Threads, // 连接池大小对齐并发线程数，见 engine.NewRequester 注释
	})
	// 扫描结束后立即释放空闲连接，不必等 IdleConnTimeout 自然超时，
	// 避免多目标扫描或频繁创建 DirScanner 时连接/goroutine 短暂堆积。
	defer requester.Close()

	filter := engine.NewFilter(engine.FilterConfig{
		IncludeStatus:   opts.IncludeStatus,
		ExcludeStatus:   opts.ExcludeStatus,
		ExcludeSize:     opts.ExcludeSize,
		ExcludeKeywords: opts.ExcludeKeywords,
		ExcludeRegex:    opts.ExcludeRegex,
	})

	wildcardScanner := engine.NewWildcardScanner(requester, baseURL)

	// 预采样：在 worker 启动前对根前缀 "/" 完成通配符采样池初始化，
	// 避免小字典场景下采样阶段吞掉真实命中。
	wildcardScanner.Precheck(ctx)

	// 总扫描时间上限 / 单目标扫描时间上限
	if opts.MaxTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.MaxTime)*time.Second)
		defer cancel()
	}
	if opts.TargetMaxTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.TargetMaxTime)*time.Second)
		defer cancel()
	}

	startTime := time.Now()
	dirResult := result.NewDirResult(task.Target, dictionary.Total())

	rc := &recursionCtl{
		opts:     opts,
		expanded: make(map[string]bool),
	}
	stats := &statsAccumulator{}

	hits := make(chan *result.DirHit, 256)

	// 实时进度输出：进度行走 stderr（单行覆写），命中流式打印走 stdout。
	// 只读 atomics 与字典长度，不与 worker/消费循环竞争。
	// progressWG 保证 goroutine 完全退出后再打印后续诊断（熔断提示等），
	// 避免最终进度行与诊断信息在同一行上竞争交错。
	var progressDone chan struct{}
	var progressWG sync.WaitGroup
	if opts.Progress {
		progressDone = make(chan struct{})
		progressWG.Add(1)
		go func() {
			defer progressWG.Done()
			reportProgress(progressDone, dictionary, stats, startTime)
		}()
	}

	var wg sync.WaitGroup
	for i := 0; i < opts.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(ctx, baseURL, dictionary, requester, filter, wildcardScanner, rc, stats, breaker, hits)
		}()
	}

	go func() {
		wg.Wait()
		close(hits)
	}()

	for h := range hits {
		dirResult.Add(h)
		if opts.Progress {
			printHit(h)
		}
	}
	if progressDone != nil {
		close(progressDone)
		progressWG.Wait() // 等最终进度行完整输出，避免与下面的诊断交错
	}
	// 熔断诊断：扫描已因持续丢包提前终止，已产出的结果仍然有效但必须让用户知情
	if breaker.Tripped() {
		fmt.Fprintf(os.Stderr, "[!] Scan aborted by circuit breaker: target dropped %d+ consecutive requests (possible WAF ban). Partial results kept. Hint: lower --rate-limit/--threads, or retry from another source IP.\n", opts.CircuitBreaker)
	}
	dirResult.Stats = stats.Finalize()
	dirResult.Finish()

	taskResults := []*model.TaskResult{
		{
			TaskID:      task.ID,
			Status:      model.TaskStatusSuccess,
			Result:      dirResult,
			ExecutedAt:  startTime,
			CompletedAt: time.Now(),
		},
	}
	return taskResults, nil
}

// recursionCtl 控制递归展开：原子记录已展开过的目录前缀。
// 一次展开等于"目录前缀 × 全量主字典"的拼接，代价极高；
// main 队列不去重，同一路径可能被请求多次并重复触发展开，
// 必须保证同一目录只展开一次。递归深度不再由此处维护——
// 深度信息编码在路径嵌套层数本身（见 dirDepth）。
type recursionCtl struct {
	opts     *DirOptions
	mu       sync.Mutex
	expanded map[string]bool
}

// tryMarkExpand 原子地检查并标记目录前缀已展开。
// 返回 false 表示该目录此前已展开过，调用方应跳过。
func (r *recursionCtl) tryMarkExpand(dir string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.expanded[dir] {
		return false
	}
	r.expanded[dir] = true
	return true
}

// circuitBreaker 连续超时熔断器（实施文档 Task 9.1）。
//
// 目标进入丢包模式（WAF 封禁扫描 IP 的典型特征）时，超时重试在数学上不可能
// 成功，但每个路径要烧掉 3×timeout+退避 ≈ 31.5s——实测对 zhutix.com 扫描中
// 87% 的时间（104s/120s）被扔进这个黑洞。熔断器区分"偶发失败"（重试有价值）
// 与"持续封禁"（重试无价值）：成功响应即清零连续计数，只有持续性丢包才会
// 达到阈值并取消整个扫描 context，worker 随之自然退出。
type circuitBreaker struct {
	threshold int64
	consec    atomic.Int64 // 连续超时计数（成功响应清零）
	tripped   atomic.Bool
	cancel    context.CancelFunc
}

// newCircuitBreaker 创建熔断器；threshold<=0 或 cancel 为空时返回 nil（禁用，
// 保持旧行为，Never break userspace）。调用方统一以 nil 判断，主路径零分支。
func newCircuitBreaker(threshold int, cancel context.CancelFunc) *circuitBreaker {
	if threshold <= 0 || cancel == nil {
		return nil
	}
	return &circuitBreaker{threshold: int64(threshold), cancel: cancel}
}

// recordTimeout 记录一次超时错误，连续计数达到阈值即熔断（只触发一次）。
func (b *circuitBreaker) recordTimeout() {
	if b == nil {
		return
	}
	if b.consec.Add(1) >= b.threshold && b.tripped.CompareAndSwap(false, true) {
		logger.Warnf("[DirScanner] circuit breaker tripped: %d consecutive timeouts, "+
			"target appears to be dropping packets (possible WAF ban), aborting scan. "+
			"Hint: lower --rate-limit/--threads, or retry later from another source IP", b.consec.Load())
		b.cancel()
	}
}

// recordSuccess 成功响应清零连续超时计数：偶发网络抖动不会累积成误判。
func (b *circuitBreaker) recordSuccess() {
	if b == nil {
		return
	}
	if b.consec.Load() != 0 {
		b.consec.Store(0)
	}
}

// Tripped 返回熔断器是否已触发（供 Run 结束后输出诊断）。
func (b *circuitBreaker) Tripped() bool {
	return b != nil && b.tripped.Load()
}

// statsAccumulator 是 worker 并发写入的统计计数器。
// 字段全部使用 atomic.Int64，避免为一组简单计数器引入额外的锁；
// RTT 相关字段以纳秒存储，扫描结束后由 Finalize() 一次性转换为
// 只读的 result.ScanStats 值对象（该转换发生在所有 worker 退出之后，
// 单线程执行，无需原子读）。
type statsAccumulator struct {
	total      atomic.Int64
	successful atomic.Int64
	filtered   atomic.Int64
	wildcard   atomic.Int64
	errored    atomic.Int64
	sumRTT     atomic.Int64
	maxRTT     atomic.Int64
	minRTT     atomic.Int64
}

// recordRTT 更新最大/最小/累计 RTT（用于最终计算平均值）。
func (s *statsAccumulator) recordRTT(rtt time.Duration) {
	n := int64(rtt)
	if n <= 0 {
		return // 时钟分辨率不足时 time.Since 可返回 0，无统计意义；
		// 且 0 是 minRTT 的"未初始化"哨兵，参与比较会覆盖已设置的正值（flaky test 根因）
	}
	s.sumRTT.Add(n)
	for {
		cur := s.maxRTT.Load()
		if n <= cur || s.maxRTT.CompareAndSwap(cur, n) {
			break
		}
	}
	for {
		cur := s.minRTT.Load()
		if cur != 0 && n >= cur || s.minRTT.CompareAndSwap(cur, n) {
			break
		}
	}
}

// Finalize 把累加结果转换为 result.ScanStats 值对象。
func (s *statsAccumulator) Finalize() result.ScanStats {
	total := s.total.Load()
	successful := s.successful.Load()
	var avg time.Duration
	if successful > 0 {
		avg = time.Duration(s.sumRTT.Load() / successful)
	}
	return result.ScanStats{
		TotalRequests:  int(total),
		SuccessfulReqs: int(successful),
		FilteredReqs:   int(s.filtered.Load()),
		WildcardReqs:   int(s.wildcard.Load()),
		ErrorReqs:      int(s.errored.Load()),
		AvgRTT:         avg,
		MaxRTT:         time.Duration(s.maxRTT.Load()),
		MinRTT:         time.Duration(s.minRTT.Load()),
	}
}

// worker 是 Worker 池中单个 goroutine 的主循环：取路径 → 限速 → 请求 →
// 过滤 → 通配符检测 → 命中上报 → 判断递归。
func (s *DirScanner) worker(
	ctx context.Context,
	baseURL string,
	dictionary *dict.Dictionary,
	requester *engine.Requester,
	filter *engine.Filter,
	wildcardScanner *engine.WildcardScanner,
	rc *recursionCtl,
	stats *statsAccumulator,
	breaker *circuitBreaker,
	hits chan<- *result.DirHit,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		path, ok := dictionary.Next()
		if !ok {
			return
		}

		if err := s.limiter.Acquire(ctx); err != nil {
			return // context 取消
		}

		// 注意：请求间隔延迟（Delay）由 requester.Do 内部统一控制，
		// worker 层不再重复 sleep（此前两处叠加导致 --delay 实际效果翻倍）。

		stats.total.Add(1)
		start := time.Now()
		resp, err := requester.Do(ctx, baseURL, path)
		if err != nil {
			s.limiter.Release()
			stats.errored.Add(1)
			// 仅超时计入熔断。超时有两个语义来源，必须都识别：
			// - ErrConnectionTimeout：网络层 i/o timeout（目标丢包，WAF 封禁特征）
			// - context.DeadlineExceeded：http.Client.Timeout 本地计时器触发，
			//   classifyError 为保留 ctx 语义会原样透传，不会映射为上面的哨兵
			// DNS失败/连接拒绝是快速失败不占线程，不计入；扫描 ctx 取消
			// （Canceled）不会被误判；MaxTime 到期导致的 DeadlineExceeded 即便
			// 计入也无害——扫描本就在结束。
			if errors.Is(err, engine.ErrConnectionTimeout) || errors.Is(err, context.DeadlineExceeded) {
				breaker.recordTimeout()
			}
			continue // 请求失败，跳过该路径
		}
		rtt := time.Since(start)
		stats.successful.Add(1)
		breaker.recordSuccess()
		stats.recordRTT(rtt)

		if !filter.Match(resp, path) {
			s.limiter.Release()
			stats.filtered.Add(1)
			continue
		}

		// 通配符检测可能触发采样探测请求（ensurePoolReady 每前缀最多 5 个），
		// 探测完成前不释放并发槽位——保证"正常请求 + 探测请求"的在飞总数
		// 不超过 Threads 承诺（此前 Release 在 Check 之前，探测可超限 2 倍）。
		isUnique, _ := wildcardScanner.Check(ctx, engine.PathPrefixOf(path), resp)
		s.limiter.Release()
		if !isUnique {
			stats.wildcard.Add(1)
			continue
		}

		select {
		case hits <- &result.DirHit{
			Path:        path,
			Status:      resp.StatusCode,
			Size:        resp.Size,
			Words:       resp.Words,
			Lines:       resp.Lines,
			Title:       resp.Title,
			Location:    resp.Location,
			ContentType: resp.ContentType,
			RTT:         rtt,
		}:
		case <-ctx.Done():
			return
		}

		if rc.opts.Recursive && shouldRecursion(resp, path, rc.opts) && !strings.Contains(path, "..") {
			// 真递归：把命中目录作为前缀，与主字典拼接展开子路径
			// （命中 /admin/ → 扫描 /admin/<字典每条目>）。
			// 深度模式额外展开命中路径的所有父级目录。
			// 含 ".." 的字典探测条目（目录穿越测试）展开父级只会生成
			// 畸形前缀并消耗整本字典，禁止其触发递归。
			dirs := []string{ensureTrailingSlash(path)}
			if rc.opts.DeepRecursive {
				dirs = parentDirsOf(path)
			}
			entries := dictionary.MainEntries()
			for _, dir := range dirs {
				depth := dirDepth(dir)
				if depth == 0 {
					continue // 根目录展开等于重放整个 main 队列，永远无意义（且 seen 不跟踪 main，会全量重复）
				}
				if depth > rc.opts.MaxRecursionDepth {
					continue
				}
				if !rc.tryMarkExpand(dir) {
					continue // 该目录已展开过，跳过
				}
				dictionary.AddExtra(generateSubpaths(dir, entries)...)
			}
		}
	}
}

// ── 实时进度输出（Progress） ────────────────────────────────────────────────

// reportProgress 每 2 秒向 stderr 输出一行进度（单行覆写），扫描结束时
// 输出最终行并换行。只读 atomics 与字典长度，无锁、不影响 worker。
func reportProgress(done <-chan struct{}, dictionary *dict.Dictionary, stats *statsAccumulator, start time.Time) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			fmt.Fprintf(os.Stderr, "\r%-110s\n", progressLine(dictionary, stats, start))
			return
		case <-ticker.C:
			fmt.Fprintf(os.Stderr, "\r%-110s", progressLine(dictionary, stats, start))
		}
	}
}

// progressLine 汇总当前扫描进度为一行文本。
// 命中数不读 dirResult（避免与主消费 goroutine 竞争），由
// successful - filtered - wildcard 推导：每个成功响应必走三者之一。
// 分母 total+remaining 是"当前已知工作量"——递归会动态追加队列，
// 进度百分比可能回退，这与 dirsearch 的行为一致。
func progressLine(dictionary *dict.Dictionary, stats *statsAccumulator, start time.Time) string {
	total := stats.total.Load()
	succ := stats.successful.Load()
	filtered := stats.filtered.Load()
	wildcard := stats.wildcard.Load()
	errored := stats.errored.Load()
	hits := succ - filtered - wildcard
	remaining := int64(dictionary.Len())
	elapsed := time.Since(start)

	rate := float64(total) / elapsed.Seconds()
	eta := "unknown"
	if rate > 0.01 && remaining > 0 {
		eta = (time.Duration(float64(remaining)/rate) * time.Second).Round(time.Second).String()
	}
	return fmt.Sprintf("[progress] %d/%d done | %.1f req/s | hits %d | wildcard %d | filtered %d | err %d | elapsed %s | ETA %s",
		total, total+remaining, rate, hits, wildcard, filtered, errored, elapsed.Round(time.Second), eta)
}

// printHit 命中即时流式打印（stdout），列与最终表格保持一致。
func printHit(h *result.DirHit) {
	loc := h.Location
	if loc == "" {
		loc = "-"
	}
	title := h.Title
	if title == "" {
		title = "-"
	}
	fmt.Printf("[hit] %d %s (size:%d, words:%d, title:%q, redirect:%s)\n",
		h.Status, h.Path, h.Size, h.Words, title, loc)
}

// shouldRecursion 判断该命中路径是否应当触发递归扫描（三种模式，见设计文档 7.1 节）。
func shouldRecursion(resp *engine.Response, path string, opts *DirOptions) bool {
	if opts.ForceRecursive {
		return true
	}
	if opts.DeepRecursive {
		// 深度递归只对目录类响应展开：403 封禁页、400 畸形请求这类
		// "命中"没有目录信息价值，对它们重放整本字典是纯粹的流量放大
		// （真实复现：对反爬统一封禁页递归导致请求量 ×6.4，跑满 2h 超时）。
		return isDirectoryStatus(resp.StatusCode)
	}
	// 基础递归：仅对目录类型路径递归（以 / 结尾 且 状态码属于目录类响应）
	return strings.HasSuffix(path, "/") && isDirectoryStatus(resp.StatusCode)
}

// isDirectoryStatus 判断状态码是否表示该路径可能是一个可继续扫描的目录。
func isDirectoryStatus(status int) bool {
	switch status {
	case 200, 301, 302, 403:
		return true
	default:
		return false
	}
}

// generateSubpaths 把目录前缀与主字典条目拼接，生成待扫描的子路径集合。
// 这是递归扫描的核心语义：命中目录 /admin/ 后，字典在该前缀下重放
// （/admin/config.php、/admin/login ...），而非旧实现的"重复请求目录本身"。
// 字典条目的前导 "/" 统一去除后拼接，避免 "//" 双斜杠。
func generateSubpaths(dir string, entries []string) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, dir+strings.TrimPrefix(e, "/"))
	}
	return out
}

// parentDirsOf 返回路径的所有父级目录前缀（含自身的目录形式）。
// 例："/a/b/c" → ["/a/", "/a/b/", "/a/b/c/"]。供深度递归模式使用。
func parentDirsOf(path string) []string {
	base := ensureTrailingSlash(path)
	parts := strings.Split(strings.Trim(base, "/"), "/")
	dirs := make([]string, 0, len(parts))
	for i := 1; i <= len(parts); i++ {
		dirs = append(dirs, "/"+strings.Join(parts[:i], "/")+"/")
	}
	return dirs
}

// dirDepth 返回目录前缀的嵌套层数："/" → 0，"/admin/" → 1，"/admin/api/" → 2。
// 递归深度直接由路径结构决定，无需额外状态（深度信息编码在路径本身）。
func dirDepth(dir string) int {
	n := strings.Count(dir, "/")
	if n == 0 {
		return 0
	}
	return n - 1
}

// ensureTrailingSlash 确保路径以 / 结尾，作为目录前缀使用。
func ensureTrailingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}
	return path + "/"
}

// normalizeTarget 把 Target/PortRange 换算成可直接发起请求的 base URL。
// 独立实现，不 import 其他扫描器的同名函数（原子扫描器隔离原则）。
func normalizeTarget(target, portRange string) string {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return strings.TrimSuffix(target, "/")
	}

	port := firstPort(portRange)
	host := target
	if port != "" && port != "80" && port != "443" && !strings.Contains(target, ":") {
		host = target + ":" + port
	}

	switch port {
	case "443", "8443", "9443", "10443":
		return "https://" + host
	default:
		return "http://" + host
	}
}

// firstPort 从逗号分隔的端口列表中取第一个非空项。
func firstPort(ports string) string {
	if ports == "" {
		return ""
	}
	parts := strings.Split(ports, ",")
	return strings.TrimSpace(parts[0])
}

// parseDirOptions 从 task.Params 中解析出 DirOptions，未提供的字段使用默认值。
// key 命名统一使用下划线（与 options.DirScanOptions.ToTask() 的约定一致）。
func parseDirOptions(params map[string]interface{}) *DirOptions {
	opts := defaultDirOptions()
	if params == nil {
		return opts
	}

	if v := paramString(params, "wordlists"); v != "" {
		opts.Wordlists = splitCSV(v)
	}
	if v := paramString(params, "category"); v != "" {
		opts.Categories = splitCSV(v)
	}
	if v := paramString(params, "extensions"); v != "" {
		opts.Extensions = splitCSV(v)
	}
	opts.ForceExt = paramBool(params, "force_extensions", opts.ForceExt)
	if v := paramString(params, "prefixes"); v != "" {
		opts.Prefixes = splitCSV(v)
	}
	if v := paramString(params, "suffixes"); v != "" {
		opts.Suffixes = splitCSV(v)
	}
	opts.Uppercase = paramBool(params, "uppercase", opts.Uppercase)
	opts.Lowercase = paramBool(params, "lowercase", opts.Lowercase)
	opts.Capital = paramBool(params, "capital", opts.Capital)

	opts.Threads = paramInt(params, "threads", opts.Threads)
	opts.Timeout = paramDurationSeconds(params, "timeout", opts.Timeout)
	opts.MaxRetries = paramInt(params, "max_retries", opts.MaxRetries)
	if v := paramString(params, "method"); v != "" {
		opts.HTTPMethod = v
	}
	opts.Recursive = paramBool(params, "recursive", opts.Recursive)
	opts.DeepRecursive = paramBool(params, "deep_recursive", opts.DeepRecursive)
	opts.ForceRecursive = paramBool(params, "force_recursive", opts.ForceRecursive)
	// ForceRecursive 优先级高于 DeepRecursive，消除语义歧义
	if opts.ForceRecursive {
		opts.DeepRecursive = false
	}
	// 对齐 dirsearch：deep/force-recursive 隐式开启递归。
	// Recursive 默认关闭后，单独指定 deep/force 不应静默退化为不递归。
	if opts.DeepRecursive || opts.ForceRecursive {
		opts.Recursive = true
	}
	opts.Progress = paramBool(params, "progress", opts.Progress)
	opts.CircuitBreaker = paramInt(params, "circuit_breaker", opts.CircuitBreaker)
	opts.MaxRecursionDepth = paramInt(params, "max_recursion_depth", opts.MaxRecursionDepth)
	opts.RateLimit = paramInt(params, "rate_limit", opts.RateLimit)
	opts.Delay = paramDurationMillis(params, "delay", opts.Delay)
	opts.MaxTime = paramInt(params, "max_time", opts.MaxTime)
	opts.TargetMaxTime = paramInt(params, "target_max_time", opts.TargetMaxTime)

	if v := paramString(params, "include_status"); v != "" {
		opts.IncludeStatus = parseIntListCSV(v)
	}
	if v := paramString(params, "exclude_status"); v != "" {
		opts.ExcludeStatus = parseIntListCSV(v)
	}
	if v := paramString(params, "exclude_size"); v != "" {
		opts.ExcludeSize = parseInt64ListCSV(v)
	}
	if v := paramString(params, "exclude_text"); v != "" {
		opts.ExcludeKeywords = splitCSV(v)
	}
	if v := paramString(params, "exclude_regex"); v != "" {
		if re, err := regexp.Compile(v); err == nil {
			opts.ExcludeRegex = []*regexp.Regexp{re}
		} else {
			logger.Warnf("[DirScanner] invalid exclude_regex %q: %v", v, err)
		}
	}

	opts.SkipBuiltin = paramBool(params, "skip_builtin", opts.SkipBuiltin)
	opts.FollowRedirects = paramBool(params, "follow_redirects", opts.FollowRedirects)
	if v := paramString(params, "proxy"); v != "" {
		opts.Proxy = v
	}
	if v := paramString(params, "ip"); v != "" {
		opts.IP = v
	}
	if v := paramString(params, "network_interface"); v != "" {
		opts.NetworkInterface = v
	}

	// User-Agent：random_agent 优先级高于单个 user_agent（内置池随机轮换，
	// 见 engine.Requester.randomUA 的多值随机选择逻辑）。
	if paramBool(params, "random_agent", false) {
		if uas, err := dict.LoadUserAgents(); err == nil && len(uas) > 0 {
			opts.UserAgents = uas
		} else if err != nil {
			logger.Warnf("[DirScanner] failed to load builtin user-agents: %v", err)
		}
	} else if v := paramString(params, "user_agent"); v != "" {
		opts.UserAgents = []string{v}
	}

	// 自定义请求头："Key: Value" 按行分隔（CLI 层 -H 可多次指定，合并为多行字符串）。
	if v := paramString(params, "headers"); v != "" {
		opts.Headers = parseHeaderLines(v)
	}
	// Basic Auth："user:pass" 直接编码为 Authorization 头，覆盖同名自定义头。
	if v := paramString(params, "auth"); v != "" {
		if opts.Headers == nil {
			opts.Headers = make(map[string]string)
		}
		opts.Headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(v))
	}

	return opts
}

// parseHeaderLines 解析形如 "Key: Value\nKey2: Value2" 的多行请求头字符串。
// 每行按第一个冒号切分，忽略空行和格式非法的行（记录警告但不中断扫描）。
func parseHeaderLines(raw string) map[string]string {
	headers := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			logger.Warnf("[DirScanner] invalid header line %q, expected \"Key: Value\"", line)
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		headers[key] = value
	}
	return headers
}

// ── task.Params 解析辅助函数 ──────────────────────────────────────────────

func paramString(params map[string]interface{}, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func paramBool(params map[string]interface{}, key string, def bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func paramInt(params map[string]interface{}, key string, def int) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return def
}

func paramDurationSeconds(params map[string]interface{}, key string, def time.Duration) time.Duration {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case int:
			return time.Duration(n) * time.Second
		case float64:
			return time.Duration(n) * time.Second
		}
	}
	return def
}

func paramDurationMillis(params map[string]interface{}, key string, def time.Duration) time.Duration {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case int:
			return time.Duration(n) * time.Millisecond
		case float64:
			return time.Duration(n) * time.Millisecond
		}
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func parseIntListCSV(s string) []int {
	parts := splitCSV(s)
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		if n, err := strconv.Atoi(p); err == nil {
			result = append(result, n)
		}
	}
	return result
}

func parseInt64ListCSV(s string) []int64 {
	parts := splitCSV(s)
	result := make([]int64, 0, len(parts))
	for _, p := range parts {
		if n, err := strconv.ParseInt(p, 10, 64); err == nil {
			result = append(result, n)
		}
	}
	return result
}

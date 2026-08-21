// Package dir 实现目录/路径扫描原子能力（DirScanner）。
// 依赖本包下的 dict/engine/result 子包，不 import 其他扫描器的内部包，
// 也不被其他扫描器 import（原子扫描器隔离原则）。
package dir

import (
	"context"
	"encoding/base64"
	"fmt"
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
		opts:  opts,
		depth: make(map[string]int),
	}
	stats := &statsAccumulator{}

	hits := make(chan *result.DirHit, 256)
	var wg sync.WaitGroup
	for i := 0; i < opts.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(ctx, baseURL, dictionary, requester, filter, wildcardScanner, rc, stats, hits)
		}()
	}

	go func() {
		wg.Wait()
		close(hits)
	}()

	for h := range hits {
		dirResult.Add(h)
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

// recursionCtl 记录已知路径的递归深度，控制递归调度不超过 MaxRecursionDepth。
// 与 Dictionary 分离维护，因为深度信息不属于字典迭代器的职责。
type recursionCtl struct {
	opts  *DirOptions
	mu    sync.Mutex
	depth map[string]int
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

// depthOf 返回路径已记录的递归深度，未记录视为第 0 层（初始字典条目）。
func (r *recursionCtl) depthOf(path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.depth[path]
}

// markDepth 记录子路径的递归深度。
func (r *recursionCtl) markDepth(path string, depth int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.depth[path] = depth
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

		if rc.opts.Delay > 0 {
			select {
			case <-ctx.Done():
				s.limiter.Release()
				return
			case <-time.After(rc.opts.Delay):
			}
		}

		stats.total.Add(1)
		start := time.Now()
		resp, err := requester.Do(ctx, baseURL, path)
		s.limiter.Release()
		if err != nil {
			stats.errored.Add(1)
			continue // 请求失败，跳过该路径
		}
		rtt := time.Since(start)
		stats.successful.Add(1)
		stats.recordRTT(rtt)

		if !filter.Match(resp, path) {
			stats.filtered.Add(1)
			continue
		}

		isUnique, _ := wildcardScanner.Check(ctx, engine.PathPrefixOf(path), resp)
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

		if rc.opts.Recursive && shouldRecursion(resp, path, rc.opts) {
			depth := rc.depthOf(path)
			if depth < rc.opts.MaxRecursionDepth {
				subpaths := generateSubpaths(path, rc.opts.DeepRecursive)
				for _, sp := range subpaths {
					rc.markDepth(sp, depth+1)
				}
				dictionary.AddExtra(subpaths...)
			}
		}
	}
}

// shouldRecursion 判断该命中路径是否应当触发递归扫描（三种模式，见设计文档 7.1 节）。
func shouldRecursion(resp *engine.Response, path string, opts *DirOptions) bool {
	if opts.ForceRecursive {
		return true
	}
	if opts.DeepRecursive {
		return true
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

// generateSubpaths 为命中路径生成待追加扫描的子路径集合。
//   - 基础模式：命中的目录路径本身作为扫描前缀，与内置字典拼接由 Dictionary 负责，
//     此处只需把命中路径原样加入 extra 队列，触发以该路径为前缀的新一轮扫描。
//   - 深度模式：额外把命中路径的每一级父目录也加入扫描队列。
//     例："/api/users/" → 追加 "/api/users/"、"/api/"。
func generateSubpaths(path string, deep bool) []string {
	base := ensureTrailingSlash(path)
	if !deep {
		return []string{base}
	}

	parts := strings.Split(strings.Trim(base, "/"), "/")
	subpaths := make([]string, 0, len(parts))
	for i := 1; i <= len(parts); i++ {
		subpaths = append(subpaths, "/"+strings.Join(parts[:i], "/")+"/")
	}
	return subpaths
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

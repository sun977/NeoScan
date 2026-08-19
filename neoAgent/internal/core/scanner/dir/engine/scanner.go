// Package engine 提供 DirScanner 的 HTTP 请求、过滤和通配符检测能力。
// 此包仅供 scanner/dir 包内部使用（原子扫描器隔离原则）。
package engine

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"path"
	"regexp"
	"strings"
	"sync"
)

// 设计说明（对齐 dirsearch lib/core/scanner.py 的 Scanner.setup()）：
//
// dirsearch 把"采样"和"判定"严格分成两个阶段：Fuzzer.setup_scanners() 在
// worker 线程启动之前，同步、阻塞地为每个 pathPrefix 创建 Scanner 并完成
// 两次探测采样；worker 线程池启动后，Scanner.check() 只读取已经采好、
// 不再变化的 self.response/self.content_parser，天然没有竞争。
//
// 我们之前的实现把这两个阶段揉进了同一个 Check() 里，用运行时状态
// （len(pool.bodies) < threshold）判断"是否还在采样"，导致高并发下
// 多个 goroutine 同时误判"未采满"、同时 append、同时触发探测请求，
// 这是产生 data race 与误判的根源。
//
// 这里对齐 dirsearch 的做法：用 sync.Once 把"采样"收拢成一次性、
// 只由第一个到达的 goroutine 执行、其他 goroutine 阻塞等待的动作；
// 采样完成后 samplePool 的字段全部固化为只读，Check() 的主路径不再
// 需要为每次调用加锁读写采样状态。

// 归一化正则，包级编译一次，避免每次调用重新编译（见实施文档约束）。
var (
	reUUID = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	reHex  = regexp.MustCompile(`[0-9a-f]{32,}`)
	reTS   = regexp.MustCompile(`\d{10,}`)
	reB64  = regexp.MustCompile(`[A-Za-z0-9+/]{40,}={0,2}`)
)

const (
	// defaultSampleThreshold 通配符采样阈值：达到该样本数后开始比对判定。
	defaultSampleThreshold = 5
	// defaultFingerprintThreshold 自校准阈值：同一指纹出现次数超过该值，直接判定为通配符。
	defaultFingerprintThreshold = 10
	// similarityThreshold 静态词命中比例阈值，达到该比例视为通配符响应。
	similarityThreshold = 0.9
	// lengthDiffThreshold 响应词数相对基准的最大允许膨胀比例。
	lengthDiffThreshold = 1.35
)

// samplePool 保存单个路径前缀的采样与判定状态。
//
// once 保证同一个 pathPrefix 的采样探测只会被第一个到达的 goroutine 执行
// 一次，其余并发到达的 goroutine 会阻塞在 once.Do 上，直到采样完成——
// 对齐 dirsearch Scanner.setup() 的"先采样、后判定"两阶段模型。
// once.Do 返回之后，下面几个字段全部固化为只读，不再需要加锁访问。
type samplePool struct {
	once           sync.Once
	bodies         []string // 采样响应体（仅采样阶段写入，之后只读）
	staticPatterns []string // 提取出的共同静态词（只读）
	baseWordCount  int      // 基准词数（只读）
	ready          bool     // 是否已完成采样固化（用于 sample 请求全部失败的兜底判断）
}

// WildcardScanner 移植 dirsearch 的 Scanner + DynamicContentParser 算法，
// 用于识别"任意路径都返回同一动态内容"的通配符响应（典型如 CDN 统一 404 页）。
type WildcardScanner struct {
	requester *Requester
	baseURL   string

	sampleThreshold   int // 采样阈值，默认 5
	fingerprintThresh int // 自校准阈值，默认 10

	poolsMu sync.Mutex             // 仅保护 pools map 本身的读写，不保护 pool 内部字段
	pools   map[string]*samplePool // pathPrefix → 采样状态

	fpMu         sync.Mutex      // 保护 fingerprints/wildCardFPs 两个 map
	fingerprints map[string]int  // fingerprint → 出现次数
	wildCardFPs  map[string]bool // 已确认的通配符指纹
}

// NewWildcardScanner 创建通配符检测引擎。
// req 用于发送采样探测请求，baseURL 是扫描目标基础地址。
func NewWildcardScanner(req *Requester, baseURL string) *WildcardScanner {
	return &WildcardScanner{
		requester:         req,
		baseURL:           baseURL,
		sampleThreshold:   defaultSampleThreshold,
		fingerprintThresh: defaultFingerprintThreshold,
		pools:             make(map[string]*samplePool),
		fingerprints:      make(map[string]int),
		wildCardFPs:       make(map[string]bool),
	}
}

// Precheck 对根前缀 "/" 预先完成采样，等价于 dirsearch 在 worker 启动前
// 调用 Fuzzer.setup_scanners() 同步初始化默认 Scanner。
// 在 worker 启动前调用，确保正式扫描开始时根前缀采样池已就绪，
// 避免小字典场景下采样阶段吞掉真实命中。
func (s *WildcardScanner) Precheck(ctx context.Context) {
	s.ensurePoolReady(ctx, "/")
}

// Check 判断给定路径的响应是否为独立、真实存在的内容。
//
// 返回 (true, "unique") 表示应当作为命中结果输出；
// 返回 (false, reason) 表示应当被丢弃（通配符/采样中/已知通配符指纹）。
//
// 并发模型对齐 dirsearch：ensurePoolReady 保证同一 pathPrefix 的采样只由
// 一个 goroutine 执行、其余并发调用阻塞等待完成，采样完成后 pool 的
// staticPatterns/baseWordCount 固化为只读，本函数不再需要为每次判定加锁。
func (s *WildcardScanner) Check(ctx context.Context, pathPrefix string, resp *Response) (bool, string) {
	fp := fingerprint(resp)

	if s.isKnownWildcardFP(fp) {
		return false, "wildcard"
	}

	pool := s.ensurePoolReady(ctx, pathPrefix)
	if !pool.ready {
		// 采样请求全部失败（如目标不可达），无基准可比对，保守地当作
		// 采样失败处理：不压制也不误判，直接放行由 Filter 层的状态码/
		// 大小规则兜底，避免因通配符引擎自身故障导致全量丢弃命中。
		return true, "unique"
	}

	if compareToStaticPatterns(pool, resp) {
		s.autoCalibrate(fp)
		return false, "match"
	}

	s.autoCalibrate(fp)
	return true, "unique"
}

// isKnownWildcardFP 判断指纹是否已被自校准标记为通配符。
func (s *WildcardScanner) isKnownWildcardFP(fp string) bool {
	s.fpMu.Lock()
	defer s.fpMu.Unlock()
	return s.wildCardFPs[fp]
}

// ensurePoolReady 返回 pathPrefix 对应、采样已完成的 samplePool。
//
// 第一个到达的 goroutine 负责发起 sampleThreshold 次探测请求（对齐
// dirsearch setup() 的 first/second probe + auto-calibrate 额外样本）；
// 同一 pathPrefix 上后续并发到达的 goroutine 会阻塞在 pool.once.Do 上，
// 直到采样完成后才继续，读到的是已经固化的只读字段——不存在"边采样
// 边被其他 goroutine 读取"的窗口，从根上消除 data race。
func (s *WildcardScanner) ensurePoolReady(ctx context.Context, pathPrefix string) *samplePool {
	pool := s.getOrCreatePool(pathPrefix)
	pool.once.Do(func() {
		if s.requester == nil {
			return
		}
		for i := 0; i < s.sampleThreshold; i++ {
			probePath := pathPrefix + randStealthWord()
			resp, err := s.requester.Do(ctx, s.baseURL, probePath)
			if err != nil {
				continue
			}
			pool.bodies = append(pool.bodies, resp.Body)
		}
		if len(pool.bodies) >= 2 {
			pool.staticPatterns, pool.baseWordCount = extractStaticPatterns(pool.bodies[0], pool.bodies[1])
			pool.ready = true
		}
	})
	return pool
}

// getOrCreatePool 按需创建路径前缀对应的采样池（只保护 map 本身，不保护
// pool 内部字段——内部字段的并发安全由 pool.once 负责）。
func (s *WildcardScanner) getOrCreatePool(pathPrefix string) *samplePool {
	s.poolsMu.Lock()
	defer s.poolsMu.Unlock()
	pool := s.pools[pathPrefix]
	if pool == nil {
		pool = &samplePool{}
		s.pools[pathPrefix] = pool
	}
	return pool
}

// compareToStaticPatterns 判断响应是否匹配采样阶段提取的静态模式
// （即为通配符响应）。pool 此时已经 ready，字段只读，无需加锁。
func compareToStaticPatterns(pool *samplePool, resp *Response) bool {
	if len(pool.staticPatterns) == 0 {
		return false
	}

	body := normalizeDynamic(resp.Body)
	for _, p := range pool.staticPatterns {
		if !strings.Contains(body, p) {
			return false
		}
	}

	wordCount := len(strings.Fields(body))
	if pool.baseWordCount > 0 && float64(wordCount) > float64(pool.baseWordCount)*lengthDiffThreshold {
		return false
	}

	return true
}

// autoCalibrate 统计指纹出现频率，超过阈值的指纹自动加入通配符黑名单。
func (s *WildcardScanner) autoCalibrate(fp string) {
	s.fpMu.Lock()
	defer s.fpMu.Unlock()

	s.fingerprints[fp]++
	if s.fingerprints[fp] > s.fingerprintThresh {
		s.wildCardFPs[fp] = true
	}
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────────

// normalizeDynamic 移除响应体中的动态内容（UUID/长hex/时间戳/Base64），
// 使得同一模板渲染出的不同响应能够被归一化为相同文本用于比对。
func normalizeDynamic(content string) string {
	content = reUUID.ReplaceAllString(content, "[UUID]")
	content = reHex.ReplaceAllString(content, "[HEX]")
	content = reTS.ReplaceAllString(content, "[TS]")
	content = reB64.ReplaceAllString(content, "[B64]")
	return content
}

// extractStaticPatterns 提取两个响应体中共同出现的词（静态模式），
// 并返回其中较短响应的词数作为基准词数。
func extractStaticPatterns(body1, body2 string) ([]string, int) {
	norm1 := normalizeDynamic(body1)
	norm2 := normalizeDynamic(body2)

	words1 := strings.Fields(norm1)
	words2 := strings.Fields(norm2)

	set2 := make(map[string]int, len(words2))
	for _, w := range words2 {
		set2[w]++
	}

	seen := make(map[string]bool, len(words1))
	var common []string
	for _, w := range words1 {
		if seen[w] {
			continue
		}
		if set2[w] > 0 {
			common = append(common, w)
			seen[w] = true
		}
	}

	baseCount := len(words1)
	if len(words2) < baseCount {
		baseCount = len(words2)
	}

	return common, baseCount
}

// fingerprint 生成响应指纹，格式：status|contentType|bodyLen/64|bodyHash前4字节。
func fingerprint(resp *Response) string {
	limit := len(resp.Body)
	if limit > 4096 {
		limit = 4096
	}
	hash := sha256.Sum256([]byte(resp.Body[:limit]))
	return fmt.Sprintf("%d|%s|%d|%x", resp.StatusCode, resp.ContentType, len(resp.Body)/64, hash[:4])
}

// randStealthWord 生成随机 8 字符字符串，用于探测不存在的路径，
// 避免与字典中真实存在的路径产生冲突。
// 仅用于探测碰撞，无需密码学安全强度，使用 math/rand 避免 crypto/rand
// 在部分系统熵不足时阻塞导致扫描卡顿。
func randStealthWord() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, 8)
	for i := range buf {
		buf[i] = charset[rand.Intn(len(charset))]
	}
	return string(buf)
}

// PathPrefixOf 提取路径所在的目录前缀，用于对同一目录下的路径共享采样池。
// 例："/admin/config.php" → "/admin/"，"/admin" → "/"。
// 导出供 dir_scanner.go 在调用 WildcardScanner.Check 前计算 pathPrefix。
func PathPrefixOf(p string) string {
	dir := path.Dir(p)
	if dir == "." {
		dir = "/"
	}
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}

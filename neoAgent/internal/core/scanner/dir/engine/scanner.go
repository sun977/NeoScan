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
type samplePool struct {
	bodies         []string // 采样响应体
	staticPatterns []string // 提取出的共同静态词
	baseWordCount  int      // 基准词数（采样阶段第一个响应的词数）
	confirmed      bool     // 是否已确认为通配符（isStatic）
}

// WildcardScanner 移植 dirsearch 的 Scanner + DynamicContentParser 算法，
// 用于识别"任意路径都返回同一动态内容"的通配符响应（典型如 CDN 统一 404 页）。
type WildcardScanner struct {
	requester *Requester
	baseURL   string

	sampleThreshold   int // 采样阈值，默认 5
	fingerprintThresh int // 自校准阈值，默认 10

	mu           sync.Mutex
	pools        map[string]*samplePool // pathPrefix → 采样状态
	fingerprints map[string]int         // fingerprint → 出现次数
	wildCardFPs  map[string]bool        // 已确认的通配符指纹
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

// Precheck 对根前缀 "/" 预先发送随机探测请求，完成采样池初始化。
// 在 worker 启动前调用，确保正式扫描开始时根前缀采样池已就绪，
// 避免小字典场景下采样阶段吞掉真实命中。
func (s *WildcardScanner) Precheck(ctx context.Context) {
	if s.requester == nil {
		return
	}
	pool := &samplePool{}
	s.pools["/"] = pool
	for i := 0; i < s.sampleThreshold; i++ {
		probePath := "/" + randStealthWord()
		resp, err := s.requester.Do(ctx, s.baseURL, probePath)
		if err != nil {
			continue
		}
		pool.bodies = append(pool.bodies, resp.Body)
	}
	if len(pool.bodies) >= s.sampleThreshold {
		s.finalizeSamples(pool)
	}
}

// Check 判断给定路径的响应是否为独立、真实存在的内容。
//
// 返回 (true, "unique") 表示应当作为命中结果输出；
// 返回 (false, reason) 表示应当被丢弃（通配符/采样中/已知通配符指纹）。
func (s *WildcardScanner) Check(ctx context.Context, pathPrefix string, resp *Response) (bool, string) {
	fp := fingerprint(resp)

	s.mu.Lock()
	if s.wildCardFPs[fp] {
		s.mu.Unlock()
		return false, "wildcard"
	}

	pool := s.pools[pathPrefix]
	if pool == nil {
		pool = &samplePool{}
		s.pools[pathPrefix] = pool
	}
	s.mu.Unlock()

	// 采样未完成：记录当前样本，发送一次探测请求补充样本，返回压制输出。
	if len(pool.bodies) < s.sampleThreshold {
		s.sample(ctx, pathPrefix, pool, resp)
		if len(pool.bodies) < s.sampleThreshold {
			return false, "sampling"
		}
		s.finalizeSamples(pool)
	}

	// 采样完成后比对：命中静态模式 → 判定为通配符。
	if s.compare(pool, resp) {
		s.autoCalibrate(fp)
		return false, "match"
	}

	s.autoCalibrate(fp)
	return true, "unique"
}

// sample 将当前响应体计入采样池，并主动发送一次随机探测请求补充样本，
// 使采样池尽快达到 sampleThreshold，减少对真实路径的依赖。
func (s *WildcardScanner) sample(ctx context.Context, pathPrefix string, pool *samplePool, resp *Response) {
	s.mu.Lock()
	pool.bodies = append(pool.bodies, resp.Body)
	needProbe := len(pool.bodies) < s.sampleThreshold
	s.mu.Unlock()

	if !needProbe || s.requester == nil {
		return
	}

	probePath := pathPrefix + randStealthWord()
	probeResp, err := s.requester.Do(ctx, s.baseURL, probePath)
	if err != nil {
		return
	}

	s.mu.Lock()
	pool.bodies = append(pool.bodies, probeResp.Body)
	s.mu.Unlock()
}

// finalizeSamples 从采样池中提取静态模式和基准词数（仅执行一次）。
func (s *WildcardScanner) finalizeSamples(pool *samplePool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if pool.confirmed || len(pool.bodies) < 2 {
		return
	}
	patterns, baseCount := extractStaticPatterns(pool.bodies[0], pool.bodies[1])
	pool.staticPatterns = patterns
	pool.baseWordCount = baseCount
	pool.confirmed = true
}

// compare 判断响应是否匹配已提取的静态模式（即为通配符响应）。
func (s *WildcardScanner) compare(pool *samplePool, resp *Response) bool {
	s.mu.Lock()
	patterns := pool.staticPatterns
	baseCount := pool.baseWordCount
	s.mu.Unlock()

	if len(patterns) == 0 {
		return false
	}

	body := normalizeDynamic(resp.Body)
	for _, p := range patterns {
		if !strings.Contains(body, p) {
			return false
		}
	}

	wordCount := len(strings.Fields(body))
	if baseCount > 0 && float64(wordCount) > float64(baseCount)*lengthDiffThreshold {
		return false
	}

	return true
}

// autoCalibrate 统计指纹出现频率，超过阈值的指纹自动加入通配符黑名单。
func (s *WildcardScanner) autoCalibrate(fp string) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

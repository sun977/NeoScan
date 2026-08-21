// Package engine 提供 DirScanner 的 HTTP 请求、过滤和通配符检测能力。
// 此包仅供 scanner/dir 包内部使用（原子扫描器隔离原则）。
package engine

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"neoagent/internal/pkg/logger"
)

// 错误定义
var (
	ErrConnectionTimeout  = errors.New("connection timed out")
	ErrConnectionRefused  = errors.New("connection refused")
	ErrDNSFailed          = errors.New("DNS resolution failed")
	ErrSSLError           = errors.New("SSL/TLS error")
	ErrOther              = errors.New("request failed")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	// ErrInvalidRequest 表示请求在本地构造阶段就失败（如字典条目含非法 URL
	// 转义序列，例如未展开的 "%EXT%"）。这类错误从未发出网络请求，重试没有
	// 任何意义，必须直接判定为不可重试，否则会在含特殊字符的字典条目上
	// 白白消耗 MaxRetries 次退避等待，拖慢整个扫描（真实复现过的性能 bug）。
	ErrInvalidRequest = errors.New("invalid request")
)

// RequesterConfig 控制 Requester 行为。
type RequesterConfig struct {
	Timeout          time.Duration     // 请求超时（默认 10s）
	MaxRetries       int               // 最大重试次数（默认 2）
	Proxy            string            // 代理 URL
	Method           string            // HTTP 方法（默认 GET）
	Headers          map[string]string // 自定义请求头
	UserAgents       []string          // User-Agent 列表（随机选择）
	RateLimit        int               // 每秒请求数限制（0 = 不限）
	Delay            time.Duration     // 请求间隔延迟
	FollowRedirects  bool              // 是否跟随重定向（默认 false）
	IP               string            // 绑定本地 IP
	NetworkInterface string            // 绑定网络接口
	MaxConns         int               // 连接池大小，通常等于扫描并发线程数（默认 100）
}

// Requester 封装 HTTP 请求能力，支持路径保留、重试、代理等。
type Requester struct {
	client      *http.Client
	cfg         RequesterConfig
	uaList      []string
	uaIndex     int
	uaMu        sync.Mutex
	rateTicker  *time.Ticker // 按 RateLimit 定期发放令牌，Do() 每次请求前消费一个
}

// NewRequester 创建请求器。
func NewRequester(cfg RequesterConfig) *Requester {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 2
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodGet
	}
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = 100
	}

	// 连接复用：对齐 dirsearch（lib/connection/requester.py 用
	// requests.Session + pool_maxsize=thread_count）的做法，按并发线程数
	// 分配空闲连接槽位，避免每个请求都重新三次握手。
	//
	// 曾经这里是 DisableKeepAlives: true（"禁用连接复用"），但设计/实施
	// 文档都没有给出任何理由——真实代价是扫一次上万条目字典要建立上万个
	// 短连接，高并发下会快速耗尽本地临时端口、堆积 TIME_WAIT，在压测
	// 中表现为同一进程内连续扫描的耗时逐轮递增（4s → 7.5s → 12s → 15s）。
	// 这是解决"假想的连接复用风险"而制造的真实性能问题，予以修正。
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // 允许自签名证书
		},
		MaxIdleConns:        cfg.MaxConns,
		MaxIdleConnsPerHost: cfg.MaxConns,
		IdleConnTimeout:     30 * time.Second,
	}

	// 代理配置
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// 本地 IP 绑定
	if cfg.IP != "" {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			localAddr, err := net.ResolveTCPAddr(network, cfg.IP+":0")
			if err != nil {
				return nil, err
			}
			dialer := &net.Dialer{
				Timeout:   cfg.Timeout,
				LocalAddr: localAddr,
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	// 速率限制器：按 1s/RateLimit 的间隔发放令牌（标准令牌桶节流），
	// Do() 每次请求前从 ticker 消费一次，避免过去"只写不读的 channel"
	// 在超过 RateLimit 个请求后永久阻塞的 bug。
	var rateTicker *time.Ticker
	if cfg.RateLimit > 0 {
		rateTicker = time.NewTicker(time.Second / time.Duration(cfg.RateLimit))
	}

	return &Requester{
		client: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if !cfg.FollowRedirects {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		cfg:        cfg,
		uaList:     cfg.UserAgents,
		rateTicker: rateTicker,
	}
}

// Close 立即释放连接池中的空闲连接。
//
// 一次扫描任务结束后应调用它：Transport 复用连接默认要等 IdleConnTimeout
// （30s）才会自然断开，扫描器进程若频繁创建/丢弃 Requester（如多目标
// 扫描、单元测试判断 goroutine 是否退出干净），不主动清理会造成连接和
// 对应读循环 goroutine 短暂堆积。
func (r *Requester) Close() {
	if transport, ok := r.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

// Do 发送 HTTP 请求并返回 Response。
// ctx 用于控制取消，baseURL 是目标地址，path 是请求路径。
func (r *Requester) Do(ctx context.Context, baseURL, path string) (*Response, error) {
	var lastErr error
	maxAttempt := r.cfg.MaxRetries + 1

	for attempt := 0; attempt < maxAttempt; attempt++ {
		if attempt > 0 {
			// 指数退避
			delay := time.Duration(1<<uint(attempt-1)) * 500 * time.Millisecond
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
			logger.Debugf("[DirReq] Retry %d/%d, waiting %v", attempt, maxAttempt, delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// 速率限制：消费一个 ticker 令牌，达到 RateLimit 后自动排队等待下一个节拍
		if r.rateTicker != nil {
			select {
			case <-r.rateTicker.C:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// 请求间隔延迟
		if r.cfg.Delay > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(r.cfg.Delay):
			}
		}

		resp, err := r.doSingle(ctx, baseURL, path)
		if err != nil {
			lastErr = err
			// 连接类错误、本地构造请求失败均不重试（重试无法改变结果，只会浪费时间）
			if isConnectionError(err) || errors.Is(err, ErrInvalidRequest) {
				return nil, err
			}
			continue
		}
		return resp, nil
	}

	return nil, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// doSingle 执行单次 HTTP 请求。
func (r *Requester) doSingle(ctx context.Context, baseURL, path string) (*Response, error) {
	// 字典条目不保证带前导斜杠（如 dirsearch 字典中的 "!.gitignore"）。
	// 缺少 "/" 时直接拼接会产生 "http://host!.gitignore" 这种畸形 URL，
	// 底层 DNS/连接尝试会长时间挂起而不是快速失败，必须在拼接前规范化。
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	target := baseURL + path

	req, err := http.NewRequestWithContext(ctx, r.cfg.Method, target, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	// 注入请求头
	req.Header.Set("User-Agent", r.randomUA())
	req.Header.Set("Accept", "*/*")
	// 不显式设置 "Connection: close"：这个请求头会让服务器和 Transport
	// 都在响应后关闭连接，即使 Transport 配置了连接池（MaxIdleConnsPerHost）
	// 也会被这里的应用层声明架空——两处配置互相矛盾，是 Transport 连接池
	// 配置形同虚设的真正原因（对齐 dirsearch 的 requests.Session 长连接
	// 复用做法，见 NewRequester 的 MaxIdleConnsPerHost 注释）。
	for k, v := range r.cfg.Headers {
		req.Header.Set(k, v)
	}

	// 路径保留：Go 1.25 已移除 Opaque 字段，需要使用自定义 Transport 实现
	// 当前版本对含特殊字符的路径不进行特殊处理，后续可扩展 PathPreservingRoundTripper
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, classifyError(err)
	}
	defer resp.Body.Close()

	// 读取响应体（上限 10MB）
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	bodyStr := string(body)

	// 计算统计信息
	lines := strings.Count(bodyStr, "\n") + 1
	words := len(strings.Fields(bodyStr))
	title := extractTitle(bodyStr)

	return &Response{
		StatusCode:  resp.StatusCode,
		Body:        bodyStr,
		Size:        int64(len(body)),
		ContentType: resp.Header.Get("Content-Type"),
		Location:    resp.Header.Get("Location"),
		Words:       words,
		Lines:       lines,
		Title:       title,
	}, nil
}

// randomUA 随机选择 User-Agent。
func (r *Requester) randomUA() string {
	r.uaMu.Lock()
	defer r.uaMu.Unlock()

	if len(r.uaList) == 0 {
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	}

	idx := r.uaIndex % len(r.uaList)
	r.uaIndex++
	return r.uaList[idx]
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────────

// classifyError 根据错误类型分类。
func classifyError(err error) error {
	if err == context.DeadlineExceeded || err == context.Canceled {
		return err
	}

	errStr := err.Error()

	// 超时
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "i/o timeout") {
		return ErrConnectionTimeout
	}

	// 连接被拒绝
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "ECONNREFUSED") {
		return ErrConnectionRefused
	}

	// DNS 失败
	if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "NXDOMAIN") {
		return ErrDNSFailed
	}

	// SSL 错误
	if strings.Contains(errStr, "ssl") || strings.Contains(errStr, "tls:") || strings.Contains(errStr, "certificate") {
		return ErrSSLError
	}

	return ErrOther
}

// isConnectionError 判断是否为连接类错误（不重试）。
func isConnectionError(err error) bool {
	return err == ErrConnectionRefused || err == ErrDNSFailed || err == ErrSSLError
}

// extractTitle 从 HTML 中提取 <title> 标签内容。
func extractTitle(body string) string {
	lower := strings.ToLower(body)
	idx := strings.Index(lower, "<title>")
	if idx == -1 {
		return ""
	}
	idx += 7 // len("<title>")
	endIdx := strings.Index(lower[idx:], "</title>")
	if endIdx == -1 {
		return ""
	}
	return strings.TrimSpace(body[idx : idx+endIdx])
}

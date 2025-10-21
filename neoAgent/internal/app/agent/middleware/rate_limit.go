/**
 * 限流中间件
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端限流中间件，用于控制请求频率
 * @func: 占位符实现，待后续完善
 */
package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"neoagent/internal/pkg/logger"
	"neoagent/internal/pkg/utils"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// 是否启用限流
	Enabled bool `json:"enabled"`
	
	// 每秒请求数限制
	RequestsPerSecond int `json:"requests_per_second"`
	
	// 突发请求数限制
	BurstSize int `json:"burst_size"`
	
	// 限流窗口大小
	WindowSize time.Duration `json:"window_size"`
	
	// 限流策略
	Strategy RateLimitStrategy `json:"strategy"`
	
	// 跳过限流的路径
	SkipPaths []string `json:"skip_paths"`
	
	// 跳过限流的IP
	SkipIPs []string `json:"skip_ips"`
	
	// 自定义限流键生成函数
	KeyGenerator func(*gin.Context) string `json:"-"`
	
	// 限流响应消息
	Message string `json:"message"`
	
	// 限流响应状态码
	StatusCode int `json:"status_code"`
}

// RateLimitStrategy 限流策略
type RateLimitStrategy string

const (
	// 令牌桶算法
	TokenBucket RateLimitStrategy = "token_bucket"
	
	// 滑动窗口算法
	SlidingWindow RateLimitStrategy = "sliding_window"
	
	// 固定窗口算法
	FixedWindow RateLimitStrategy = "fixed_window"
)

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	config   *RateLimitConfig
	logger   *logger.LoggerManager
	limiters map[string]*RateLimiter
	mutex    sync.RWMutex
}

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow() bool
	Reset()
	GetRemaining() int
	GetResetTime() time.Time
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	tokens    int
	maxTokens int
	refillRate int
	lastRefill time.Time
	mutex     sync.Mutex
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	requests   []time.Time
	maxRequests int
	windowSize time.Duration
	mutex      sync.Mutex
}

// FixedWindowLimiter 固定窗口限流器
type FixedWindowLimiter struct {
	count      int
	maxCount   int
	windowStart time.Time
	windowSize time.Duration
	mutex      sync.Mutex
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(config *RateLimitConfig) *RateLimitMiddleware {
	if config == nil {
		config = &RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 100,
			BurstSize:         200,
			WindowSize:        time.Minute,
			Strategy:          TokenBucket,
			Message:           "Rate limit exceeded",
			StatusCode:        http.StatusTooManyRequests,
			SkipPaths: []string{
				"/health",
				"/ping",
			},
		}
	}
	
	if config.KeyGenerator == nil {
		config.KeyGenerator = defaultKeyGenerator
	}
	
	return &RateLimitMiddleware{
		config:   config,
		logger:   logger.LoggerInstance,
		limiters: make(map[string]*RateLimiter),
	}
}

// Handler 限流处理器
func (m *RateLimitMiddleware) Handler() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// TODO: 实现限流逻辑
		// 1. 检查是否启用限流
		// 2. 检查是否跳过限流
		// 3. 生成限流键
		// 4. 获取或创建限流器
		// 5. 检查是否允许请求
		// 6. 设置响应头
		
		if !m.config.Enabled {
			c.Next()
			return
		}
		
		// 检查是否跳过限流
		if m.shouldSkipRateLimit(c) {
			c.Next()
			return
		}
		
		// 生成限流键
		key := m.config.KeyGenerator(c)
		
		// 获取限流器
		limiter := m.getLimiter(key)
		
		// 检查是否允许请求
		if !limiter.Allow() {
			m.handleRateLimitExceeded(c, limiter)
			return
		}
		
		// 设置限流响应头
		m.setRateLimitHeaders(c, limiter)
		
		c.Next()
	})
}

// shouldSkipRateLimit 检查是否应该跳过限流
func (m *RateLimitMiddleware) shouldSkipRateLimit(c *gin.Context) bool {
	path := c.Request.URL.Path
	ip := utils.GetClientIP(c)
	
	// 检查跳过路径
	for _, skipPath := range m.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	
	// 检查跳过IP
	for _, skipIP := range m.config.SkipIPs {
		if ip == skipIP {
			return true
		}
	}
	
	return false
}

// getLimiter 获取限流器
func (m *RateLimitMiddleware) getLimiter(key string) RateLimiter {
	m.mutex.RLock()
	limiter, exists := m.limiters[key]
	m.mutex.RUnlock()
	
	if exists {
		return *limiter
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// 双重检查
	if limiter, exists := m.limiters[key]; exists {
		return *limiter
	}
	
	// 创建新的限流器
	newLimiter := m.createLimiter()
	m.limiters[key] = &newLimiter
	
	return newLimiter
}

// createLimiter 创建限流器
func (m *RateLimitMiddleware) createLimiter() RateLimiter {
	switch m.config.Strategy {
	case TokenBucket:
		return &TokenBucketLimiter{
			tokens:     m.config.BurstSize,
			maxTokens:  m.config.BurstSize,
			refillRate: m.config.RequestsPerSecond,
			lastRefill: time.Now(),
		}
	case SlidingWindow:
		return &SlidingWindowLimiter{
			requests:    make([]time.Time, 0),
			maxRequests: m.config.RequestsPerSecond,
			windowSize:  m.config.WindowSize,
		}
	case FixedWindow:
		return &FixedWindowLimiter{
			count:       0,
			maxCount:    m.config.RequestsPerSecond,
			windowStart: time.Now(),
			windowSize:  m.config.WindowSize,
		}
	default:
		return &TokenBucketLimiter{
			tokens:     m.config.BurstSize,
			maxTokens:  m.config.BurstSize,
			refillRate: m.config.RequestsPerSecond,
			lastRefill: time.Now(),
		}
	}
}

// handleRateLimitExceeded 处理限流超出
func (m *RateLimitMiddleware) handleRateLimitExceeded(c *gin.Context, limiter RateLimiter) {
	logger.Warn("Rate limit exceeded")
	
	// 设置限流响应头
	m.setRateLimitHeaders(c, limiter)
	
	c.JSON(m.config.StatusCode, gin.H{
		"error":   "rate_limit_exceeded",
		"message": m.config.Message,
		"retry_after": limiter.GetResetTime().Unix(),
	})
	
	c.Abort()
}

// setRateLimitHeaders 设置限流响应头
func (m *RateLimitMiddleware) setRateLimitHeaders(c *gin.Context, limiter RateLimiter) {
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", m.config.RequestsPerSecond))
	c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limiter.GetRemaining()))
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", limiter.GetResetTime().Unix()))
}

// defaultKeyGenerator 默认键生成器
func defaultKeyGenerator(c *gin.Context) string {
	return utils.GetClientIP(c)
}

// TokenBucketLimiter 实现

// Allow 检查是否允许请求
func (t *TokenBucketLimiter) Allow() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	now := time.Now()
	
	// 计算需要添加的令牌数
	elapsed := now.Sub(t.lastRefill)
	tokensToAdd := int(elapsed.Seconds()) * t.refillRate
	
	if tokensToAdd > 0 {
		t.tokens = min(t.maxTokens, t.tokens+tokensToAdd)
		t.lastRefill = now
	}
	
	if t.tokens > 0 {
		t.tokens--
		return true
	}
	
	return false
}

// Reset 重置限流器
func (t *TokenBucketLimiter) Reset() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.tokens = t.maxTokens
	t.lastRefill = time.Now()
}

// GetRemaining 获取剩余令牌数
func (t *TokenBucketLimiter) GetRemaining() int {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	return t.tokens
}

// GetResetTime 获取重置时间
func (t *TokenBucketLimiter) GetResetTime() time.Time {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if t.tokens >= t.maxTokens {
		return time.Now()
	}
	
	tokensNeeded := t.maxTokens - t.tokens
	secondsToWait := float64(tokensNeeded) / float64(t.refillRate)
	
	return time.Now().Add(time.Duration(secondsToWait) * time.Second)
}

// SlidingWindowLimiter 实现

// Allow 检查是否允许请求
func (s *SlidingWindowLimiter) Allow() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-s.windowSize)
	
	// 移除过期的请求
	validRequests := make([]time.Time, 0)
	for _, req := range s.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	s.requests = validRequests
	
	// 检查是否超出限制
	if len(s.requests) >= s.maxRequests {
		return false
	}
	
	// 添加当前请求
	s.requests = append(s.requests, now)
	return true
}

// Reset 重置限流器
func (s *SlidingWindowLimiter) Reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.requests = make([]time.Time, 0)
}

// GetRemaining 获取剩余请求数
func (s *SlidingWindowLimiter) GetRemaining() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	return s.maxRequests - len(s.requests)
}

// GetResetTime 获取重置时间
func (s *SlidingWindowLimiter) GetResetTime() time.Time {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if len(s.requests) == 0 {
		return time.Now()
	}
	
	return s.requests[0].Add(s.windowSize)
}

// FixedWindowLimiter 实现

// Allow 检查是否允许请求
func (f *FixedWindowLimiter) Allow() bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	now := time.Now()
	
	// 检查是否需要重置窗口
	if now.Sub(f.windowStart) >= f.windowSize {
		f.count = 0
		f.windowStart = now
	}
	
	// 检查是否超出限制
	if f.count >= f.maxCount {
		return false
	}
	
	f.count++
	return true
}

// Reset 重置限流器
func (f *FixedWindowLimiter) Reset() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	f.count = 0
	f.windowStart = time.Now()
}

// GetRemaining 获取剩余请求数
func (f *FixedWindowLimiter) GetRemaining() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	return f.maxCount - f.count
}

// GetResetTime 获取重置时间
func (f *FixedWindowLimiter) GetResetTime() time.Time {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	return f.windowStart.Add(f.windowSize)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// UpdateConfig 更新限流配置
func (m *RateLimitMiddleware) UpdateConfig(config *RateLimitConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.config = config
	
	// 清空现有限流器，强制重新创建
	m.limiters = make(map[string]*RateLimiter)
	
	logger.Info("Rate limit middleware config updated")
	
	return nil
}

// GetConfig 获取当前配置
func (m *RateLimitMiddleware) GetConfig() *RateLimitConfig {
	return m.config
}

// GetStats 获取限流统计信息
func (m *RateLimitMiddleware) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	stats := map[string]interface{}{
		"total_limiters": len(m.limiters),
		"config": map[string]interface{}{
			"enabled":             m.config.Enabled,
			"requests_per_second": m.config.RequestsPerSecond,
			"burst_size":          m.config.BurstSize,
			"strategy":            string(m.config.Strategy),
		},
	}
	
	return stats
}
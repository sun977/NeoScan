/**
 * 中间件:限流器中间件 - 尚未使用
 * @author: sun977
 * @date: 2025.10.10
 * @description: 定义限流器中间件
 * @func:
 *   - GinRateLimitMiddleware 调用 GinRateLimitMiddlewareWithLimiter 默认限流器中间件[根据客户端IP进行限流]
 *   - GinAPIRateLimitMiddleware API接口限流器[针对API接口的专用限流，限制更严格]
 *   - GinAuthRateLimitMiddleware 认证接口限流器[针对认证接口的专用限流，限制更严格]
 *   - GinUserBasedRateLimitMiddleware 用户个性化限流器[针对已认证用户的个性化限流]
 */
package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"

	"github.com/gin-gonic/gin"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(key string) bool
	Reset(key string)
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	buckets map[string]*TokenBucket
	mutex   sync.RWMutex
	rate    int           // 每秒生成的令牌数
	burst   int           // 桶的容量
	cleanup time.Duration // 清理间隔
}

// TokenBucket 令牌桶
type TokenBucket struct {
	tokens   int       // 当前令牌数
	capacity int       // 桶容量
	rate     int       // 令牌生成速率（每秒）
	lastTime time.Time // 上次更新时间
	mutex    sync.Mutex
}

// NewTokenBucketLimiter 创建新的令牌桶限流器
func NewTokenBucketLimiter(rate, burst int, cleanup time.Duration) *TokenBucketLimiter {
	limiter := &TokenBucketLimiter{
		buckets: make(map[string]*TokenBucket),
		rate:    rate,
		burst:   burst,
		cleanup: cleanup,
	}

	// 启动清理协程
	go limiter.cleanupExpiredBuckets()

	return limiter
}

// Allow 检查是否允许请求
func (tbl *TokenBucketLimiter) Allow(key string) bool {
	tbl.mutex.Lock()
	bucket, exists := tbl.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			tokens:   tbl.burst,
			capacity: tbl.burst,
			rate:     tbl.rate,
			lastTime: time.Now(),
		}
		tbl.buckets[key] = bucket
	}
	tbl.mutex.Unlock()

	return bucket.consume()
}

// Reset 重置指定key的限流状态
func (tbl *TokenBucketLimiter) Reset(key string) {
	tbl.mutex.Lock()
	delete(tbl.buckets, key)
	tbl.mutex.Unlock()
}

// consume 消费一个令牌
func (tb *TokenBucket) consume() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()

	// 添加新令牌
	newTokens := int(elapsed * float64(tb.rate))
	tb.tokens += newTokens
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	tb.lastTime = now

	// 尝试消费一个令牌
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// cleanupExpiredBuckets 清理过期的令牌桶
func (tbl *TokenBucketLimiter) cleanupExpiredBuckets() {
	ticker := time.NewTicker(tbl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		tbl.mutex.Lock()
		now := time.Now()
		for key, bucket := range tbl.buckets {
			bucket.mutex.Lock()
			// 如果桶超过清理间隔时间没有使用，则删除
			if now.Sub(bucket.lastTime) > tbl.cleanup {
				delete(tbl.buckets, key)
			}
			bucket.mutex.Unlock()
		}
		tbl.mutex.Unlock()
	}
}

// GinRateLimitMiddleware 默认限流中间件
// 使用配置文件中的限流策略
func (m *MiddlewareManager) GinRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用限流
		if !m.securityConfig.RateLimit.Enabled {
			c.Next()
			return
		}

		// 检查是否跳过限流
		if m.shouldSkipRateLimit(c) {
			c.Next()
			return
		}

		// 获取客户端IP作为限流key
		clientIP := utils.GetClientIP(c)

		// 根据配置创建限流器
		limiter := m.getRateLimiter()

		// 检查是否允许请求
		if !limiter.Allow(clientIP) {
			// 记录限流日志
			logger.LogWarn("Rate limit exceeded for client", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
				"operation": "rate_limit_exceeded",
				"option":    "block_request",
				"func_name": "middleware.ratelimit.GinRateLimitMiddleware",
				"method":    c.Request.Method,
			})

			// 返回配置的限流错误
			c.JSON(m.securityConfig.RateLimit.StatusCode, gin.H{
				"error":   "Rate limit exceeded",
				"message": m.securityConfig.RateLimit.Message,
				"code":    "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		// 记录通过限流的日志
		logger.LogInfo("Request passed rate limit check", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"operation": "rate_limit_check",
			"option":    "allow_request",
			"func_name": "middleware.ratelimit.GinRateLimitMiddleware",
		})

		// 继续处理请求
		c.Next()
	}
}

// shouldSkipRateLimit 检查是否应该跳过限流
func (m *MiddlewareManager) shouldSkipRateLimit(c *gin.Context) bool {
	path := c.Request.URL.Path

	// 检查跳过路径
	for _, skipPath := range m.securityConfig.RateLimit.SkipPaths {
		if path == skipPath {
			return true
		}
	}

	// 检查跳过IP
	clientIP := utils.GetClientIP(c)
	for _, skipIP := range m.securityConfig.RateLimit.SkipIPs {
		if clientIP == skipIP {
			return true
		}
	}

	return false
}

// getRateLimiter 根据配置获取限流器
func (m *MiddlewareManager) getRateLimiter() RateLimiter {
	config := &m.securityConfig.RateLimit

	// 解析窗口大小字符串为time.Duration
	windowSize, err := time.ParseDuration(config.WindowSize)
	if err != nil {
		// 如果解析失败，使用默认值
		windowSize = 15 * time.Minute
	}

	// 根据策略创建不同的限流器
	switch config.Strategy {
	case "token_bucket":
		return NewTokenBucketLimiter(
			config.RequestsPerSecond,
			config.BurstSize,
			windowSize,
		)
	default:
		// 默认使用令牌桶算法
		return NewTokenBucketLimiter(
			config.RequestsPerSecond,
			config.BurstSize,
			windowSize,
		)
	}
}

// GinRateLimitMiddlewareWithLimiter 通用限流中间件
// 基于IP地址进行限流，可配置限流策略
func (m *MiddlewareManager) GinRateLimitMiddlewareWithLimiter(limiter RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端IP作为限流key
		// clientIP := c.ClientIP()
		clientIP := utils.GetClientIP(c)

		// 检查是否允许请求
		if !limiter.Allow(clientIP) {
			// 记录限流日志
			logger.LogWarn("Rate limit exceeded for client", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
				"operation": "rate_limit_exceeded",
				"option":    "block_request",
				"func_name": "middleware.ratelimit.GinRateLimitMiddlewareWithLimiter",
				"method":    c.Request.Method,
			})

			// 返回限流错误
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests, please try again later",
				"code":    "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		// 记录通过限流的日志
		logger.LogInfo("Request passed rate limit check", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"operation": "rate_limit_check",
			"option":    "allow_request",
			"func_name": "middleware.ratelimit.GinRateLimitMiddlewareWithLimiter",
		})

		// 继续处理请求
		c.Next()
	}
}

// GinAPIRateLimitMiddleware API接口限流中间件
// 针对API接口的专用限流，限制更严格
func (m *MiddlewareManager) GinAPIRateLimitMiddleware() gin.HandlerFunc {
	// 创建API专用限流器：每秒10个请求，突发20个
	limiter := NewTokenBucketLimiter(10, 20, 5*time.Minute)

	return m.GinRateLimitMiddlewareWithLimiter(limiter)
}

// GinAuthRateLimitMiddleware 认证接口限流中间件
// 针对登录、注册等认证接口的严格限流
func (m *MiddlewareManager) GinAuthRateLimitMiddleware() gin.HandlerFunc {
	// 创建认证专用限流器：每秒2个请求，突发5个
	limiter := NewTokenBucketLimiter(2, 5, 10*time.Minute)
	// clientIP := utils.GetClientIP(c)

	return func(c *gin.Context) {
		clientIP := utils.GetClientIP(c)
		// 使用IP+路径作为限流key，更精确的限流
		key := fmt.Sprintf("%s:%s", utils.GetClientIP(c), c.Request.URL.Path)

		if !limiter.Allow(key) {
			// 记录认证限流日志
			logger.LogWarn("Authentication rate limit exceeded", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
				"operation": "auth_rate_limit_exceeded",
				"option":    "block_auth_request",
				"func_name": "middleware.ratelimit.GinAuthRateLimitMiddleware",
				"method":    c.Request.Method,
			})

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Authentication rate limit exceeded",
				"message": "Too many authentication attempts, please try again later",
				"code":    "AUTH_RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		// 记录通过认证限流的日志
		logger.LogInfo("Authentication request passed rate limit check", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"operation": "auth_rate_limit_check",
			"option":    "allow_auth_request",
			"func_name": "middleware.ratelimit.GinAuthRateLimitMiddleware",
		})

		c.Next()
	}
}

// GinUserBasedRateLimitMiddleware 基于用户的限流中间件
// 针对已认证用户的个性化限流
func (m *MiddlewareManager) GinUserBasedRateLimitMiddleware() gin.HandlerFunc {
	// 创建用户专用限流器：每秒30个请求，突发50个
	limiter := NewTokenBucketLimiter(30, 50, 15*time.Minute)
	// clientIP := utils.GetClientIP(c)

	return func(c *gin.Context) {
		clientIP := utils.GetClientIP(c)
		// 尝试获取用户ID
		userID, exists := c.Get("user_id")
		var key string

		if exists && userID != nil {
			// 已认证用户使用用户ID作为key
			key = fmt.Sprintf("user:%v", userID)
		} else {
			// 未认证用户使用IP作为key
			key = fmt.Sprintf("ip:%s", clientIP)
		}

		if !limiter.Allow(key) {
			// 记录用户限流日志
			logger.LogWarn("User-based rate limit exceeded", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
				"operation": "user_rate_limit_exceeded",
				"option":    "block_user_request",
				"func_name": "middleware.ratelimit.GinUserBasedRateLimitMiddleware",
				"user_id":   userID,
				"key":       key,
			})

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "User rate limit exceeded",
				"message": "Too many requests from this user, please try again later",
				"code":    "USER_RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		// 记录通过用户限流的日志
		logger.LogInfo("User request passed rate limit check", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"operation": "user_rate_limit_check",
			"option":    "allow_user_request",
			"func_name": "middleware.ratelimit.GinUserBasedRateLimitMiddleware",
			"user_id":   userID,
			"key":       key,
		})

		c.Next()
	}
}

package qos

import (
	"context"
	"sync"
	"sync/atomic"
)

// AdaptiveLimiter 实现了 AIMD (Additive Increase Multiplicative Decrease) 拥塞控制算法
// 用于动态调整并发限制，以适应网络状况
// - 成功时：线性增加并发数 (Additive Increase)
// - 失败时：乘性减少并发数 (Multiplicative Decrease)
type AdaptiveLimiter struct {
	sem             chan struct{} // 信号量通道，用于控制并发令牌
	reductionNeeded int32         // 需要减少的令牌数量 (待偿还的债务)
	
	currentLimit    int // 当前并发限制
	minLimit        int // 最小并发限制 (保底值)
	maxLimit        int // 最大并发限制 (天花板)

	successCount    int        // 连续成功计数
	mu              sync.Mutex // 互斥锁，保护 limit 和 successCount 的更新
}

// NewAdaptiveLimiter 创建一个新的自适应限流器
// initial: 初始并发数
// min: 最小并发数
// max: 最大并发数
func NewAdaptiveLimiter(initial, min, max int) *AdaptiveLimiter {
	// 参数校验与修正
	if initial < min {
		initial = min
	}
	if initial > max {
		initial = max
	}

	l := &AdaptiveLimiter{
		sem:          make(chan struct{}, max), // 通道容量设置为最大值，方便扩容
		currentLimit: initial,
		minLimit:     min,
		maxLimit:     max,
	}

	// 填充初始令牌
	for i := 0; i < initial; i++ {
		l.sem <- struct{}{}
	}

	return l
}

// Acquire 获取一个并发令牌
// 如果没有令牌可用，会阻塞直到有令牌释放或 context 取消
func (l *AdaptiveLimiter) Acquire(ctx context.Context) error {
	select {
	case <-l.sem:
		return nil // 成功获取令牌
	case <-ctx.Done():
		return ctx.Err() // 上下文取消
	}
}

// Release 释放一个并发令牌
// 会检查是否有"债务"需要偿还 (reductionNeeded)，如果有则销毁令牌而不是归还
func (l *AdaptiveLimiter) Release() {
	// 检查是否需要缩减容量 (偿还债务)
	if atomic.LoadInt32(&l.reductionNeeded) > 0 {
		// 尝试减少 reductionNeeded 计数
		// 使用循环 CAS 确保原子性操作正确
		for {
			val := atomic.LoadInt32(&l.reductionNeeded)
			if val <= 0 {
				break // 没有债务了，不需要销毁
			}
			// 尝试将债务 -1
			if atomic.CompareAndSwapInt32(&l.reductionNeeded, val, val-1) {
				return // 成功销毁一个令牌 (不归还到 channel)，直接返回
			}
		}
	}

	// 归还令牌到 channel
	select {
	case l.sem <- struct{}{}:
		// 成功归还
	default:
		// 理论上不应该发生，除非 release 次数超过了 acquire 次数
		// 或者在缩容时逻辑有误，这里作为防御性编程忽略
	}
}

// OnSuccess 通知一次成功的操作
// AIMD 策略：线性增长 (Additive Increase)
// 只有在连续成功达到当前 Limit 次数时，才增加 1 个并发额度 (拥塞避免算法)
func (l *AdaptiveLimiter) OnSuccess() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.successCount++
	
	// 增长策略：每完成 currentLimit 次成功，Limit + 1
	// 这比"每成功一次就 +1" (慢启动) 要温和，适合稳定期
	if l.successCount >= l.currentLimit {
		l.successCount = 0
		l.increaseLimit(1)
	}
}

// OnFailure 通知一次失败的操作 (通常是超时)
// AIMD 策略：乘性减少 (Multiplicative Decrease)
// 一旦发生拥塞(超时)，立即大幅降低并发数
func (l *AdaptiveLimiter) OnFailure() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 减少策略：当前 Limit * 0.7
	// 这是一个经典的拥塞控制参数，比 TCP 的 0.5 温和一些
	newLimit := int(float64(l.currentLimit) * 0.7)
	decrease := l.currentLimit - newLimit
	
	// 至少减少 1 个
	if decrease < 1 {
		decrease = 1
	}
	
	l.decreaseLimit(decrease)
	// 失败后重置成功计数，重新开始积累
	l.successCount = 0
}

// increaseLimit 增加并发限制
// n: 增加的数量
func (l *AdaptiveLimiter) increaseLimit(n int) {
	target := l.currentLimit + n
	if target > l.maxLimit {
		target = l.maxLimit
	}
	
	diff := target - l.currentLimit
	if diff <= 0 {
		return
	}

	l.currentLimit = target
	// 注入新的令牌到 channel
	for i := 0; i < diff; i++ {
		select {
		case l.sem <- struct{}{}:
		default:
			// Channel 已满 (达到 maxLimit)，不再注入
		}
	}
}

// decreaseLimit 减少并发限制
// n: 减少的数量
func (l *AdaptiveLimiter) decreaseLimit(n int) {
	target := l.currentLimit - n
	if target < l.minLimit {
		target = l.minLimit
	}

	diff := l.currentLimit - target
	if diff <= 0 {
		return
	}

	l.currentLimit = target
	
	// 我们需要移除 diff 个令牌。
	// 策略：
	// 1. 先尝试从 channel 中直接取走空闲令牌 (立即生效)
	// 2. 如果取不到 (令牌都被借出去了)，则记录到 reductionNeeded (债务)
	//    等后续 Release() 时销毁
	removed := 0
	for i := 0; i < diff; i++ {
		select {
		case <-l.sem:
			removed++
		default:
			// Channel 空了，说明并发跑满了
		}
	}

	remaining := diff - removed
	if remaining > 0 {
		// 记账，欠的令牌需要在 Release 时偿还
		atomic.AddInt32(&l.reductionNeeded, int32(remaining))
	}
}

// CurrentLimit 获取当前并发限制数
func (l *AdaptiveLimiter) CurrentLimit() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.currentLimit
}

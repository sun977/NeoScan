package qos

import (
	"context"
	"testing"
	"time"
)

func TestRttEstimator(t *testing.T) {
	e := NewRttEstimator()

	// 初始状态应该等于 defaultInitialRTO
	if e.Timeout() != defaultInitialRTO {
		t.Errorf("Expected initial RTO %v, got %v", defaultInitialRTO, e.Timeout())
	}

	// 1. 第一次更新: RTT = 100ms
	// SRTT = 100, RTTVAR = 50, RTO = 100 + 4*50 = 300ms
	e.Update(100 * time.Millisecond)
	rto := e.Timeout()
	if rto != 300*time.Millisecond {
		t.Errorf("First update failed. Expected 300ms, got %v", rto)
	}

	// 2. 第二次更新: RTT = 200ms (变慢了)
	// Delta = |100 - 200| = 100
	// RTTVAR = (0.75 * 50) + (0.25 * 100) = 37.5 + 25 = 62.5
	// SRTT = (0.875 * 100) + (0.125 * 200) = 87.5 + 25 = 112.5
	// RTO = 112.5 + 4*62.5 = 112.5 + 250 = 362.5ms
	e.Update(200 * time.Millisecond)
	rto = e.Timeout()
	if rto != 362500*time.Microsecond {
		t.Errorf("Second update failed. Expected 362.5ms, got %v", rto)
	}
}

func TestAdaptiveLimiter_Increase(t *testing.T) {
	// 初始 10, 最小 1, 最大 20
	l := NewAdaptiveLimiter(10, 1, 20)

	// 模拟 10 次成功 -> 应该增加 1
	for i := 0; i < 10; i++ {
		l.OnSuccess()
	}

	if l.CurrentLimit() != 11 {
		t.Errorf("Expected limit increase to 11, got %d", l.CurrentLimit())
	}

	// 再模拟 11 次成功 -> 应该增加 1 -> 12
	for i := 0; i < 11; i++ {
		l.OnSuccess()
	}

	if l.CurrentLimit() != 12 {
		t.Errorf("Expected limit increase to 12, got %d", l.CurrentLimit())
	}
}

func TestAdaptiveLimiter_Decrease(t *testing.T) {
	// 初始 100
	l := NewAdaptiveLimiter(100, 1, 200)

	// 模拟一次失败 -> 100 * 0.7 = 70
	l.OnFailure()

	if l.CurrentLimit() != 70 {
		t.Errorf("Expected limit decrease to 70, got %d", l.CurrentLimit())
	}
}

func TestAdaptiveLimiter_AcquireRelease(t *testing.T) {
	// 初始 2
	l := NewAdaptiveLimiter(2, 1, 10)
	ctx := context.Background()

	// 获取 2 个
	if err := l.Acquire(ctx); err != nil {
		t.Fatal(err)
	}
	if err := l.Acquire(ctx); err != nil {
		t.Fatal(err)
	}

	// 第 3 个应该阻塞 (用 select 检查)
	select {
	case <-l.sem:
		t.Fatal("Should verify blocked")
	default:
		// OK
	}

	// 释放 1 个
	l.Release()

	// 现在应该能获取
	if err := l.Acquire(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAdaptiveLimiter_DynamicAdjustment(t *testing.T) {
	// 初始 5, 最小 1
	l := NewAdaptiveLimiter(5, 1, 100)
	ctx := context.Background()

	// 借出所有 5 个令牌
	for i := 0; i < 5; i++ {
		l.Acquire(ctx)
	}

	// 触发失败，导致缩容
	// Limit: 5 -> 3 (5 * 0.7 = 3.5 -> 3)
	// 需要减少 2 个令牌。
	// 因为现在所有令牌都被借出去了，reductionNeeded 应该变为 2
	l.OnFailure()

	if l.CurrentLimit() != 3 {
		t.Errorf("Limit should be 3, got %d", l.CurrentLimit())
	}

	if l.reductionNeeded != 2 {
		t.Errorf("ReductionNeeded should be 2, got %d", l.reductionNeeded)
	}

	// 释放 1 个 -> 应该被销毁 (reductionNeeded -> 1)
	l.Release()
	if l.reductionNeeded != 1 {
		t.Errorf("ReductionNeeded should be 1 after release, got %d", l.reductionNeeded)
	}
	
	// 此时 channel 应该还是空的
	select {
	case <-l.sem:
		t.Fatal("Channel should be empty, token should have been destroyed")
	default:
		// OK
	}

	// 释放第 2 个 -> 应该被销毁 (reductionNeeded -> 0)
	l.Release()
	if l.reductionNeeded != 0 {
		t.Errorf("ReductionNeeded should be 0, got %d", l.reductionNeeded)
	}

	// 释放第 3 个 -> 应该归还到 channel
	l.Release()
	select {
	case <-l.sem:
		// OK
	default:
		t.Fatal("Channel should have 1 token")
	}
}

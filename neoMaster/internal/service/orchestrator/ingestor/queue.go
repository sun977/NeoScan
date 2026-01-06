// ResultQueue 结果缓冲队列接口
// 职责: 解耦 Agent 提交与 Master 处理速率，实现削峰填谷
package ingestor

import (
	"context"
	"errors"
	"sync"

	orcModel "neomaster/internal/model/orchestrator"
)

var (
	// ErrQueueFull 队列已满
	ErrQueueFull = errors.New("queue is full")
	// ErrQueueEmpty 队列为空
	ErrQueueEmpty = errors.New("queue is empty")
)

// ResultQueue 结果缓冲队列接口
type ResultQueue interface {
	// Push 推送结果到队列
	Push(ctx context.Context, result *orcModel.StageResult) error
	// Pop 从队列获取结果 (供 Processor 消费)
	// 如果队列为空，应返回 ErrQueueEmpty 或阻塞等待(取决于具体实现)
	Pop(ctx context.Context) (*orcModel.StageResult, error)
	// Len 获取队列长度 (用于监控)
	Len(ctx context.Context) (int64, error)
}

// MemoryQueue 基于 Channel 的内存队列实现
// 适用于单机部署或测试环境
type MemoryQueue struct {
	queue chan *orcModel.StageResult
	mu    sync.RWMutex
}

// NewMemoryQueue 创建内存队列
// bufferSize: 队列缓冲区大小
func NewMemoryQueue(bufferSize int) *MemoryQueue {
	if bufferSize <= 0 {
		bufferSize = 1000 // 默认缓冲 1000 条
	}
	return &MemoryQueue{
		queue: make(chan *orcModel.StageResult, bufferSize),
	}
}

// Push 推送结果到队列
// 非阻塞模式：如果队列满了，返回 ErrQueueFull
func (q *MemoryQueue) Push(ctx context.Context, result *orcModel.StageResult) error {
	select {
	case q.queue <- result:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

// Pop 从队列获取结果
// 阻塞模式：如果队列为空，会一直等待直到有数据或 Context 取消
func (q *MemoryQueue) Pop(ctx context.Context) (*orcModel.StageResult, error) {
	select {
	case result := <-q.queue:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Len 获取队列当前长度
func (q *MemoryQueue) Len(ctx context.Context) (int64, error) {
	return int64(len(q.queue)), nil
}

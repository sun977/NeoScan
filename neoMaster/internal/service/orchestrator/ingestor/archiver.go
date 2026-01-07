// EvidenceArchiver 原始证据归档器接口
// 职责: 将 Agent 上报的原始 JSON/XML/截图 存入对象存储，作为审计依据
package ingestor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// EvidenceArchiver 原始证据归档器接口
type EvidenceArchiver interface {
	// Archive 归档证据数据
	// key: 存储路径/Key (如: task_id/result_id.json)
	// data: 原始数据
	Archive(ctx context.Context, key string, data []byte) error
}

// FileArchiver 本地文件系统归档器实现
type FileArchiver struct {
	basePath string
}

// NewFileArchiver 创建本地文件系统归档器
// basePath: 基础存储路径 (如: data/evidence)
func NewFileArchiver(basePath string) *FileArchiver {
	return &FileArchiver{
		basePath: basePath,
	}
}

// Archive 归档证据数据
func (a *FileArchiver) Archive(ctx context.Context, key string, data []byte) error {
	fullPath := filepath.Join(a.basePath, key)
	dir := filepath.Dir(fullPath)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

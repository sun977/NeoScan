// AssetETLError 资产ETL错误记录
// 实现 AssetETLError 表的查询筛选功能
package asset

import (
	"context"
	"fmt"

	assetModel "neomaster/internal/model/asset"
	assetRepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/asset/etl"
)

// AssetETLErrorService ETL 错误服务接口
type AssetETLErrorService interface {
	// ListErrors 获取错误列表 (带筛选)
	ListErrors(ctx context.Context, filter assetRepo.ETLErrorFilter) ([]*assetModel.AssetETLError, int64, error)
	// GetError 获取错误详情
	GetError(ctx context.Context, id uint64) (*assetModel.AssetETLError, error)
	// TriggerReplay 触发错误重放
	TriggerReplay(ctx context.Context) (int, error)
}

type assetETLErrorService struct {
	repo      assetRepo.ETLErrorRepository
	processor etl.ResultProcessor
}

// NewAssetETLErrorService 创建 AssetETLErrorService 实例
func NewAssetETLErrorService(repo assetRepo.ETLErrorRepository, processor etl.ResultProcessor) AssetETLErrorService {
	return &assetETLErrorService{
		repo:      repo,
		processor: processor,
	}
}

// ListErrors 获取错误列表 (带筛选)
func (s *assetETLErrorService) ListErrors(ctx context.Context, filter assetRepo.ETLErrorFilter) ([]*assetModel.AssetETLError, int64, error) {
	return s.repo.List(ctx, filter)
}

// GetError 获取错误详情
func (s *assetETLErrorService) GetError(ctx context.Context, id uint64) (*assetModel.AssetETLError, error) {
	return s.repo.GetByID(ctx, id)
}

// TriggerReplay 触发错误重放
func (s *assetETLErrorService) TriggerReplay(ctx context.Context) (int, error) {
	if s.processor == nil {
		return 0, fmt.Errorf("etl processor is not initialized")
	}
	// TODO: 可以添加一些前置检查，比如系统负载保护
	return s.processor.ReplayErrors(ctx)
}

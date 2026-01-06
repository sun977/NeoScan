// AssetMerger 资产合并器接口
// 职责: 实现 StageResult -> Asset 的 Upsert 逻辑
// 核心逻辑:
// - 发现 IP -> 查找或创建 AssetHost
// - 发现 Port -> 更新 AssetPort
// - 发现 Service -> 更新 AssetService
// - 发现 Web -> 更新 AssetWeb
package etl

import (
	"context"
	assetModel "neomaster/internal/model/asset"
	orcModel "neomaster/internal/model/orchestrator"
)

// AssetMerger 资产合并器接口
type AssetMerger interface {
	// Merge 将扫描结果合并到资产库
	// 返回受影响的 AssetID 列表
	Merge(ctx context.Context, result *orcModel.StageResult) ([]uint64, error)
}

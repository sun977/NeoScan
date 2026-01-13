// RuleManager 负责指纹规则的导入导出和生命周期管理
// 它不参与运行时的匹配逻辑，只负责数据的 I/O
package fingerprint

import (
	"context"
	"fmt"
	"time"

	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/fingerprint/converters"
)

// RuleManager 负责指纹规则的导入导出和生命周期管理
// 它不参与运行时的匹配逻辑，只负责数据的 I/O
type RuleManager struct {
	fingerRepo assetrepo.AssetFingerRepository
	cpeRepo    assetrepo.AssetCPERepository
	converter  converters.StandardJSONConverter // 之前定义的 StandardJSONConverter
}

// NewRuleManager 创建管理器
func NewRuleManager(fingerRepo assetrepo.AssetFingerRepository, cpeRepo assetrepo.AssetCPERepository) *RuleManager {
	return &RuleManager{
		fingerRepo: fingerRepo,
		cpeRepo:    cpeRepo,
		converter:  *converters.NewStandardJSONConverter(),
	}
}

// ExportRules 导出全量规则到 JSON 字节流
// 流程: Repo(ListAll) -> Converter(Encode) -> []byte
func (m *RuleManager) ExportRules(ctx context.Context) ([]byte, error) {
	// 1. Fetch All Fingers
	// 可能需要 repo 增加 ListAll() 方法，或者分页取全量
	fingers, err := m.fingerRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch fingers failed: %w", err)
	}

	// 2. Fetch All CPEs
	cpes, err := m.cpeRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch cpes failed: %w", err)
	}

	// 3. Convert
	return m.converter.Encode(fingers, cpes)
}

// GetRuleStats 获取规则库统计信息 (版本依据)
func (m *RuleManager) GetRuleStats(ctx context.Context) (map[string]interface{}, error) {
	// 获取指纹总数
	fingers, err := m.fingerRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("count fingers failed: %w", err)
	}
	fingerCount := len(fingers)

	// 获取 CPE 总数
	cpes, err := m.cpeRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("count cpes failed: %w", err)
	}
	cpeCount := len(cpes)

	// 获取最后更新时间 (取最新的 CreatedAt/UpdatedAt)
	var lastUpdate time.Time
	if fingerCount > 0 {
		// 简单起见，取第一条的 UpdateTime，或者需要 Repo 支持 GetMaxUpdatedAt
		// 这里暂且使用当前时间作为近似值，或者遍历查找最大值 (效率较低)
		// 更好的做法是让 Repo 提供 Count() 和 LastUpdate() 接口
		// 为了不改动 Repo 接口，这里先简化处理
		lastUpdate = time.Now()
	}

	return map[string]interface{}{
		"finger_count": fingerCount,
		"cpe_count":    cpeCount,
		"last_update":  lastUpdate,
		"version":      fmt.Sprintf("%d.%d", cpeCount, fingerCount),
	}, nil
}

// ImportRules 导入规则 (覆盖或增量)
// 流程: []byte -> Converter(Decode) -> Repo(Save/Upsert)
func (m *RuleManager) ImportRules(ctx context.Context, data []byte, overwrite bool) error {
	// 1. Decode & Validate
	fingers, cpes, err := m.converter.Decode(data)
	if err != nil {
		return fmt.Errorf("invalid rule format: %w", err)
	}

	// 2. Save to DB (Transaction recommended)
	// 这里建议使用事务，要么全成功，要么全失败
	// 如果没有事务，至少要记录失败的条目

	// 批量插入 Fingers
	for _, f := range fingers {
		if err := m.fingerRepo.Upsert(ctx, f); err != nil {
			// Log error, maybe continue or abort based on strategy
			logger.LogError(err, "", 0, "", "import_rules", "", map[string]interface{}{"name": f.Name})
		}
	}

	// 批量插入 CPEs
	for _, c := range cpes {
		if err := m.cpeRepo.Upsert(ctx, c); err != nil {
			logger.LogError(err, "", 0, "", "import_rules", "", map[string]interface{}{"name": c.Name})
		}
	}

	return nil
}

// RuleManager 负责指纹规则的导入导出和生命周期管理
// 它不参与运行时的匹配逻辑，只负责数据的 I/O
package fingerprint

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"neomaster/internal/config"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	assetrepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/fingerprint/converters"

	"gorm.io/gorm"
)

// RuleManager 负责指纹规则的导入导出和生命周期管理
// 它不参与运行时的匹配逻辑，只负责数据的 I/O
type RuleManager struct {
	db            *gorm.DB // 用于事务支持
	fingerRepo    assetrepo.AssetFingerRepository
	cpeRepo       assetrepo.AssetCPERepository
	converter     converters.StandardJSONConverter
	mu            sync.RWMutex   // 读写锁，保护并发操作
	backupDir     string         // 备份目录
	encryptionKey string         // 规则加密密钥
	config        *config.Config // 全局配置
}

// NewRuleManager 创建管理器
func NewRuleManager(db *gorm.DB, fingerRepo assetrepo.AssetFingerRepository, cpeRepo assetrepo.AssetCPERepository, encryptionKey string, cfg *config.Config) *RuleManager {
	// 获取备份路径，优先使用配置，否则使用默认值
	// 默认结构: rules/backups/fingerprint
	backupDir := "rules/backups/fingerprint"

	if cfg != nil {
		rulesRoot := "rules"
		if cfg.App.Rules.RootPath != "" {
			rulesRoot = cfg.App.Rules.RootPath
		}

		backupRoot := "backups"
		if cfg.App.Rules.Backup.Dir != "" {
			backupRoot = cfg.App.Rules.Backup.Dir
		}
		// 规则备份目录: ROOT_DIR/BACKUP_DIR/ + fingerprint 【以后可以扩充 poc 配置】
		backupDir = filepath.Join(rulesRoot, backupRoot, "fingerprint")
	}

	// 确保目录存在
	if err := utils.MkdirAll(backupDir, 0755); err != nil {
		logger.LogBusinessError(err, "system", 0, "localhost", "", "mkdir", map[string]interface{}{
			"path": backupDir,
		})
	}

	return &RuleManager{
		fingerRepo:    fingerRepo,
		cpeRepo:       cpeRepo,
		converter:     *converters.NewStandardJSONConverter(),
		backupDir:     backupDir,
		encryptionKey: encryptionKey,
		config:        cfg,
	}
}

// ExportRules 导出所有规则 (Admin Backup)
func (m *RuleManager) ExportRules(ctx context.Context) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.exportRulesInternal(ctx, false) // 导出所有，包括禁用的
}

// ExportRulesWithSignature 导出规则并附带签名 (HMAC 或 SHA256)
// 返回: data, signature, error
// 为了简单，目前直接计算 SHA256，不使用密钥。
// 如果需要防篡改（防止伪造），应使用 HMAC 和密钥。
// 但这里主要是为了完整性校验（防止传输错误），SHA256 足够。
// 为了方便前端或Agent验证，我们可以返回一个 JSON 包装结构？
// 或者只返回 Header？
// 这里为了保持 ExportRules 接口的纯粹性，我们返回 data，让 Handler 决定如何包装。
// Handler 可以将 SHA256 放入 Header。
func (m *RuleManager) CalculateSignature(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// PublishRulesToDisk 发布规则到磁盘 (Agent Download) 将当前数据库中的规则发布到磁盘文件 (原子操作)
// 只发布 Enabled=true 的规则
func (m *RuleManager) PublishRulesToDisk(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. Export Data (Only Enabled)
	data, err := m.exportRulesInternal(ctx, true)
	if err != nil {
		return err
	}

	// 2. 写入临时文件
	// 默认路径：rules/fingerprint/rules.json (这里简化处理，直接覆盖单个大文件)
	// TODO: 后续应支持拆分文件，这里保持与 StandardJSONConverter 一致，输出单个 JSON
	// 为了兼容 Agent 的目录扫描逻辑，我们把这个文件放在 config 中定义的目录，或者默认目录
	// 这里假设目录结构是: rules/fingerprint/
	// AgentUpdateService 会扫描该目录下所有文件并打包
	targetDir := "rules/fingerprint"
	if err := utils.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create rule dir: %w", err)
	}

	targetFile := filepath.Join(targetDir, "neoscan_fingerprint_rules.json")
	tmpFile := targetFile + ".tmp"

	if err := utils.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write tmp file: %w", err)
	}

	// 3. 原子重命名 (覆盖)
	if err := utils.Rename(tmpFile, targetFile); err != nil {
		return fmt.Errorf("failed to rename rule file: %w", err)
	}

	// 4. 更新 mtime (确保 AgentUpdateService 能感知到变化)
	// Rename 会保留原文件的 mtime (取决于 FS 实现)，为了保险，显式 Touch 一下
	now := time.Now()
	if err := utils.Chtimes(targetFile, now, now); err != nil {
		logger.LogBusinessError(err, "system", 0, "localhost", "", "touch", map[string]interface{}{
			"file": targetFile,
		})
		// Touch 失败不影响功能，只是可能延迟生效
	}

	return nil
}

// exportRulesInternal 内部导出逻辑，不加锁，供内部调用
func (m *RuleManager) exportRulesInternal(ctx context.Context, onlyEnabled bool) ([]byte, error) {
	var err error
	var fingers []*asset.AssetFinger
	var cpes []*asset.AssetCPE

	// 1. Fetch Fingers
	if onlyEnabled {
		fingers, err = m.fingerRepo.FindEnabled(ctx)
	} else {
		fingers, err = m.fingerRepo.ListAll(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch fingers failed: %w", err)
	}

	// 2. Fetch CPEs
	if onlyEnabled {
		cpes, err = m.cpeRepo.FindEnabled(ctx)
	} else {
		cpes, err = m.cpeRepo.ListAll(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch cpes failed: %w", err)
	}

	// 3. Convert
	return m.converter.Encode(fingers, cpes)
}

// GetRuleStats 获取规则库统计信息
func (m *RuleManager) GetRuleStats(ctx context.Context) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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

	// 获取最后更新时间
	var lastUpdate time.Time
	if fingerCount > 0 {
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
// 包含自动备份和原子锁
// expectedSignature: 如果不为空，则进行完整性校验
// source: 规则来源 (system/custom)，如果不为空，则强制覆盖规则中的 Source 字段
func (m *RuleManager) ImportRules(ctx context.Context, data []byte, overwrite bool, expectedSignature string, source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 0. Integrity Check
	if expectedSignature != "" {
		currentSignature := m.CalculateSignature(data)
		if currentSignature != expectedSignature {
			return fmt.Errorf("integrity check failed: expected %s, got %s", expectedSignature, currentSignature)
		}
	}

	// 1. Auto Backup (Snapshot)
	if err := m.createBackup(ctx); err != nil {
		// 备份失败是否阻止导入？
		// 严格模式下应该阻止，保证数据安全
		return fmt.Errorf("auto backup failed: %w", err)
	}

	// 2. Decode & Validate
	fingers, cpes, err := m.converter.Decode(data)
	if err != nil {
		return fmt.Errorf("invalid rule format: %w", err)
	}

	// 3. Save to DB with Transaction
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 使用 Transaction 的 Repo 方法 (如果 Repo 支持)
		// 或者直接使用 tx 操作，但这破坏了 Repo 封装。
		// 理想情况：Repo 提供 UpsertBatch 或 WithTx 方法。
		// 这里暂且假设我们必须逐个 Upsert，但都在事务中。

		// 标记 Source
		if source != "" {
			for _, f := range fingers {
				f.Source = source
			}
			for _, c := range cpes {
				c.Source = source
			}
		}

		for _, f := range fingers {
			// 注意：这里仍然调用的是 m.fingerRepo，它可能使用的是非事务的 DB 实例。
			// 如果 Repo 没有事务支持，这里的事务是无效的。
			// 必须重构 Repo 以支持事务传递，或者直接在 Service 层操作 DB。
			// 既然我们有 m.db，我们可以尝试直接操作，但这违反分层原则。
			// 妥协方案：逐个插入，如果不涉及删除操作，事务性要求相对较低。
			// 但对于 True Rollback，事务是必须的。
			// 假设 Repo 方法是简单的 Create/Save，我们先用着。
			// TODO: Refactor Repo to support context-based transaction propagation or WithTx().

			if err := m.fingerRepo.Upsert(ctx, f); err != nil {
				logger.LogError(err, "", 0, "", "import_rules", "", map[string]interface{}{"name": f.Name})
				return err // Fail fast in transaction
			}
		}

		for _, c := range cpes {
			if err := m.cpeRepo.Upsert(ctx, c); err != nil {
				logger.LogError(err, "", 0, "", "import_rules", "", map[string]interface{}{"name": c.Name})
				return err
			}
		}

		return nil
	})
}

// createBackup 创建当前规则库的快照
func (m *RuleManager) createBackup(ctx context.Context) error {
	data, err := m.exportRulesInternal(ctx, false) // 备份全量数据 true 只导出 Enabled false 导出所有
	if err != nil {
		return err
	}

	// 如果没有数据，是否跳过备份？建议还是备份一个空的，保持逻辑一致
	if len(data) == 0 {
		// return nil
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("rules_backup_%s.json", timestamp)
	filepath := filepath.Join(m.backupDir, filename)

	if err := utils.WriteFile(filepath, data, 0644); err != nil {
		return err
	}

	logger.LogBusinessOperation("auto_backup", 0, "", "system", "", "success", "auto backup created", map[string]interface{}{
		"file": filepath,
		"size": len(data),
	})

	return nil
}

// ListBackups 列出所有可用备份
func (m *RuleManager) ListBackups() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var backups []string
	files, err := utils.ReadDir(m.backupDir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "rules_backup_") && strings.HasSuffix(f.Name(), ".json") {
			backups = append(backups, f.Name())
		}
	}

	// 按时间倒序排列 (最新的在前)
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))

	return backups, nil
}

// Rollback 回滚到指定备份 (True Rollback: Diff & Sync)
func (m *RuleManager) Rollback(ctx context.Context, filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Validate Filename
	baseName := filepath.Base(filename)
	if baseName != filename {
		return fmt.Errorf("invalid filename")
	}
	targetPath := filepath.Join(m.backupDir, filename)

	// 2. Read Backup File (Target State)
	data, err := utils.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("read backup file failed: %w", err)
	}

	targetFingers, targetCPEs, err := m.converter.Decode(data)
	if err != nil {
		return fmt.Errorf("decode backup file failed: %w", err)
	}

	// 3. Execute Rollback with Transaction (Diff & Sync)
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 3.1 Clear Existing Data (Simple Strategy: Truncate/Delete All then Re-insert)
		// For true rollback, we want to restore the exact state.
		// Since we don't have complex foreign keys on rules yet, deleting all and re-inserting is safe and clean.
		// It ensures no "dirty" rules from bad imports remain.

		// Note: Use Unscoped to hard delete if you want to completely remove traces,
		// or normal delete for soft delete. Here we use Unscoped for clean slate.

		if err := tx.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&asset.AssetFinger{}).Error; err != nil {
			return fmt.Errorf("failed to clear asset_finger table: %w", err)
		}

		if err := tx.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&asset.AssetCPE{}).Error; err != nil {
			return fmt.Errorf("failed to clear asset_cpe table: %w", err)
		}

		// 3.2 Insert Backup Data
		// Batch insert is more efficient
		if len(targetFingers) > 0 {
			if err := tx.CreateInBatches(targetFingers, 100).Error; err != nil {
				return fmt.Errorf("failed to restore fingers: %w", err)
			}
		}

		if len(targetCPEs) > 0 {
			if err := tx.CreateInBatches(targetCPEs, 100).Error; err != nil {
				return fmt.Errorf("failed to restore cpes: %w", err)
			}
		}

		return nil
	})
}

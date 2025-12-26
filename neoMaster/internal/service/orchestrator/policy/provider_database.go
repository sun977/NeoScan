// DatabaseProvider 数据库来源提供者
// 实现 TargetProvider 接口，从数据库查询目标
// 功能：
// 1. 查询数据库表/视图/自定义SQL，返回目标列表
// 2. 支持 WHERE 子句过滤、分页、排序
// 3. 解析结果，将 指定列 作为 Target.Value，其他列作为 Target.Meta【强制要求必须指定列】
// 4. 支持自定义 SQL 语句查询

package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	orcmodel "neomaster/internal/model/orchestrator"

	"gorm.io/gorm"
)

// DatabaseProvider 数据库来源提供者
type DatabaseProvider struct {
	db *gorm.DB // 全局数据库连接
}

// NewDatabaseProvider 创建数据库提供者
func NewDatabaseProvider(db *gorm.DB) *DatabaseProvider {
	return &DatabaseProvider{
		db: db,
	}
}

// DBQueryConfig 数据库查询配置 (对应 filter_rules)
type DBQueryConfig struct {
	Where []DBFilterRule `json:"where"`
	Limit int            `json:"limit"`
}

// DBFilterRule 单条过滤规则
type DBFilterRule struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}

// DBParserConfig 数据库解析配置 (对应 parser_config)
type DBParserConfig struct {
	ValueColumn string   `json:"value_column"` // 核心：哪一列是 Target.Value (必填)
	MetaColumns []string `json:"meta_columns"` // 可选：哪些列放入 Target.Meta
}

func (d *DatabaseProvider) Name() string { return "database" }

func (d *DatabaseProvider) Provide(ctx context.Context, config orcmodel.TargetSource, seedTargets []string) ([]Target, error) {
	// 暂时仅支持 MySQL 数据库
	// TODO: 支持更多数据库类型
	// 注意：目前设计为复用系统全局 DB，如果连接外部 DB 需在 AuthConfig 中配置连接信息并在此处建立临时连接

	// 1. 获取数据库连接
	// 如果配置了 AuthConfig，说明是外部数据库，需要建立连接
	// 如果没有配置 AuthConfig，默认查询系统内部数据库 (复用 d.db)
	var db *gorm.DB
	if len(config.AuthConfig) > 0 {
		// TODO: 实现连接外部数据库逻辑
		return nil, fmt.Errorf("external database connection not supported yet")
	} else {
		if d.db == nil {
			return nil, fmt.Errorf("system database connection is not initialized in DatabaseProvider")
		}
		db = d.db
	}

	// 2. 解析配置
	if config.QueryMode != "table" && config.QueryMode != "view" && config.QueryMode != "sql" {
		return nil, fmt.Errorf("unsupported query_mode: %s (only table/view/sql supported currently)", config.QueryMode)
	}

	var parserConfig DBParserConfig
	if len(config.ParserConfig) > 0 {
		if err := json.Unmarshal(config.ParserConfig, &parserConfig); err != nil {
			return nil, fmt.Errorf("invalid parser_config: %w", err)
		}
	}
	// 不论哪种模式，value_column 都是必须的，用于映射结果
	if parserConfig.ValueColumn == "" {
		return nil, fmt.Errorf("value_column is required in parser_config")
	}

	var tx *gorm.DB

	// 分支处理：SQL模式 vs Table/View模式
	if config.QueryMode == "sql" {
		// --- SQL 模式 ---
		sqlStr, ok := config.SourceValue.(string)
		if !ok || sqlStr == "" {
			return nil, fmt.Errorf("source_value (sql) is required and must be string when query_mode is sql")
		}

		// 安全检查：仅允许 SELECT 语句
		// 简单的字符串前缀检查，忽略大小写和空格
		trimmedSQL := strings.TrimSpace(sqlStr)
		if len(trimmedSQL) < 6 || !strings.EqualFold(trimmedSQL[:6], "SELECT") {
			return nil, fmt.Errorf("security violation: only SELECT statements are allowed in custom_sql")
		}

		// 使用 Raw SQL
		tx = db.Raw(sqlStr)

	} else {
		// --- Table/View 模式 ---
		tableName, ok := config.SourceValue.(string)
		if !ok || tableName == "" {
			return nil, fmt.Errorf("source_value (table/view name) is required and must be string")
		}
		// 简单校验表名防止注入 (只允许字母数字下划线)
		if !isValidTableName(tableName) {
			return nil, fmt.Errorf("invalid table name: %s", tableName)
		}

		var queryConfig DBQueryConfig
		if len(config.FilterRules) > 0 {
			if err := json.Unmarshal(config.FilterRules, &queryConfig); err != nil {
				return nil, fmt.Errorf("invalid filter_rules: %w", err)
			}
		}

		// 构建查询
		tx = db.Table(tableName)

		// 应用 Where 条件
		for _, rule := range queryConfig.Where {
			if !isValidFieldName(rule.Field) {
				return nil, fmt.Errorf("invalid field name: %s", rule.Field)
			}
			if !isValidOp(rule.Op) {
				return nil, fmt.Errorf("invalid operator: %s", rule.Op)
			}
			// 安全构建 Where 子句，GORM 会处理 Value 的参数化
			tx = tx.Where(fmt.Sprintf("%s %s ?", rule.Field, rule.Op), rule.Value)
		}

		if queryConfig.Limit > 0 {
			tx = tx.Limit(queryConfig.Limit)
		}

		// 指定 Select 列
		selectCols := append([]string{parserConfig.ValueColumn}, parserConfig.MetaColumns...)
		tx = tx.Select(selectCols)
	}

	// 4. 执行查询
	var results []map[string]interface{}
	if err := tx.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// 5. 结果映射
	targets := make([]Target, 0, len(results))
	for _, row := range results {
		val, ok := row[parserConfig.ValueColumn]
		if !ok || val == nil {
			continue
		}
		valStr := fmt.Sprintf("%v", val)
		if valStr = strings.TrimSpace(valStr); valStr == "" {
			continue
		}

		meta := make(map[string]string)
		for _, metaCol := range parserConfig.MetaColumns {
			if v, exists := row[metaCol]; exists && v != nil {
				meta[metaCol] = fmt.Sprintf("%v", v)
			}
		}

		targets = append(targets, Target{
			Type:   config.TargetType,
			Value:  valStr,
			Source: fmt.Sprintf("database:%v", config.SourceValue),
			Meta: orcmodel.TargetMeta{
				Custom: meta,
			},
		})
	}

	return targets, nil
}

func (d *DatabaseProvider) HealthCheck(ctx context.Context) error {
	if d.db == nil {
		return fmt.Errorf("database connection not initialized")
	}
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// 辅助校验函数
func isValidTableName(name string) bool {
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func isValidFieldName(name string) bool {
	return isValidTableName(name) // 规则相同
}

func isValidOp(op string) bool {
	validOps := map[string]bool{
		"=": true, ">": true, "<": true, ">=": true, "<=": true,
		"!=": true, "LIKE": true, "IN": true,
	}
	return validOps[strings.ToUpper(op)]
}

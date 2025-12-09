// 目标提供者
// 处理多元异构数据，将其转换为统一的 Target 格式作为扫描目标
// 工作流程如下：
// 1.初始化：创建TargetProvider实例并注册各种SourceProvider（包括实际实现和桩实现）
// 2.策略解析：通过ResolveTargets方法解析目标策略：
// - 如果策略为空，使用种子目标作为回退
// - 解析JSON策略配置
// - 遍历所有目标源配置
// 3.目标获取：针对每个目标源配置：
// - 根据源类型查找对应的Provider
// - 调用Provider的Provide方法获取目标
// - 处理异常情况（未知源类型或获取失败）
// 4.结果处理：
// - 合并所有获取到的目标
// - 基于目标值进行去重
// - 返回统一格式的目标列表
// 目前已实现的Provider包括：
// ManualProvider：处理手动输入的目标
// ProjectTargetProvider：处理项目种子目标
// 其他Provider：以桩形式存在，待后续实现
// 这种设计使得系统可以灵活地从多种来源获取扫描目标，并将其统一为标准格式，便于后续处理。

package policy

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"neomaster/internal/pkg/logger"
	"os"
	"strings"
	"sync"
)

// TargetProvider 目标提供者服务接口
type TargetProvider interface {
	// ResolveTargets 解析策略并返回目标列表
	ResolveTargets(ctx context.Context, policyJSON string, seedTargets []string) ([]Target, error)
	// RegisterProvider 注册新的目标源提供者
	RegisterProvider(name string, provider SourceProvider)
	// CheckHealth 检查所有已注册 Provider 的健康状态
	CheckHealth(ctx context.Context) map[string]error
}

// SourceProvider 单个目标源提供者接口
// 对应设计文档中的 TargetProvider Interface
type SourceProvider interface {
	// Name 返回 Provider 名称
	Name() string

	// Provide 执行获取逻辑
	// ctx: 上下文
	// config: 目标源配置
	// seedTargets: 种子目标(用于 project_target)
	Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error)

	// HealthCheck 检查 Provider 健康状态 (如数据库连接、文件访问权限)
	HealthCheck(ctx context.Context) error
}

// Target 标准目标对象
// 对应设计文档: 运行时对象 (Target Object)
type Target struct {
	Type   string            `json:"type"`   // 目标类型: ip, domain, url
	Value  string            `json:"value"`  // 目标值
	Source string            `json:"source"` // 来源标识
	Meta   map[string]string `json:"meta"`   // 元数据
}

// TargetPolicyConfig 目标策略配置结构
// 对应 ScanStage.target_policy
type TargetPolicyConfig struct {
	TargetSources []TargetSourceConfig `json:"target_sources"`
}

// TargetSourceConfig 目标源配置详细结构
// 对应设计文档: 配置结构 (TargetSource Config)
type TargetSourceConfig struct {
	SourceType   string          `json:"source_type"`             // manual, project_target, file, database, api, previous_stage
	QueryMode    string          `json:"query_mode,omitempty"`    // table, view, sql (仅用于数据库)
	TargetType   string          `json:"target_type"`             // ip, ip_range, domain, url
	SourceValue  string          `json:"source_value,omitempty"`  // 具体值
	CustomSQL    string          `json:"custom_sql,omitempty"`    // custom_sql
	FilterRules  json.RawMessage `json:"filter_rules,omitempty"`  // 过滤规则
	AuthConfig   json.RawMessage `json:"auth_config,omitempty"`   // 认证配置
	ParserConfig json.RawMessage `json:"parser_config,omitempty"` // 解析配置
}

// 定义目标提供者服务
type targetProviderService struct {
	providers map[string]SourceProvider // 储存已注册的目标源提供者
	mu        sync.RWMutex              // 读写互斥锁, 保护 providers  map 并发访问
}

// NewTargetProvider 创建目标提供者实例
func NewTargetProvider() TargetProvider {
	svc := &targetProviderService{
		providers: make(map[string]SourceProvider),
	}
	// 注册内置提供者
	svc.RegisterProvider("manual", &ManualProvider{})                // 手动输入来源提供(前端用户手动输入目标)
	svc.RegisterProvider("project_target", &ProjectTargetProvider{}) // 项目种子目标提供(直接用项目目标覆盖当前目标)
	svc.RegisterProvider("file", &FileProvider{})                    // 注册文件提供者

	// 注册待实现的提供者 (占位符)
	svc.RegisterProvider("database", &StubProvider{name: "database"})
	svc.RegisterProvider("api", &StubProvider{name: "api"})
	svc.RegisterProvider("previous_stage", &StubProvider{name: "previous_stage"})

	return svc
}

// RegisterProvider 注册新的目标源提供者
func (p *targetProviderService) RegisterProvider(name string, provider SourceProvider) {
	p.mu.Lock() // 使用写锁保证并发安全
	defer p.mu.Unlock()
	p.providers[name] = provider // 注册新的目标源提供者
}

// CheckHealth 检查所有已注册 Provider 的健康状态
func (p *targetProviderService) CheckHealth(ctx context.Context) map[string]error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	results := make(map[string]error)
	for name, provider := range p.providers {
		results[name] = provider.HealthCheck(ctx)
	}
	return results
}

// ResolveTargets 解析策略并返回目标列表 --- 核心实现逻辑
// 策略解析器：对应 ScanStage.target_policy
// 1. 如果策略为空，默认使用种子目标
// 2. 不为空解析 json 策略
// 3. 并发/顺序获取所有目标
// 4. 去重 (基于 Value)
// 策略样例：
//
//	{
//	  "target_sources": [
//	    {
//	      "source_type": "file",           // 来源类型：file/db/view/sql/manual/api/previous_stage【上一个阶段结果】
//	      "source_value": "/path/to/targets.txt",  // 根据类型的具体值
//	      "target_type": "ip_range"        // 目标类型：ip/ip_range/domain/url
//	    }
//	  ],
//	  "whitelist_enabled": true,           // 是否启用白名单
//	  "whitelist_sources": [               // 白名单来源
//	    {
//	      "source_type": "file",
//	      "source_value": "/path/to/whitelist.txt"
//	    }
//	  ],
//	  "skip_enabled": true,                // 是否启用跳过条件
//	  "skip_conditions": [                 // 跳过条件,列表中可添加多个条件
//	    {
//	      "condition_field": "device_type",
//	      "operator": "equals",
//	      "value": "honeypot"
//	    }
//	  ]
//	}
func (p *targetProviderService) ResolveTargets(ctx context.Context, policyJSON string, seedTargets []string) ([]Target, error) {
	// 1. 如果策略为空，默认使用种子目标
	if policyJSON == "" || policyJSON == "{}" {
		return p.fallbackToSeed(seedTargets), nil // 种子目标转换成 Target 对象,调用 fallbackToSeed 方法返回
	}

	// 策略不为空则解析 json 策略
	var config TargetPolicyConfig
	if err := json.Unmarshal([]byte(policyJSON), &config); err != nil {
		logger.LogWarn("Failed to parse target policy, using seed targets", "", 0, "", "ResolveTargets", "", map[string]interface{}{
			"error":  err.Error(),
			"policy": policyJSON,
		})
		return p.fallbackToSeed(seedTargets), nil
	}

	// 2. 如果没有配置源，也使用种子目标
	if len(config.TargetSources) == 0 {
		return p.fallbackToSeed(seedTargets), nil
	}

	// 3. 并发/顺序获取所有目标
	allTargets := make([]Target, 0)
	// 遍历所有目标源配置,查找对应的目标源提供者
	for _, sourceConfig := range config.TargetSources {
		p.mu.RLock()
		provider, exists := p.providers[sourceConfig.SourceType] // 查找对应的目标源提供者
		p.mu.RUnlock()

		// 目标源提供者不存在, 记录警告日志, 跳过该源
		if !exists {
			logger.LogWarn("Unknown source type", "", 0, "", "ResolveTargets", "", map[string]interface{}{
				"type": sourceConfig.SourceType,
			})
			continue // 跳过未知的源类型
		}

		// 目标源提供者存在 --- 调用目标源提供者的Provide方法获取目标
		targets, err := provider.Provide(ctx, sourceConfig, seedTargets)
		if err != nil {
			logger.LogError(err, "Provider failed to get targets", 0, "", "ResolveTargets", "", map[string]interface{}{
				"type": sourceConfig.SourceType,
			})
			continue // 跳过获取目标失败的源
		}

		// 目标源提供者成功获取目标 --- 合并到 allTargets 中
		allTargets = append(allTargets, targets...)
	}

	// 4. 去重 (基于 Value)
	targetSet := make(map[string]struct{})
	result := make([]Target, 0, len(allTargets))
	for _, t := range allTargets {
		if _, ok := targetSet[t.Value]; !ok {
			targetSet[t.Value] = struct{}{}
			result = append(result, t)
		}
	}

	return result, nil
}

// fallbackToSeed 辅助方法：将种子目标转换为 Target 对象
func (p *targetProviderService) fallbackToSeed(seedTargets []string) []Target {
	targets := make([]Target, 0, len(seedTargets))
	for _, t := range seedTargets {
		targets = append(targets, Target{
			Type:   "unknown",
			Value:  t,
			Source: "seed",
			Meta:   nil,
		})
	}
	return targets
}

// --- 具体 Provider 实现 ---

// ManualProvider 人工输入
type ManualProvider struct{}

func (m *ManualProvider) Name() string { return "manual" }

func (m *ManualProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	parts := strings.FieldsFunc(config.SourceValue, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	targets := make([]Target, 0, len(parts))
	for _, part := range parts {
		t := strings.TrimSpace(part)
		if t != "" {
			targets = append(targets, Target{
				Type:   config.TargetType, // 使用配置中的类型
				Value:  t,
				Source: "manual",
				Meta:   nil,
			})
		}
	}
	return targets, nil
}

func (m *ManualProvider) HealthCheck(ctx context.Context) error {
	return nil
}

// ProjectTargetProvider 项目种子目标
type ProjectTargetProvider struct{}

func (p *ProjectTargetProvider) Name() string { return "project_target" }

func (p *ProjectTargetProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	targets := make([]Target, 0, len(seedTargets))
	for _, t := range seedTargets {
		targets = append(targets, Target{
			Type:   config.TargetType, // 假设种子目标类型与配置一致，或者需要自动检测
			Value:  t,
			Source: "project_target",
			Meta:   nil,
		})
	}
	return targets, nil
}

func (p *ProjectTargetProvider) HealthCheck(ctx context.Context) error {
	return nil
}

// FileProvider 文件导入
type FileProvider struct{}

// FileParserConfig 文件解析配置
type FileParserConfig struct {
	Format    string `json:"format"`     // line, csv, json_array
	CSVColumn string `json:"csv_column"` // 如果是 CSV，指定列名
	JSONPath  string `json:"json_path"`  // 如果是 JSON，指定提取路径
}

func (f *FileProvider) Name() string { return "file" }

func (f *FileProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	// 1. 验证文件路径
	if config.SourceValue == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	// 2. 解析配置
	var parserConfig FileParserConfig
	if len(config.ParserConfig) > 0 {
		if err := json.Unmarshal(config.ParserConfig, &parserConfig); err != nil {
			return nil, fmt.Errorf("invalid parser config: %w", err)
		}
	}
	// 默认格式为 line
	if parserConfig.Format == "" {
		parserConfig.Format = "line"
	}

	// 3. 读取文件
	file, err := os.Open(config.SourceValue)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", config.SourceValue, err)
	}
	defer file.Close()

	// 4. 根据格式解析
	var values []string
	switch parserConfig.Format {
	case "line":
		values, err = f.parseLine(file)
	case "csv":
		values, err = f.parseCSV(file, parserConfig.CSVColumn)
	case "json_array":
		values, err = f.parseJSON(file, parserConfig.JSONPath)
	default:
		return nil, fmt.Errorf("unsupported format: %s", parserConfig.Format)
	}

	if err != nil {
		return nil, err
	}

	// 5. 转换为 Target 对象
	targets := make([]Target, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			targets = append(targets, Target{
				Type:   config.TargetType,
				Value:  v,
				Source: "file:" + config.SourceValue,
				Meta:   nil,
			})
		}
	}

	return targets, nil
}

func (f *FileProvider) parseLine(r io.Reader) ([]string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	return lines, nil
}

func (f *FileProvider) parseCSV(r io.Reader, column string) ([]string, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	// 查找列索引
	colIdx := 0 // 默认第一列
	if column != "" {
		header := records[0]
		found := false
		for i, h := range header {
			if h == column {
				colIdx = i
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("column %s not found in csv header", column)
		}
		// 如果指定了列名，假设第一行是 header，跳过
		records = records[1:]
	}

	result := make([]string, 0, len(records))
	for _, record := range records {
		if len(record) > colIdx {
			result = append(result, record[colIdx])
		}
	}
	return result, nil
}

func (f *FileProvider) parseJSON(r io.Reader, jsonPath string) ([]string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// 简单实现：支持 string array 或 object array (with json_path pointing to field)
	// 暂时只支持 root 是 array 的情况
	var root interface{}
	if err := json.Unmarshal(content, &root); err != nil {
		return nil, err
	}

	arr, ok := root.([]interface{})
	if !ok {
		return nil, fmt.Errorf("json content must be an array")
	}

	result := make([]string, 0, len(arr))
	for _, item := range arr {
		switch v := item.(type) {
		case string:
			result = append(result, v)
		case map[string]interface{}:
			// 简单的路径提取: "data.targets[*].ip" -> 我们只看最后一段 "ip"
			// 这是一个简化实现，不支持复杂的 json path
			if jsonPath != "" {
				// 提取字段名 (这里假设 path 类似于 "xxx.field" 或只是 "field")
				parts := strings.Split(jsonPath, ".")
				fieldName := parts[len(parts)-1]

				if val, exists := v[fieldName]; exists {
					if strVal, ok := val.(string); ok {
						result = append(result, strVal)
					}
				}
			}
		}
	}
	return result, nil
}

func (f *FileProvider) HealthCheck(ctx context.Context) error {
	// 检查是否有文件读取权限，或者不做特殊检查
	return nil
}

// StubProvider 桩实现
type StubProvider struct {
	name string
}

func (s *StubProvider) Name() string { return s.name }

func (s *StubProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	logger.LogWarn(fmt.Sprintf("%s source not supported yet", s.name), "", 0, "", "Provide", "", nil)
	return nil, nil
}

func (s *StubProvider) HealthCheck(ctx context.Context) error {
	return nil
}

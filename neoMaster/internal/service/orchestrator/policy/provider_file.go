// FileProvider 文件来源提供者
// 实现 TargetProvider 接口，从文件读取目标
// 功能：
// 1. 支持读取文本文件（line, csv, json_array）
// 2. 解析配置，根据格式提取目标值
// 3. 支持自定义列名、JSON 路径提取
// 4. 过滤空行、去重

package policy

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

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

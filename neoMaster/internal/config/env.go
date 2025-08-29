package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvManager 环境变量管理器
type EnvManager struct {
	prefix string // 环境变量前缀
}

// NewEnvManager 创建环境变量管理器
func NewEnvManager(prefix string) *EnvManager {
	if prefix == "" {
		prefix = "NEOSCAN"
	}
	return &EnvManager{
		prefix: prefix,
	}
}

// GetString 获取字符串类型环境变量
func (em *EnvManager) GetString(key, defaultValue string) string {
	envKey := em.buildEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetInt 获取整数类型环境变量
func (em *EnvManager) GetInt(key string, defaultValue int) int {
	envKey := em.buildEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// GetInt64 获取64位整数类型环境变量
func (em *EnvManager) GetInt64(key string, defaultValue int64) int64 {
	envKey := em.buildEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}

	int64Value, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return int64Value
}

// GetBool 获取布尔类型环境变量
func (em *EnvManager) GetBool(key string, defaultValue bool) bool {
	envKey := em.buildEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

// GetFloat64 获取浮点数类型环境变量
func (em *EnvManager) GetFloat64(key string, defaultValue float64) float64 {
	envKey := em.buildEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return floatValue
}

// GetDuration 获取时间间隔类型环境变量
func (em *EnvManager) GetDuration(key string, defaultValue time.Duration) time.Duration {
	envKey := em.buildEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

// GetStringSlice 获取字符串切片类型环境变量（逗号分隔）
func (em *EnvManager) GetStringSlice(key string, defaultValue []string) []string {
	envKey := em.buildEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}

	// 按逗号分割并去除空白
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}
	return result
}

// SetString 设置字符串类型环境变量
func (em *EnvManager) SetString(key, value string) error {
	envKey := em.buildEnvKey(key)
	return os.Setenv(envKey, value)
}

// SetInt 设置整数类型环境变量
func (em *EnvManager) SetInt(key string, value int) error {
	envKey := em.buildEnvKey(key)
	return os.Setenv(envKey, strconv.Itoa(value))
}

// SetBool 设置布尔类型环境变量
func (em *EnvManager) SetBool(key string, value bool) error {
	envKey := em.buildEnvKey(key)
	return os.Setenv(envKey, strconv.FormatBool(value))
}

// SetDuration 设置时间间隔类型环境变量
func (em *EnvManager) SetDuration(key string, value time.Duration) error {
	envKey := em.buildEnvKey(key)
	return os.Setenv(envKey, value.String())
}

// SetStringSlice 设置字符串切片类型环境变量（逗号分隔）
func (em *EnvManager) SetStringSlice(key string, value []string) error {
	envKey := em.buildEnvKey(key)
	return os.Setenv(envKey, strings.Join(value, ","))
}

// Unset 删除环境变量
func (em *EnvManager) Unset(key string) error {
	envKey := em.buildEnvKey(key)
	return os.Unsetenv(envKey)
}

// Exists 检查环境变量是否存在
func (em *EnvManager) Exists(key string) bool {
	envKey := em.buildEnvKey(key)
	_, exists := os.LookupEnv(envKey)
	return exists
}

// GetAll 获取所有带前缀的环境变量
func (em *EnvManager) GetAll() map[string]string {
	result := make(map[string]string)
	prefix := em.prefix + "_"

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		if strings.HasPrefix(key, prefix) {
			// 移除前缀
			shortKey := strings.TrimPrefix(key, prefix)
			result[shortKey] = value
		}
	}

	return result
}

// buildEnvKey 构建环境变量键名
func (em *EnvManager) buildEnvKey(key string) string {
	if em.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s_%s", em.prefix, strings.ToUpper(key))
}

// 全局环境变量管理器实例
var DefaultEnvManager = NewEnvManager("NEOSCAN")

// 便捷函数，使用默认环境变量管理器

// GetEnvString 获取字符串类型环境变量
func GetEnvString(key, defaultValue string) string {
	return DefaultEnvManager.GetString(key, defaultValue)
}

// GetEnvInt 获取整数类型环境变量
func GetEnvInt(key string, defaultValue int) int {
	return DefaultEnvManager.GetInt(key, defaultValue)
}

// GetEnvInt64 获取64位整数类型环境变量
func GetEnvInt64(key string, defaultValue int64) int64 {
	return DefaultEnvManager.GetInt64(key, defaultValue)
}

// GetEnvBool 获取布尔类型环境变量
func GetEnvBool(key string, defaultValue bool) bool {
	return DefaultEnvManager.GetBool(key, defaultValue)
}

// GetEnvFloat64 获取浮点数类型环境变量
func GetEnvFloat64(key string, defaultValue float64) float64 {
	return DefaultEnvManager.GetFloat64(key, defaultValue)
}

// GetEnvDuration 获取时间间隔类型环境变量
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	return DefaultEnvManager.GetDuration(key, defaultValue)
}

// GetEnvStringSlice 获取字符串切片类型环境变量
func GetEnvStringSlice(key string, defaultValue []string) []string {
	return DefaultEnvManager.GetStringSlice(key, defaultValue)
}

// SetEnvString 设置字符串类型环境变量
func SetEnvString(key, value string) error {
	return DefaultEnvManager.SetString(key, value)
}

// SetEnvInt 设置整数类型环境变量
func SetEnvInt(key string, value int) error {
	return DefaultEnvManager.SetInt(key, value)
}

// SetEnvBool 设置布尔类型环境变量
func SetEnvBool(key string, value bool) error {
	return DefaultEnvManager.SetBool(key, value)
}

// SetEnvDuration 设置时间间隔类型环境变量
func SetEnvDuration(key string, value time.Duration) error {
	return DefaultEnvManager.SetDuration(key, value)
}

// SetEnvStringSlice 设置字符串切片类型环境变量
func SetEnvStringSlice(key string, value []string) error {
	return DefaultEnvManager.SetStringSlice(key, value)
}

// UnsetEnv 删除环境变量
func UnsetEnv(key string) error {
	return DefaultEnvManager.Unset(key)
}

// EnvExists 检查环境变量是否存在
func EnvExists(key string) bool {
	return DefaultEnvManager.Exists(key)
}

// GetAllEnvs 获取所有带前缀的环境变量
func GetAllEnvs() map[string]string {
	return DefaultEnvManager.GetAll()
}

// LoadEnvFile 从.env文件加载环境变量
func LoadEnvFile(filename string) error {
	if filename == "" {
		filename = ".env"
	}

	// 检查文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil // 文件不存在时不报错
	}

	// 读取文件内容
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read env file %s: %w", filename, err)
	}

	// 解析环境变量
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和注释行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析键值对
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid env line %d in file %s: %s", i+1, filename, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 移除引号
		if len(value) >= 2 {
			if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		// 设置环境变量
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set env variable %s: %w", key, err)
		}
	}

	return nil
}

// SaveEnvFile 保存环境变量到.env文件
func SaveEnvFile(filename string, envs map[string]string) error {
	if filename == "" {
		filename = ".env"
	}

	// 构建文件内容
	var lines []string
	lines = append(lines, "# NeoScan Environment Variables")
	lines = append(lines, fmt.Sprintf("# Generated at %s", time.Now().Format(time.RFC3339)))
	lines = append(lines, "")

	for key, value := range envs {
		// 如果值包含空格或特殊字符，添加引号
		if strings.ContainsAny(value, " \t\n\r") {
			value = fmt.Sprintf(`"%s"`, value)
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	content := strings.Join(lines, "\n")

	// 写入文件
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write env file %s: %w", filename, err)
	}

	return nil
}

// ValidateRequiredEnvs 验证必需的环境变量是否存在
func ValidateRequiredEnvs(requiredEnvs []string) error {
	var missingEnvs []string

	for _, env := range requiredEnvs {
		if !EnvExists(env) {
			missingEnvs = append(missingEnvs, DefaultEnvManager.buildEnvKey(env))
		}
	}

	if len(missingEnvs) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missingEnvs, ", "))
	}

	return nil
}
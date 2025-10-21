package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// EnvManager 环境变量管理器
type EnvManager struct {
	prefix string // 环境变量前缀
}

// NewEnvManager 创建环境变量管理器
func NewEnvManager(prefix string) *EnvManager {
	if prefix == "" {
		prefix = "NEOAGENT"
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

// buildEnvKey 构建环境变量键名
func (em *EnvManager) buildEnvKey(key string) string {
	if em.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s_%s", em.prefix, key)
}

// EnvLoader 环境变量加载器
// @author: sun977
// @date: 2025.01.14
// @description: 负责从环境变量和.env文件加载配置
type EnvLoader struct {
	envFiles []string // .env文件路径列表
	loaded   bool     // 是否已加载
}

// NewEnvLoader 创建环境变量加载器
func NewEnvLoader(envFiles ...string) *EnvLoader {
	if len(envFiles) == 0 {
		envFiles = []string{".env"}
	}
	return &EnvLoader{
		envFiles: envFiles,
		loaded:   false,
	}
}

// Load 加载环境变量
func (e *EnvLoader) Load() error {
	if e.loaded {
		return nil
	}

	// 加载.env文件
	for _, envFile := range e.envFiles {
		if err := e.loadEnvFile(envFile); err != nil {
			// .env文件不存在不算错误
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to load env file %s: %w", envFile, err)
			}
		}
	}

	e.loaded = true
	return nil
}

// loadEnvFile 加载单个.env文件
func (e *EnvLoader) loadEnvFile(envFile string) error {
	// 检查文件是否存在
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return err
	}

	// 加载.env文件
	if err := godotenv.Load(envFile); err != nil {
		return fmt.Errorf("failed to load %s: %w", envFile, err)
	}

	return nil
}

// GetString 获取字符串环境变量
func (e *EnvLoader) GetString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetInt 获取整数环境变量
func (e *EnvLoader) GetInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetInt64 获取64位整数环境变量
func (e *EnvLoader) GetInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetFloat64 获取浮点数环境变量
func (e *EnvLoader) GetFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// GetBool 获取布尔值环境变量
func (e *EnvLoader) GetBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetDuration 获取时间间隔环境变量
func (e *EnvLoader) GetDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetStringSlice 获取字符串切片环境变量 (逗号分隔)
func (e *EnvLoader) GetStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// 按逗号分割并去除空白
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

// GetIntSlice 获取整数切片环境变量 (逗号分隔)
func (e *EnvLoader) GetIntSlice(key string, defaultValue []int) []int {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]int, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				if intValue, err := strconv.Atoi(trimmed); err == nil {
					result = append(result, intValue)
				}
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

// GetPath 获取路径环境变量 (自动处理相对路径)
func (e *EnvLoader) GetPath(key, defaultValue string) string {
	path := e.GetString(key, defaultValue)
	if path == "" {
		return ""
	}

	// 如果是相对路径，转换为绝对路径
	if !filepath.IsAbs(path) {
		if absPath, err := filepath.Abs(path); err == nil {
			return absPath
		}
	}

	return path
}

// IsSet 检查环境变量是否已设置
func (e *EnvLoader) IsSet(key string) bool {
	_, exists := os.LookupEnv(key)
	return exists
}

// MustGetString 获取必需的字符串环境变量
func (e *EnvLoader) MustGetString(key string) (string, error) {
	if value := os.Getenv(key); value != "" {
		return value, nil
	}
	return "", fmt.Errorf("required environment variable %s is not set", key)
}

// MustGetInt 获取必需的整数环境变量
func (e *EnvLoader) MustGetInt(key string) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, fmt.Errorf("required environment variable %s is not set", key)
	}
	
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s is not a valid integer: %w", key, err)
	}
	
	return intValue, nil
}

// MustGetBool 获取必需的布尔值环境变量
func (e *EnvLoader) MustGetBool(key string) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return false, fmt.Errorf("required environment variable %s is not set", key)
	}
	
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("environment variable %s is not a valid boolean: %w", key, err)
	}
	
	return boolValue, nil
}

// MustGetDuration 获取必需的时间间隔环境变量
func (e *EnvLoader) MustGetDuration(key string) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, fmt.Errorf("required environment variable %s is not set", key)
	}
	
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s is not a valid duration: %w", key, err)
	}
	
	return duration, nil
}

// SetEnv 设置环境变量 (主要用于测试)
func (e *EnvLoader) SetEnv(key, value string) error {
	return os.Setenv(key, value)
}

// UnsetEnv 取消设置环境变量 (主要用于测试)
func (e *EnvLoader) UnsetEnv(key string) error {
	return os.Unsetenv(key)
}

// GetAllEnv 获取所有环境变量
func (e *EnvLoader) GetAllEnv() map[string]string {
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	return envMap
}

// GetEnvWithPrefix 获取指定前缀的所有环境变量
func (e *EnvLoader) GetEnvWithPrefix(prefix string) map[string]string {
	envMap := make(map[string]string)
	for key, value := range e.GetAllEnv() {
		if strings.HasPrefix(key, prefix) {
			// 移除前缀
			newKey := strings.TrimPrefix(key, prefix)
			if newKey != "" {
				envMap[newKey] = value
			}
		}
	}
	return envMap
}

// ValidateRequired 验证必需的环境变量
func (e *EnvLoader) ValidateRequired(requiredKeys []string) error {
	var missingKeys []string
	
	for _, key := range requiredKeys {
		if !e.IsSet(key) {
			missingKeys = append(missingKeys, key)
		}
	}
	
	if len(missingKeys) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missingKeys, ", "))
	}
	
	return nil
}

// 全局环境变量加载器实例
var globalEnvLoader *EnvLoader

// InitGlobalEnvLoader 初始化全局环境变量加载器
func InitGlobalEnvLoader(envFiles ...string) error {
	globalEnvLoader = NewEnvLoader(envFiles...)
	return globalEnvLoader.Load()
}

// GetGlobalEnvLoader 获取全局环境变量加载器
func GetGlobalEnvLoader() *EnvLoader {
	if globalEnvLoader == nil {
		globalEnvLoader = NewEnvLoader()
		_ = globalEnvLoader.Load() // 忽略错误，使用默认值
	}
	return globalEnvLoader
}

// 便捷函数，使用全局加载器

// EnvString 获取字符串环境变量
func EnvString(key, defaultValue string) string {
	return GetGlobalEnvLoader().GetString(key, defaultValue)
}

// EnvInt 获取整数环境变量
func EnvInt(key string, defaultValue int) int {
	return GetGlobalEnvLoader().GetInt(key, defaultValue)
}

// EnvBool 获取布尔值环境变量
func EnvBool(key string, defaultValue bool) bool {
	return GetGlobalEnvLoader().GetBool(key, defaultValue)
}

// EnvDuration 获取时间间隔环境变量
func EnvDuration(key string, defaultValue time.Duration) time.Duration {
	return GetGlobalEnvLoader().GetDuration(key, defaultValue)
}

// EnvStringSlice 获取字符串切片环境变量
func EnvStringSlice(key string, defaultValue []string) []string {
	return GetGlobalEnvLoader().GetStringSlice(key, defaultValue)
}

// EnvPath 获取路径环境变量
func EnvPath(key, defaultValue string) string {
	return GetGlobalEnvLoader().GetPath(key, defaultValue)
}
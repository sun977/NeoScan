package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var (
	// GlobalConfig 全局配置实例
	GlobalConfig *Config
)

// LoadConfig 加载配置文件
// configPath: 配置文件路径，如果为空则使用默认路径
// env: 环境标识，支持 development, test, production
func LoadConfig(configPath, env string) (*Config, error) {
	// 设置默认环境
	if env == "" {
		env = getEnvFromEnvironment()
	}

	// 创建viper实例
	v := viper.New()

	// 设置配置文件类型
	v.SetConfigType("yaml")

	// 设置配置文件路径
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// 根据环境选择配置文件
	configFile := getConfigFileName(configPath, env)
	v.SetConfigFile(configFile)

	// 设置环境变量前缀
	v.SetEnvPrefix("NEOSCAN")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 绑定环境变量
	bindEnvironmentVariables(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	// 解析配置到结构体
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	applyDefaultRulesConfig(&config)

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// 设置全局配置
	GlobalConfig = &config

	return &config, nil
}

// getEnvFromEnvironment 从环境变量获取环境标识
func getEnvFromEnvironment() string {
	env := os.Getenv("NEOSCAN_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = "development" // 默认开发环境
	}
	return env
}

// getDefaultConfigPath 获取默认配置文件路径
func getDefaultConfigPath() string {
	// 尝试从环境变量获取配置路径
	if configPath := os.Getenv("NEOSCAN_CONFIG_PATH"); configPath != "" {
		return configPath
	}

	// 使用默认路径
	return "configs"
}

// getConfigFileName 根据环境获取配置文件名
func getConfigFileName(configPath, env string) string {
	var configFile string

	switch env {
	case "production", "prod":
		configFile = filepath.Join(configPath, "config.prod.yaml")
	case "test", "testing":
		configFile = filepath.Join(configPath, "config.test.yaml")
	default:
		configFile = filepath.Join(configPath, "config.yaml")
	}

	// 检查文件是否存在，如果不存在则使用默认配置文件
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		defaultConfig := filepath.Join(configPath, "config.yaml")
		if _, err := os.Stat(defaultConfig); err == nil {
			return defaultConfig
		}
	}

	return configFile
}

// bindEnvironmentVariables 绑定环境变量
func bindEnvironmentVariables(v *viper.Viper) {
	// 数据库配置
	v.BindEnv("database.mysql.host", "NEOSCAN_MYSQL_HOST")
	v.BindEnv("database.mysql.port", "NEOSCAN_MYSQL_PORT")
	v.BindEnv("database.mysql.username", "NEOSCAN_MYSQL_USERNAME")
	v.BindEnv("database.mysql.password", "NEOSCAN_MYSQL_PASSWORD")
	v.BindEnv("database.mysql.database", "NEOSCAN_MYSQL_DATABASE")

	v.BindEnv("database.redis.host", "NEOSCAN_REDIS_HOST")
	v.BindEnv("database.redis.port", "NEOSCAN_REDIS_PORT")
	v.BindEnv("database.redis.password", "NEOSCAN_REDIS_PASSWORD")
	v.BindEnv("database.redis.database", "NEOSCAN_REDIS_DATABASE")

	// JWT配置 (更新为嵌套在security下的路径)
	v.BindEnv("security.jwt.secret", "NEOSCAN_JWT_SECRET")
	v.BindEnv("security.jwt.access_token_expire", "NEOSCAN_JWT_ACCESS_TOKEN_EXPIRE")
	v.BindEnv("security.jwt.refresh_token_expire", "NEOSCAN_JWT_REFRESH_TOKEN_EXPIRE")
	v.BindEnv("security.jwt.issuer", "NEOSCAN_JWT_ISSUER")
	v.BindEnv("security.jwt.algorithm", "NEOSCAN_JWT_ALGORITHM")

	// 安全配置
	v.BindEnv("security.cors.allow_origins", "NEOSCAN_CORS_ALLOW_ORIGINS")
	v.BindEnv("security.csrf.secret", "NEOSCAN_CSRF_SECRET")

	// 邮件配置
	v.BindEnv("mail.smtp_host", "NEOSCAN_MAIL_SMTP_HOST")
	v.BindEnv("mail.smtp_port", "NEOSCAN_MAIL_SMTP_PORT")
	v.BindEnv("mail.username", "NEOSCAN_MAIL_USERNAME")
	v.BindEnv("mail.password", "NEOSCAN_MAIL_PASSWORD")
	v.BindEnv("mail.from_email", "NEOSCAN_MAIL_FROM_EMAIL")

	// 服务器配置
	v.BindEnv("server.host", "NEOSCAN_SERVER_HOST")
	v.BindEnv("server.port", "NEOSCAN_SERVER_PORT")
	v.BindEnv("server.mode", "NEOSCAN_SERVER_MODE")

	// 应用配置
	v.BindEnv("app.environment", "NEOSCAN_APP_ENVIRONMENT")
	v.BindEnv("app.debug", "NEOSCAN_APP_DEBUG")
	v.BindEnv("app.rules.root_path", "NEOSCAN_APP_RULES_ROOT_PATH")
	v.BindEnv("app.rules.fingerprint.dir", "NEOSCAN_APP_RULES_FINGERPRINT_DIR")
	v.BindEnv("app.rules.poc.dir", "NEOSCAN_APP_RULES_POC_DIR")
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证服务器配置
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Server.Mode != "debug" && config.Server.Mode != "release" && config.Server.Mode != "test" {
		return fmt.Errorf("invalid server mode: %s", config.Server.Mode)
	}

	// 验证数据库配置
	if config.Database.MySQL.Host == "" {
		return fmt.Errorf("mysql host is required")
	}

	if config.Database.MySQL.Database == "" {
		return fmt.Errorf("mysql database name is required")
	}

	if config.Database.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}

	// 验证JWT配置 (更新为嵌套在security下的路径)
	if config.Security.JWT.Secret == "" {
		return fmt.Errorf("jwt secret is required")
	}

	if len(config.Security.JWT.Secret) < 32 {
		return fmt.Errorf("jwt secret must be at least 32 characters long")
	}

	// 验证日志配置
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
	if !contains(validLogLevels, config.Log.Level) {
		return fmt.Errorf("invalid log level: %s", config.Log.Level)
	}

	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, config.Log.Format) {
		return fmt.Errorf("invalid log format: %s", config.Log.Format)
	}

	validLogOutputs := []string{"stdout", "stderr", "file"}
	if !contains(validLogOutputs, config.Log.Output) {
		return fmt.Errorf("invalid log output: %s", config.Log.Output)
	}

	// 如果日志输出到文件，验证文件路径
	if config.Log.Output == "file" && config.Log.FilePath == "" {
		return fmt.Errorf("log file path is required when output is file")
	}

	// 验证会话配置
	validSessionStores := []string{"memory", "redis"}
	if !contains(validSessionStores, config.Session.Store) {
		return fmt.Errorf("invalid session store: %s", config.Session.Store)
	}

	validSameSiteValues := []string{"strict", "lax", "none"}
	if !contains(validSameSiteValues, config.Session.SameSite) {
		return fmt.Errorf("invalid session same_site value: %s", config.Session.SameSite)
	}

	if strings.TrimSpace(config.App.Rules.RootPath) == "" {
		return fmt.Errorf("app.rules.root_path is required")
	}
	if strings.TrimSpace(config.App.Rules.Fingerprint.Dir) == "" {
		return fmt.Errorf("app.rules.fingerprint.dir is required")
	}
	if strings.TrimSpace(config.App.Rules.POC.Dir) == "" {
		return fmt.Errorf("app.rules.poc.dir is required")
	}

	return nil
}

func applyDefaultRulesConfig(config *Config) {
	if config == nil {
		return
	}

	if strings.TrimSpace(config.App.Rules.RootPath) == "" {
		config.App.Rules.RootPath = "rules"
	}
	if strings.TrimSpace(config.App.Rules.Fingerprint.Dir) == "" {
		config.App.Rules.Fingerprint.Dir = "fingerprint"
	}
	if strings.TrimSpace(config.App.Rules.POC.Dir) == "" {
		config.App.Rules.POC.Dir = "poc"
	}
}

// contains 检查切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return GlobalConfig
}

// MustLoadConfig 加载配置，如果失败则panic
func MustLoadConfig(configPath, env string) *Config {
	config, err := LoadConfig(configPath, env)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	return config
}

// ReloadConfig 重新加载配置
func ReloadConfig() error {
	if GlobalConfig == nil {
		return fmt.Errorf("global config is not initialized")
	}

	// 重新加载配置
	config, err := LoadConfig("", "")
	if err != nil {
		return err
	}

	GlobalConfig = config
	return nil
}

// GetEnv 获取当前环境
func GetEnv() string {
	if GlobalConfig != nil {
		return GlobalConfig.App.Environment
	}
	return getEnvFromEnvironment()
}

// IsDevelopment 判断是否为开发环境
func IsDevelopment() bool {
	if GlobalConfig != nil {
		return GlobalConfig.App.IsDevelopment()
	}
	return GetEnv() == "development"
}

// IsProduction 判断是否为生产环境
func IsProduction() bool {
	if GlobalConfig != nil {
		return GlobalConfig.App.IsProduction()
	}
	return GetEnv() == "production"
}

// IsTest 判断是否为测试环境
func IsTest() bool {
	if GlobalConfig != nil {
		return GlobalConfig.App.IsTest()
	}
	return GetEnv() == "test"
}

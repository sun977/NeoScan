/**
 * Agent端配置管理
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端配置管理，负责加载和管理所有配置
 * @func: 占位符实现，待后续完善
 */
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config Agent配置
type Config struct {
	// 应用配置
	App *AppConfig `yaml:"app" mapstructure:"app"`
	
	// 服务器配置
	Server *ServerConfig `yaml:"server" mapstructure:"server"`
	
	// 日志配置
	Log *LogConfig `yaml:"log" mapstructure:"log"`
	
	// 数据库配置
	Database *DatabaseConfig `yaml:"database" mapstructure:"database"`
	
	// Master连接配置
	Master *MasterConfig `yaml:"master" mapstructure:"master"`
	
	// Agent配置
	Agent *AgentConfig `yaml:"agent" mapstructure:"agent"`
	
	// 中间件配置
	Middleware *MiddlewareConfig `yaml:"middleware" mapstructure:"middleware"`
	
	// 执行器配置
	Executor *ExecutorConfig `yaml:"executor" mapstructure:"executor"`
	
	// 监控配置
	Monitor *MonitorConfig `yaml:"monitor" mapstructure:"monitor"`
	
	// 安全配置
	Security *SecurityConfig `yaml:"security" mapstructure:"security"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string `yaml:"name" mapstructure:"name"`               // 应用名称
	Version     string `yaml:"version" mapstructure:"version"`         // 应用版本
	Environment string `yaml:"environment" mapstructure:"environment"` // 运行环境
	Debug       bool   `yaml:"debug" mapstructure:"debug"`             // 调试模式
	Timezone    string `yaml:"timezone" mapstructure:"timezone"`       // 时区
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host           string        `yaml:"host" mapstructure:"host"`                       // 监听地址
	Port           int           `yaml:"port" mapstructure:"port"`                       // 监听端口
	Mode           string        `yaml:"mode" mapstructure:"mode"`                       // 运行模式 (debug/release/test)
	APIVersion     string        `yaml:"api_version" mapstructure:"api_version"`         // API版本
	Prefix         string        `yaml:"prefix" mapstructure:"prefix"`                   // 路由前缀
	ReadTimeout    time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`       // 读取超时时间
	WriteTimeout   time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`     // 写入超时时间
	IdleTimeout    time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`       // 空闲超时时间
	MaxHeaderBytes int           `yaml:"max_header_bytes" mapstructure:"max_header_bytes"` // 最大头部字节数
	TLS            TLSConfig     `yaml:"tls" mapstructure:"tls"`                         // TLS配置
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`   // 是否启用TLS
	CertFile string `yaml:"cert_file" mapstructure:"cert_file"` // 证书文件路径
	KeyFile  string `yaml:"key_file" mapstructure:"key_file"`   // 私钥文件路径
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level" mapstructure:"level"`           // 日志级别 (debug/info/warn/error)
	Format     string `yaml:"format" mapstructure:"format"`         // 日志格式 (json/text)
	Output     string `yaml:"output" mapstructure:"output"`         // 日志输出 (stdout/file/both)
	FilePath   string `yaml:"file_path" mapstructure:"file_path"`   // 日志文件路径
	MaxSize    int    `yaml:"max_size" mapstructure:"max_size"`     // 最大文件大小（MB）
	MaxBackups int    `yaml:"max_backups" mapstructure:"max_backups"` // 最大备份数
	MaxAge     int    `yaml:"max_age" mapstructure:"max_age"`       // 最大保留天数
	Compress   bool   `yaml:"compress" mapstructure:"compress"`     // 是否压缩
	Caller     bool   `yaml:"caller" mapstructure:"caller"`         // 是否显示调用者信息
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type            string        `yaml:"type" mapstructure:"type"`                         // 数据库类型 (mysql/postgres/sqlite)
	Host            string        `yaml:"host" mapstructure:"host"`                         // 主机地址
	Port            int           `yaml:"port" mapstructure:"port"`                         // 端口
	Username        string        `yaml:"username" mapstructure:"username"`                 // 用户名
	Password        string        `yaml:"password" mapstructure:"password"`                 // 密码
	Database        string        `yaml:"database" mapstructure:"database"`                 // 数据库名
	Charset         string        `yaml:"charset" mapstructure:"charset"`                   // 字符集
	ParseTime       bool          `yaml:"parse_time" mapstructure:"parse_time"`             // 是否解析时间
	Loc             string        `yaml:"loc" mapstructure:"loc"`                           // 时区
	MaxIdleConns    int           `yaml:"max_idle_conns" mapstructure:"max_idle_conns"`     // 最大空闲连接数
	MaxOpenConns    int           `yaml:"max_open_conns" mapstructure:"max_open_conns"`     // 最大打开连接数
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"` // 连接最大生存时间
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" mapstructure:"conn_max_idle_time"` // 连接最大空闲时间
}

// MasterConfig Master连接配置
type MasterConfig struct {
	Address               string        `yaml:"address" mapstructure:"address"`                               // Master地址
	Port                  int           `yaml:"port" mapstructure:"port"`                                     // Master端口
	Protocol              string        `yaml:"protocol" mapstructure:"protocol"`                             // 连接协议 (http/https/grpc)
	TLS                   TLSConfig     `yaml:"tls" mapstructure:"tls"`                                       // TLS配置
	ConnectTimeout        time.Duration `yaml:"connect_timeout" mapstructure:"connect_timeout"`               // 连接超时时间
	RequestTimeout        time.Duration `yaml:"request_timeout" mapstructure:"request_timeout"`               // 请求超时时间
	HeartbeatInterval     time.Duration `yaml:"heartbeat_interval" mapstructure:"heartbeat_interval"`         // 心跳间隔
	ReconnectInterval     time.Duration `yaml:"reconnect_interval" mapstructure:"reconnect_interval"`         // 重连间隔
	MaxReconnectAttempts  int           `yaml:"max_reconnect_attempts" mapstructure:"max_reconnect_attempts"` // 最大重连次数
	SkipTLSVerify         bool          `yaml:"skip_tls_verify" mapstructure:"skip_tls_verify"`               // 跳过TLS验证
}

// AgentConfig Agent配置
type AgentConfig struct {
	ID                 string        `yaml:"id" mapstructure:"id"`                                   // Agent ID
	Name               string        `yaml:"name" mapstructure:"name"`                               // Agent名称
	Version            string        `yaml:"version" mapstructure:"version"`                         // Agent版本
	Type               string        `yaml:"type" mapstructure:"type"`                               // Agent类型
	Tags               []string      `yaml:"tags" mapstructure:"tags"`                               // Agent标签
	WorkDir            string        `yaml:"work_dir" mapstructure:"work_dir"`                       // 工作目录
	TempDir            string        `yaml:"temp_dir" mapstructure:"temp_dir"`                       // 临时目录
	LogDir             string        `yaml:"log_dir" mapstructure:"log_dir"`                         // 日志目录
	DataDir            string        `yaml:"data_dir" mapstructure:"data_dir"`                       // 数据目录
	MaxConcurrentTasks int           `yaml:"max_concurrent_tasks" mapstructure:"max_concurrent_tasks"` // 最大并发任务数
	TaskTimeout        time.Duration `yaml:"task_timeout" mapstructure:"task_timeout"`               // 任务超时时间
	AutoRegister       bool          `yaml:"auto_register" mapstructure:"auto_register"`             // 是否自动注册
	Resources          ResourceConfig `yaml:"resources" mapstructure:"resources"`                    // 资源配置
}

// ResourceConfig 资源配置
type ResourceConfig struct {
	CPU    string `yaml:"cpu" mapstructure:"cpu"`       // CPU限制
	Memory string `yaml:"memory" mapstructure:"memory"` // 内存限制
	Disk   string `yaml:"disk" mapstructure:"disk"`     // 磁盘限制
}

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	// 认证中间件配置
	Auth *AuthConfig `yaml:"auth" json:"auth"`
	
	// 日志中间件配置
	Logging *LoggingConfig `yaml:"logging" json:"logging"`
	
	// CORS中间件配置
	CORS *CORSConfig `yaml:"cors" json:"cors"`
	
	// 限流中间件配置
	RateLimit *RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
}

// AuthConfig 认证中间件配置
type AuthConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	AuthMethod   string   `yaml:"auth_method" json:"auth_method"`
	WhitelistIPs []string `yaml:"whitelist_ips" json:"whitelist_ips"`
	SkipPaths    []string `yaml:"skip_paths" json:"skip_paths"`
}

// LoggingConfig 日志中间件配置
type LoggingConfig struct {
	EnableRequestLog      bool          `yaml:"enable_request_log" json:"enable_request_log"`
	EnableResponseLog     bool          `yaml:"enable_response_log" json:"enable_response_log"`
	LogRequestBody        bool          `yaml:"log_request_body" json:"log_request_body"`
	LogResponseBody       bool          `yaml:"log_response_body" json:"log_response_body"`
	LogHeaders            bool          `yaml:"log_headers" json:"log_headers"`
	SlowRequestThreshold  time.Duration `yaml:"slow_request_threshold" json:"slow_request_threshold"`
	MaxBodySize           int64         `yaml:"max_body_size" json:"max_body_size"`
	SkipPaths             []string      `yaml:"skip_paths" json:"skip_paths"`
}

// CORSConfig CORS中间件配置
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	AllowAllOrigins  bool     `yaml:"allow_all_origins" json:"allow_all_origins"`
	AllowOrigins     []string `yaml:"allow_origins" json:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers" json:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers" json:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
	MaxAge           int      `yaml:"max_age" json:"max_age"`
}

// RateLimitConfig 限流中间件配置
type RateLimitConfig struct {
	Enabled           bool   `yaml:"enabled" json:"enabled"`
	RequestsPerSecond int    `yaml:"requests_per_second" json:"requests_per_second"`
	BurstSize         int    `yaml:"burst_size" json:"burst_size"`
	Strategy          string `yaml:"strategy" json:"strategy"`
	KeyGenerator      string `yaml:"key_generator" json:"key_generator"`
	SkipPaths         []string `yaml:"skip_paths" json:"skip_paths"`
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	// 执行器类型
	Types []string `yaml:"types" json:"types"`
	
	// 默认执行器
	Default string `yaml:"default" json:"default"`
	
	// 执行器配置
	Configs map[string]interface{} `yaml:"configs" json:"configs"`
	
	// 资源限制
	ResourceLimits *ResourceLimitsConfig `yaml:"resource_limits" json:"resource_limits"`
	
	// 工具路径
	ToolPaths map[string]string `yaml:"tool_paths" json:"tool_paths"`
}

// ResourceLimitsConfig 资源限制配置
type ResourceLimitsConfig struct {
	// CPU限制（核数）
	CPU float64 `yaml:"cpu" json:"cpu"`
	
	// 内存限制（MB）
	Memory int64 `yaml:"memory" json:"memory"`
	
	// 磁盘限制（MB）
	Disk int64 `yaml:"disk" json:"disk"`
	
	// 网络带宽限制（Mbps）
	Network int64 `yaml:"network" json:"network"`
	
	// 进程数限制
	Processes int `yaml:"processes" json:"processes"`
	
	// 文件描述符限制
	FileDescriptors int `yaml:"file_descriptors" json:"file_descriptors"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	// 是否启用监控
	Enabled bool `yaml:"enabled" json:"enabled"`
	
	// 监控间隔
	Interval time.Duration `yaml:"interval" json:"interval"`
	
	// 指标收集器
	Collectors []string `yaml:"collectors" json:"collectors"`
	
	// 告警规则
	AlertRules []AlertRuleConfig `yaml:"alert_rules" json:"alert_rules"`
	
	// 数据保留时间
	RetentionPeriod time.Duration `yaml:"retention_period" json:"retention_period"`
}

// AlertRuleConfig 告警规则配置
type AlertRuleConfig struct {
	// 规则名称
	Name string `yaml:"name" json:"name"`
	
	// 指标名称
	Metric string `yaml:"metric" json:"metric"`
	
	// 条件
	Condition string `yaml:"condition" json:"condition"`
	
	// 阈值
	Threshold float64 `yaml:"threshold" json:"threshold"`
	
	// 持续时间
	Duration time.Duration `yaml:"duration" json:"duration"`
	
	// 告警级别
	Severity string `yaml:"severity" json:"severity"`
	
	// 告警消息
	Message string `yaml:"message" json:"message"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	// API密钥
	APIKey string `yaml:"api_key" json:"api_key" env:"AGENT_API_KEY"`
	
	// JWT密钥
	JWTSecret string `yaml:"jwt_secret" json:"jwt_secret" env:"JWT_SECRET"`
	
	// JWT过期时间
	JWTExpiration time.Duration `yaml:"jwt_expiration" json:"jwt_expiration"`
	
	// 加密密钥
	EncryptionKey string `yaml:"encryption_key" json:"encryption_key" env:"ENCRYPTION_KEY"`
	
	// IP白名单
	IPWhitelist []string `yaml:"ip_whitelist" json:"ip_whitelist"`
	
	// 是否启用IP白名单
	EnableIPWhitelist bool `yaml:"enable_ip_whitelist" json:"enable_ip_whitelist"`
}

// LoadConfig 加载配置
func LoadConfig(configPath ...string) (*Config, error) {
	// 使用新的配置加载器
	var path string
	if len(configPath) > 0 && configPath[0] != "" {
		path = configPath[0]
	}
	
	loader := NewConfigLoader(path, "NEOAGENT")
	config, err := loader.LoadConfig()
	if err != nil {
		return nil, err
	}
	
	// 设置全局配置
	globalConfig = config
	return config, nil
}

// loadConfigFile 从配置文件加载
// loadConfigFileAuto 自动查找并加载配置文件
func loadConfigFileAuto(config *Config) error {
	// 查找配置文件
	configPaths := []string{
		"config.yaml",
		"config.yml",
		"configs/config.yaml",
		"configs/config.yml",
		"/etc/neoagent/config.yaml",
		"/etc/neoagent/config.yml",
	}
	
	// 从环境变量获取配置文件路径
	if configPath := os.Getenv("AGENT_CONFIG_PATH"); configPath != "" {
		configPaths = append([]string{configPath}, configPaths...)
	}
	
	var configFile string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configFile = path
			break
		}
	}
	
	if configFile == "" {
		// 如果没有找到配置文件，使用默认配置
		return nil
	}
	
	// 使用统一的loadConfigFile函数
	return loadConfigFile(config, configFile)
}

// loadFromEnv 从环境变量加载
func loadFromEnv(config *Config) error {
	// TODO: 实现从环境变量加载配置的逻辑
	// 使用反射或手动设置环境变量
	
	// 服务器配置
	if config.Server == nil {
		config.Server = &ServerConfig{}
	}
	
	if port := os.Getenv("AGENT_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	
	if host := os.Getenv("AGENT_HOST"); host != "" {
		config.Server.Host = host
	}
	// 从环境变量覆盖配置
	if debug := os.Getenv("NEOAGENT_DEBUG"); debug != "" {
		config.App.Debug = strings.ToLower(debug) == "true"
	}
	
	// 日志配置
	if config.Log == nil {
		config.Log = &LogConfig{}
	}
	
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Log.Level = level
	}
	
	if filePath := os.Getenv("LOG_FILE_PATH"); filePath != "" {
		config.Log.FilePath = filePath
	}
	
	// 数据库配置
	if config.Database == nil {
		config.Database = &DatabaseConfig{}
	}
	
	if dbType := os.Getenv("DB_TYPE"); dbType != "" {
		config.Database.Type = dbType
	}
	// 数据库配置
	if host := os.Getenv("NEOAGENT_DATABASE_HOST"); host != "" {
		config.Database.Host = host
	}
	
	if port := os.Getenv("NEOAGENT_DATABASE_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Database.Port = p
		}
	}
	
	if username := os.Getenv("NEOAGENT_DATABASE_USERNAME"); username != "" {
		config.Database.Username = username
	}
	
	if password := os.Getenv("NEOAGENT_DATABASE_PASSWORD"); password != "" {
		config.Database.Password = password
	}
	
	if database := os.Getenv("NEOAGENT_DATABASE_NAME"); database != "" {
		config.Database.Database = database
	}
	
	if host := os.Getenv("DB_HOST"); host != "" {
		config.Database.Host = host
	}
	
	if port := os.Getenv("DB_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Database.Port = p
		}
	}
	
	if username := os.Getenv("DB_USERNAME"); username != "" {
		config.Database.Username = username
	}
	
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		config.Database.Password = password
	}
	
	if database := os.Getenv("DB_DATABASE"); database != "" {
		config.Database.Database = database
	}
	
	// Master配置
	if config.Master == nil {
		config.Master = &MasterConfig{}
	}
	
	if address := os.Getenv("MASTER_ADDRESS"); address != "" {
		config.Master.Address = address
	}
	
	if port := os.Getenv("MASTER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Master.Port = p
		}
	}
	
	if enableTLS := os.Getenv("MASTER_ENABLE_TLS"); enableTLS != "" {
		config.Master.TLS.Enabled = strings.ToLower(enableTLS) == "true"
	}
	
	if tlsCertPath := os.Getenv("MASTER_TLS_CERT_PATH"); tlsCertPath != "" {
		config.Master.TLS.CertFile = tlsCertPath
	}
	
	if tlsKeyPath := os.Getenv("MASTER_TLS_KEY_PATH"); tlsKeyPath != "" {
		config.Master.TLS.KeyFile = tlsKeyPath
	}
	
	// Agent配置
	if config.Agent == nil {
		config.Agent = &AgentConfig{}
	}
	
	if id := os.Getenv("AGENT_ID"); id != "" {
		config.Agent.ID = id
	}
	
	if name := os.Getenv("AGENT_NAME"); name != "" {
		config.Agent.Name = name
	}
	
	if workDir := os.Getenv("AGENT_WORK_DIR"); workDir != "" {
		config.Agent.WorkDir = workDir
	}
	
	if tempDir := os.Getenv("AGENT_TEMP_DIR"); tempDir != "" {
		config.Agent.TempDir = tempDir
	}
	
	if logDir := os.Getenv("AGENT_LOG_DIR"); logDir != "" {
		config.Agent.LogDir = logDir
	}
	
	if dataDir := os.Getenv("AGENT_DATA_DIR"); dataDir != "" {
		config.Agent.DataDir = dataDir
	}
	
	// 安全配置
	if config.Security == nil {
		config.Security = &SecurityConfig{}
	}
	
	if apiKey := os.Getenv("AGENT_API_KEY"); apiKey != "" {
		config.Security.APIKey = apiKey
	}
	
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.Security.JWTSecret = jwtSecret
	}
	
	if encryptionKey := os.Getenv("ENCRYPTION_KEY"); encryptionKey != "" {
		config.Security.EncryptionKey = encryptionKey
	}
	
	return nil
}

// setDefaults 设置默认值
func setDefaults(config *Config) {
	// 服务器默认配置
	if config.Server == nil {
		config.Server = &ServerConfig{}
	}
	
	if config.Server.Port == 0 {
		config.Server.Port = 8081
	}
	
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	
	if config.Server.APIVersion == "" {
		config.Server.APIVersion = "v1"
	}
	
	if config.Server.Prefix == "" {
		config.Server.Prefix = "/api"
	}
	
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30
	}
	
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30
	}
	
	if config.Server.IdleTimeout == 0 {
		config.Server.IdleTimeout = 60
	}
	
	if config.Server.MaxHeaderBytes == 0 {
		config.Server.MaxHeaderBytes = 1 << 20 // 1MB
	}
	
	// 日志默认配置
	if config.Log == nil {
		config.Log = &LogConfig{}
	}
	
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}
	
	if config.Log.Format == "" {
		config.Log.Format = "json"
	}
	
	if config.Log.Output == "" {
		config.Log.Output = "file"
	}
	
	if config.Log.FilePath == "" {
		config.Log.FilePath = "logs/agent.log"
	}
	
	if config.Log.MaxSize == 0 {
		config.Log.MaxSize = 100
	}
	
	if config.Log.MaxBackups == 0 {
		config.Log.MaxBackups = 10
	}
	
	if config.Log.MaxAge == 0 {
		config.Log.MaxAge = 30
	}
	
	// Master默认配置
	if config.Master == nil {
		config.Master = &MasterConfig{}
	}
	
	if config.Master.Address == "" {
		config.Master.Address = "localhost"
	}
	
	if config.Master.Port == 0 {
		config.Master.Port = 8080
	}
	
	if config.Master.Protocol == "" {
		config.Master.Protocol = "http"
	}
	
	if config.Master.ConnectTimeout == 0 {
		config.Master.ConnectTimeout = 10 * time.Second
	}
	
	if config.Master.RequestTimeout == 0 {
		config.Master.RequestTimeout = 30 * time.Second
	}
	
	if config.Master.HeartbeatInterval == 0 {
		config.Master.HeartbeatInterval = 30 * time.Second
	}
	
	if config.Master.ReconnectInterval == 0 {
		config.Master.ReconnectInterval = 5 * time.Second
	}
	
	if config.Master.MaxReconnectAttempts == 0 {
		config.Master.MaxReconnectAttempts = 10
	}
	
	// Agent默认配置
	if config.Agent == nil {
		config.Agent = &AgentConfig{}
	}
	
	if config.Agent.ID == "" {
		config.Agent.ID = generateAgentID()
	}
	
	if config.Agent.Name == "" {
		config.Agent.Name = "neoagent"
	}
	
	if config.Agent.Version == "" {
		config.Agent.Version = "1.0.0"
	}
	
	if config.Agent.Type == "" {
		config.Agent.Type = "scanner"
	}
	
	if config.Agent.WorkDir == "" {
		config.Agent.WorkDir = "./work"
	}
	
	if config.Agent.TempDir == "" {
		config.Agent.TempDir = "./temp"
	}
	
	if config.Agent.LogDir == "" {
		config.Agent.LogDir = "./logs"
	}
	
	if config.Agent.DataDir == "" {
		config.Agent.DataDir = "./data"
	}
	
	if config.Agent.MaxConcurrentTasks == 0 {
		config.Agent.MaxConcurrentTasks = 10
	}
	
	if config.Agent.TaskTimeout == 0 {
		config.Agent.TaskTimeout = 30 * time.Minute
	}
	
	// 执行器默认配置
	if config.Executor == nil {
		config.Executor = &ExecutorConfig{}
	}
	
	if len(config.Executor.Types) == 0 {
		config.Executor.Types = []string{"system", "nmap", "nuclei", "masscan"}
	}
	
	if config.Executor.Default == "" {
		config.Executor.Default = "system"
	}
	
	// 监控默认配置
	if config.Monitor == nil {
		config.Monitor = &MonitorConfig{}
	}
	
	if config.Monitor.Interval == 0 {
		config.Monitor.Interval = 30 * time.Second
	}
	
	if config.Monitor.RetentionPeriod == 0 {
		config.Monitor.RetentionPeriod = 24 * time.Hour
	}
	
	// 安全默认配置
	if config.Security == nil {
		config.Security = &SecurityConfig{}
	}
	
	if config.Security.JWTExpiration == 0 {
		config.Security.JWTExpiration = 24 * time.Hour
	}
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// TODO: 实现配置验证逻辑
	// 1. 验证必需字段
	// 2. 验证字段格式
	// 3. 验证字段范围
	
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	
	if config.Master.Port <= 0 || config.Master.Port > 65535 {
		return fmt.Errorf("invalid master port: %d", config.Master.Port)
	}
	
	if config.Agent.MaxConcurrentTasks <= 0 {
		return fmt.Errorf("invalid max concurrent tasks: %d", config.Agent.MaxConcurrentTasks)
	}
	
	// 验证目录路径
	dirs := []string{
		config.Agent.WorkDir,
		config.Agent.TempDir,
		config.Agent.LogDir,
		config.Agent.DataDir,
	}
	
	for _, dir := range dirs {
		if err := ensureDir(dir); err != nil {
			return fmt.Errorf("failed to ensure directory %s: %w", dir, err)
		}
	}
	
	return nil
}

// loadConfigFile 从配置文件加载
func loadConfigFile(cfg *Config, configPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configPath)
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 根据文件扩展名选择解析方式
	ext := filepath.Ext(configPath)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	return nil
}

// ensureDirectories 确保所有必需的目录存在
func ensureDirectories(cfg *Config) error {
	dirs := []string{
		cfg.Agent.WorkDir,
		cfg.Agent.TempDir,
		cfg.Agent.LogDir,
		cfg.Agent.DataDir,
	}

	for _, dir := range dirs {
		if dir != "" {
			if err := ensureDir(dir); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	return nil
}

// ensureDir 确保目录存在
func ensureDir(dir string) error {
	if dir == "" {
		return nil
	}
	
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	
	return os.MkdirAll(absDir, 0755)
}

// generateAgentID 生成Agent ID
func generateAgentID() string {
	// TODO: 实现更好的ID生成逻辑
	// 可以使用UUID、主机名+时间戳等
	
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	
	return fmt.Sprintf("agent-%s-%d", hostname, time.Now().Unix())
}

// GetConfig 获取配置（单例模式）
var globalConfig *Config

func GetConfig() *Config {
	if globalConfig == nil {
		var err error
		globalConfig, err = LoadConfig("")
		if err != nil {
			panic(fmt.Sprintf("Failed to load config: %v", err))
		}
	}
	return globalConfig
}

// ReloadConfig 重新加载配置
func ReloadConfig() error {
	newConfig, err := LoadConfig("")
	if err != nil {
		return err
	}
	
	globalConfig = newConfig
	return nil
}
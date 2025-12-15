package config

import (
	"fmt"
	"time"
)

// Config 应用配置结构体 [这里的字段和配置文件中一级字段保持一致，否则会没有值]
type Config struct {
	Server   ServerConfig   `yaml:"server" mapstructure:"server"`     // 服务器配置
	Database DatabaseConfig `yaml:"database" mapstructure:"database"` // 数据库配置
	// JWT        JWTConfig        `yaml:"jwt" mapstructure:"jwt"`                 // JWT配置 合并到 安全配置中
	Log        LogConfig        `yaml:"log" mapstructure:"log"`                 // 日志配置
	Security   SecurityConfig   `yaml:"security" mapstructure:"security"`       // 安全配置
	Session    SessionConfig    `yaml:"session" mapstructure:"session"`         // 会话配置
	WebSocket  WebSocketConfig  `yaml:"websocket" mapstructure:"websocket"`     // WebSocket配置
	Upload     UploadConfig     `yaml:"upload" mapstructure:"upload"`           // 文件上传配置
	Mail       MailConfig       `yaml:"mail" mapstructure:"mail"`               // 邮件配置
	Monitor    MonitorConfig    `yaml:"monitor" mapstructure:"monitor"`         // 监控配置
	App        AppConfig        `yaml:"app" mapstructure:"app"`                 // 应用配置
	ThirdParty ThirdPartyConfig `yaml:"third_party" mapstructure:"third_party"` // 第三方服务配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host           string        `yaml:"host" mapstructure:"host"`                         // 服务器主机地址
	Port           int           `yaml:"port" mapstructure:"port"`                         // 服务器端口
	Mode           string        `yaml:"mode" mapstructure:"mode"`                         // 运行模式: debug, release, test
	ReadTimeout    time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`         // 读取超时时间
	WriteTimeout   time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`       // 写入超时时间
	IdleTimeout    time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`         // 空闲超时时间
	MaxHeaderBytes int           `yaml:"max_header_bytes" mapstructure:"max_header_bytes"` // 最大请求头字节数
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL MySQLConfig `yaml:"mysql" mapstructure:"mysql"` // MySQL配置
	Redis RedisConfig `yaml:"redis" mapstructure:"redis"` // Redis配置
}

// MySQLConfig MySQL数据库配置
type MySQLConfig struct {
	Host            string        `yaml:"host" mapstructure:"host"`                             // 数据库主机
	Port            int           `yaml:"port" mapstructure:"port"`                             // 数据库端口
	Username        string        `yaml:"username" mapstructure:"username"`                     // 用户名
	Password        string        `yaml:"password" mapstructure:"password"`                     // 密码
	Database        string        `yaml:"database" mapstructure:"database"`                     // 数据库名
	Charset         string        `yaml:"charset" mapstructure:"charset"`                       // 字符集
	ParseTime       bool          `yaml:"parse_time" mapstructure:"parse_time"`                 // 是否解析时间
	Loc             string        `yaml:"loc" mapstructure:"loc"`                               // 时区
	MaxIdleConns    int           `yaml:"max_idle_conns" mapstructure:"max_idle_conns"`         // 最大空闲连接数
	MaxOpenConns    int           `yaml:"max_open_conns" mapstructure:"max_open_conns"`         // 最大打开连接数
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`   // 连接最大生存时间
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" mapstructure:"conn_max_idle_time"` // 连接最大空闲时间
	LogLevel        string        `yaml:"log_level" mapstructure:"log_level"`                   // 日志级别
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host         string        `yaml:"host" mapstructure:"host"`                     // Redis主机
	Port         int           `yaml:"port" mapstructure:"port"`                     // Redis端口
	Password     string        `yaml:"password" mapstructure:"password"`             // Redis密码
	Database     int           `yaml:"database" mapstructure:"database"`             // Redis数据库索引
	PoolSize     int           `yaml:"pool_size" mapstructure:"pool_size"`           // 连接池大小
	MinIdleConns int           `yaml:"min_idle_conns" mapstructure:"min_idle_conns"` // 最小空闲连接数
	DialTimeout  time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout"`     // 连接超时
	ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`     // 读取超时
	WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`   // 写入超时
	PoolTimeout  time.Duration `yaml:"pool_timeout" mapstructure:"pool_timeout"`     // 连接池超时
	IdleTimeout  time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`     // 空闲超时
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level" mapstructure:"level"`             // 日志级别
	Format     string `yaml:"format" mapstructure:"format"`           // 日志格式: json, text
	Output     string `yaml:"output" mapstructure:"output"`           // 输出方式: stdout, stderr, file
	FilePath   string `yaml:"file_path" mapstructure:"file_path"`     // 日志文件路径
	MaxSize    int    `yaml:"max_size" mapstructure:"max_size"`       // 单个日志文件最大大小(MB)
	MaxBackups int    `yaml:"max_backups" mapstructure:"max_backups"` // 保留的日志文件数量
	MaxAge     int    `yaml:"max_age" mapstructure:"max_age"`         // 日志文件保留天数
	Compress   bool   `yaml:"compress" mapstructure:"compress"`       // 是否压缩日志文件
	Caller     bool   `yaml:"caller" mapstructure:"caller"`           // 是否显示调用者信息
	StackTrace bool   `yaml:"stack_trace" mapstructure:"stack_trace"` // 是否显示堆栈跟踪
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	JWT       JWTConfig       `yaml:"jwt" mapstructure:"jwt"`               // JWT配置
	Auth      AuthConfig      `yaml:"auth" mapstructure:"auth"`             // 认证配置
	Logging   LoggingConfig   `yaml:"logging" mapstructure:"logging"`       // 日志中间件配置
	CORS      CORSConfig      `yaml:"cors" mapstructure:"cors"`             // CORS配置
	RateLimit RateLimitConfig `yaml:"rate_limit" mapstructure:"rate_limit"` // 限流配置
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret             string        `yaml:"secret" mapstructure:"secret"`                             // JWT密钥
	Issuer             string        `yaml:"issuer" mapstructure:"issuer"`                             // 签发者
	AccessTokenExpire  time.Duration `yaml:"access_token_expire" mapstructure:"access_token_expire"`   // 访问令牌过期时间
	RefreshTokenExpire time.Duration `yaml:"refresh_token_expire" mapstructure:"refresh_token_expire"` // 刷新令牌过期时间
	Algorithm          string        `yaml:"algorithm" mapstructure:"algorithm"`                       // 签名算法
}

// AuthConfig 认证中间件配置
type AuthConfig struct {
	AuthMethod        string   `yaml:"auth_method" mapstructure:"auth_method"`                 // 认证方式
	APIKey            string   `yaml:"api_key" mapstructure:"api_key"`                         // API密钥
	APIKeyHeader      string   `yaml:"api_key_header" mapstructure:"api_key_header"`           // API密钥请求头
	WhitelistIPs      []string `yaml:"whitelist_ips" mapstructure:"whitelist_ips"`             // IP白名单
	EnableIPWhitelist bool     `yaml:"enable_ip_whitelist" mapstructure:"enable_ip_whitelist"` // 是否启用IP白名单
	SkipPaths         []string `yaml:"skip_paths" mapstructure:"skip_paths"`                   // 跳过认证的路径
}

// LoggingConfig 日志中间件配置
type LoggingConfig struct {
	EnableRequestLog     bool          `yaml:"enable_request_log" mapstructure:"enable_request_log"`         // 是否启用请求日志
	EnableResponseLog    bool          `yaml:"enable_response_log" mapstructure:"enable_response_log"`       // 是否启用响应日志
	LogRequestBody       bool          `yaml:"log_request_body" mapstructure:"log_request_body"`             // 是否记录请求体
	LogResponseBody      bool          `yaml:"log_response_body" mapstructure:"log_response_body"`           // 是否记录响应体
	SlowRequestThreshold time.Duration `yaml:"slow_request_threshold" mapstructure:"slow_request_threshold"` // 慢请求阈值
	SkipPaths            []string      `yaml:"skip_paths" mapstructure:"skip_paths"`                         // 跳过日志记录的路径
	MaxRequestBodySize   int           `yaml:"max_request_body_size" mapstructure:"max_request_body_size"`   // 最大请求体记录大小
	MaxResponseBodySize  int           `yaml:"max_response_body_size" mapstructure:"max_response_body_size"` // 最大响应体记录大小
}

// CORSConfig CORS配置
type CORSConfig struct {
	Enabled          bool          `yaml:"enabled" mapstructure:"enabled"`                     // 是否启用CORS
	AllowAllOrigins  bool          `yaml:"allow_all_origins" mapstructure:"allow_all_origins"` // 是否允许所有源
	AllowOrigins     []string      `yaml:"allow_origins" mapstructure:"allow_origins"`         // 允许的源
	AllowMethods     []string      `yaml:"allow_methods" mapstructure:"allow_methods"`         // 允许的方法
	AllowHeaders     []string      `yaml:"allow_headers" mapstructure:"allow_headers"`         // 允许的请求头
	ExposeHeaders    []string      `yaml:"expose_headers" mapstructure:"expose_headers"`       // 暴露的响应头
	AllowCredentials bool          `yaml:"allow_credentials" mapstructure:"allow_credentials"` // 是否允许凭证
	MaxAge           time.Duration `yaml:"max_age" mapstructure:"max_age"`                     // 预检请求缓存时间
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool     `yaml:"enabled" mapstructure:"enabled"`                         // 是否启用限流
	RequestsPerSecond int      `yaml:"requests_per_second" mapstructure:"requests_per_second"` // 每秒请求数限制
	BurstSize         int      `yaml:"burst_size" mapstructure:"burst_size"`                   // 突发请求数
	Strategy          string   `yaml:"strategy" mapstructure:"strategy"`                       // 限流策略
	WindowSize        string   `yaml:"window_size" mapstructure:"window_size"`                 // 窗口大小
	StatusCode        int      `yaml:"status_code" mapstructure:"status_code"`                 // 限流时返回的状态码
	Message           string   `yaml:"message" mapstructure:"message"`                         // 限流时返回的消息
	SkipPaths         []string `yaml:"skip_paths" mapstructure:"skip_paths"`                   // 跳过限流的路径
	SkipIPs           []string `yaml:"skip_ips" mapstructure:"skip_ips"`                       // 跳过限流的IP
}

// SessionConfig 会话配置
type SessionConfig struct {
	Store    string `yaml:"store" mapstructure:"store"`         // 存储方式: memory, redis
	Key      string `yaml:"key" mapstructure:"key"`             // 会话键名
	MaxAge   int    `yaml:"max_age" mapstructure:"max_age"`     // 会话最大存活时间(秒)
	Secure   bool   `yaml:"secure" mapstructure:"secure"`       // 是否仅HTTPS
	HTTPOnly bool   `yaml:"http_only" mapstructure:"http_only"` // 是否仅HTTP访问
	SameSite string `yaml:"same_site" mapstructure:"same_site"` // SameSite策略: strict, lax, none
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	Enabled           bool          `yaml:"enabled" mapstructure:"enabled"`                       // 是否启用WebSocket
	Path              string        `yaml:"path" mapstructure:"path"`                             // WebSocket路径
	ReadBufferSize    int           `yaml:"read_buffer_size" mapstructure:"read_buffer_size"`     // 读缓冲区大小
	WriteBufferSize   int           `yaml:"write_buffer_size" mapstructure:"write_buffer_size"`   // 写缓冲区大小
	CheckOrigin       bool          `yaml:"check_origin" mapstructure:"check_origin"`             // 是否检查来源
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" mapstructure:"heartbeat_interval"` // 心跳间隔
	MaxConnections    int           `yaml:"max_connections" mapstructure:"max_connections"`       // 最大连接数
}

// UploadConfig 文件上传配置
type UploadConfig struct {
	MaxSize      int64    `yaml:"max_size" mapstructure:"max_size"`           // 最大文件大小(字节)
	AllowedTypes []string `yaml:"allowed_types" mapstructure:"allowed_types"` // 允许的文件类型
	UploadPath   string   `yaml:"upload_path" mapstructure:"upload_path"`     // 上传路径
	URLPrefix    string   `yaml:"url_prefix" mapstructure:"url_prefix"`       // URL前缀
}

// MailConfig 邮件配置
type MailConfig struct {
	Enabled   bool   `yaml:"enabled" mapstructure:"enabled"`       // 是否启用邮件功能
	SMTPHost  string `yaml:"smtp_host" mapstructure:"smtp_host"`   // SMTP服务器地址
	SMTPPort  int    `yaml:"smtp_port" mapstructure:"smtp_port"`   // SMTP服务器端口
	Username  string `yaml:"username" mapstructure:"username"`     // SMTP用户名
	Password  string `yaml:"password" mapstructure:"password"`     // SMTP密码
	FromEmail string `yaml:"from_email" mapstructure:"from_email"` // 发件人邮箱
	FromName  string `yaml:"from_name" mapstructure:"from_name"`   // 发件人名称
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Metrics MetricsConfig `yaml:"metrics" mapstructure:"metrics"` // 指标监控配置
	Health  HealthConfig  `yaml:"health" mapstructure:"health"`   // 健康检查配置
	Pprof   PprofConfig   `yaml:"pprof" mapstructure:"pprof"`     // 性能分析配置
}

// MetricsConfig 指标监控配置
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"` // 是否启用指标监控
	Path    string `yaml:"path" mapstructure:"path"`       // 指标接口路径
}

// HealthConfig 健康检查配置
type HealthConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"` // 是否启用健康检查
	Path    string `yaml:"path" mapstructure:"path"`       // 健康检查接口路径
}

// PprofConfig 性能分析配置
type PprofConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"` // 是否启用性能分析
	Path    string `yaml:"path" mapstructure:"path"`       // 性能分析接口路径
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string         `yaml:"name" mapstructure:"name"`               // 应用名称
	Version     string         `yaml:"version" mapstructure:"version"`         // 应用版本
	Environment string         `yaml:"environment" mapstructure:"environment"` // 运行环境
	Debug       bool           `yaml:"debug" mapstructure:"debug"`             // 是否调试模式
	Timezone    string         `yaml:"timezone" mapstructure:"timezone"`       // 时区
	Language    string         `yaml:"language" mapstructure:"language"`       // 语言
	Master      MasterConfig   `yaml:"master" mapstructure:"master"`           // Master配置
	Features    FeaturesConfig `yaml:"features" mapstructure:"features"`       // 功能开关配置
}

// MasterConfig Master节点配置
type MasterConfig struct {
	Task TaskConfig `yaml:"task" mapstructure:"task"` // 任务配置
}

// TaskConfig 任务配置
type TaskConfig struct {
	ChunkSize     int `yaml:"chunk_size" mapstructure:"chunk_size"`         // 每个任务分块大小
	Timeout       int `yaml:"timeout" mapstructure:"timeout"`               // 任务超时时间(秒)
	MaxRetries    int `yaml:"max_retries" mapstructure:"max_retries"`       // 任务最大重试次数
	RetryInterval int `yaml:"retry_interval" mapstructure:"retry_interval"` // 任务重试间隔(秒)
}

// FeaturesConfig 功能开关配置
type FeaturesConfig struct {
	UserRegistration  bool `yaml:"user_registration" mapstructure:"user_registration"`   // 用户注册功能
	EmailVerification bool `yaml:"email_verification" mapstructure:"email_verification"` // 邮箱验证功能
	PasswordReset     bool `yaml:"password_reset" mapstructure:"password_reset"`         // 密码重置功能
	TwoFactorAuth     bool `yaml:"two_factor_auth" mapstructure:"two_factor_auth"`       // 双因子认证功能
	AuditLog          bool `yaml:"audit_log" mapstructure:"audit_log"`                   // 审计日志功能
	APIDocumentation  bool `yaml:"api_documentation" mapstructure:"api_documentation"`   // API文档功能
}

// ThirdPartyConfig 第三方服务配置
type ThirdPartyConfig struct {
	Placeholder bool `yaml:"placeholder" mapstructure:"placeholder"` // 占位符，可根据需要添加具体的第三方服务配置
}

// GetAddress 获取服务器完整地址
func (s *ServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// IsDevelopment 判断是否为开发环境
func (a *AppConfig) IsDevelopment() bool {
	return a.Environment == "development"
}

// IsProduction 判断是否为生产环境
func (a *AppConfig) IsProduction() bool {
	return a.Environment == "production"
}

// IsTest 判断是否为测试环境
func (a *AppConfig) IsTest() bool {
	return a.Environment == "test"
}

// GetMySQLDSN 获取MySQL数据源名称
func (m *MySQLConfig) GetMySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		m.Username, m.Password, m.Host, m.Port, m.Database, m.Charset, m.ParseTime, m.Loc)
}

// GetRedisAddress 获取Redis地址
func (r *RedisConfig) GetRedisAddress() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

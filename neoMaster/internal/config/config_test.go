package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoadConfig 测试配置加载功能
func TestLoadConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configContent := `
server:
  host: "localhost"
  port: 8080
  mode: "test"
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  max_header_bytes: 1048576

database:
  mysql:
    host: "localhost"
    port: 3306
    username: "test_user"
    password: "test_password"
    database: "test_db"
    charset: "utf8mb4"
    parse_time: true
    loc: "Local"
    max_idle_conns: 10
    max_open_conns: 100
    conn_max_lifetime: 3600s
    conn_max_idle_time: 1800s
    log_level: "info"
  redis:
    host: "localhost"
    port: 6379
    password: ""
    database: 0
    pool_size: 10
    min_idle_conns: 5
    dial_timeout: 5s
    read_timeout: 3s
    write_timeout: 3s
    pool_timeout: 4s
    idle_timeout: 300s

jwt:
  secret: "test_jwt_secret_key_at_least_32_chars"
  issuer: "neoscan-test"
  access_token_expire: 24h
  refresh_token_expire: 168h
  algorithm: "HS256"

log:
  level: "info"
  format: "json"
  output: "stdout"
  file_path: "logs/app.log"
  max_size: 100
  max_backups: 5
  max_age: 30
  compress: true
  caller: true
  stack_trace: true

security:
  cors:
    allow_origins: ["*"]
    allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers: ["*"]
    expose_headers: ["Content-Length"]
    allow_credentials: true
    max_age: 12h
  rate_limit:
    enabled: true
    requests_per_minute: 100
    burst: 200
  csrf:
    enabled: false
    secret: ""

session:
  store: "memory"
  key: "neoscan_session"
  max_age: 86400
  secure: false
  http_only: true
  same_site: "lax"

websocket:
  enabled: true
  path: "/ws"
  read_buffer_size: 1024
  write_buffer_size: 1024
  check_origin: false
  heartbeat_interval: 30s
  max_connections: 1000

upload:
  max_size: 10485760
  allowed_types: [".jpg", ".jpeg", ".png"]
  upload_path: "uploads/"
  url_prefix: "/uploads/"

mail:
  enabled: false
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  username: ""
  password: ""
  from_email: "noreply@neoscan.com"
  from_name: "NeoScan System"

monitoring:
  metrics:
    enabled: true
    path: "/metrics"
  health:
    enabled: true
    path: "/health"
  pprof:
    enabled: true
    path: "/debug/pprof"

app:
  name: "NeoScan Master Test"
  version: "1.0.0"
  environment: "test"
  debug: true
  timezone: "Asia/Shanghai"
  language: "zh-CN"
  features:
    user_registration: true
    email_verification: false
    password_reset: true
    two_factor_auth: false
    audit_log: true
    api_documentation: true

third_party:
  placeholder: true
`

	// 写入配置文件
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 测试加载配置
	config, err := LoadConfig(tempDir, "test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证配置值
	if config.Server.Host != "localhost" {
		t.Errorf("Expected server host 'localhost', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 8080 {
		t.Errorf("Expected server port 8080, got %d", config.Server.Port)
	}

	if config.Database.MySQL.Database != "test_db" {
		t.Errorf("Expected database name 'test_db', got '%s'", config.Database.MySQL.Database)
	}

	if config.Security.JWT.Secret != "test_jwt_secret_key_at_least_32_chars" {
		t.Errorf("Expected JWT secret, got '%s'", config.Security.JWT.Secret)
	}

	if config.App.Environment != "test" {
		t.Errorf("Expected environment 'test', got '%s'", config.App.Environment)
	}
}

// TestLoadConfigWithEnvVars 测试环境变量覆盖配置
func TestLoadConfigWithEnvVars(t *testing.T) {
	// 设置环境变量
	os.Setenv("NEOSCAN_SERVER_PORT", "9090")
	os.Setenv("NEOSCAN_MYSQL_HOST", "env_mysql_host")
	os.Setenv("NEOSCAN_JWT_SECRET", "env_jwt_secret_key_at_least_32_chars")
	defer func() {
		os.Unsetenv("NEOSCAN_SERVER_PORT")
		os.Unsetenv("NEOSCAN_MYSQL_HOST")
		os.Unsetenv("NEOSCAN_JWT_SECRET")
	}()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configContent := `
server:
  host: "localhost"
  port: 8080
  mode: "test"
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  max_header_bytes: 1048576

database:
  mysql:
    host: "localhost"
    port: 3306
    username: "test_user"
    password: "test_password"
    database: "test_db"
    charset: "utf8mb4"
    parse_time: true
    loc: "Local"
    max_idle_conns: 10
    max_open_conns: 100
    conn_max_lifetime: 3600s
    conn_max_idle_time: 1800s
    log_level: "info"
  redis:
    host: "localhost"
    port: 6379
    password: ""
    database: 0
    pool_size: 10
    min_idle_conns: 5
    dial_timeout: 5s
    read_timeout: 3s
    write_timeout: 3s
    pool_timeout: 4s
    idle_timeout: 300s

jwt:
  secret: "original_jwt_secret_key_at_least_32_chars"
  issuer: "neoscan-test"
  access_token_expire: 24h
  refresh_token_expire: 168h
  algorithm: "HS256"

log:
  level: "info"
  format: "json"
  output: "stdout"
  file_path: "logs/app.log"
  max_size: 100
  max_backups: 5
  max_age: 30
  compress: true
  caller: true
  stack_trace: true

security:
  cors:
    allow_origins: ["*"]
    allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers: ["*"]
    expose_headers: ["Content-Length"]
    allow_credentials: true
    max_age: 12h
  rate_limit:
    enabled: true
    requests_per_minute: 100
    burst: 200
  csrf:
    enabled: false
    secret: ""

session:
  store: "memory"
  key: "neoscan_session"
  max_age: 86400
  secure: false
  http_only: true
  same_site: "lax"

websocket:
  enabled: true
  path: "/ws"
  read_buffer_size: 1024
  write_buffer_size: 1024
  check_origin: false
  heartbeat_interval: 30s
  max_connections: 1000

upload:
  max_size: 10485760
  allowed_types: [".jpg", ".jpeg", ".png"]
  upload_path: "uploads/"
  url_prefix: "/uploads/"

mail:
  enabled: false
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  username: ""
  password: ""
  from_email: "noreply@neoscan.com"
  from_name: "NeoScan System"

monitoring:
  metrics:
    enabled: true
    path: "/metrics"
  health:
    enabled: true
    path: "/health"
  pprof:
    enabled: true
    path: "/debug/pprof"

app:
  name: "NeoScan Master Test"
  version: "1.0.0"
  environment: "test"
  debug: true
  timezone: "Asia/Shanghai"
  language: "zh-CN"
  features:
    user_registration: true
    email_verification: false
    password_reset: true
    two_factor_auth: false
    audit_log: true
    api_documentation: true

third_party:
  placeholder: true
`

	// 写入配置文件
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 测试加载配置
	config, err := LoadConfig(tempDir, "test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证环境变量覆盖了配置文件的值
	if config.Server.Port != 9090 {
		t.Errorf("Expected server port 9090 (from env), got %d", config.Server.Port)
	}

	if config.Database.MySQL.Host != "env_mysql_host" {
		t.Errorf("Expected mysql host 'env_mysql_host' (from env), got '%s'", config.Database.MySQL.Host)
	}

	if config.Security.JWT.Secret != "env_jwt_secret_key_at_least_32_chars" {
		t.Errorf("Expected JWT secret from env, got '%s'", config.Security.JWT.Secret)
	}
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
					Mode: "debug",
				},
				Database: DatabaseConfig{
					MySQL: MySQLConfig{
						Host:     "localhost",
						Database: "test_db",
					},
					Redis: RedisConfig{
						Host: "localhost",
					},
				},
				Security: SecurityConfig{
					JWT: JWTConfig{
						Secret: "test_jwt_secret_key_at_least_32_chars",
					},
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				Session: SessionConfig{
					Store:    "memory",
					SameSite: "lax",
				},
			},
			expectError: false,
		},
		{
			name: "invalid port",
			config: &Config{
				Server: ServerConfig{
					Port: -1,
				},
			},
			expectError: true,
			errorMsg:    "invalid server port",
		},
		{
			name: "short jwt secret",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Mode: "debug",
				},
				Database: DatabaseConfig{
					MySQL: MySQLConfig{
						Host:     "localhost",
						Database: "test_db",
					},
					Redis: RedisConfig{
						Host: "localhost",
					},
				},
				Security: SecurityConfig{
					JWT: JWTConfig{
						Secret: "short", // 太短的密钥
					},
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				Session: SessionConfig{
					Store:    "memory",
					SameSite: "lax",
				},
			},
			expectError: true,
			errorMsg:    "jwt secret must be at least 32 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestEnvManager 测试环境变量管理器
func TestEnvManager(t *testing.T) {
	em := NewEnvManager("TEST")

	// 测试字符串类型
	em.SetString("STRING_VAL", "test_value")
	if val := em.GetString("STRING_VAL", "default"); val != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", val)
	}

	// 测试整数类型
	em.SetInt("INT_VAL", 42)
	if val := em.GetInt("INT_VAL", 0); val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}

	// 测试布尔类型
	em.SetBool("BOOL_VAL", true)
	if val := em.GetBool("BOOL_VAL", false); val != true {
		t.Errorf("Expected true, got %t", val)
	}

	// 测试时间间隔类型
	duration := 5 * time.Minute
	em.SetDuration("DURATION_VAL", duration)
	if val := em.GetDuration("DURATION_VAL", 0); val != duration {
		t.Errorf("Expected %v, got %v", duration, val)
	}

	// 测试字符串切片类型
	slice := []string{"a", "b", "c"}
	em.SetStringSlice("SLICE_VAL", slice)
	if val := em.GetStringSlice("SLICE_VAL", nil); len(val) != 3 || val[0] != "a" {
		t.Errorf("Expected %v, got %v", slice, val)
	}

	// 测试不存在的环境变量
	if val := em.GetString("NON_EXISTENT", "default"); val != "default" {
		t.Errorf("Expected 'default', got '%s'", val)
	}

	// 测试环境变量是否存在
	if !em.Exists("STRING_VAL") {
		t.Error("Expected environment variable to exist")
	}

	if em.Exists("NON_EXISTENT") {
		t.Error("Expected environment variable to not exist")
	}

	// 清理环境变量
	em.Unset("STRING_VAL")
	em.Unset("INT_VAL")
	em.Unset("BOOL_VAL")
	em.Unset("DURATION_VAL")
	em.Unset("SLICE_VAL")
}

// TestConfigHelperMethods 测试配置辅助方法
func TestConfigHelperMethods(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		App: AppConfig{
			Environment: "development",
		},
		Database: DatabaseConfig{
			MySQL: MySQLConfig{
				Host:      "localhost",
				Port:      3306,
				Username:  "user",
				Password:  "pass",
				Database:  "test",
				Charset:   "utf8mb4",
				ParseTime: true,
				Loc:       "Local",
			},
			Redis: RedisConfig{
				Host: "localhost",
				Port: 6379,
			},
		},
	}

	// 测试服务器地址
	expectedAddr := "localhost:8080"
	if addr := config.Server.GetAddress(); addr != expectedAddr {
		t.Errorf("Expected address '%s', got '%s'", expectedAddr, addr)
	}

	// 测试环境判断
	if !config.App.IsDevelopment() {
		t.Error("Expected to be development environment")
	}

	if config.App.IsProduction() {
		t.Error("Expected not to be production environment")
	}

	// 测试MySQL DSN
	expectedDSN := "user:pass@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=true&loc=Local"
	if dsn := config.Database.MySQL.GetMySQLDSN(); dsn != expectedDSN {
		t.Errorf("Expected DSN '%s', got '%s'", expectedDSN, dsn)
	}

	// 测试Redis地址
	expectedRedisAddr := "localhost:6379"
	if addr := config.Database.Redis.GetRedisAddress(); addr != expectedRedisAddr {
		t.Errorf("Expected Redis address '%s', got '%s'", expectedRedisAddr, addr)
	}
}

package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"neoagent/internal/config"
)

// TestConfigLoader 测试配置加载器
func TestConfigLoader(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	
	configContent := `
app:
  name: "neoAgent"
  version: "1.0.0"
  environment: "test"
  debug: true
  timezone: "Asia/Shanghai"

server:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"
  api_version: "v1"
  prefix: "/api"
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  max_header_bytes: 1048576

log:
  level: "debug"
  format: "json"
  output: "stdout"
  file_path: "logs/agent.log"
  max_size: 100
  max_backups: 5
  max_age: 30
  compress: true
  caller: true

database:
  type: "sqlite"
  host: "localhost"
  port: 3306
  username: "root"
  password: "password"
  database: "neoagent"
  charset: "utf8mb4"
  parse_time: true
  loc: "Local"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600s
  conn_max_idle_time: 1800s

master:
  address: "localhost"
  port: 9090
  protocol: "http"
  heartbeat_interval: 30s
  reconnect_interval: 5s
  timeout: 10s
  max_retries: 3
  skip_tls_verify: false

agent:
  id: "test-agent-001"
  name: "Test Agent"
  tags: ["test", "development"]
  auto_register: true
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// 测试配置加载
	loader := config.NewConfigLoader(tempDir, "NEOAGENT")
	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// 验证配置
	if cfg.App.Name != "neoAgent" {
		t.Errorf("Expected app name 'neoAgent', got '%s'", cfg.App.Name)
	}
	
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected server port 8080, got %d", cfg.Server.Port)
	}
	
	if cfg.Database.Type != "sqlite" {
		t.Errorf("Expected database type 'sqlite', got '%s'", cfg.Database.Type)
	}
	
	if cfg.Master.Address != "localhost" {
		t.Errorf("Expected master address 'localhost', got '%s'", cfg.Master.Address)
	}
}

// TestConfigWatcher 测试配置文件监听器
func TestConfigWatcher(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	
	initialConfig := `
app:
  name: "neoAgent"
  debug: false
server:
  port: 8080
agent:
  id: "test-agent-002"
  name: "Test Agent 2"
`
	
	err := os.WriteFile(configFile, []byte(initialConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// 加载初始配置
	loader := config.NewConfigLoader(tempDir, "NEOAGENT")
	_, err = loader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// 创建配置监听器
	watcher, err := config.NewConfigWatcher(configFile)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	
	// 添加回调函数
	configChanged := false
	watcher.AddCallback(func(oldConfig, newConfig *config.Config) error {
		configChanged = true
		t.Logf("Config changed: debug %v -> %v", oldConfig.App.Debug, newConfig.App.Debug)
		return nil
	})
	
	// 启动监听器
	err = watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()
	
	// 等待监听器启动
	time.Sleep(100 * time.Millisecond)
	
	// 修改配置文件
	updatedConfig := `
app:
  name: "neoAgent"
  debug: true
server:
  port: 8080
agent:
  id: "test-agent-002"
  name: "Test Agent 2"
`
	
	err = os.WriteFile(configFile, []byte(updatedConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}
	
	// 等待文件变更被检测到
	time.Sleep(2 * time.Second) // 增加等待时间
	
	// 验证配置是否更新
	currentConfig := watcher.GetConfig()
	if !currentConfig.App.Debug {
		t.Error("Expected debug to be true after config update")
	}
	
	if !configChanged {
		t.Error("Expected config change callback to be called")
	}
}

// TestEnvironmentVariableOverride 测试环境变量覆盖
func TestEnvironmentVariableOverride(t *testing.T) {
	// 设置环境变量
	os.Setenv("NEOAGENT_APP_DEBUG", "true")
	os.Setenv("NEOAGENT_SERVER_PORT", "9090")
	os.Setenv("NEOAGENT_DATABASE_HOST", "testhost")
	defer func() {
		os.Unsetenv("NEOAGENT_APP_DEBUG")
		os.Unsetenv("NEOAGENT_SERVER_PORT")
		os.Unsetenv("NEOAGENT_DATABASE_HOST")
	}()
	
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	
	configContent := `
app:
  debug: false
server:
  port: 8080
database:
  host: "localhost"
agent:
  id: "test-agent-003"
  name: "Test Agent 3"
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// 加载配置
	loader := config.NewConfigLoader(tempDir, "NEOAGENT")
	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// 验证环境变量覆盖
	if !cfg.App.Debug {
		t.Error("Expected debug to be true (overridden by env var)")
	}
	
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected server port 9090 (overridden by env var), got %d", cfg.Server.Port)
	}
	
	if cfg.Database.Host != "testhost" {
		t.Errorf("Expected database host 'testhost' (overridden by env var), got '%s'", cfg.Database.Host)
	}
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	
	// 测试无效配置
	invalidConfig := `
server:
  port: -1  # 无效端口
database:
  type: ""  # 空类型
`
	
	err := os.WriteFile(configFile, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// 尝试加载无效配置
	loader := config.NewConfigLoader(tempDir, "NEOAGENT")
	_, err = loader.LoadConfig()
	if err == nil {
		t.Error("Expected validation error for invalid config")
	}
}

// TestDefaultValues 测试默认值设置
func TestDefaultValues(t *testing.T) {
	// 创建空配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	
	configContent := `
agent:
  id: "test-agent-004"
  name: "Test Agent 4"
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// 加载配置
	loader := config.NewConfigLoader(tempDir, "NEOAGENT")
	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// 验证默认值
	if cfg.App.Name == "" {
		t.Error("Expected default app name to be set")
	}
	
	if cfg.Server.Port == 0 {
		t.Error("Expected default server port to be set")
	}
	
	if cfg.Log.Level == "" {
		t.Error("Expected default log level to be set")
	}
}
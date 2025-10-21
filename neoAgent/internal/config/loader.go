package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// ConfigLoader 配置加载器
type ConfigLoader struct {
	configPath string
	envPrefix  string
	viper      *viper.Viper
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(configPath, envPrefix string) *ConfigLoader {
	if envPrefix == "" {
		envPrefix = "NEOAGENT"
	}
	
	return &ConfigLoader{
		configPath: configPath,
		envPrefix:  envPrefix,
		viper:      viper.New(),
	}
}

// LoadConfig 加载配置
func (cl *ConfigLoader) LoadConfig() (*Config, error) {
	// 设置配置文件类型
	cl.viper.SetConfigType("yaml")
	
	// 设置环境变量前缀
	cl.viper.SetEnvPrefix(cl.envPrefix)
	cl.viper.AutomaticEnv()
	cl.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	// 绑定环境变量
	cl.bindEnvVars()
	
	// 设置默认值
	cl.setDefaults()
	
	// 加载配置文件
	if err := cl.loadConfigFile(); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}
	
	// 解析配置
	var config Config
	if err := cl.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// 验证配置
	if err := cl.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &config, nil
}

// loadConfigFile 加载配置文件
func (cl *ConfigLoader) loadConfigFile() error {
	if cl.configPath == "" {
		// 尝试从环境变量获取配置文件路径
		if envPath := os.Getenv("NEOAGENT_CONFIG_PATH"); envPath != "" {
			cl.configPath = envPath
		} else {
			// 默认配置文件路径
			cl.configPath = "./configs"
		}
	}
	
	// 获取环境
	env := cl.getEnvironment()
	
	// 设置配置文件搜索路径
	cl.viper.AddConfigPath(cl.configPath)
	cl.viper.AddConfigPath("./configs")
	cl.viper.AddConfigPath(".")
	
	// 尝试加载环境特定的配置文件
	configName := fmt.Sprintf("config.%s", env)
	cl.viper.SetConfigName(configName)
	
	if err := cl.viper.ReadInConfig(); err != nil {
		// 如果环境特定配置文件不存在，尝试加载默认配置文件
		cl.viper.SetConfigName("config")
		if err := cl.viper.ReadInConfig(); err != nil {
			return fmt.Errorf("config file not found: %w", err)
		}
	}
	
	return nil
}

// getEnvironment 获取运行环境
func (cl *ConfigLoader) getEnvironment() string {
	env := os.Getenv("NEOAGENT_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = "development"
	}
	return env
}

// bindEnvVars 绑定环境变量
func (cl *ConfigLoader) bindEnvVars() {
	// App配置
	cl.viper.BindEnv("app.name", "NEOAGENT_APP_NAME")
	cl.viper.BindEnv("app.version", "NEOAGENT_APP_VERSION")
	cl.viper.BindEnv("app.environment", "NEOAGENT_APP_ENVIRONMENT")
	cl.viper.BindEnv("app.debug", "NEOAGENT_APP_DEBUG")
	cl.viper.BindEnv("app.timezone", "NEOAGENT_APP_TIMEZONE")
	
	// Server配置
	cl.viper.BindEnv("server.host", "NEOAGENT_SERVER_HOST")
	cl.viper.BindEnv("server.port", "NEOAGENT_SERVER_PORT")
	cl.viper.BindEnv("server.mode", "NEOAGENT_SERVER_MODE")
	
	// 数据库配置
	cl.viper.BindEnv("database.type", "NEOAGENT_DB_TYPE")
	cl.viper.BindEnv("database.host", "NEOAGENT_DB_HOST")
	cl.viper.BindEnv("database.port", "NEOAGENT_DB_PORT")
	cl.viper.BindEnv("database.username", "NEOAGENT_DB_USERNAME")
	cl.viper.BindEnv("database.password", "NEOAGENT_DB_PASSWORD")
	cl.viper.BindEnv("database.database", "NEOAGENT_DB_DATABASE")
	
	// Master配置
	cl.viper.BindEnv("master.address", "NEOAGENT_MASTER_ADDRESS")
	cl.viper.BindEnv("master.port", "NEOAGENT_MASTER_PORT")
	cl.viper.BindEnv("master.protocol", "NEOAGENT_MASTER_PROTOCOL")
	
	// Agent配置
	cl.viper.BindEnv("agent.id", "NEOAGENT_AGENT_ID")
	cl.viper.BindEnv("agent.name", "NEOAGENT_AGENT_NAME")
	cl.viper.BindEnv("agent.work_dir", "NEOAGENT_AGENT_WORK_DIR")
	
	// 日志配置
	cl.viper.BindEnv("log.level", "NEOAGENT_LOG_LEVEL")
	cl.viper.BindEnv("log.file_path", "NEOAGENT_LOG_FILE_PATH")
}

// setDefaults 设置默认值
func (cl *ConfigLoader) setDefaults() {
	// App默认值
	cl.viper.SetDefault("app.name", "NeoAgent")
	cl.viper.SetDefault("app.version", "1.0.0")
	cl.viper.SetDefault("app.environment", "development")
	cl.viper.SetDefault("app.debug", false)
	cl.viper.SetDefault("app.timezone", "UTC")
	
	// Server默认值
	cl.viper.SetDefault("server.host", "0.0.0.0")
	cl.viper.SetDefault("server.port", 8080)
	cl.viper.SetDefault("server.mode", "debug")
	cl.viper.SetDefault("server.api_version", "v1")
	cl.viper.SetDefault("server.prefix", "/api")
	cl.viper.SetDefault("server.read_timeout", "30s")
	cl.viper.SetDefault("server.write_timeout", "30s")
	cl.viper.SetDefault("server.idle_timeout", "60s")
	cl.viper.SetDefault("server.max_header_bytes", 1048576)
	
	// 数据库默认值
	cl.viper.SetDefault("database.type", "mysql")
	cl.viper.SetDefault("database.host", "localhost")
	cl.viper.SetDefault("database.port", 3306)
	cl.viper.SetDefault("database.charset", "utf8mb4")
	cl.viper.SetDefault("database.parse_time", true)
	cl.viper.SetDefault("database.loc", "Local")
	cl.viper.SetDefault("database.max_idle_conns", 10)
	cl.viper.SetDefault("database.max_open_conns", 100)
	cl.viper.SetDefault("database.conn_max_lifetime", "1h")
	cl.viper.SetDefault("database.conn_max_idle_time", "10m")
	
	// Master默认值
	cl.viper.SetDefault("master.address", "localhost")
	cl.viper.SetDefault("master.port", 8081)
	cl.viper.SetDefault("master.protocol", "http")
	cl.viper.SetDefault("master.connect_timeout", "10s")
	cl.viper.SetDefault("master.request_timeout", "30s")
	cl.viper.SetDefault("master.heartbeat_interval", "30s")
	cl.viper.SetDefault("master.reconnect_interval", "5s")
	cl.viper.SetDefault("master.max_reconnect_attempts", 10)
	cl.viper.SetDefault("master.skip_tls_verify", false)
	
	// Agent默认值
	cl.viper.SetDefault("agent.type", "worker")
	cl.viper.SetDefault("agent.work_dir", "./work")
	cl.viper.SetDefault("agent.temp_dir", "./temp")
	cl.viper.SetDefault("agent.log_dir", "./logs")
	cl.viper.SetDefault("agent.data_dir", "./data")
	cl.viper.SetDefault("agent.max_concurrent_tasks", 10)
	cl.viper.SetDefault("agent.task_timeout", "5m")
	cl.viper.SetDefault("agent.auto_register", true)
	
	// 日志默认值
	cl.viper.SetDefault("log.level", "info")
	cl.viper.SetDefault("log.format", "json")
	cl.viper.SetDefault("log.output", "stdout")
	cl.viper.SetDefault("log.file_path", "./logs/agent.log")
	cl.viper.SetDefault("log.max_size", 100)
	cl.viper.SetDefault("log.max_backups", 3)
	cl.viper.SetDefault("log.max_age", 28)
	cl.viper.SetDefault("log.compress", true)
	cl.viper.SetDefault("log.caller", true)
}

// validateConfig 验证配置
func (cl *ConfigLoader) validateConfig(config *Config) error {
	// 验证必需字段
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	
	if config.Database.Type == "" {
		return fmt.Errorf("database type is required")
	}
	
	if config.Master.Address == "" {
		return fmt.Errorf("master address is required")
	}
	
	if config.Agent.ID == "" {
		return fmt.Errorf("agent ID is required")
	}
	
	// 验证目录路径
	if err := cl.validateDirectories(config); err != nil {
		return err
	}
	
	return nil
}

// validateDirectories 验证目录路径
func (cl *ConfigLoader) validateDirectories(config *Config) error {
	dirs := []string{
		config.Agent.WorkDir,
		config.Agent.TempDir,
		config.Agent.LogDir,
		config.Agent.DataDir,
	}
	
	for _, dir := range dirs {
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}
	
	return nil
}

// GetConfigPath 获取配置文件路径
func (cl *ConfigLoader) GetConfigPath() string {
	return cl.viper.ConfigFileUsed()
}

// LoadConfigFromFile 从指定文件加载配置
func LoadConfigFromFile(configFile string) (*Config, error) {
	configPath := filepath.Dir(configFile)
	loader := NewConfigLoader(configPath, "NEOAGENT")
	return loader.LoadConfig()
}
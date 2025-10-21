package config

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigWatcher 配置文件监听器
// 
// 工作原理：
// 1. 使用 fsnotify 监听配置文件变化
// 2. 当文件发生变化时，重新加载配置
// 3. 通过回调函数通知配置变更
// 
// 优势：
// - 支持热重载，无需重启服务
// - 线程安全的配置更新
// - 可自定义配置变更处理逻辑
// 
// 注意事项：
// - 配置变更时会有短暂的不一致状态
// - 需要确保配置变更的原子性
// - 建议在配置变更时进行验证
type ConfigWatcher struct {
	configPath   string
	config       *Config
	loader       *ConfigLoader
	watcher      *fsnotify.Watcher
	callbacks    []ConfigChangeCallback
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	reloadDelay  time.Duration
	lastReload   time.Time
}

// ConfigChangeCallback 配置变更回调函数
type ConfigChangeCallback func(oldConfig, newConfig *Config) error

// NewConfigWatcher 创建配置监听器
func NewConfigWatcher(configPath string) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ConfigWatcher{
		configPath:  configPath,
		loader:      NewConfigLoader(filepath.Dir(configPath), "NEOAGENT"),
		watcher:     watcher,
		callbacks:   make([]ConfigChangeCallback, 0),
		ctx:         ctx,
		cancel:      cancel,
		reloadDelay: 1 * time.Second, // 防抖延迟
	}, nil
}

// Start 启动配置监听
func (cw *ConfigWatcher) Start() error {
	// 初始加载配置
	config, err := cw.loader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load initial config: %w", err)
	}
	
	cw.mu.Lock()
	cw.config = config
	cw.mu.Unlock()
	
	// 添加配置文件到监听列表
	configFile := cw.loader.GetConfigPath()
	if configFile == "" {
		return fmt.Errorf("config file path is empty")
	}
	
	if err := cw.watcher.Add(configFile); err != nil {
		return fmt.Errorf("failed to watch config file %s: %w", configFile, err)
	}
	
	// 启动监听协程
	go cw.watchLoop()
	
	return nil
}

// Stop 停止配置监听
func (cw *ConfigWatcher) Stop() error {
	cw.cancel()
	return cw.watcher.Close()
}

// GetConfig 获取当前配置
func (cw *ConfigWatcher) GetConfig() *Config {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.config
}

// AddCallback 添加配置变更回调
func (cw *ConfigWatcher) AddCallback(callback ConfigChangeCallback) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

// watchLoop 监听循环
func (cw *ConfigWatcher) watchLoop() {
	for {
		select {
		case <-cw.ctx.Done():
			return
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			cw.handleFileEvent(event)
		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Config watcher error: %v\n", err)
		}
	}
}

// handleFileEvent 处理文件事件
func (cw *ConfigWatcher) handleFileEvent(event fsnotify.Event) {
	// 只处理写入和创建事件
	if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
		// 防抖处理，避免频繁重载
		now := time.Now()
		if now.Sub(cw.lastReload) < cw.reloadDelay {
			return
		}
		cw.lastReload = now
		
		// 延迟重载，确保文件写入完成
		time.AfterFunc(cw.reloadDelay, func() {
			if err := cw.reloadConfig(); err != nil {
				fmt.Printf("Failed to reload config: %v\n", err)
			}
		})
	}
}

// reloadConfig 重新加载配置
func (cw *ConfigWatcher) reloadConfig() error {
	// 加载新配置
	newConfig, err := cw.loader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}
	
	// 获取旧配置
	cw.mu.RLock()
	oldConfig := cw.config
	cw.mu.RUnlock()
	
	// 执行回调函数
	for _, callback := range cw.callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			return fmt.Errorf("config change callback failed: %w", err)
		}
	}
	
	// 更新配置
	cw.mu.Lock()
	cw.config = newConfig
	cw.mu.Unlock()
	
	fmt.Println("Config reloaded successfully")
	return nil
}

// WatchConfig 监听配置变更（便捷函数）
func WatchConfig(configPath string, callback ConfigChangeCallback) (*ConfigWatcher, error) {
	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		return nil, err
	}
	
	if callback != nil {
		watcher.AddCallback(callback)
	}
	
	if err := watcher.Start(); err != nil {
		return nil, err
	}
	
	return watcher, nil
}

// DefaultConfigChangeCallback 默认配置变更回调
func DefaultConfigChangeCallback(oldConfig, newConfig *Config) error {
	fmt.Printf("Config changed: %s -> %s\n", 
		oldConfig.App.Version, 
		newConfig.App.Version)
	
	// 这里可以添加配置变更的处理逻辑
	// 例如：重新初始化数据库连接、更新日志级别等
	
	return nil
}

// ValidateConfigChange 验证配置变更
func ValidateConfigChange(oldConfig, newConfig *Config) error {
	// 验证关键配置不能变更
	if oldConfig.Agent.ID != newConfig.Agent.ID {
		return fmt.Errorf("agent ID cannot be changed during runtime")
	}
	
	if oldConfig.Database.Type != newConfig.Database.Type {
		return fmt.Errorf("database type cannot be changed during runtime")
	}
	
	// 验证新配置的有效性
	if newConfig.Server.Port <= 0 || newConfig.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", newConfig.Server.Port)
	}
	
	return nil
}
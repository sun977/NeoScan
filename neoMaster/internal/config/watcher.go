/*
ConfigWatcher 配置文件监听器
监听配置文件的变化，当配置文件发生变化时，调用注册的回调函数。

配置文件监听器的工作原理如下：
1. 监听配置文件所在目录的变化。
2. 当配置文件发生变化时，触发文件系统事件。
3. 调用注册的回调函数，将旧配置和新配置作为参数传递。
4. 回调函数可以根据需要进行配置的重载或其他操作。

配置文件监听器的优势在于：
- 实时监听配置文件变化，无需手动重启服务。
- 支持动态配置更新，无需重启服务即可生效。
- 可以在运行时动态添加或移除回调函数，实现灵活的配置管理。

注意事项：
- 配置文件监听器基于文件系统事件触发，因此对文件系统的性能有一定影响。
- 配置文件监听器只能监听文件的变化，无法监听目录的变化。
- 配置文件监听器只能监听当前进程所在的目录，无法监听其他进程的目录。
*/
package config

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify" // 文件系统监听库
)

// ConfigWatcher 配置文件监听器
type ConfigWatcher struct {
	watcher    *fsnotify.Watcher  // 文件系统监听器
	configPath string             // 配置文件路径
	env        string             // 环境标识
	callbacks  []ReloadCallback   // 重载回调函数列表
	mu         sync.RWMutex       // 读写锁
	ctx        context.Context    // 上下文
	cancel     context.CancelFunc // 取消函数
	done       chan struct{}      // 完成信号
}

// ReloadCallback 配置重载回调函数类型
type ReloadCallback func(oldConfig, newConfig *Config) error

// NewConfigWatcher 创建配置文件监听器
func NewConfigWatcher(configPath, env string) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cw := &ConfigWatcher{
		watcher:    watcher,
		configPath: configPath,
		env:        env,
		callbacks:  make([]ReloadCallback, 0),
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
	}

	return cw, nil
}

// Start 启动配置文件监听
func (cw *ConfigWatcher) Start() error {
	// 获取配置文件路径
	if cw.configPath == "" {
		cw.configPath = getDefaultConfigPath()
	}

	// 添加监听目录
	if err := cw.watcher.Add(cw.configPath); err != nil {
		return fmt.Errorf("failed to add config path to watcher: %w", err)
	}

	// 启动监听协程
	go cw.watchLoop()

	log.Printf("Config watcher started, watching path: %s", cw.configPath)
	return nil
}

// Stop 停止配置文件监听
func (cw *ConfigWatcher) Stop() error {
	cw.cancel()

	// 等待监听协程结束
	select {
	case <-cw.done:
	case <-time.After(5 * time.Second):
		log.Println("Config watcher stop timeout")
	}

	return cw.watcher.Close()
}

// AddCallback 添加配置重载回调函数
func (cw *ConfigWatcher) AddCallback(callback ReloadCallback) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

// watchLoop 监听循环
func (cw *ConfigWatcher) watchLoop() {
	defer close(cw.done)

	// 防抖动定时器
	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		<-debounceTimer.C
	}

	for {
		select {
		case <-cw.ctx.Done():
			log.Println("Config watcher stopped")
			return // 停止监听

		case event, ok := <-cw.watcher.Events:
			if !ok {
				log.Println("Config watcher events channel closed")
				return
			}

			// 只处理写入和创建事件
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// 检查是否为配置文件
				if cw.isConfigFile(event.Name) {
					log.Printf("Config file changed: %s", event.Name)

					// 重置防抖动定时器
					debounceTimer.Reset(500 * time.Millisecond)
				}
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				log.Println("Config watcher errors channel closed")
				return
			}
			log.Printf("Config watcher error: %v", err)

		case <-debounceTimer.C:
			// 执行配置重载
			if err := cw.reloadConfig(); err != nil {
				log.Printf("Failed to reload config: %v", err)
			}
		}
	}
}

// isConfigFile 检查是否为配置文件
func (cw *ConfigWatcher) isConfigFile(filename string) bool {
	baseName := filepath.Base(filename)

	// 检查是否为YAML配置文件
	if filepath.Ext(baseName) != ".yaml" && filepath.Ext(baseName) != ".yml" {
		return false
	}

	// 检查是否为配置文件
	configFiles := []string{
		"config.yaml",
		"config.yml",
		"config.dev.yaml",
		"config.dev.yml",
		"config.test.yaml",
		"config.test.yml",
		"config.prod.yaml",
		"config.prod.yml",
	}

	for _, configFile := range configFiles {
		if baseName == configFile {
			return true
		}
	}

	return false
}

// reloadConfig 重载配置
func (cw *ConfigWatcher) reloadConfig() error {
	// 保存旧配置
	oldConfig := GlobalConfig

	// 加载新配置
	newConfig, err := LoadConfig(cw.configPath, cw.env)
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}

	// 执行回调函数
	cw.mu.RLock()
	callbacks := make([]ReloadCallback, len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.mu.RUnlock()

	for _, callback := range callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			log.Printf("Config reload callback error: %v", err)
			// 继续执行其他回调，不因为一个回调失败而中断
		}
	}

	log.Println("Config reloaded successfully")
	return nil
}

// 全局配置监听器实例
var (
	globalWatcher *ConfigWatcher
	watcherMu     sync.Mutex
)

// StartConfigWatcher 启动全局配置文件监听器
func StartConfigWatcher(configPath, env string) error {
	watcherMu.Lock()
	defer watcherMu.Unlock()

	if globalWatcher != nil {
		return fmt.Errorf("config watcher is already running")
	}

	watcher, err := NewConfigWatcher(configPath, env)
	if err != nil {
		return err
	}

	if err := watcher.Start(); err != nil {
		return err
	}

	globalWatcher = watcher
	return nil
}

// StopConfigWatcher 停止全局配置文件监听器
func StopConfigWatcher() error {
	watcherMu.Lock()
	defer watcherMu.Unlock()

	if globalWatcher == nil {
		return nil
	}

	err := globalWatcher.Stop()
	globalWatcher = nil
	return err
}

// AddConfigReloadCallback 添加配置重载回调函数
func AddConfigReloadCallback(callback ReloadCallback) error {
	watcherMu.Lock()
	defer watcherMu.Unlock()

	if globalWatcher == nil {
		return fmt.Errorf("config watcher is not running")
	}

	globalWatcher.AddCallback(callback)
	return nil
}

// 预定义的配置重载回调函数

// LogConfigReloadCallback 日志配置重载回调
func LogConfigReloadCallback(oldConfig, newConfig *Config) error {
	if oldConfig == nil {
		return nil
	}

	// 检查日志配置是否发生变化
	if oldConfig.Log.Level != newConfig.Log.Level ||
		oldConfig.Log.Format != newConfig.Log.Format ||
		oldConfig.Log.Output != newConfig.Log.Output {
		log.Printf("Log configuration changed, old level: %s, new level: %s",
			oldConfig.Log.Level, newConfig.Log.Level)
		// 这里可以添加重新初始化日志器的逻辑
	}

	return nil
}

// DatabaseConfigReloadCallback 数据库配置重载回调
func DatabaseConfigReloadCallback(oldConfig, newConfig *Config) error {
	if oldConfig == nil {
		return nil
	}

	// 检查数据库配置是否发生变化
	if oldConfig.Database.MySQL.Host != newConfig.Database.MySQL.Host ||
		oldConfig.Database.MySQL.Port != newConfig.Database.MySQL.Port ||
		oldConfig.Database.MySQL.Database != newConfig.Database.MySQL.Database {
		log.Println("Database configuration changed, connection pool may need to be recreated")
		// 这里可以添加重新初始化数据库连接池的逻辑
	}

	return nil
}

// SecurityConfigReloadCallback 安全配置重载回调
func SecurityConfigReloadCallback(oldConfig, newConfig *Config) error {
	if oldConfig == nil {
		return nil
	}

	// 检查安全配置是否发生变化
	if len(oldConfig.Security.CORS.AllowOrigins) != len(newConfig.Security.CORS.AllowOrigins) {
		log.Println("CORS configuration changed")
		// 这里可以添加重新配置CORS中间件的逻辑
	}

	if oldConfig.Security.RateLimit.Enabled != newConfig.Security.RateLimit.Enabled ||
		oldConfig.Security.RateLimit.RequestsPerSecond != newConfig.Security.RateLimit.RequestsPerSecond {
		log.Println("Rate limit configuration changed")
		// 这里可以添加重新配置限流中间件的逻辑
	}

	return nil
}

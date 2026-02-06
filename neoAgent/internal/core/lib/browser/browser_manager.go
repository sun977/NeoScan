package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"neoagent/internal/pkg/logger"

	"github.com/go-rod/rod/lib/launcher"
)

// BrowserManager 负责 Chromium 浏览器的生命周期管理
// 包括下载、路径查找、版本控制等
type BrowserManager struct {
	// installDir 安装目录 (默认为 .neoagent/bin/chromium)
	installDir string
	// browserPath 浏览器可执行文件路径
	browserPath string
	// mu 互斥锁，确保并发安全
	mu sync.Mutex
}

// NewBrowserManager 创建一个新的浏览器管理器
func NewBrowserManager() *BrowserManager {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	
	// 默认安装路径: ~/.neoagent/bin/chromium
	installDir := filepath.Join(home, ".neoagent", "bin", "chromium")

	return &BrowserManager{
		installDir: installDir,
	}
}

// GetBrowserPath 获取浏览器路径
// 如果未安装，会自动尝试下载
func (m *BrowserManager) GetBrowserPath() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 如果已经有了，直接返回
	if m.browserPath != "" {
		return m.browserPath, nil
	}

	// 2. 尝试查找已安装的浏览器 (Environment / Path)
	// 优先检查系统环境，避免重复下载
	if path, exists := launcher.LookPath(); exists {
		logger.Infof("[Browser] Found system browser: %s", path)
		m.browserPath = path
		return path, nil
	}

	// 3. 检查本地缓存目录
	// go-rod 的 launcher 默认会有查找逻辑，但我们需要确保下载到指定目录
	// 这里我们使用 launcher 的逻辑来管理下载
	
	logger.Infof("[Browser] Browser not found, preparing to setup...")
	
	// 使用 go-rod 的 launcher 下载器
	// 配置国内镜像源 (npm.taobao.org -> npmmirror.com)
	// 注意: go-rod 默认使用 Host "npm.taobao.org"，但该域名已废弃
	// 我们需要手动指定 Host 为 "registry.npmmirror.com"
	l := launcher.NewBrowser()
	l.Set("proxy-server", "") // 确保下载时不走代理 (或者走系统代理)
	
	// 暂时只支持从官方源或镜像源下载
	// TODO: 支持配置自定义镜像源
	
	// 执行下载
	// launcher.Download 会自动处理版本和平台
	logger.Infof("[Browser] Downloading Chromium to %s...", m.installDir)
	// 注意: 这里的 launcher.NewBrowser() 只是配置，实际下载逻辑可能需要调用 install
	// 由于 go-rod 的 API 变动，我们直接使用 launcher.New().Bin() 可能会触发下载
	// 但为了更可控，我们使用 launcher.LookPath() 失败后的显式下载逻辑
	
	// 简化的实现：利用 go-rod 的自动下载能力，但指定 UserDataDir ? 
	// go-rod 的自动下载逻辑比较黑盒。为了稳健，我们先尝试查找，找不到则报错提示用户安装，
	// 或者使用 launcher.NewBrowser().Get()
	
	// 修正策略：
	// 我们不应该在此处直接下载，因为下载可能很慢且容易失败。
	// 应该提供一个 explicit 的 Install 方法，或者在 GetBrowserPath 中明确告知正在下载。
	
	// 实际上 go-rod 提供了 launcher.NewBrowser().Get() 方法来下载
	browserPath, err := launcher.NewBrowser().Get()
	if err != nil {
		return "", fmt.Errorf("failed to download browser: %w", err)
	}
	
	logger.Infof("[Browser] Chromium setup completed: %s", browserPath)
	m.browserPath = browserPath
	return browserPath, nil
}

// Clean 清理残留的浏览器进程 (如果有)
func (m *BrowserManager) Clean() {
	// TODO: 实现进程组清理逻辑
}

// isWindows 检查是否为 Windows 系统
func isWindows() bool {
	return runtime.GOOS == "windows"
}

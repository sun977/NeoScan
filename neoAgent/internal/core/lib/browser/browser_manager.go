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
	// installDir 安装目录 (默认为 ./bin/chromium)
	installDir string
	// browserPath 浏览器可执行文件路径
	browserPath string
	// mu 互斥锁，确保并发安全
	mu sync.Mutex
}

// NewBrowserManager 创建一个新的浏览器管理器
func NewBrowserManager() *BrowserManager {
	// 获取当前执行文件所在目录
	ex, err := os.Executable()
	if err != nil {
		// Fallback to current working directory if executable path cannot be determined
		ex, _ = os.Getwd()
	}
	baseDir := filepath.Dir(ex)

	// 默认安装路径: ./bin/chromium (相对于 neoAgent 二进制文件)
	// 这样设计更符合便携式应用 (Portable App) 的理念，解压即用，删除即卸载
	installDir := filepath.Join(baseDir, "bin", "chromium")

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

	logger.Warnf("[Browser] Chromium binary not found. Starting automatic download...")
	logger.Warnf("[Browser] This is a one-time setup and may take a few minutes depending on your network.")

	// 使用 go-rod 的 launcher 下载器
	l := launcher.NewBrowser()
	// 强制指定安装目录为 ~/.neoagent/bin/chromium
	l.Dir = m.installDir

	// 执行下载
	// go-rod's launcher prints progress to stdout by default
	browserPath, err := l.Get()
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

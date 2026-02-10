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
	// 构造 go-rod launcher 实例
	l := launcher.NewBrowser()
	l.Dir = m.installDir

	// 尝试在指定目录查找浏览器
	// Validate() 在旧版本 rod 可能不存在，或者名字不同。
	// 但 launcher.Browser.Get() 内部其实已经有了缓存检查逻辑。
	// 问题在于，我们手动 new 了一个 Browser，如果它没有正确指向已存在的路径，Get() 就会重新下载。
	// l.Get() 会调用 l.find()，如果找到就返回。
	// 但 l.find() 依赖 l.Dir。
	// 让我们检查一下目录是否存在，并且是否包含可执行文件。

	// 手动检查目录是否存在
	if _, err := os.Stat(m.installDir); err == nil {
		// 如果目录存在，我们让 l.Get() 去决定是否完整。
		// 但为了避免"Found system browser"没有触发而直接进下载，
		// 我们可以尝试利用 LookPath 的变体?
		// 其实 l.Get() 只要配置了 Dir，就应该能找到。

		// 也许问题出在版本号匹配上？
		// rod 会下载特定版本的 chromium。如果本地目录有，但版本不对（或者 rod 认为不对），它就会重下。
		// 我们先尝试不带下载逻辑的查找。
	}

	// 在 go-rod 中，可以使用 launcher.LookPath()，但它只看系统路径。
	// 我们需要看 m.installDir。

	// 让我们相信 l.Get() 的缓存机制，但为了避免重复下载，我们可以先尝试查找。
	// 由于 rod 的 Browser 结构体没有暴露 Find 方法，我们只能依赖 Get。
	// 用户的反馈是"每次都下载"，说明 l.Get() 没找到。
	// 这可能是因为 l.Dir 虽然设置了，但 rod 期望的子目录结构不匹配。

	// 修正：我们先不要 Warn，先调用 Get，如果它很快返回了，就是找到了。
	// 但 Get() 会打印下载进度，所以还是会有干扰。

	// 让我们用一个更简单的方法：检查 m.installDir 下是否有文件。
	// rod 下载的目录结构通常是: installDir/chromium-123456/chrome-linux/chrome
	// 只要 m.installDir 不为空，我们就假设它可能存在。

	// 更好的方法：遍历 m.installDir，寻找可执行文件
	if foundPath := findChromeInDir(m.installDir); foundPath != "" {
		logger.Infof("[Browser] Found cached browser: %s", foundPath)
		m.browserPath = foundPath
		return foundPath, nil
	}

	// 如果没找到，再下载
	logger.Warnf("[Browser] Chromium binary not found. Starting automatic download...")
	logger.Warnf("[Browser] This is a one-time setup and may take a few minutes depending on your network.")

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

// findChromeInDir 递归查找目录下的 chrome 可执行文件
func findChromeInDir(dir string) string {
	var found string
	exeName := "chrome"
	if isWindows() {
		exeName = "chrome.exe"
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == exeName {
			// 简单的权限检查 (Linux)
			if !isWindows() && info.Mode()&0111 == 0 {
				// 尝试赋予执行权限，防止因复制导致的权限丢失
				if err := os.Chmod(path, 0755); err != nil {
					logger.Warnf("[Browser] Found chrome at %s but cannot chmod: %v", path, err)
					return nil
				}
				logger.Infof("[Browser] Fixed permissions for %s", path)
			}
			found = path
			return filepath.SkipDir // 找到一个就行
		}
		return nil
	})
	return found
}

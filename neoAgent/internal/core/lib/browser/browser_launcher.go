package browser

import (
	"context"
	"fmt"
	"sync"

	"neoagent/internal/pkg/logger"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// BrowserLauncher 负责启动浏览器实例
type BrowserLauncher struct {
	manager  *BrowserManager
	proxy    string // 代理地址 (e.g. socks5://127.0.0.1:1080)
	headless bool

	// 浏览器实例 (全局复用)
	browser *rod.Browser
	mu      sync.Mutex
}

// NewLauncher 创建启动器
func NewLauncher(manager *BrowserManager) *BrowserLauncher {
	return &BrowserLauncher{
		manager:  manager,
		headless: true, // 默认无头模式
	}
}

// SetProxy 设置代理
func (l *BrowserLauncher) SetProxy(proxy string) {
	l.proxy = proxy
}

// SetHeadless 设置是否无头模式
func (l *BrowserLauncher) SetHeadless(headless bool) {
	l.headless = headless
}

// Launch 启动或获取已启动的浏览器实例
func (l *BrowserLauncher) Launch(ctx context.Context) (*rod.Browser, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 1. 如果已经启动且连接正常，直接返回
	if l.browser != nil {
		// 简单的健康检查
		if _, err := l.browser.Version(); err == nil {
			return l.browser, nil
		}
		// 连接断开，尝试清理并重启
		l.browser.Close()
		l.browser = nil
	}

	// 2. 获取浏览器路径
	binPath, err := l.manager.GetBrowserPath()
	if err != nil {
		return nil, err
	}

	// 3. 配置启动参数
	u := launcher.New().
		Bin(binPath).
		Headless(l.headless).
		// 禁用沙箱 (在 Docker/Root 环境下必须)
		NoSandbox(true).
		// 禁用 GPU (Headless 模式下不需要)
		Set("disable-gpu").
		// 禁用 dev-shm (避免内存不足崩溃)
		Set("disable-dev-shm-usage").
		// 禁用默认扩展
		Set("disable-extensions").
		// 忽略证书错误 (关键! 否则无法扫描 HTTPS 站点)
		Set("ignore-certificate-errors").
		// 设置 User-Agent
		Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 4. 配置代理 (Proxy Integration)
	if l.proxy != "" {
		// Chromium 代理参数格式: --proxy-server="scheme://host:port"
		// 注意: Chromium 对 SOCKS5 的支持可能需要 scheme (socks5://)
		u = u.Set("proxy-server", l.proxy)
		logger.Debugf("[Browser] Launching with proxy: %s", l.proxy)
	}

	// 5. 启动
	controlURL, err := u.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	// 6. 连接
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	// 7. 配置 Browser Context
	// 忽略证书错误 (再次确认)
	// 这一步对于 rod 来说可能不是必须的，因为启动参数已经加了，但为了保险
	// browser.IgnoreCertErrors(true) // rod v0.113+ 已废弃此方法，改为启动参数控制

	l.browser = browser
	return browser, nil
}

// Close 关闭浏览器
func (l *BrowserLauncher) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.browser != nil {
		err := l.browser.Close()
		l.browser = nil
		return err
	}
	return nil
}

// OpenPage 打开新页面并应用通用配置
func (l *BrowserLauncher) OpenPage(ctx context.Context, browser *rod.Browser, targetURL string) (*rod.Page, error) {
	// 创建新页面 (Incognito Context 更好，但 rod 默认是 default context)
	// 建议使用 MustIncognito().Page(url) 来隔离 Cookie

	// 这里使用默认上下文，后续可以优化为 Incognito
	page, err := browser.Page(proto.TargetCreateTarget{URL: targetURL})
	if err != nil {
		return nil, err
	}

	// 设置视口大小 (影响截图和响应式页面)
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             1920,
		Height:            1080,
		DeviceScaleFactor: 1,
		Mobile:            false,
	}); err != nil {
		// 非致命错误，记录即可
		logger.Warnf("[Browser] Failed to set viewport: %v", err)
	}

	return page, nil
}

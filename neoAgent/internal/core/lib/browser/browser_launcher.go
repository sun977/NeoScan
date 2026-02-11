package browser

import (
	"context"
	"fmt"
	"sync"

	"neoagent/internal/pkg/logger"
	"neoagent/internal/pkg/version"

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
		if _, err := l.browser.Pages(); err == nil {
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
		// 允许所有 Mixed Content (HTTPS 页面加载 HTTP 资源) --- 可选
		Set("allow-insecure-localhost").
		// 允许运行不安全的内容 (如 HTTP 资源加载 HTTPS 页面) --- 可选
		Set("allow-running-insecure-content").
		// 设置 User-Agent --- 自定义 User-Agent 以标识 NeoScan 代理
		Set("user-agent", version.GetUserAgent())

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

	// 7. 设置全局 IgnoreCertErrors
	// 虽然启动参数加了 ignore-certificate-errors，但有些场景下(如 iframe)可能仍需通过 CDP 命令忽略
	// 这是一个保险措施
	// browser.IgnoreCertErrors(true) // rod v0.106+ 不需要手动调，参数已足够

	l.browser = browser
	return l.browser, nil
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

// OpenPage 打开新页面并配置 (如忽略证书错误)
func (l *BrowserLauncher) OpenPage(ctx context.Context, browser *rod.Browser, url string) (*rod.Page, error) {
	// 创建新页面 (incognito context by default if browser was launched incognito, but here we share context)
	// 使用 MustIncognito? No, we use default context for now.

	// CreateTarget allows us to create page with url directly
	// But usually we create blank page then navigate to capture events
	var page *rod.Page
	var err error

	if url == "" {
		page, err = browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	} else {
		page, err = browser.Page(proto.TargetCreateTarget{URL: url})
	}

	if err != nil {
		return nil, err
	}

	// 关联 Context
	page = page.Context(ctx)

	// 忽略证书错误 (再次确保)
	// 虽然启动参数已设置，但在某些版本 Chromium 中，Page 级别的 Security 检查可能仍需 Bypass
	// 通过 CDP 命令 Security.setIgnoreCertificateErrors
	// 这是一个保险措施，防止 HTTPS 报错
	// 注意: 需要先启用 Security 域，但在 rod 中直接调用即可
	// err = proto.SecuritySetIgnoreCertificateErrors{Ignore: true}.Call(page)
	// 简化: rod 并没有直接暴露这个 helper 在 Page 上?
	// 实际上，启动参数 --ignore-certificate-errors 通常已足够全局生效。
	// 如果需要，可以使用 page.MustEval(`...`) 或者 hijack。
	// 这里暂不额外处理，除非发现问题。

	return page, nil
}

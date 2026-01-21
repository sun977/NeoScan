# 可以通过 go-rod 实现自动获取 Chromium

package main

import (
    "github.com/go-rod/rod"
    "github.com/go-rod/rod/lib/launcher"
)

func main() {
    // launcher.New() 会自动下载 Chromium（如果需要）
    url := launcher.New().MustLaunch()

    // 连接到浏览器
    browser := rod.New().ControlURL(url).MustConnect()

    // 打开新页面
    page := browser.MustPage("https://example.com")

    // 等待页面加载完成并截图
    page.MustWaitLoad().MustScreenshot("example.png")
}


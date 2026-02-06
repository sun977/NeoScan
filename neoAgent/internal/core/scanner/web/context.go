package web

import (
	"encoding/json"

	"github.com/go-rod/rod"
)

// ExtractRichContext 从页面提取富上下文信息
// 包括: HTML, Headers, Meta, Scripts, Cookies, Global Variables
func ExtractRichContext(page *rod.Page) (map[string]interface{}, error) {
	ctx := make(map[string]interface{})

	// 1. 获取 HTML 内容
	html, err := page.HTML()
	if err == nil {
		ctx["body"] = html
	}

	// 2. 获取标题
	// rod 提供了 MustInfo().Title，但这里使用非 panic 版本
	if info, err1 := page.Info(); err1 == nil {
		ctx["title"] = info.Title
	}

	// 3. 提取 Meta 标签
	// 使用 JS 执行提取
	metaMap := make(map[string]string)
	// 使用 IIFE 确保立即执行并返回值
	metaRes, err := page.Eval(`(() => {
		const metas = document.getElementsByTagName('meta');
		const result = {};
		for (let i = 0; i < metas.length; i++) {
			const name = metas[i].getAttribute('name') || metas[i].getAttribute('property');
			const content = metas[i].getAttribute('content');
			if (name && content) {
				result[name] = content;
			}
		}
		return result;
	})()`)
	if err == nil {
		// metaRes.Value 是 gson.JSON，将其序列化为 bytes 再反序列化到 map
		if valBytes, e := json.Marshal(metaRes.Value); e == nil {
			_ = json.Unmarshal(valBytes, &metaMap)
		}
	}
	ctx["meta"] = metaMap

	// 4. 提取 Script 标签 (src)
	var scripts []string
	scriptRes, err := page.Eval(`(() => {
		const scripts = document.getElementsByTagName('script');
		const result = [];
		for (let i = 0; i < scripts.length; i++) {
			if (scripts[i].src) {
				result.push(scripts[i].src);
			}
		}
		return result;
	})()`)
	if err == nil {
		if valBytes, e := json.Marshal(scriptRes.Value); e == nil {
			_ = json.Unmarshal(valBytes, &scripts)
		}
	}
	// 放入 dom.scripts，方便 matcher 匹配
	ctx["dom"] = map[string]interface{}{
		"scripts": scripts,
	}

	// 5. 提取 JS 全局变量 (针对 Wappalyzer 规则)
	// Wappalyzer 的规则里有 "js": {"wp": ...} 这种，意味着检查 window.wp 是否存在
	// 这里我们无法预知所有变量，只能根据规则按需提取，或者提取一些常见的
	// 目前先留空，等待后续根据规则动态生成 JS 代码
	ctx["js"] = map[string]interface{}{}

	// 6. 提取 Headers
	// go-rod 比较难直接获取 Response Headers，除非开启了 Network 监听
	// 这是一个比较复杂的话题，通常需要 page.HijackRequests 或者监听 Event
	// 简单起见，L1 阶段的 HTTP 请求已经获取了 Headers，这里主要关注 DOM 信息

	// 7. 提取 Cookies
	cookies, err := page.Cookies(nil)
	if err == nil {
		cookieMap := make(map[string]string)
		for _, c := range cookies {
			cookieMap[c.Name] = c.Value
		}
		ctx["cookies"] = cookieMap
	}

	// 8. 提取 Favicon URL
	// 注意: 这里只提取 URL，后续由 Scanner 决定是否下载并转换为 Base64
	faviconURL, err := page.Eval(`(() => {
		let link = document.querySelector("link[rel*='icon']");
		return link ? link.href : "";
	})()`)
	if err == nil {
		var favStr string
		if valBytes, e := json.Marshal(faviconURL.Value); e == nil {
			if err := json.Unmarshal(valBytes, &favStr); err == nil {
				ctx["favicon_url"] = favStr
			}
		}
	}

	return ctx, nil
}

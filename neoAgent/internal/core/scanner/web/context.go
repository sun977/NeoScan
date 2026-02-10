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
	// 策略：提取 window 对象下所有非内置的 Key，或者简单粗暴提取所有第一层 Key
	// 为了性能和兼容性，我们提取 window 的所有属性名，后续由 Matcher 决定匹配哪个
	// 注意：只提取 Key，不提取 Value，因为 Value 可能是复杂对象导致序列化失败
	// 如果规则需要匹配 Value (如版本号)，需要更复杂的逻辑，目前 V1 版本先支持"存在性"检查
	var jsKeys []string
	jsRes, err := page.Eval(`(() => {
		const keys = [];
		// 遍历 window 对象的可枚举属性
		for (const key in window) {
			keys.push(key);
		}
		// 也可以补充一些不可枚举但常见的，或者直接使用 Object.getOwnPropertyNames(window)
		// 这里为了保险起见，结合两者，去重
		const allKeys = new Set(Object.getOwnPropertyNames(window));
		for (const key in window) {
			allKeys.add(key);
		}
		return Array.from(allKeys);
	})()`)
	if err == nil {
		if valBytes, e := json.Marshal(jsRes.Value); e == nil {
			_ = json.Unmarshal(valBytes, &jsKeys)
		}
	}
	// 将 keys 转为 map[string]interface{} 格式，Value 暂时为空，方便统一接口
	// 后续如果 Matcher 需要检查 Value，这里需要改为提取具体 Value
	jsMap := make(map[string]interface{})
	for _, key := range jsKeys {
		jsMap[key] = "" // 占位，表示存在
	}
	ctx["js"] = jsMap

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

package crawler

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
	// 注意：传给 page.Eval 的必须是一个函数本身（() => {...}），不能写成 IIFE
	// （(() => {...})()）。rod 内部会把传入的 JS 字符串包装成
	// `function() { return (js).apply(this, arguments) }`，如果 js 写成 IIFE，
	// 包装后就变成对 IIFE 的返回值（一个普通对象/数组）调用 .apply()，
	// 而这个返回值根本没有 .apply 方法，会抛出 TypeError，导致 err != nil。
	metaRes, err := page.Eval(`() => {
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
	}`)
	if err == nil {
		// metaRes.Value 是 gson.JSON，将其序列化为 bytes 再反序列化到 map
		if valBytes, e := json.Marshal(metaRes.Value); e == nil {
			_ = json.Unmarshal(valBytes, &metaMap)
		}
	}
	ctx["meta"] = metaMap

	// 4. 提取 Script 标签 (src)
	var scripts []string
	scriptRes, err := page.Eval(`() => {
		const scripts = document.getElementsByTagName('script');
		const result = [];
		for (let i = 0; i < scripts.length; i++) {
			if (scripts[i].src) {
				result.push(scripts[i].src);
			}
		}
		return result;
	}`)
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
	// 使用 Iframe Trick 过滤掉浏览器内置的全局变量，只保留用户/框架定义的变量
	// 同时尝试提取值 (仅限基本类型)，用于后续可能的版本匹配
	var jsVars map[string]interface{}
	jsRes, err := page.Eval(`() => {
		try {
			// 1. 创建一个干净的 iframe 用于获取标准全局变量列表
			const iframe = document.createElement('iframe');
			iframe.style.display = 'none';
			document.body.appendChild(iframe);
			// 某些情况下 iframe.contentWindow 可能为空或受限，需防御性编程
			if (!iframe.contentWindow) {
				document.body.removeChild(iframe);
				return {}; 
			}
			
			const cleanWindow = iframe.contentWindow;
			const standardGlobals = new Set(Object.getOwnPropertyNames(cleanWindow));
			
			// 补充一些常见的标准但在 iframe 中可能缺失的
			['alert', 'confirm', 'prompt', 'print', 'postMessage'].forEach(k => standardGlobals.add(k));

			document.body.removeChild(iframe);

			// 2. 获取当前窗口的全局变量
			const currentGlobals = Object.getOwnPropertyNames(window);
			const customGlobals = {};

			// 3. 差集计算
			for (const key of currentGlobals) {
				if (!standardGlobals.has(key)) {
					try {
						const val = window[key];
						const type = typeof val;
						
						if (val === null) {
							customGlobals[key] = null;
						} else if (type === 'string') {
							// 限制字符串长度，防止过大
							customGlobals[key] = val.length > 100 ? val.substring(0, 100) + '...' : val;
						} else if (type === 'number' || type === 'boolean') {
							customGlobals[key] = val;
						} else if (type === 'object') {
							// 对于对象，尝试简单的特征描述，或者是特定的已知对象提取版本
							// 这里简单标记为 [Object]
							customGlobals[key] = '[Object]';
						} else if (type === 'function') {
							customGlobals[key] = '[Function]';
						} else {
							customGlobals[key] = '[' + type + ']';
						}
					} catch (e) {
						// 某些属性访问可能抛出异常 (如 cross-origin 限制)
						customGlobals[key] = '[Error]';
					}
				}
			}
			return customGlobals;
		} catch (e) {
			return { 'error': e.toString() };
		}
	}`)

	if err == nil {
		// jsRes.Value 是 gson.JSON
		if valBytes, e := json.Marshal(jsRes.Value); e == nil {
			_ = json.Unmarshal(valBytes, &jsVars)
		}
	} else {
		// 如果执行失败，初始化为空 map
		jsVars = make(map[string]interface{})
	}

	ctx["js"] = jsVars

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
	faviconURL, err := page.Eval(`() => {
		let link = document.querySelector("link[rel*='icon']");
		return link ? link.href : "";
	}`)
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

// ExtractLinks 从 go-rod 已渲染完成的页面中提取 <a href> 链接列表，
// 供爬虫（crawler 包）BFS 遍历使用的"第一层种子链接"。
//
// 为什么单独写一个函数，而不是复用 crawler 包里基于 goquery 的 ExtractLinksAndForms？
//   两者处理的输入完全不同：
//     - 这里的输入是 go-rod 已经跑完 JS、渲染出最终 DOM 之后的浏览器页面对象，
//       页面上所有通过 JS 动态生成的链接（比如 Vue/React 渲染出来的 <a> 标签）
//       在这一步都已经真实存在于 DOM 里了。
//     - ExtractLinksAndForms 处理的输入是 net/http 拿到的原始 HTML 文本字符串，
//       如果页面是 JS 动态渲染的（SPA），这段原始 HTML 里可能只有一个空的
//       <div id="root"></div>，压根看不到真正的链接。
//   所以首页种子链接优先用这个函数（如果 go-rod 可用），只有在 go-rod 启动/
//   导航失败、降级到 net/http 的场景下，才会改用 ExtractLinksAndForms 提取种子。
//
// 原理：
//   用 page.Eval 在浏览器上下文里执行一段 JS，遍历 document.getElementsByTagName('a')，
//   取每个 <a> 标签的 .href 属性（注意不是 getAttribute('href')）。这是刻意的选择：
//   DOM 元素的 .href 属性经过浏览器内部处理后，天然就是解析好的绝对 URL
//   （浏览器自己已经用当前页面地址做了一次相对路径换算），比如页面里写的
//   `<a href="foo.html">`，读取 .href 会直接得到 "http://a.com/dir/foo.html"，
//   不需要再在 Go 侧手动调用 url.ResolveReference 之类的函数二次处理。
//   而 getAttribute('href') 拿到的是标签里写的原始字符串（可能是相对路径），
//   还得自己再解析一遍，多此一举。
//
//   JS 执行完的返回值是 rod 的 gson.JSON 类型，这里统一走"先 json.Marshal
//   成 []byte，再 json.Unmarshal 回 Go 的 []string"这个转换套路——这和
//   ExtractRichContext 里提取 meta/scripts/js 全局变量用的是同一套模式，
//   本文件保持风格一致，不额外发明新的类型转换方式。
func ExtractLinks(page *rod.Page) []string {
	res, err := page.Eval(`() => {
		const anchors = document.getElementsByTagName('a');
		const result = [];
		for (let i = 0; i < anchors.length; i++) {
			if (anchors[i].href) {
				result.push(anchors[i].href);
			}
		}
		return result;
	}`)
	if err != nil {
		// JS 执行失败（比如页面已经关闭、上下文丢失），种子链接拿不到就返回 nil，
		// 调用方（WebScanner）看到空切片自然就不会触发深度爬取，是安全的降级。
		return nil
	}
	var links []string
	if valBytes, e := json.Marshal(res.Value); e == nil {
		_ = json.Unmarshal(valBytes, &links)
	}
	return links
}

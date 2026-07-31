package crawler

// ==============================================================================
// extract.go：攻击面提取器
// ------------------------------------------------------------------------------
// 职责（只做这一件事）：
//   给定一个页面的原始 HTML 字符串，把里面隐藏的"攻击面信息"抠出来，包括：
//     1. 链接 (<a href="...">)      —— 用于 BFS 继续往下爬
//     2. 表单 (<form>...</form>)    —— 用于后续漏洞扫描的输入点（比如找 XSS/SQLi 注入点）
//     3. URL 参数 (?id=1&type=2)   —— 同样是攻击面输入点，参数名本身就是有价值的信息
//
// 为什么用 goquery 而不是手写正则/字符串查找？
//   HTML 不是正则能"正确"解析的格式（标签可以嵌套、属性顺序任意、单双引号混用、
//   甚至标签不闭合浏览器也能容错渲染）。goquery 内部用 golang.org/x/net/html 做了
//   一次真正的 DOM 树解析，然后提供类似 jQuery 的选择器语法（如 `a[href]`）来查询，
//   这样"找出所有带 href 属性的 <a> 标签"这种需求只需要一行选择器，且对畸形 HTML
//   有天然的容错能力。Sprint 1 里那个用字符串查找 `href=` 的占位实现，本质上是
//   "能跑但不严谨"的临时方案，Sprint 2 就是用这个文件把它换成正规实现。
// ==============================================================================

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"neoagent/internal/core/model"
)

// ExtractLinksAndForms 是本文件对外暴露的唯一入口函数。
//
// 输入：
//   - baseURL: 当前页面自己的 URL（用于把页面里的相对路径链接换算成绝对路径，
//     比如页面是 http://a.com/dir/page.html，页面里写的 <a href="foo.html">
//     必须换算成 http://a.com/dir/foo.html，否则爬虫拿到的链接根本没法访问）。
//   - body: 页面的原始 HTML 文本（从 net/http 响应体或 go-rod 渲染结果里拿到的）。
//
// 输出（三个返回值，互相独立，各自失败不影响彼此）：
//   - links:  页面里所有"看起来能继续访问"的绝对 URL，去掉了 javascript:/mailto:/
//     tel:/纯锚点这类不是真实页面的伪链接。
//   - forms:  页面里所有 <form> 标签的结构化信息（提交地址、方法、字段名）。
//   - params: 当前页面自己 URL 里的 Query 参数名集合（不是链接里的参数，是
//     baseURL 自身携带的参数，比如 baseURL 是 ?id=1&type=admin，就返回
//     ["id", "type"]）。
//
// 原理：
//  1. 用 goquery.NewDocumentFromReader 把 HTML 字符串解析成一棵可查询的 DOM 树。
//  2. 用两条选择器分别扫描 <a href> 和 <form>，逐个处理后放进结果切片。
//  3. baseURL 自身的参数直接用 net/url 解析 Query() 拿到，不需要 goquery。
//
// 任何一步解析失败（比如 body 根本不是合法 HTML，或者 baseURL 本身解析不出来），
// 函数只会让对应的那一部分返回值为空，不会 panic，也不会让另外两个返回值受影响
// ——因为攻击面提取是"尽力而为"的辅助功能，不应该因为某个页面格式古怪就让整个
// 爬虫任务失败。
func ExtractLinksAndForms(baseURL string, body string) (links []string, forms []model.FormInfo, params []string) {
	base, err := url.Parse(baseURL)
	if err != nil {
		// baseURL 都解析不出来，后面的相对路径换算无从谈起，直接放弃整个函数。
		return nil, nil, nil
	}

	// 先把当前页面自身携带的 Query 参数抠出来，这一步和 HTML 解析无关，
	// 即便下面 HTML 解析失败，这部分信息也应该尽量保留。
	params = extractParamNames(base)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		// HTML 解析失败（比如 body 是空的、是一段 JSON、是二进制内容），
		// links/forms 保持 nil，但 params 已经算出来了，照常返回。
		return nil, nil, params
	}

	links = extractLinks(doc, base)
	forms = extractForms(doc)
	return links, forms, params
}

// extractLinks 从已解析的 DOM 树里提取所有 <a href="..."> 标签对应的绝对 URL。
//
// 原理：goquery 的 `Find("a[href]")` 等价于 CSS 选择器"所有带 href 属性的 a 标签"，
// `.Each` 会对匹配到的每一个节点执行一次回调（回调签名固定是 (索引, *Selection)，
// 这里用不到索引所以用 `_` 忽略）。每个节点取出 href 原始值后交给 resolve 做
// "相对路径 -> 绝对路径"的换算和"是否是可爬取的真实链接"的过滤，resolve 返回
// 空字符串就表示这个 href 不值得收录（细节见 resolve 函数注释）。
func extractLinks(doc *goquery.Document, base *url.URL) []string {
	var links []string
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		if abs := resolve(base, href); abs != "" {
			links = append(links, abs)
		}
	})
	return links
}

// extractForms 从已解析的 DOM 树里提取所有 <form> 标签的结构化信息。
//
// 表单为什么是攻击面：漏洞扫描（比如后续 Sprint 要做的 XSS/SQLi 检测）需要知道
// "这个页面有哪些地方可以提交用户输入"，<form> 就是最典型的输入点——它告诉你
// 提交到哪个地址（action）、用什么 HTTP 方法（method）、有哪些字段名（fields）。
// 拿到这些信息后，后续漏洞扫描器就能针对每个字段构造 payload 发起测试，而不需要
// 自己再重新解析一遍页面。
//
// 原理：
//  1. `Find("form")` 找出所有表单节点。
//  2. 每个表单节点内部再用 `Find("input[name],select[name],textarea[name]")`
//     找出所有"带 name 属性"的输入控件——服务端只会收到有 name 的字段，没有
//     name 的 input（比如纯展示用的按钮）提交时根本不会被带上，所以不需要收录。
//  3. method 属性如果没写（HTML 规范里 <form> 不写 method 时浏览器默认按 GET
//     提交），这里手动补全为 "GET"，避免后续使用方要重复判断这个 HTML 规范细节。
func extractForms(doc *goquery.Document) []model.FormInfo {
	var forms []model.FormInfo
	doc.Find("form").Each(func(_ int, s *goquery.Selection) {
		action, _ := s.Attr("action")
		method, hasMethod := s.Attr("method")
		if !hasMethod || strings.TrimSpace(method) == "" {
			method = "GET"
		} else {
			method = strings.ToUpper(strings.TrimSpace(method))
		}

		var fields []string
		s.Find("input[name],select[name],textarea[name]").Each(func(_ int, f *goquery.Selection) {
			if name, ok := f.Attr("name"); ok && name != "" {
				fields = append(fields, name)
			}
		})

		forms = append(forms, model.FormInfo{
			Action: action,
			Method: method,
			Fields: fields,
		})
	})
	return forms
}

// extractParamNames 从一个已解析的 URL 中取出所有 Query 参数的名字（不含值）。
//
// 只取参数名、不取参数值的原因：这里的用途是"记录这个页面暴露了哪些输入点名字"
// （比如告诉后续扫描器"这个页面接受 id 和 type 两个参数"），而不是记录具体的
// 业务数据；参数值本身可能包含敏感信息或者只是当前这一次访问的随机取值，
// 收录参数值没有额外价值，反而可能造成信息冗余。
func extractParamNames(u *url.URL) []string {
	query := u.Query()
	if len(query) == 0 {
		return nil
	}
	names := make([]string, 0, len(query))
	for k := range query {
		names = append(names, k)
	}
	return names
}

// resolve 把页面里写的原始 href（可能是相对路径，也可能已经是绝对路径）
// 换算成一个可以直接发起 HTTP 请求的绝对 URL；如果这个 href 根本不是一个
// "可爬取的页面地址"，返回空字符串，调用方看到空字符串就应该丢弃它。
//
// 需要过滤掉的四类伪链接（这是本函数存在的核心原因，不是可有可无的边界处理）：
//   - javascript:xxx   —— 这是一段 JS 代码，不是页面地址，请求它没有意义。
//   - mailto:xxx       —— 打开邮件客户端用的，不是网页。
//   - tel:xxx          —— 打开电话拨号用的，不是网页。
//   - #xxx / 空字符串   —— 纯页内锚点跳转（或者 href 干脆是空的），
//     指向的还是当前这同一个页面，爬取它只是在原地重复访问，没有意义。
//
// 如果不做这层过滤，这些伪链接会被塞进 BFS 队列，crawler 会尝试对
// "javascript:void(0)" 这样的字符串发起 net/http 请求，必然请求失败，
// 白白浪费一次网络请求配额，还会把限流器的失败计数拉高，属于纯粹的噪音。
//
// 过滤通过之后，用 net/url 的 ResolveReference 做真正的换算：它会按照
// RFC 3986 定义的规则，结合 base（当前页面地址）和 ref（href 解析出的相对/
// 绝对引用），算出正确的绝对地址——这是标准库提供的能力，不需要自己手写
// "拼路径"的逻辑（手写很容易在 ../ 这类上级目录引用上出错）。
func resolve(base *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" ||
		strings.HasPrefix(href, "javascript:") ||
		strings.HasPrefix(href, "mailto:") ||
		strings.HasPrefix(href, "tel:") ||
		strings.HasPrefix(href, "#") {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}

# JSFinder / URLFinder / dirsearch 项目分析（面向 Web-JS 接口提取方案的参考调研）

> 文档版本：v1.0
> 修改时间：2026-08-04
> 分析对象：`docs/references/JSFinder/`、`docs/references/URLFinder/`、`docs/references/dirsearch/`（均为本地已缓存的第三方源码，只读分析，不做代码改动）
> 文档性质：调研文档，为 [`Web-JS接口提取方案.md`](./Web-JS接口提取方案.md) 提供原理/实现/亮点的参考依据，末节给出三者对比后的最终选型结论。

---

## 一、JSFinder（Python，最早的同类工具，正则出自 LinkFinder）

### 1.1 原理

一句话：**"抓页面 → 收集所有 `<script>`（内联 + 外链）→ 对拼起来的 JS 文本跑一条大正则 → 按前缀规则把提取结果拼成绝对 URL"**。是这一类工具里最简单、最早、后续几乎所有同类工具都在"抄它的正则再魔改"的原型。

### 1.2 实现拆解

**抓取**：`Extract_html` 用 `requests` 直接 GET，`BeautifulSoup` 解析 `<script>` 标签，有 `src` 的再发一次请求拿外链 JS 内容，无 `src` 的直接取内联文本，全部拼接成一个大字符串统一跑正则（不区分来源）：

```101:123:c:/mytools/code/go/NeoScan/neoAgent/docs/references/JSFinder/JSFinder.py
def find_by_url(url, js = False):
	...
	html_scripts = html.findAll("script")
	script_array = {}
	script_temp = ""
	for html_script in html_scripts:
		script_src = html_script.get("src")
		if script_src == None:
			script_temp += html_script.get_text() + "\n"
		else:
			purl = process_url(url, script_src)
			script_array[purl] = Extract_html(purl)
	script_array[url] = script_temp
```

**提取正则**（直接照搬自 LinkFinder，是这个领域事实上的"标准正则"）：

```23:46:c:/mytools/code/go/NeoScan/neoAgent/docs/references/JSFinder/JSFinder.py
def extract_URL(JS):
	pattern_raw = r"""
	  (?:"|')                               # Start newline delimiter
	  (
	    ((?:[a-zA-Z]{1,10}://|//)           # Match a scheme [a-Z]*1-10 or //
	    [^"'/]{1,}\.                        # Match a domainname (any character + dot)
	    [a-zA-Z]{2,}[^"']{0,})              # The domainextension and/or path
	    |
	    ((?:/|\.\./|\./)                    # Start with /,../,./
	    [^"'><,;| *()(%%$^/\\\[\]]          # Next character can't be...
	    [^"'><,;|()]{1,})                   # Rest of the characters can't be
	    |
	    ([a-zA-Z0-9_\-/]{1,}/               # Relative endpoint with /
	    [a-zA-Z0-9_\-/]{1,}                 # Resource name
	    \.(?:[a-zA-Z]{1,4}|action)          # Rest + extension (length 1-4 or action)
	    (?:[\?|/][^"|']{0,}|))              # ? mark with parameters
	    |
	    ([a-zA-Z0-9_\-]{1,}                 # filename
	    \.(?:php|asp|aspx|jsp|json|
	         action|html|js|txt|xml)             # . + extension
	    (?:\?[^"|']{0,}|))                  # ? mark with parameters
	  )
	  (?:"|')                               # End newline delimiter
	"""
```

这条正则是四选一的"或"结构，分别覆盖：完整协议+域名的绝对 URL；`/`、`../`、`./` 开头的相对路径；`xxx/xxx.ext` 形态的资源路径；纯文件名+固定后缀名单（含 `.php/.json/.action` 等）。**注意它的后缀白名单里没有把"看起来像接口路径但没有后缀"（如 `/api/v1/user`，没有扩展名）作为一个独立分支**——这是它最大的短板，纯 RESTful 无后缀的接口路径命中率不高。

**相对路径转绝对路径**：`process_url` 按四种前缀（`//`、`http`、`/`、其他相对路径）分别处理，`../` 只是简单裁掉两个字符再拼接，不是标准的 RFC 3986 处理，遇到多层 `../../` 会算错：

```66:89:c:/mytools/code/go/NeoScan/neoAgent/docs/references/JSFinder/JSFinder.py
def process_url(URL, re_URL):
	black_url = ["javascript:"]	# Add some keyword for filter url.
	URL_raw = urlparse(URL)
	ab_URL = URL_raw.netloc
	host_URL = URL_raw.scheme
	if re_URL[0:2] == "//":
		result = host_URL  + ":" + re_URL
	elif re_URL[0:4] == "http":
		result = re_URL
	elif re_URL[0:2] != "//" and re_URL not in black_url:
		if re_URL[0:1] == "/":
			result = host_URL + "://" + ab_URL + re_URL
		else:
			if re_URL[0:1] == ".":
				if re_URL[0:2] == "..":
					result = host_URL + "://" + ab_URL + re_URL[2:]
				else:
					result = host_URL + "://" + ab_URL + re_URL[1:]
			else:
				result = host_URL + "://" + ab_URL + "/" + re_URL
	else:
		result = URL
	return result
```

**子域名收集是提取 URL 后的副产品**：从提取出的所有 URL 里再解析一遍 `netloc`，和主域名做包含关系比对，不是独立的检测逻辑。

**深度爬取**：`-d` 参数只多做一层——先爬首页的所有 `<a href>`，再对每个链接页面各跑一遍 `find_by_url`，是最朴素的"广度爬取 1 层"，没有队列、没有去重（`if link not in links` 是唯一的去重手段）、没有并发。

### 1.3 亮点

1. **正则本身是被后续所有同类工具验证过的、覆盖面广的基线**——四个分支覆盖了绝对 URL、相对路径、带后缀的资源路径、裸文件名四种最常见形态，是"从零设计正则"时最值得参考的起点，不需要重新发明。
2. **代码量极小（不到 260 行）却完整覆盖了这个问题的核心链路**：抓取→提取→归一化→去重→子域名衍生，是理解这一整类工具"最小可行实现长什么样"的最佳样本。
3. **`process_url` 里"没有 `-c` cookie 就匿名请求，有就带上"的简单退化处理**，提醒我们这类工具经常要考虑"目标页面需要登录态才能看到真实接口列表"的场景，但 JSFinder 本身没有解决这个问题，只是留了个接口。

### 1.4 短板（不建议照搬的地方）

- 无后缀的 RESTful 路径命中率低（正则设计年代还是 `.php/.jsp` 后缀 API 为主流的时代背景）。
- 完全没有降噪机制——凡是正则命中的一律收录，无置信度分层，静态资源（`.png/.css` 混在同一条正则的后缀名单里，虽然不属于接口但会一起被收录）。
- 无并发、无限流、无超时/重试策略，纯同步阻塞，扫大站会很慢。
- `../` 相对路径处理不是标准实现，多层嵌套会出错。

---

## 二、URLFinder（Go，pingc0y 作者，社区里公认的 JSFinder"精神继承者"）

作者在 README 里明确写了开发动机——"经常使用 JSFinder 时会返回空或链接不完整……萌生了自己开发一款类似工具的想法"，是同一问题域下的"工程化重写"，用 Go 解决了 JSFinder 的并发/规则可配置/漏抓等问题。

### 2.1 原理

一句话：**"并发 BFS 爬取（Spider）→ 用一组可配置的正则分别抽取 JS 链接 / URL 链接 / 敏感信息 → 分层过滤降噪 → （可选）对所有候选发请求验证状态码 → （可选）对 404 目录做字典化 Fuzz 补全"**。相比 JSFinder，多了**规则可配置化**、**主动状态验证**、**目录 Fuzz** 三个显著升级。

### 2.2 实现拆解

**抓取主循环**：`Spider` 是整个工具的核心节点，用 `chan int` 做并发数控制（类似信号量），拿到响应后依次调用 `jsFind`/`urlFind`/`infoFind` 三个独立的提取函数，互不干扰：

```1:25:c:/mytools/code/go/NeoScan/neoAgent/docs/references/URLFinder/crawler/crawler.go
package crawler

import (
	"compress/gzip"
	"fmt"
	"github.com/pingc0y/URLFinder/cmd"
	"github.com/pingc0y/URLFinder/config"
	"github.com/pingc0y/URLFinder/result"
	"github.com/pingc0y/URLFinder/util"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// 蜘蛛抓取页面内容
func Spider(u string, num int) {
	is := true
	defer func() {
		config.Wg.Done()
		if is {
			<-config.Ch
		}

	}()
```

**规则外置为配置（`config.yaml`），而不是硬编码在代码里**——这是相比 JSFinder 最大的架构升级：`jsFind`/`urlFind`/`infoFind`（敏感信息：手机号/邮箱/身份证/JWT/其他）都是字符串数组，可以在不改代码的情况下加规则：

```34:60:c:/mytools/code/go/NeoScan/neoAgent/docs/references/URLFinder/config/config.go
	JsFind = []string{
		"(https{0,1}:[-a-zA-Z0-9（）@:%_\\+.~#?&//=]{2,250}?[-a-zA-Z0-9（）@:%_\\+.~#?&//=]{3}[.]js)",
		"[\"'‘“`]\\s{0,6}(/{0,1}[-a-zA-Z0-9（）@:%_\\+.~#?&//=]{2,250}?[-a-zA-Z0-9（）@:%_\\+.~#?&//=]{3}[.]js)",
		"=\\s{0,6}[\",',’,”]{0,1}\\s{0,6}(/{0,1}[-a-zA-Z0-9（）@:%_\\+.~#?&//=]{2,250}?[-a-zA-Z0-9（）@:%_\\+.~#?&//=]{3}[.]js)",
	}
	UrlFind = []string{
		"[\"'‘“`]\\s{0,6}(https{0,1}:[-a-zA-Z0-9()@:%_\\+.~#?&//={}]{2,250}?)\\s{0,6}[\"'‘“`]",
		"=\\s{0,6}(https{0,1}:[-a-zA-Z0-9()@:%_\\+.~#?&//={}]{2,250})",
		"[\"'‘“`]\\s{0,6}([#,.]{0,2}/[-a-zA-Z0-9()@:%_\\+.~#?&//={}]{2,250}?)\\s{0,6}[\"'‘“`]",
		"\"([-a-zA-Z0-9()@:%_\\+.~#?&//={}]+?[/]{1}[-a-zA-Z0-9()@:%_\\+.~#?&//={}]+?)\"",
		"href\\s{0,6}=\\s{0,6}[\"'‘“`]{0,1}\\s{0,6}([-a-zA-Z0-9()@:%_\\+.~#?&//={}]{2,250})|action\\s{0,6}=\\s{0,6}[\"'‘“`]{0,1}\\s{0,6}([-a-zA-Z0-9()@:%_\\+.~#?&//={}]{2,250})",
	}
```

`urlFind` 的第三条正则 `[\"'‘“\`]\s{0,6}([#,.]{0,2}/...` 没有强制要求后缀名，这是相比 JSFinder 提升"无后缀 RESTful 路径"召回率的关键改动。

**过滤（降噪）是独立的一步，和提取正则解耦**：`jsFilter`/`urlFilter` 先做统一清洗（去转义、去空格、URL 解码），再套一层可配置黑名单正则（`JsFiler`/`UrlFiler`），黑名单里专门列了一批"提取正则容易误命中、但明显不是接口"的模式（`.src`、`.href`、`location.href`、`javascript:`、各类静态资源后缀）：

```42:82:c:/mytools/code/go/NeoScan/neoAgent/docs/references/URLFinder/crawler/filter.go
func urlFilter(str [][]string) [][]string {
	for i := range str {
		str[i][0], _ = url.QueryUnescape(str[i][1])
		str[i][0] = strings.TrimSpace(str[i][0])
		str[i][0] = strings.Replace(str[i][0], " ", "", -1)
		str[i][0] = strings.Replace(str[i][0], "\\/", "/", -1)
		str[i][0] = strings.Replace(str[i][0], "%3A", ":", -1)
		str[i][0] = strings.Replace(str[i][0], "%2F", "/", -1)
		match, _ := regexp.MatchString("[a-zA-Z]+|[0-9]+", str[i][0])
		if !match {
			str[i][0] = ""
			continue
		}
		if isNonFetchableReference(str[i][0]) {
			str[i][0] = ""
			continue
		}
		...
		for i2 := range config.UrlFiler {
			re := regexp.MustCompile(config.UrlFiler[i2])
			is := re.MatchString(str[i][0])
			if is {
				str[i][0] = ""
				break
			}
		}
	}
	return str
}
```

黑名单正则本身也值得参考，覆盖了非常具体的实战踩坑经验（`UrlFiler` 第一条）：

```51:53:c:/mytools/code/go/NeoScan/neoAgent/docs/references/URLFinder/config/config.go
	UrlFiler = []string{
		"\\.js\\?|\\.css\\?|\\.jpeg\\?|\\.jpg\\?|\\.png\\?|.gif\\?|www\\.w3\\.org|example\\.com|\\<|\\>|\\{|\\}|\\[|\\]|\\||\\^|;|/js/|\\.src|\\.replace|\\.url|\\.att|\\.href|location\\.href|javascript:|location:|application/x-www-form-urlencoded|\\.createObject|:location|\\.path|\\*#__PURE__\\*|\\*\\$0\\*|\\n",
		".*\\.js$|.*\\.css$|.*\\.scss$|.*,$|.*\\.jpeg$|.*\\.jpg$|.*\\.png$|.*\\.gif$|.*\\.ico$|.*\\.svg$|.*\\.vue$|.*\\.ts$",
	}
```

**主动状态验证（`-s` 参数触发）是独立的第二阶段**，不影响提取阶段的性能，对每个候选 URL/JS 单独发一次请求拿状态码、响应体大小、标题，还处理了 302 重定向链：

```16:27:c:/mytools/code/go/NeoScan/neoAgent/docs/references/URLFinder/crawler/state.go
// 检测js访问状态码
func JsState(u string, i int, sou string) {

	defer func() {
		config.Wg.Done()
		<-config.Jsch
		PrintProgress()
	}()
	if cmd.S == "" {
		result.ResultJs[i].Url = u
		return
	}
```

**404 目录 Fuzz 是提取完成之后的补充能力**：从已发现的 404 链接里提炼出目录层级，再和内置的常见文件名字典（`login.js`/`app.js`/`config.js` 等）组合碰撞，试图找到"因为拼接错误没被提取到、但实际存在"的资源：

```11:33:c:/mytools/code/go/NeoScan/neoAgent/docs/references/URLFinder/crawler/jsFuzz.go
func JsFuzz() {
	paths := []string{}
	for i := range result.ResultJs {
		re := regexp.MustCompile("(.+/)[^/]+.js").FindAllStringSubmatch(result.ResultJs[i].Url, -1)
		if len(re) != 0 {
			paths = append(paths, re[0][1])
		}
		...
	}
	paths = util.UniqueArr(paths)
	for i := range paths {
		for i2 := range config.JsFuzzPath {
			result.ResultJs = append(result.ResultJs, mode.Link{
				Url:    paths[i] + config.JsFuzzPath[i2],
				Source: "Fuzz",
			})
		}
	}
}
```

**安全模式（`-m 3`）显式跳过危险路由**——这是唯一一个"主动避免误伤"的设计，值得注意：

```18:19:c:/mytools/code/go/NeoScan/neoAgent/docs/references/URLFinder/config/config.go
var (
	Risks = []string{"remove", "delete", "insert", "update", "logout"}
```

抓取阶段和状态验证阶段命中这些关键词的路径都会被跳过不发请求（`crawler.go`/`state.go` 均有对应判断），因为状态验证是真实发起 GET 请求，如果目标是一个 `/api/user/delete` 这样的接口，即便是 GET 请求也可能被后端设计成有副作用的操作，这是这类工具在"要不要主动验证"这个问题上给出的一种折中方案。

**结果去重和最大数量熔断在 `Append` 环节统一处理**，而不是在提取正则层面处理：

```279:309:c:/mytools/code/go/NeoScan/neoAgent/docs/references/URLFinder/crawler/run.go
func AppendJs(ur string, urltjs string) int {
	config.Lock.Lock()
	defer config.Lock.Unlock()
	if len(result.ResultUrl)+len(result.ResultJs) >= cmd.MA {
		return 1
	}
	_, err := url.Parse(ur)
	if err != nil {
		return 2
	}
	for _, eachItem := range result.ResultJs {
		if eachItem.Url == ur {
			return 0
		}
	}
	result.ResultJs = append(result.ResultJs, mode.Link{Url: ur})
	...
```

### 2.3 亮点

1. **提取规则外置为可配置项，而不是编译期常量**——这让"规则库需要持续迭代"这件事变成运维层面的配置更新，不需要重新编译发版，是相比 JSFinder 最大的工程化进步。
2. **提取 → 过滤 → 验证三阶段清晰分离**，每一阶段职责单一，互不耦合：提取只管"正则命中"，过滤只管"排除已知噪音模式"，验证是独立的可选开关，不强制发生。这个三段式结构本身就是一份很好的架构参考。
3. **`Risks` 危险路由黑名单是"主动探测边界意识"的具体体现**——工具作者意识到"验证状态码"这个动作本身对某些路径（delete/remove/update/logout）是有副作用的主动操作，用一个显式的黑名单去规避，而不是假装所有 GET 请求都是安全幂等的。这是所有安全工具设计"主动验证功能"时都应该考虑的一条边界。
4. **404 目录 Fuzz 弥补了"静态提取天生的召回率上限"**——JS 代码里能提取到的接口终究是"开发者写在代码里的"，Fuzz 是对这个上限的一种主动补偿手段，用字典去撞"提取不到但可能存在"的资源。
5. **`isNonFetchableReference` 显式区分"引用类协议"（如 `tel:`/`mailto:`/纯锚点）和真正可访问的地址**，这个判断和我们项目 `crawler/extract.go` 里 `resolve` 函数过滤 `javascript:`/`mailto:`/`tel:`/`#` 的逻辑是完全一致的思路，说明这是这个问题域的共识做法，不是某一家的独创。

### 2.4 短板 / 需要辩证看待的地方

- **状态验证和目录 Fuzz 都是主动探测行为**——对我们的场景（安全扫描工具，需要克制主动请求的量级和副作用）需要谨慎借鉴，不能直接照搬"默认开启"的设计。
- **正则规则数量多、维护成本不低**——`UrlFind` 5 条 + `UrlFiler` 2 条黑名单，长期需要人工根据实战反馈持续调整，本身也是一种技术债，只是用配置化的方式把维护成本从"改代码重新编译"降低到了"改配置文件"。
- **`urlFind` 第四条正则 `"([-a-zA-Z0-9()@:%_\+.~#?&//={}]+?[/]{1}...)"` 几乎是"只要带斜杠的双引号字符串就收"，召回率高但注定伴随较高误报率**，README 自己也承认这一点（"为了更好的兼容和防止漏抓链接，放弃了低误报率，错误的链接会变多但漏抓概率变低"）——这是一个**明确的、作者自己也认可的取舍**，不是无意识的缺陷，但如果直接照搬这条正则，需要接受"结果里会有相当比例噪音"这个代价。

---

## 三、dirsearch（Python，目录爆破工具，字典驱动，和前两者是"互补"而非"同类"关系）

### 3.1 原理

dirsearch 解决的不是"从 JS 里挖 URL"，而是"用字典暴力探测路径是否存在"，这是完全不同的问题（主动 Fuzz vs 被动提取）。用户点名它，价值不在"抄它的核心功能"，而在于它为了让**暴力探测**这件事做得靠谱，磨出来的几个通用能力——**降噪（避免海量误报）**、**用已抓到的页面内容反哺字典（爬虫辅助字典）**——这两点对我们判断"提取出的 API 该不该收录/该不该进一步验证"有直接借鉴意义。

### 3.2 实现拆解

**核心亮点：Wildcard（通配符/软 404）检测机制**，这是 dirsearch 相比简单状态码判断最大的工程含金量所在。很多网站对"任意不存在的路径"都返回 200（SPA 前端路由兜底页、自定义美化 404 页），如果只看状态码，会把这些页面全部误判为"发现了一个新资产"。dirsearch 的做法是：先探测一个几乎不可能真实存在的随机路径，拿到的响应作为"这个网站对不存在路径的标准反应模板"，后续每一次真正的探测结果都要先和这个模板比对，只有"明显不同于模板"的响应才会被认为是真实存在的资源：

```67:114:c:/mytools/code/go/NeoScan/neoAgent/docs/references/dirsearch/lib/core/scanner.py
    def check(self, path: str, response: BaseResponse) -> bool:
        """
        Perform analyzing to see if the response is wildcard or not
        """
        decision = self.classify(path, response)
        return decision != "wildcard"

    def classify(self, path: str, response: BaseResponse) -> str:
        """Classify response against this wildcard profile."""

        if self.response.status != response.status:
            self.reason = "status differs from wildcard profile"
            return "unique"
        ...
        if self.is_wildcard(response):
            self.reason = "matches wildcard profile"
            return "wildcard"

        if self.is_probable_wildcard(path, response):
            self.reason = "matches ambiguous wildcard profile"
            ...
            return "wildcard"

        self.reason = "response is unique enough"
        return "unique"
```

不仅比较状态码，还比较响应体内容相似度（`content_similarity`）、重定向目标是否符合"通配符重定向"的模式（把路径部分替换成占位符后生成正则去匹配，`generate_matching_regex`/`replace_path`），是一套相当严谨的"排除噪音"链路。

**用已抓到的页面内容反哺后续探测的字典（"爬虫辅助字典"）**——这是 `lib/utils/crawl.py` 里的 `Crawler`，和 URLFinder/JSFinder 的"提取链接"看似同一件事，但用途完全不同：dirsearch 提取出来的路径不是"最终结果"，而是**追加进当前扫描的路径候选队列**，让下一轮字典爆破也测试这些路径。更关键的是它的提取方式和 JSFinder/URLFinder 的"猜测式正则"不同，是**锚定 scope（当前域名）的精确匹配**：

```37:58:c:/mytools/code/go/NeoScan/neoAgent/docs/references/dirsearch/lib/utils/crawl.py
class Crawler:
    @classmethod
    def crawl(cls, response):
        scope = "/".join(response.url.split("/")[:3]) + "/"

        if "text/html" in response.headers.get("content-type", ""):
            return cls.html_crawl(response.url, scope, response.content)
        elif response.path == "robots.txt":
            return cls.robots_crawl(response.url, scope, response.content)
        else:
            return cls.text_crawl(response.url, scope, response.content)

    @staticmethod
    @lru_cache(maxsize=None)
    def text_crawl(url, scope, content):
        results = []
        regex = re.escape(scope) + "[a-zA-Z0-9-._~!$&*+,;=:@?%]+"

        for match in re.findall(regex, content):
            results.append(match[len(scope):])

        return _filter(results)
```

`text_crawl` 的正则是 `re.escape(scope) + "[合法URL字符集]+"`——先用当前域名的完整前缀（`scope`，例如 `https://a.com/`）做精确锚定，再匹配后面跟着的合法路径字符。这和 JSFinder/URLFinder"不知道域名是什么，靠猜测协议前缀/斜杠开头去泛化匹配"的思路完全不同：**因为明确知道自己正在扫描的目标是谁，所以可以把正则收窄到"必须以这个域名开头"，天然排除了大量属于第三方域名（CDN、统计脚本、广告）的噪音链接**，误报率显著更低，只是代价是"必须先知道 scope"，无法用于"不知道目标域名、纯粹拿到一段 JS 文本就要分析"的场景。

`html_crawl` 则是结构化提取（用 `BeautifulSoup` 认标签+认属性），比正则更精确，覆盖了 `CRAWL_TAGS`（`a/area/base/blockquote/button/embed/form/frame/...`）× `CRAWL_ATTRIBUTES`（`action/cite/data/formaction/href/poster/src/srcset/...`）的笛卡尔积，比只看 `<a href>` 和 `<form action>` 覆盖面更广。

**媒体文件后缀白名单式排除**，是最简单直接的降噪手段：

```33:34:c:/mytools/code/go/NeoScan/neoAgent/docs/references/dirsearch/lib/utils/crawl.py
def _filter(paths):
    return {clean_path(path, keep_queries=True) for path in paths if not path.endswith(MEDIA_EXTENSIONS)}
```

**黑名单字典是按状态码分类维护的独立文本文件**，而不是写死在代码里，这也是一种"规则外置"，只是外置的形式是纯文本字典而非 YAML 配置：

```1:9:c:/mytools/code/go/NeoScan/neoAgent/docs/references/dirsearch/db/403_blacklist.txt
%2e%2e//google.com
%ff
%2e%2e/%2e%2e/%2e%2e/%2e%2e/%2e%2e/%2e%2e/etc/passwd
%2e%2e;/test
%3f/
%C0%AE%C0%AE%C0%AF
../../../../../../etc/passwd
..;/
cgi-bin/.%2e/%2e%2e/%2e%2e/%2e%2e/etc/passwd
```

### 3.3 亮点

1. **Wildcard/软 404 检测是这个领域里"降噪"做得最深入的实现**，思路可以完整迁移到我们的场景：如果未来要对提取出的 `APIEndpoint` 做"存活验证"（第 7.1 节里提到的、独立于被动提取的能力），必须先解决"网站对不存在的接口路径统一返回 200 + 固定 JSON 结构（如 `{code: 404, msg: "not found"}`）"这种情况，否则验证结果会被大量误判为"接口存在"污染。
2. **`scope` 锚定式正则（"已知域名前缀 + 合法路径字符集"）是比"猜测式正则"更精确的一种提取策略**，在明确知道当前扫描目标是谁的场景下（我们的 `crawler` 包恰好总是知道 `baseURL`），这个思路比 JSFinder/URLFinder 的"泛化猜测"更值得借鉴，可以用来设计一条"精确锚定当前站点域名/相对路径"的高置信度补充规则。
3. **提取标签+属性用笛卡尔积覆盖（`CRAWL_TAGS × CRAWL_ATTRIBUTES`）**，比只看 `<a href>` 更全面，这个思路可以用于我们 `extract.go` 未来如果要扩展"还有哪些标签属性值得提取"时的参考基准。
4. **黑名单字典独立成外置文本文件**（`403_blacklist.txt` 这类按状态码分类的纯文本），管理和更新比正则黑名单更简单直接，适合"经验积累型"的规则（不需要正则表达能力，纯粹是关键词枚举）。

### 3.4 和本方案的关系（先说清楚：不是同一类问题）

dirsearch 的核心能力（字典驱动的主动路径爆破）和"从 JS 提取 API"是两种完全不同的信息获取方式：前者是"不知道有什么，靠字典去撞"，后者是"信息已经写在代码里，只是要抠出来"。**dirsearch 本身不适合被整体照搬进本方案**，它对我们的价值仅限于"降噪思路"和"scope 锚定提取"这两个可迁移的子能力，不构成"是否要引入字典爆破能力"的立项理由——这件事本身在我们的架构里已经有独立的 `dir_scan`（目录扫描）能力承接，不应该和"被动 JS 提取"混在一起讨论。

---

## 四、三者横向对比

| 维度 | JSFinder | URLFinder | dirsearch |
|---|---|---|---|
| 语言 | Python | Go | Python |
| 解决的问题 | 从 JS 提取 URL/子域名 | 从页面+JS 提取 URL/JS/敏感信息 | 字典驱动的路径爆破 |
| 提取方式 | 单条大正则（源自 LinkFinder） | 多条可配置正则，提取/过滤/验证三阶段分离 | 精确锚定 scope 的正则 + 结构化标签解析（辅助字典，非主提取手段） |
| 规则可配置性 | 无，硬编码在代码里 | 有，YAML 外置 | 部分（黑名单字典外置为文本文件） |
| 降噪机制 | 无 | 黑名单正则过滤 | Wildcard/软 404 检测（本领域最严谨） |
| 是否主动验证存活 | 否 | 是（可选开关 `-s`） | 是（爆破的本质就是主动验证） |
| 对危险操作的边界意识 | 无 | 有（`Risks` 黑名单跳过 delete/remove 等路径） | 有（黑名单字典机制类似） |
| 并发/性能 | 无并发，同步阻塞 | 有并发（channel 控制） | 有并发（asyncio） |
| 对本方案的核心参考点 | 提取正则的基线（四类形态覆盖） | 三阶段架构 + 规则外置 + 危险路由黑名单意识 | Wildcard 降噪思路 + scope 锚定提取 |

---

## 五、结合三者分析后，我们自己的 JS 接口发现最合适逻辑

结合 [`Web-JS接口提取方案.md`](./Web-JS接口提取方案.md) 已经定好的边界（被动提取、不主动验证、不新建模块），三个项目对我们的价值排序是：**URLFinder 的三阶段架构思路 > dirsearch 的 scope 锚定提取思路 > JSFinder 的基线正则**。落地上做以下几处针对性调整，覆盖 `Web-JS接口提取方案.md` 第四节：

### 5.1 采纳：正则分层设计，但用 dirsearch 的思路加固"低置信度"这一层

方案原有的两层设计（高置信度结构化调用 / 低置信度泛化路径）保持不变，但低置信度层新增一条参考 dirsearch `text_crawl` 思路的规则：**锚定当前页面所在域名，提取"域名 + 合法路径字符"的完整 URL**，而不是只认 `/api/`、`/v1/` 这类固定前缀。原因：我们的 `crawler` 包在处理每一段 JS 文本时，永远知道这段 JS 来自哪个 `baseURL`（`fetchAndExtract` 里明确传入 `it.URL`），具备 dirsearch 那种"精确锚定 scope"的前提条件，没有理由放弃这个天然优势去做更容易误报的泛化匹配。

新增规则示例（伪正则，具体到实施文档时细化）：`re.QuoteMeta(host) + 合法URL字符集`，命中优先级高于"泛化前缀匹配"、低于"结构化调用匹配"，作为中等置信度的一档。

### 5.2 采纳：URLFinder 的"过滤与提取解耦"结构，作为 `jsapi.go` 内部的两段式实现

不是"正则里加否定环排除噪音"，而是像 `jsFilter`/`urlFilter` 那样，提取和过滤是两个独立函数：`extractAPICandidates`（只管正则命中，不做任何过滤判断）→ `filterAPICandidates`（统一做转义清理、静态资源后缀排除、黑名单关键词排除）。这样规则演进时，"新增一条提取正则"和"新增一条排除规则"是两件互不影响的事，不需要每次都去改一条大正则的否定环。

### 5.3 采纳：URLFinder 的 `Risks` 危险路由黑名单意识，但用法不同

URLFinder 用黑名单是为了"验证状态码时跳过有副作用的路径"，我们的场景不做主动验证，所以不需要真正跳过任何路径的**提取**，但这提醒我们：**未来一旦要做本方案 7.1 节提到的"接口存活验证"独立能力时，必须内置类似的危险路由黑名单**（`delete/remove/drop/truncate` 等关键词），这个点补充进方案文档第 7.1 节作为"以后做存活验证时的前置约束"，现在不需要写代码，只需要在文档里明确记下这个心智模型，避免以后设计验证功能时重新踩一遍坑。

### 5.4 不采纳：JSFinder/URLFinder 的"无后缀裸文件名/裸路径也收"策略

两者的正则都倾向于"召回优先，误报可接受"（URLFinder README 原话）。这不适合我们的场景——我们的 `ApiResult.APIs`（详见 [`Web-JS接口提取方案.md`](./Web-JS接口提取方案.md) v2.0 第三节）是要直接呈现给用户看的攻击面清单，不是给人工二次筛选的原始素材库，噪音过多会直接损害这个功能的可用性。因此保持方案原定的"宁可漏报不要噪音"的取舍，不因为参考了 URLFinder 就放宽标准。

### 5.5 不采纳：URLFinder 的目录 Fuzz、dirsearch 的字典爆破

两者都是主动探测能力，和 `Web-JS接口提取方案.md` 第 7.1 节"被动提取到底为止，不做延伸动作"的边界冲突，明确不纳入本方案范围。若未来确实需要"根据提取到的 API 做规律性猜测补全"（比如发现 `/api/v1/user/get`，尝试猜测 `/api/v1/user/list`、`/api/v1/user/delete` 是否存在），这属于独立的、需要用户显式开启的主动能力，性质上更接近现有的 `dir_scan`，不应该在被动的 `crawler` 包里实现。

### 5.6 不采纳：dirsearch 的 Wildcard 软 404 检测（现阶段）

这是三者里工程含金量最高的降噪技术，但它服务于"主动验证响应是否可信"这个问题，我们当前的方案不做主动验证，暂时用不上。**明确记录在案**：如果未来做"接口存活验证"独立能力，Wildcard 检测是必须引入的核心机制，不这么做的话，遇到"任意路径都返回 200 + 统一错误 JSON"的网站，验证结果会产出大量假阳性，到时候应直接参照 dirsearch `scanner.py` 的 `classify`/`is_wildcard`/`is_probable_wildcard` 思路实现，不需要重新设计。

---

## 六、结论：需要同步更新到 `Web-JS接口提取方案.md` 的内容

1. 第四节"提取规则设计"补充第三种规则档位——"锚定当前域名的中等置信度规则"（来自 dirsearch 的 scope 锚定思路），置信度介于 high 和 low 之间，或者归入 `low` 但优先级更高，具体分档方式留给实施文档决定。
2. 第四节内部结构调整为"提取"与"过滤"两个独立函数（来自 URLFinder 的架构经验），而不是在正则里堆否定环。
3. 第七节"不做的事"追加一条心智模型记录：未来如果做"接口存活验证"独立能力，必须前置引入 dirsearch 式的 Wildcard/软 404 检测机制，并内置危险路由关键词黑名单（参考 URLFinder 的 `Risks`），这不是本次方案要做的事，只是提前记录避免以后重新踩坑。

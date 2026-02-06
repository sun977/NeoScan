# SweetBabyScan的Chromium下载、go-rod库使用和爬虫处理流程

## 一、Chromium自动下载实现

### 1.1 实现原理
SweetBabyScan使用[go-rod](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBrowserScan/vendor/github.com/go-rod/rod/lib/utils.go#L4-L4)库来处理浏览器自动化任务，该库会在需要时自动下载并管理Chromium浏览器实例。

### 1.2 自动下载触发机制
- 当程序调用[DoFullScreenshot](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/core/plugins/plugin_scan_site/plugin.go#L59-L69)函数时，执行`rod.New().Timeout(timeout).MustConnect()`
- 如果系统中没有合适的浏览器实例，[go-rod](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/vendor/github.com/go-rod/rod/lib/utils.go#L4-L4)库会自动下载匹配当前操作系统的Chromium浏览器
- 下载动作由[go-rod](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/vendor/github.com/go-rod/rod/lib/utils.go#L4-L4)库内部处理，封装在库的实现中

### 1.3 下载初始化代码
在[initialize_screenshot/initialize.go](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/initializes/initialize_screenshot/initialize.go)中：

```go
func Download(ctx context.Context, domain, path string) bool {
    ch := make(chan bool, 1)
    go func() {
        fmt.Println("downloading chrome headless......")
        plugin_scan_site.DoFullScreenshot(fmt.Sprintf("http://%s/", domain), path, 60*time.Second)
        ch <- true
    }()
    select {
    case <-ch:
        fmt.Println("download finish !")
        return true
    case <-ctx.Done():
        fmt.Println("download timeout !")
        return false
    }
}

func InitScreenShot() bool {
    path := "./static/ip.png"
    status, _ := utils.PathExists(path)
    if status {
        return status
    }

    domain := "myip.ipip.net"
    status = plugin_scan_host.ScanHostByPing(domain)
    if status {
        ctx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
        status = Download(ctx, domain, path)
        cancel()
    }
    return status
}
```

## 二、go-rod库的使用

### 2.1 库的作用
- 提供浏览器自动化功能
- 自动管理Chromium浏览器的下载和版本兼容
- 支持网页截图、页面交互等功能

### 2.2 在项目中的使用
在[plugin_scan_site/plugin.go](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/core/plugins/plugin_scan_site/plugin.go)中：

```go
/*执行截图*/
func DoFullScreenshot(url, path string, timeout time.Duration) bool {
    err := rod.Try(func() {
        browser := rod.New().Timeout(timeout).MustConnect()
        defer browser.MustClose()
        page := browser.MustPage(url)
        page.MustWaitLoad().MustScreenshot(path)
    })

    if err != nil {
        //fmt.Println(err)
        return false
    }

    if status, _ := utils.PathExists(path); status {
        return true
    }

    return false
}
```

## 三、爬虫处理流程

### 3.1 爬虫相关文件
- [plugin_scan_site/plugin.go](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/core/plugins/plugin_scan_site/plugin.go) - 网站扫描插件
- [task_scan_site/task.go](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/core/tasks/task_scan_site/task.go) - 网站扫描任务

### 3.2 爬虫处理流程
1. **目标获取**：获取需要爬取的URL列表
2. **浏览器初始化**：使用go-rod库连接浏览器实例
3. **页面访问**：访问目标URL
4. **页面加载等待**：等待页面完全加载
5. **内容提取**：提取页面标题、状态码等信息
6. **截图保存**：对页面进行截图并保存
7. **CMS识别**：尝试识别网站使用的CMS类型
8. **结果输出**：将结果保存到Excel或TXT文件

### 3.3 任务执行流程
在[task_scan_site/task.go](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/core/tasks/task_scan_site/task.go)中定义了爬虫的具体执行流程：

```go
func (t *taskScanSite) doTask(data models.WaitScanSite) error {
    // 1. 访问URL并获取页面信息
    title, statusCode, packetSend, packetRecv := plugin_scan_site.DoGetTitle(data.Url, t.config.Timeout)
    
    // 2. 获取网站图标并计算hash值
    iconPath := fmt.Sprintf("./static/%s.ico", data.Host)
    plugin_scan_site.DoGetIcon(fmt.Sprintf("http://%s/favicon.ico", data.Host), iconPath, t.config.Timeout)
    iconHash := utils.GetMd5ByFile(iconPath)
    
    // 3. 保存截图
    screenshotPath := fmt.Sprintf("./static/%s.png", data.Host)
    plugin_scan_site.DoFullScreenshot(data.Url, screenshotPath, t.config.Timeout)
    
    // 4. 识别CMS
    cms := plugin_scan_cms2.GetCmsByRule(title, iconHash, data.Url)
    
    // 5. 保存结果
    // ...
}
```

## 四、主要功能

### 4.1 浏览器自动化功能
- **网页截图**：对目标网站进行完整页面截图
- **页面信息提取**：获取页面标题、状态码、响应内容等
- **网站图标获取**：下载网站favicon并计算hash值用于识别

### 4.2 爬虫功能
- **批量URL处理**：支持并发处理多个目标URL
- **页面内容分析**：提取页面标题、状态码等信息
- **CMS识别**：基于页面内容和图标hash识别网站CMS类型
- **结果导出**：将爬取结果导出为Excel或TXT格式

### 4.3 自动化管理功能
- **Chromium自动下载**：根据操作系统自动下载匹配的Chromium版本
- **浏览器实例管理**：自动管理浏览器实例的创建和销毁
- **资源清理**：自动清理临时文件和浏览器实例

### 4.4 安全与稳定性
- **超时控制**：对每个操作设置超时限制
- **错误处理**：对各种异常情况进行处理
- **并发控制**：支持并发爬取，提高效率

## 五、技术特点

1. **自动化程度高**：从Chromium下载到页面处理全程自动化
2. **跨平台兼容**：支持Windows/Linux/macOS等多平台
3. **高效并发**：支持多协程并发爬取
4. **智能识别**：结合多种特征进行CMS识别
5. **结果丰富**：同时输出截图、页面信息、CMS识别结果等
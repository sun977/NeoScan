# SweetBabyScan Web扫描实现原理

## 一、概述

SweetBabyScan的Web扫描功能主要包括网站爬虫、页面截图、CMS识别等功能，它通过浏览器自动化技术来实现对Web目标的深度探测和分析。

## 二、核心组件

### 2.1 浏览器自动化引擎
- **库选择**：使用[go-rod](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBrowserScan/vendor/github.com/go-rod/rod/lib/utils.go#L4-L4)库实现浏览器自动化
- **自动下载**：自动检测并下载适配当前操作系统的Chromium浏览器
- **实例管理**：自动管理浏览器实例的生命周期，包含超时控制和资源清理

### 2.2 Web扫描插件
- **文件位置**：[core/plugins/plugin_scan_site/plugin.go](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/core/plugins/plugin_scan_site/plugin.go)
- **核心功能**：
  - 获取页面标题和状态信息
  - 下载网站图标(Favicon)并计算哈希值
  - 对网页进行全页面截图

### 2.3 Web扫描任务
- **文件位置**：[core/tasks/task_scan_site/task.go](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/core/tasks/task_scan_site/task.go)
- **任务调度**：负责管理和执行Web扫描任务

## 三、Web扫描实现流程

### 3.1 目标URL准备
- 从扫描参数中获取待扫描的IP或域名列表
- 为每个IP或域名生成HTTP和HTTPS协议的URL

### 3.2 页面信息提取
- **访问URL**：使用浏览器访问目标URL
- **等待加载**：等待页面完全加载完成
- **信息获取**：
  - 提取页面标题
  - 获取HTTP状态码
  - 记录请求和响应数据

### 3.3 网站图标处理
- **下载Favicon**：从目标网站下载favicon.ico文件
- **哈希计算**：计算图标文件的MD5哈希值
- **存储路径**：将图标保存到./static目录下

### 3.4 网页截图
- **启动浏览器**：使用[go-rod](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/vendor/github.com/go-rod/rod/lib/utils.go#L4-L4)连接浏览器实例
- **页面加载**：导航到目标URL
- **等待加载**：等待页面元素完全加载
- **截图保存**：对整个页面进行截图并保存到本地

### 3.5 CMS识别
- **识别方法**：使用[plugin_scan_cms2.GetCmsByRule](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/SweetBabyScan/core/plugins/plugin_scan_cms2/plugin.go#L28-L53)函数
- **识别依据**：基于页面标题、图标哈希值和URL进行匹配
- **规则库**：使用预定义的CMS识别规则

## 四、关键技术实现

### 4.1 页面标题获取实现
```go
func DoGetTitle(url string, timeout time.Duration) (title, statusCode, packetSend, packetRecv string) {
    // 实现页面标题和状态信息获取
}
```

### 4.2 网站截图实现
```go
func DoFullScreenshot(url, path string, timeout time.Duration) bool {
    err := rod.Try(func() {
        browser := rod.New().Timeout(timeout).MustConnect()
        defer browser.MustClose()
        page := browser.MustPage(url)
        page.MustWaitLoad().MustScreenshot(path)
    })

    if err != nil {
        return false
    }

    if status, _ := utils.PathExists(path); status {
        return true
    }

    return false
}
```

### 4.3 图标下载实现
```go
func DoGetIcon(url, path string, timeout time.Duration) bool {
    // 实现图标下载功能
}
```

## 五、并发控制机制

### 5.1 协程池管理
- **并发数控制**：通过参数`-wss`（workerScanSite）控制爬虫并发数
- **默认值**：runtime.NumCPU()*2，充分利用多核处理器性能

### 5.2 超时控制
- **单次扫描超时**：通过参数`-tss`（timeOutScanSite）控制单次扫描超时时间
- **截图超时**：通过参数`-ts`（timeOutScreen）控制截图超时时间

## 六、结果处理

### 6.1 数据保存
- **Excel导出**：将扫描结果保存到Excel文件
- **文本导出**：将结果保存到文本文件
- **截图存储**：将网页截图保存到static目录

### 6.2 信息提取
- **基础信息**：IP、端口、协议、标题、状态码
- **扩展信息**：CMS类型、网站图标、响应内容
- **网络信息**：请求包、响应包

## 七、技术特点

### 7.1 自动化部署
- 无头浏览器的全自动部署，无需外部依赖
- 自动下载和管理Chromium浏览器

### 7.2 高效识别
- 基于多特征（标题、图标哈希、URL）的CMS识别
- 支持批量URL并发处理

### 7.3 稳定可靠
- 完善的超时控制机制
- 错误处理和资源清理机制
- 防止协程泄漏的设计

### 7.4 结果丰富
- 提供页面截图
- 包含详细的状态信息
- 支持多种输出格式
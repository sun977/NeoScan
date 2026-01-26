
# ScopeSentry 项目分析文档

## 一、项目概述

ScopeSentry 是一款具有分布式资产测绘、子域名枚举、信息泄露检测、漏洞扫描、目录扫描、子域名接管、爬虫、页面监控等功能的安全扫描平台。它采用分布式架构，允许用户通过构建多个节点，自由选择节点运行扫描任务，从而在出现新漏洞时能够快速排查关注资产是否存在相关组件。

## 二、项目目录功能说明

```
ScopeSentry/
├── cmd/                    # 应用程序入口
│   └── main/               # 主程序入口
│       ├── static/         # 静态资源文件
│       │   ├── assets/     # 前端资源（JS、CSS等）
│       │   └── index.html  # 前端入口页面
│       └── main.go         # Go应用主入口文件
├── internal/               # 内部应用程序代码
│   ├── api/                # API层
│   │   ├── handlers/       # 请求处理器
│   │   │   ├── assets/     # 资产相关处理
│   │   │   ├── configuration/ # 配置相关处理
│   │   │   ├── dictionary/ # 字典相关处理
│   │   │   ├── export/     # 导出相关处理
│   │   │   ├── fingerprint/ # 指纹相关处理
│   │   │   ├── node/       # 节点相关处理
│   │   │   ├── plugin/     # 插件相关处理
│   │   │   ├── poc/        # POC相关处理
│   │   │   ├── project/    # 项目相关处理
│   │   │   ├── role/       # 角色相关处理
│   │   │   ├── sensitiveRule/ # 敏感规则相关处理
│   │   │   ├── system/     # 系统相关处理
│   │   │   ├── task/       # 任务相关处理
│   │   │   └── user/       # 用户相关处理
│   │   ├── middleware/     # 中间件
│   │   │   ├── auth.go     # 认证中间件
│   │   │   ├── cors.go     # CORS中间件
│   │   │   ├── i18n.go     # 国际化中间件
│   │   │   ├── logger.go   # 日志中间件
│   │   │   └── permission.go # 权限中间件
│   │   ├── response/       # 响应结构
│   │   │   ├── data.go     # 数据响应结构
│   │   │   └── response.go # 通用响应结构
│   │   └── routes/         # 路由定义
│   │       ├── assets/     # 资产路由
│   │       ├── common/     # 通用路由
│   │       ├── configuration/ # 配置路由
│   │       ├── dictionary/ # 字典路由
│   │       ├── export/     # 导出路由
│   │       ├── fingerprint/ # 指纹路由
│   │       ├── node/       # 节点路由
│   │       ├── plugin/     # 插件路由
│   │       ├── poc/        # POC路由
│   │       ├── project/    # 项目路由
│   │       ├── role/       # 角色路由
│   │       ├── sensitive_rule/ # 敏感规则路由
│   │       ├── system/     # 系统路由
│   │       ├── task/       # 任务路由
│   │       ├── user/       # 用户路由
│   │       └── routes.go   # 主路由配置
│   ├── bootstrap/          # 应用初始化
│   │   ├── bootstrap.go    # 启动配置
│   │   └── initdata.go     # 初始化数据
│   ├── config/             # 配置管理
│   │   └── config.go       # 全局配置
│   ├── constants/          # 常量定义
│   │   ├── assets/         # 资产相关常量
│   │   │   └── ScopeSentry.SensitiveRule.json # 敏感规则JSON
│   │   ├── defaults.go     # 默认配置
│   │   └── permissions.go  # 权限常量
│   ├── database/           # 数据库相关
│   │   ├── mongodb/        # MongoDB相关
│   │   │   ├── gridfs.go   # GridFS操作
│   │   │   ├── initdb.go   # 数据库初始化
│   │   │   └── mongodb.go  # MongoDB连接
│   │   └── redis/          # Redis相关
│   │       └── redis.go    # Redis连接
│   ├── i18n/               # 国际化
│   │   ├── locales/        # 本地化文件
│   │   │   ├── en-US.json  # 英文翻译
│   │   │   └── zh-CN.json  # 中文翻译
│   │   └── i18n.go         # 国际化配置
│   ├── logger/             # 日志系统
│   │   ├── gin.go          # Gin框架日志
│   │   └── logger.go       # 通用日志
│   ├── models/             # 数据模型
│   │   ├── app.go          # APP模型
│   │   ├── asset.go        # 资产模型
│   │   ├── common.go       # 通用模型
│   │   ├── configuration.go # 配置模型
│   │   ├── crawler.go      # 爬虫模型
│   │   ├── dictionary.go   # 字典模型
│   │   ├── dirscan.go      # 目录扫描模型
│   │   ├── export.go       # 导出模型
│   │   ├── field.go        # 字段模型
│   │   ├── fingerprint.go  # 指纹模型
│   │   ├── miniprogram.go  # 小程序模型
│   │   ├── mp.go           # 微信公众号模型
│   │   ├── node.go         # 节点模型
│   │   ├── page_monitoring.go # 页面监控模型
│   │   ├── plugin.go       # 插件模型
│   │   ├── poc.go          # POC模型
│   │   ├── project.go      # 项目模型
│   │   ├── role.go         # 角色模型
│   │   ├── root_domain.go  # 根域名模型
│   │   ├── route.go        # 路由模型
│   │   ├── scan.go         # 扫描模型
│   │   ├── search.go       # 搜索模型
│   │   ├── sensitive.go    # 敏感信息模型
│   │   ├── statistics.go   # 统计模型
│   │   ├── subdomain.go    # 子域名模型
│   │   ├── task.go         # 任务模型
│   │   ├── template.go     # 模板模型
│   │   ├── url.go          # URL模型
│   │   ├── user.go         # 用户模型
│   │   └── vulnerability.go # 漏洞模型
│   ├── repositories/       # 数据访问层
│   │   ├── assets/         # 资产数据访问
│   │   │   ├── app/        # APP数据访问
│   │   │   ├── asset/      # 资产数据访问
│   │   │   ├── common/     # 通用数据访问
│   │   │   ├── crawler/    # 爬虫数据访问
│   │   │   ├── dirscan/    # 目录扫描数据访问
│   │   │   ├── ip/         # IP数据访问
│   │   │   ├── mp/         # 小程序数据访问
│   │   │   ├── page_monitoring/ # 页面监控数据访问
│   │   │   ├── root_domain/ # 根域名数据访问
│   │   │   ├── sensitive/  # 敏感数据访问
│   │   │   ├── statistics/ # 统计数据访问
│   │   │   ├── subdomain/  # 子域名数据访问
│   │   │   ├── url/        # URL数据访问
│   │   │   └── vulnerability/ # 漏洞数据访问
│   │   ├── common/         # 通用数据访问
│   │   │   └── common.go   # 通用数据访问
│   │   ├── dictionary/     # 字典数据访问
│   │   │   ├── manage.go   # 管理字典
│   │   │   └── port.go     # 端口字典
│   │   ├── export/         # 导出数据访问
│   │   │   └── export.go   # 导出数据访问
│   │   ├── fingerprint/    # 指纹数据访问
│   │   │   └── fingerprint.go # 指纹数据访问
│   │   ├── node/           # 节点数据访问
│   │   │   └── node.go     # 节点数据访问
│   │   ├── plugin/         # 插件数据访问
│   │   │   └── plugin.go   # 插件数据访问
│   │   ├── poc/            # POC数据访问
│   │   │   └── poc.go      # POC数据访问
│   │   ├── project/        # 项目数据访问
│   │   │   └── project.go  # 项目数据访问
│   │   ├── role/           # 角色数据访问
│   │   │   └── role.go     # 角色数据访问
│   │   ├── sensitive_rule/ # 敏感规则数据访问
│   │   │   └── sensitive_rule.go # 敏感规则数据访问
│   │   ├── task/           # 任务数据访问
│   │   │   ├── scheduler/  # 任务调度器
│   │   │   ├── task/       # 任务数据访问
│   │   │   └── template/   # 任务模板
│   │   └── user/           # 用户数据访问
│   │       └── user.go     # 用户数据访问
│   ├── scheduler/          # 任务调度器
│   │   ├── jobs/           # 调度任务
│   │   │   ├── page_monitoring.go # 页面监控任务
│   │   │   └── scan.go     # 扫描任务
│   │   └── scheduler.go    # 调度器
│   ├── services/           # 业务逻辑层
│   │   ├── assets/         # 资产业务逻辑
│   │   ├── common/         # 通用业务逻辑
│   │   ├── configuration/  # 配置业务逻辑
│   │   ├── dictionary/     # 字典业务逻辑
│   │   ├── export/         # 导出业务逻辑
│   │   ├── fingerprint/    # 指纹业务逻辑
│   │   ├── node/           # 节点业务逻辑
│   │   ├── plugin/         # 插件业务逻辑
│   │   ├── poc/            # POC业务逻辑
│   │   ├── project/        # 项目业务逻辑
│   │   ├── redis_log_subscriber/ # Redis日志订阅业务逻辑
│   │   ├── scheduler/      # 调度业务逻辑
│   │   ├── sensitive_rule/ # 敏感规则业务逻辑
│   │   ├── system/         # 系统业务逻辑
│   │   ├── task/           # 任务业务逻辑
│   │   └── user/           # 用户业务逻辑
│   ├── update/             # 更新逻辑
│   │   ├── update.go       # 更新逻辑
│   │   └── update18.go     # 版本1.8更新逻辑
│   ├── utils/              # 工具函数
│   │   ├── helper/         # 辅助函数
│   │   │   ├── search.go   # 搜索辅助函数
│   │   │   └── ...         # 其他辅助函数
│   │   ├── random/         # 随机数工具
│   │   │   └── ...         # 随机数相关函数
│   │   └── filetool.go     # 文件工具
│   └── worker/             # 后台工作者
│       ├── iconhandle.go   # 图标处理
│       ├── ip_asset.go     # IP资产处理
│       ├── screenshot.go   # 截图处理
│       └── start.go        # 启动工作者
├── pkg/                    # 公共包
│   └── response/           # 通用响应
│       └── response.go     # 通用响应结构
├── README.md               # 项目说明文档
├── README_CN.md            # 中文项目说明文档
├── dockerfile              # Docker镜像构建文件
├── single-host-deployment.yml # 单主机部署配置
├── single-host-deployment-fix.yml # 修复版单主机部署配置
└── version.json            # 版本信息文件
```

## 三、项目运行原理和逻辑

### 1. 项目架构

ScopeSentry 采用分布式架构，包含三个主要部分：

- **服务端（Server）**: 使用 Python + FastAPI 构建，提供 REST API
- **扫描端（Scanner）**: 使用 Go 语言编写，独立运行并与主服务通信
- **前端界面**: 使用 Vue3 + Element Plus Admin 构建

### 2. 主要技术栈

- Go 1.23+ (扫描端)
- Python 3.7+ (服务端)
- MongoDB 4.0+ (数据存储)
- Redis 6.0+ (缓存和消息队列)
- Gin 框架 (Go Web框架)
- Docker (容器化部署)

### 3. 项目运行流程

#### 3.1 启动流程 ([main.go](file://c:\Users\root\Desktop\code\GoCode\code_03\ScopeSentry\cmd\main\main.go))

1. **初始化配置**：
   - 设置 Gin 框架运行模式（debug/release）
   - 设置系统时区
   - 输出当前版本号

2. **执行更新**：
   - 调用 [update.Update()](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/ScopeSentry/internal/update/update.go#L24-L48) 检查并更新系统

3. **初始化数据**：
   - 获取项目列表 ([bootstrap.GetProjectList()](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/ScopeSentry/internal/bootstrap/initdata.go#L17-L34))
   - 初始化全局调度器 ([scheduler.InitializeGlobalScheduler()](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/ScopeSentry/internal/scheduler/scheduler.go#L10-L15))

4. **启动服务**：
   - 启动 Redis 日志订阅服务
   - 设置路由
   - 配置静态资源和 Swagger 文档
   - 启动资产处理工作者
   - 启动 HTTP 服务器

5. **优雅关闭**：
   - 监听系统信号（SIGINT, SIGTERM）
   - 优雅关闭服务器

#### 3.2 数据流向

1. **用户通过前端 UI 创建扫描任务**
2. **后端将任务写入数据库并推送到指定节点**
3. **Go 扫描节点拉取任务并执行插件化扫描**
4. **扫描结果通过 Redis 或 HTTP 回传至主服务**
5. **主服务处理数据并存入 MongoDB**
6. **前端实时展示资产、漏洞、进度等信息**

#### 3.3 数据库初始化流程

1. **连接 MongoDB 数据库**
2. **检查数据库是否存在**
3. **创建用户集合并设置初始密码**
4. **创建配置集合并插入系统配置**
5. **创建通知配置**
6. **创建任务模板**

#### 3.4 任务调度机制

- 使用 [gocron](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/ScopeSentry/go.mod#L10-L10) 库实现定时任务
- 支持周期性页面监控
- 支持计划任务执行

#### 3.5 插件系统

- 通过插件市场动态加载扫描模块
- 支持任意工具集成
- 提供统一的插件接口

#### 3.6 认证和权限

- 使用 JWT 令牌进行身份验证
- 实现基于角色的权限控制
- 提供细粒度的权限管理

### 4. 核心功能模块

#### 4.1 资产管理
- 资产识别和分类
- 子域名枚举和接管检测
- IP 资产管理
- APP 和小程序自动收集

#### 4.2 扫描功能
- 端口扫描
- 目录扫描
- 漏洞扫描（支持 POC 导入）
- 敏感信息泄露检测
- URL 提取和爬虫

#### 4.3 监控功能
- 页面内容变更监控
- 自定义 WEB 指纹
- 资产分组管理

#### 4.4 分布式架构
- 多节点扫描
- 任务分发和负载均衡
- 节点管理和监控

### 5. 部署方式

项目提供 Docker Compose 部署方案，包含：
- MongoDB 数据库服务
- Redis 缓存服务
- ScopeSentry 主服务
- ScopeSentry 扫描节点服务

### 6. 数据存储

- **MongoDB**: 存储资产、任务、配置等结构化数据
- **Redis**: 缓存、任务队列、日志订阅、会话管理
- **GridFS**: 存储文件类数据（如截图等）

### 7. 消息传递

- 使用 Redis 的发布/订阅模式实现日志订阅
- 通过 Redis 实现任务队列
- 实现实时日志查看功能

这个项目是一个功能强大的分布式安全扫描平台，通过合理的架构设计和模块化开发，实现了多种安全扫描功能的集成和管理。
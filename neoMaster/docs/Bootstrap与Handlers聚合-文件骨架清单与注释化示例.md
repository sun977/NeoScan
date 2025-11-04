# Bootstrap 层与 Handlers 聚合 — 文件骨架清单与注释化示例

> 目的：在不改变现有代码的前提下，为 neoMaster 的“初始化职责前移（bootstrap/container 层）”与“Handlers 聚合”提供清晰的文件骨架与注释化示例，方便团队分阶段落地迁移，避免 router_manager.go 随功能膨胀。

适用范围
- 项目：NeoScan/neoMaster
- 约束：
  - Web 框架：Gin
  - 依赖管理：Go Modules
  - ORM：GORM
  - 日志：统一使用 internal/pkg/logger
  - 类型转换：统一使用 internal/pkg/convert
  - 严格遵守调用层级：Controller/Handler → Service → Repository → Database

-------------------------

一、总体目标与原则
- 职责收敛：RouterManager 仅负责“路由编排 + 中间件挂载”，不再承担“服务/仓库/工具的装配”。
- 结构化初始化：将配置、基础设施（DB/Redis/Logger）、安全组件（JWT/Password/SessionRepo）、仓库、服务、处理器、以及中间件管理器统一前移到 bootstrap 层的组合根（Composition Root）。
- 聚合处理器：以 Handlers 聚合结构向路由层暴露各领域 Handler，避免 Router 结构体字段无限增长。
- 可插拔路由：为后续引入 RouteRegistrar 接口留出结构，便于按照配置开关模块路由。
- 日志规范一致：在路由/处理器中统一采集关键字段 path/operation/option/func_name。

-------------------------

二、建议的目录与文件骨架

建议新增目录（仅文档示例，实际落地时按需创建）：

```
neoMaster/internal/app/master/bootstrap/
├── config.go        // 配置加载与校验
├── db.go            // GORM 初始化
├── redis.go         // Redis 客户端初始化
├── logger.go        // 日志管理器初始化
├── security.go      // JWT/Password/SessionRepo 初始化
├── repo.go          // 仓库（system/orchestrator/agent）初始化
├── service.go       // 服务装配（含循环依赖处理）
├── handler.go       // 处理器初始化 + Handlers 聚合构造
├── middleware.go    // 中间件管理器初始化（全局中间件）
└── container.go     // 组合根构建入口（BuildContainer）

neoMaster/internal/app/master/router/
├── handlers.go      // Handlers 聚合结构定义（仅示例，便于统一暴露）
├── registrar.go     // 路由注册器接口（RouteRegistrar）定义（为可插拔注册做准备）
├── ...              // 现有 routes_xxx.go 保持不变（admin/public/user/orchestrator/agent/health）
```

说明：
- 本文档仅提供“文件骨架与示例”，不直接修改现有代码；团队可按阶段引入相应文件并替换掉 router_manager.go 中的装配逻辑。

-------------------------

三、container.go（组合根）示例

```go
// 文件：internal/app/master/bootstrap/container.go
// 作用：统一构建项目运行所需的所有实例，并以 Container 聚合输出给入口与路由层使用。

package bootstrap

import (
    "neomaster/internal/config"
    "neomaster/internal/app/master/middleware"
    "neomaster/internal/app/master/router" // 引用 Handlers 聚合定义（示例）
    "github.com/go-redis/redis/v8"
    "gorm.io/gorm"
)

// Container 作为运行时的组合根，聚合所有初始化后的组件。
type Container struct {
    // 基础设施
    DB           *gorm.DB
    RedisClient  *redis.Client
    // 配置
    Config       *config.Config
    // 中间件管理器（全局中间件）
    Middleware   *middleware.MiddlewareManager
    // 处理器聚合（面向路由层暴露）
    Handlers     router.Handlers
}

// BuildContainer 构建组合根，内部按顺序完成所有初始化。
// 注意：这里只是结构化示例，具体实现应拆分到各 *.go 文件中。
func BuildContainer(cfg *config.Config) (*Container, error) {
    // 1. 配置已由上层加载；此处可做额外校验/默认值填充

    // 2. 初始化基础设施：DB/Redis/Logger（分别放到 db.go / redis.go / logger.go）
    db, err := InitGormDB(cfg)
    if err != nil { return nil, err }

    redisClient, err := InitRedisClient(cfg)
    if err != nil { return nil, err }

    if err := InitLogger(cfg); err != nil { return nil, err }

    // 3. 初始化安全组件：JWT/Password/SessionRepo（security.go）
    jwtManager := NewJWTManager(cfg)              // 示例函数名，实际在 security.go
    passwordMgr := NewPasswordManager(cfg)        // 示例函数名
    sessionRepo := NewSessionRepository(redisClient) // 示例函数名

    // 4. 初始化仓库（repo.go）
    sysRepos := InitSystemRepos(db)           // 用户/角色/权限等
    orchRepos := InitOrchestratorRepos(db)    // 项目/工作流/工具/规则等
    agentRepos := InitAgentRepos(db)          // Agent 相关

    // 5. 初始化服务（service.go），处理循环依赖（如 SessionService 与 JWTService）
    svc := InitServices(sysRepos, orchRepos, agentRepos, jwtManager, passwordMgr, sessionRepo)

    // 6. 初始化处理器（handler.go），并构造 Handlers 聚合
    handlers := InitHandlers(svc)

    // 7. 初始化中间件管理器（middleware.go）
    mm := InitMiddlewareManager(cfg, svc)

    return &Container{
        DB:          db,
        RedisClient: redisClient,
        Config:      cfg,
        Middleware:  mm,
        Handlers:    handlers,
    }, nil
}
```

设计要点与原因：
- 按“配置→基础设施→安全→仓库→服务→处理器→中间件”的顺序初始化，避免循环依赖并符合层级约束。
- Container 只做聚合与输出，不在此处做路由注册；路由注册仍由 router 包负责。
- 将复杂装配与依赖管理集中到 bootstrap 层，RouterManager 得以瘦身。

-------------------------

四、Handlers 聚合结构示例

```go
// 文件：internal/app/master/router/handlers.go
// 作用：统一聚合各领域 Handler，供路由层与注册器使用，避免 Router 持有大量散列字段。

package router

import (
    authHandler "neomaster/internal/handler/auth"
    systemHandler "neomaster/internal/handler/system"
    scanConfigHandler "neomaster/internal/handler/orchestrator"
    agentHandler "neomaster/internal/handler/agent"
)

// AuthHandlers 认证相关处理器聚合
type AuthHandlers struct {
    Login      *authHandler.LoginHandler
    Logout     *authHandler.LogoutHandler
    Refresh    *authHandler.RefreshHandler
    Register   *authHandler.RegisterHandler
}

// SystemHandlers 系统（用户/角色/权限/会话）相关处理器聚合
type SystemHandlers struct {
    User        *systemHandler.UserHandler
    Role        *systemHandler.RoleHandler
    Permission  *systemHandler.PermissionHandler
    Session     *systemHandler.SessionHandler
}

// OrchestratorHandlers 扫描编排器相关处理器聚合
type OrchestratorHandlers struct {
    ProjectConfig *scanConfigHandler.ProjectConfigHandler
    Workflow      *scanConfigHandler.WorkflowHandler
    ScanTool      *scanConfigHandler.ScanToolHandler
    ScanRule      *scanConfigHandler.ScanRuleHandler
    RuleEngine    *scanConfigHandler.RuleEngineHandler
}

// AgentHandlers Agent 管理相关处理器聚合
type AgentHandlers struct {
    Agent *agentHandler.AgentHandler
}

// Handlers 顶层聚合，面向 Router 暴露
type Handlers struct {
    Auth         AuthHandlers
    System       SystemHandlers
    Orchestrator OrchestratorHandlers
    Agent        AgentHandlers
}
```

设计要点与原因：
- 通过领域聚合减少 RouterManager 的字段数量，新增模块只需扩展聚合结构，不影响已有路由编排。
- 命名保持与现有 handler 包一致，降低认知负担。

-------------------------

五、handler.go（处理器初始化）示例

```go
// 文件：internal/app/master/bootstrap/handler.go
// 作用：用服务实例化各 Handler，并返回 Handlers 聚合。

package bootstrap

import (
    authHandler "neomaster/internal/handler/auth"
    systemHandler "neomaster/internal/handler/system"
    scanConfigHandler "neomaster/internal/handler/orchestrator"
    agentHandler "neomaster/internal/handler/agent"
    "neomaster/internal/app/master/router"
)

// Services 聚合（示例结构）：实际在 service.go 内构造并传入本方法
type Services struct {
    // Auth
    SessionSvc   interface{} // 替换为具体类型
    UserSvc      interface{}
    RoleSvc      interface{}
    PermissionSvc interface{}
    RBACSvc      interface{}
    JWTService   interface{}
    PasswordSvc  interface{}
    // Orchestrator
    ProjectConfigSvc interface{}
    WorkflowSvc      interface{}
    ScanToolSvc      interface{}
    ScanRuleSvc      interface{}
    // Agent
    AgentManagerSvc  interface{}
    AgentMonitorSvc  interface{}
    AgentConfigSvc   interface{}
    AgentTaskSvc     interface{}
}

// InitHandlers 使用服务构造各领域 Handler，并聚合为 router.Handlers。
func InitHandlers(svc *Services) router.Handlers {
    // 认证与用户体系
    login := authHandler.NewLoginHandler(svc.SessionSvc)
    logout := authHandler.NewLogoutHandler(svc.SessionSvc)
    refresh := authHandler.NewRefreshHandler(svc.SessionSvc)
    register := authHandler.NewRegisterHandler(svc.UserSvc)

    user := systemHandler.NewUserHandler(svc.UserSvc, svc.PasswordSvc)
    role := systemHandler.NewRoleHandler(svc.RoleSvc)
    permission := systemHandler.NewPermissionHandler(svc.PermissionSvc)
    session := systemHandler.NewSessionHandler(svc.SessionSvc)

    // 扫描编排器
    project := scanConfigHandler.NewProjectConfigHandler(svc.ProjectConfigSvc)
    workflow := scanConfigHandler.NewWorkflowHandler(svc.WorkflowSvc)
    scanTool := scanConfigHandler.NewScanToolHandler(svc.ScanToolSvc)
    scanRule := scanConfigHandler.NewScanRuleHandler(*svc.ScanRuleSvc) // 注意：示例沿用现有签名
    ruleEngine := scanConfigHandler.NewRuleEngineHandler(nil, svc.ScanRuleSvc)

    // Agent 管理
    agent := agentHandler.NewAgentHandler(
        svc.AgentManagerSvc,
        svc.AgentMonitorSvc,
        svc.AgentConfigSvc,
        svc.AgentTaskSvc,
    )

    return router.Handlers{
        Auth: router.AuthHandlers{
            Login:    login,
            Logout:   logout,
            Refresh:  refresh,
            Register: register,
        },
        System: router.SystemHandlers{
            User:       user,
            Role:       role,
            Permission: permission,
            Session:    session,
        },
        Orchestrator: router.OrchestratorHandlers{
            ProjectConfig: project,
            Workflow:      workflow,
            ScanTool:      scanTool,
            ScanRule:      scanRule,
            RuleEngine:    ruleEngine,
        },
        Agent: router.AgentHandlers{
            Agent: agent,
        },
    }
}
```

设计要点与原因：
- 处理器的构造统一集中，路由层只接收聚合结果，避免 RouterManager 被“处理器列表”拖长。
- 保持现有 handler 的构造顺序与依赖一致，迁移时不改变逻辑与签名。

-------------------------

六、middleware.go（中间件管理器初始化）示例

```go
// 文件：internal/app/master/bootstrap/middleware.go
// 作用：统一初始化 MiddlewareManager，并约定好“全局中间件”与“分组中间件”的边界。

package bootstrap

import (
    "neomaster/internal/app/master/middleware"
    "neomaster/internal/config"
)

// InitMiddlewareManager 根据安全配置等，初始化中间件管理器。
func InitMiddlewareManager(cfg *config.Config, svc *Services) *middleware.MiddlewareManager {
    // 注意：实际 manager 内部可能依赖 Session/JWT/RBAC 等服务
    mm := middleware.NewMiddlewareManager(
        svc.SessionSvc,
        svc.RBACSvc,
        svc.JWTService,
        &cfg.Security,
    )
    return mm
}
```

设计要点与原因：
- 将中间件管理器的构造抽离，路由层只调用 mm.GinCORSMiddleware() 等方法挂载，职责清晰。
- 与“分组中间件”（例如管理员权限校验）在各 routes_xxx.go 或 registrar 内挂载，避免所有校验都堆在 RouterManager。

-------------------------

七、RouteRegistrar 接口（预留）示例

```go
// 文件：internal/app/master/router/registrar.go
// 作用：定义统一的模块路由注册接口，便于将来按配置开关模块、版本化路由。

package router

import (
    "github.com/gin-gonic/gin"
    "neomaster/internal/app/master/middleware"
)

// RouteRegistrar 路由注册器接口
type RouteRegistrar interface {
    Name() string
    Register(group *gin.RouterGroup, h Handlers, mm *middleware.MiddlewareManager)
}

// 示例：Auth 模块注册器（仅结构示例）
type authRegistrar struct{}

func (r *authRegistrar) Name() string { return "auth" }
func (r *authRegistrar) Register(group *gin.RouterGroup, h Handlers, mm *middleware.MiddlewareManager) {
    // 公共路由
    group.POST("/auth/register", h.Auth.Register.Register)
    group.POST("/auth/login",    h.Auth.Login.Login)

    // 需要认证的路由（示例：在子分组挂载 JWT 校验/密码版本校验等中间件）
    auth := group.Group("/auth")
    // 示例：auth.Use(mm.GinJWTAuthMiddleware()) // 具体函数名以实际中间件实现为准
    auth.POST("/refresh", h.Auth.Refresh.Refresh)
    auth.POST("/logout",  h.Auth.Logout.Logout)
}
```

设计要点与原因：
- Registrar 让 RouterManager 通过循环调用完成注册，减少枚举调用，便于开关模块。
- 分组挂载中间件，明确全局与模块的边界，避免误用。

-------------------------

八、入口 main.go 使用示例（仅示例）

```go
// 文件：neoMaster/cmd/master/main.go
// 作用：加载配置 → 构建组合根 → 创建 Router → 注册路由 → 启动服务

package main

import (
    "neomaster/internal/config"
    "neomaster/internal/app/master/bootstrap"
    "neomaster/internal/app/master/router"
)

func main() {
    // 1) 加载配置（示例：实际在 bootstrap/config.go 内封装）
    cfg := config.LoadOrDie()

    // 2) 构建组合根
    container, err := bootstrap.BuildContainer(cfg)
    if err != nil { panic(err) }

    // 3) 创建 Router（示例；实际 router.NewRouter 只接收 Handlers 与 Middleware）
    r := router.NewRouter(container.Handlers, container.Middleware, cfg)
    r.SetupRoutes()

    // 4) 启动服务（省略：端口/优雅关闭等）
}
```

设计要点与原因：
- 入口仅负责“顺序调用”，组合根负责“装配”，路由层负责“编排”。职责分离，提升可维护性。

-------------------------

九、日志与审计字段统一采集建议
- 在中间件 logging.go 或具体 handler 内，统一使用 internal/pkg/logger，记录以下字段：
  - path：c.Request.URL.String()
  - operation：如 login/register/workflow.create 等
  - option：具体步骤，如 userService.Register
  - func_name：路径化函数名，如 handler.auth.register.Register（“.” 表示包含关系）
- 原因：统一的审计与问题定位标准，便于跨模块检索与统计。

-------------------------

十、TrustedProxies 与真实 Client IP 获取
- 在 Gin 引擎初始化时设置 engine.SetTrustedProxies（或 Gin 版本对应方式），确保在反向代理（Nginx/K8s ingress）下 utils.GetClientIP(c) 能获取到真实 IP（X-Forwarded-For / X-Real-IP）。
- 建议在 bootstrap/server.go 或 router 初始化处统一设置，并以配置驱动。

-------------------------

十一、测试策略与目录约定
- 单元测试优先：为各模块路由注册编写表驱动测试，验证以下要点：
  - 路由存在性（如 GET/POST /api/v1/... 能被命中）
  - 鉴权链条（需要认证的路由必须触发相应中间件）
  - 权限链条（管理员路由必须触发 RBAC 校验）
  - 健康路由不需要鉴权
- 目录约定：neoMaster/test/日期，例如 neoMaster/test/20251104/20251104_router_registrar_test.go
- 原因：迁移过程中保障行为不变，降低回归风险。

-------------------------

十二、渐进式迁移路线（不改变代码逻辑的前提下）
- 第1步：新增 bootstrap 目录与 container.go（空实现或最小实现），在文档层面明确职责，不影响编译。
  - 原因：先固化结构与职责边界，给团队统一认知。
- 第2步：新增 router/handlers.go，定义 Handlers 聚合结构；router_manager.go 暂不改动，仅为后续迁移做准备。
  - 原因：减少 Router 字段增长的趋势，形成统一暴露的处理器集合。
- 第3步：新增 router/registrar.go，定义 RouteRegistrar 接口；现有 routes_xxx.go 保持不变。
  - 原因：为未来的“可插拔路由注册”留出抽象层，不立即改造现有注册方式。
- 第4步：在 bootstrap 下补齐 config/db/redis/logger/security/repo/service/handler/middleware 的初始化函数，实现真正的装配迁移。
  - 原因：彻底将装配从 RouterManager 移走，但保证外部行为不变。
- 第5步：为各模块补充路由与中间件链条的单元测试，覆盖核心路径与边界情况。
  - 原因：以测试兜底，防止迁移回归。

-------------------------

十三、常见问题（FAQ）
1) 初始化完成后，所有初始化代码在哪里？
   - 统一前移至 neoMaster/internal/app/master/bootstrap（组合根），入口 main.go 顺序调用，RouterManager 不再承载装配逻辑。
2) 如果某个模块暂时不需要启用，如何控制？
   - 在 RouterConfig/MiddlewareConfig 中引入开关，并在 RouteRegistrar 选择性注册对应模块路由。
3) 会不会影响现有路由定义或路径？
   - 不会。迁移过程中严格保持现有路由路径与处理逻辑，只改变“初始化发生的位置”和“注册调用方式”。
4) 循环依赖如何处理？
   - 在 service.go 中处理：先构造 SessionService，再构造 JWTService，最后通过 SetTokenGenerator 解决循环引用。

-------------------------

十四、总结
- 通过“bootstrap 组合根 + Handlers 聚合 + （预留）RouteRegistrar”三件套，router_manager.go 将回归到仅负责路由编排与中间件挂载的职责，避免随功能无限膨胀。
- 本文档提供了文件骨架与注释化示例，团队可按阶段逐步落地，过程中以测试兜底，确保不改变现有行为与对外接口。

（本文档仅为结构设计与示例，暂不修改现有源代码）
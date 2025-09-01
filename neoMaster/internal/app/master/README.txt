# app.go
          
文件是 NeoMaster 应用程序的核心应用层文件，它的主要作用包括：

## 核心功能

### 1. **应用程序生命周期管理**
- 定义了 `App` 结构体，作为整个应用程序的主要容器
- 提供 `Start()` 和 `Stop()` 方法来管理应用程序的启动和停止
- 负责优雅地关闭数据库和Redis连接

### 2. **依赖注入和初始化**
- `NewApp()` 函数负责创建和初始化所有核心组件：
  - 加载配置文件（支持不同环境）
  - 初始化MySQL数据库连接
  - 初始化Redis连接
  - 创建和配置路由器

### 3. **组件集成**
- 将数据库、Redis、配置和路由器等组件整合到一个统一的应用程序实例中
- 提供 `GetRouter()` 和 `GetConfig()` 方法供外部访问核心组件

### 4. **错误处理和容错机制**
- 对数据库和Redis连接失败采用容错处理
- 在开发阶段允许部分组件连接失败但应用程序仍能启动
- 记录警告日志而不是直接终止程序

## 架构作用

这个文件实现了**应用程序层**的核心职责，是整个 NeoMaster 系统的入口点和协调中心。它遵循了依赖注入模式，将各个组件的初始化和生命周期管理集中在一处，使得系统架构更加清晰和可维护。

通过这种设计，main.go 只需要创建 App 实例并启动，而所有复杂的初始化逻辑都封装在 app.go 中。
        



# middleware.go 
文件是 NeoMaster 系统的**HTTP中间件管理中心**，负责处理所有HTTP请求的预处理和后处理逻辑。

## 核心作用

### 1. **认证与授权管理**
- **JWT认证中间件**：验证用户身份，提取用户信息
- **权限验证中间件**：检查用户是否具有特定权限
- **角色验证中间件**：验证用户角色（如管理员权限）
- **用户状态中间件**：确保用户账户处于活跃状态
- **密码版本验证**：确保密码修改后旧token失效

### 2. **安全防护**
- **CORS中间件**：处理跨域请求
- **安全头中间件**：设置各种安全HTTP头
- **限流中间件**：防止API滥用（待实现）

### 3. **系统功能**
- **日志中间件**：记录请求信息
- **响应处理**：统一错误响应格式
- **中间件链管理**：组合不同中间件形成处理链

### 4. **框架适配**
- 同时支持标准HTTP和Gin框架
- 提供两套中间件实现（标准HTTP和Gin适配器）

## 后续添加新模块功能的修改方式

### 1. **添加新的认证/授权中间件**
```go
// 在MiddlewareManager中添加新方法
func (m *MiddlewareManager) GinNewFeatureMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 新功能的验证逻辑
        c.Next()
    }
}
```

### 2. **扩展权限检查**
- 在现有的 `RequirePermission` 和 `RequireRole` 基础上添加新的权限类型
- 修改 `rbacService` 调用以支持新的权限模型

### 3. **添加新的安全中间件**
```go
// 例如：API版本控制中间件
func (m *MiddlewareManager) GinAPIVersionMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 检查API版本兼容性
        c.Next()
    }
}
```

### 4. **更新中间件链**
在相应的中间件链方法中添加新中间件：
```go
func (m *MiddlewareManager) AuthChain() *Chain {
    return m.PublicChain().Append(
        m.JWTAuthMiddleware,
        m.RequireActiveUser,
        m.GinNewFeatureMiddleware, // 新增
    )
}
```

### 5. **依赖注入更新**
如果新功能需要额外的服务依赖：
```go
type MiddlewareManager struct {
    sessionService *auth.SessionService
    rbacService    *auth.RBACService
    jwtService     *auth.JWTService
    newService     *NewService // 新增服务
}
```

### 6. **路由集成**
router.go中应用新中间件到相应的路由组。

## 最佳实践建议

1. **保持单一职责**：每个中间件只处理一个特定功能
2. **错误处理统一**：使用现有的错误响应格式
3. **性能考虑**：避免在中间件中进行重复的数据库查询
4. **测试覆盖**：为新中间件编写单元测试
5. **文档更新**：在API文档中说明新的认证/授权要求

这个文件是系统安全和功能扩展的核心，所有新模块的HTTP层安全控制都应该通过这里进行管理。
        

# router.go

文件是 NeoMaster 系统的**HTTP路由管理中心**，负责定义和组织所有API端点的路由规则和访问控制。

## 核心作用

### 1. **路由架构管理**
- **API版本控制**：通过 `/api/v1` 前缀实现版本化管理
- **路由分组**：按功能模块组织路由（认证、用户、管理员、健康检查）
- **中间件集成**：统一应用安全、认证、日志等中间件

### 2. **访问控制分层**
- **公共路由**：无需认证的接口（登录、刷新token等）
- **认证路由**：需要JWT认证的用户接口
- **管理员路由**：需要管理员权限的管理接口

### 3. **依赖注入和初始化**
- 集成数据库、Redis、JWT等核心组件
- 初始化各种服务（认证、RBAC、会话管理）
- 创建和配置处理器（登录、登出、刷新）

### 4. **当前API端点结构**
```
/api/v1/
├── auth/          # 认证相关
│   ├── login      # 登录
│   ├── logout     # 登出
│   ├── refresh    # 刷新token
│   └── ...
├── user/          # 用户功能
│   ├── profile    # 用户资料
│   ├── permissions # 用户权限
│   └── ...
├── admin/         # 管理功能
│   ├── users/     # 用户管理
│   ├── roles/     # 角色管理
│   ├── permissions/ # 权限管理
│   └── sessions/  # 会话管理
└── health         # 健康检查
```

## 后续新增模块的使用方式

### 1. **添加新的功能模块路由组**
```go
// 在 setupAuthRoutes 或 setupAdminRoutes 中添加
func (r *Router) setupAuthRoutes(v1 *gin.RouterGroup) {
    // 现有代码...
    
    // 新增模块路由组
    newModule := v1.Group("/new-module")
    newModule.Use(r.middlewareManager.GinJWTAuthMiddleware())
    newModule.Use(r.middlewareManager.GinUserActiveMiddleware())
    {
        newModule.GET("/list", r.listNewModuleItems)
        newModule.POST("/create", r.createNewModuleItem)
        newModule.GET("/:id", r.getNewModuleItem)
        newModule.PUT("/:id", r.updateNewModuleItem)
        newModule.DELETE("/:id", r.deleteNewModuleItem)
    }
}
```

### 2. **添加处理器方法**
```go
// 在文件末尾添加新的处理器方法
func (r *Router) listNewModuleItems(c *gin.Context) {
    // TODO: 实现新模块列表功能
    c.JSON(http.StatusOK, gin.H{"message": "list new module items - not implemented yet"})
}

func (r *Router) createNewModuleItem(c *gin.Context) {
    // TODO: 实现新模块创建功能
    c.JSON(http.StatusOK, gin.H{"message": "create new module item - not implemented yet"})
}
```

### 3. **集成新的服务和处理器**
在 `NewRouter` 函数中：
```go
func NewRouter(db *gorm.DB, redisClient *redis.Client, jwtSecret string) *Router {
    // 现有初始化代码...
    
    // 初始化新模块的服务
    newModuleService := newmodule.NewService(db, redisClient)
    
    // 初始化新模块的处理器
    newModuleHandler := newmodule.NewHandler(newModuleService)
    
    return &Router{
        engine:            engine,
        middlewareManager: middlewareManager,
        // 现有处理器...
        newModuleHandler:  newModuleHandler, // 新增
    }
}
```

### 4. **权限控制集成**
为新模块添加特定权限要求：
```go
// 需要特定权限的路由
newModule.GET("/sensitive", 
    r.middlewareManager.GinRequirePermission("newmodule:read"),
    r.getSensitiveData)

// 需要特定角色的路由
newModule.POST("/admin-only",
    r.middlewareManager.GinRequireRole("module-admin"),
    r.adminOnlyAction)
```

### 5. **健康检查扩展**
在 `readinessCheck` 中添加新模块的健康检查：
```go
func (r *Router) readinessCheck(c *gin.Context) {
    // 检查新模块依赖的服务状态
    // 例如：检查新模块的数据库连接、外部API等
    c.JSON(http.StatusOK, gin.H{
        "status":    "ready",
        "timestamp": time.Now(),
        "modules":   []string{"auth", "user", "admin", "new-module"},
    })
}
```

## 最佳实践建议

1. **模块化组织**：每个功能模块使用独立的路由组
2. **中间件分层**：根据安全级别应用不同的中间件组合
3. **版本控制**：新功能优先在当前版本实现，重大变更考虑新版本
4. **错误处理**：使用统一的错误响应格式
5. **文档同步**：更新API文档和Postman集合
6. **测试覆盖**：为新路由编写集成测试

这个文件是系统HTTP层的核心，所有新模块的Web API都需要通过这里进行路由注册和访问控制配置。
        
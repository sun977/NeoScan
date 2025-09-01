# NeoScan Master API 接口文档 v2.0

## 📋 版本更新说明

**版本**: v2.0  
**更新日期**: 2025-09-01  
**主要变更**:
- 重构用户服务架构：UserService功能合并到UserRepository
- 优化数据库表结构：统一使用自增主键，移除复合主键
- 增强测试覆盖：完整的单元测试和集成测试
- 改进权限管理：优化角色权限关联表结构
- 提升性能：数据库索引优化和查询性能提升

## 🌐 服务器信息

- **基础URL**: `http://localhost:8123`
- **API版本**: v1
- **认证方式**: JWT Bearer Token
- **内容类型**: `application/json`
- **服务器版本**: NeoScan Master v2.0
- **数据库**: MySQL 8.0+ (UTF8MB4编码)
- **缓存**: Redis 6.0+

## 📊 通用响应格式

### 成功响应
```json
{
  "success": true,
  "message": "操作成功",
  "data": {},
  "timestamp": "2025-09-01T12:00:00Z"
}
```

### 错误响应
```json
{
  "success": false,
  "error": "错误代码",
  "message": "错误描述",
  "timestamp": "2025-09-01T12:00:00Z",
  "details": "详细错误信息（开发模式）"
}
```

### 分页响应
```json
{
  "success": true,
  "data": {
    "items": [],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 100,
      "pages": 10
    }
  }
}
```

## 🏥 健康检查接口

### 1. 健康检查
- **URL**: `/api/health`
- **方法**: `GET`
- **描述**: 检查服务器健康状态
- **认证**: 无需认证

**响应示例**:
```json
{
  "status": "healthy",
  "timestamp": "2025-09-01T12:00:00Z",
  "version": "v2.0",
  "uptime": "24h30m15s"
}
```

### 2. 就绪检查
- **URL**: `/api/ready`
- **方法**: `GET`
- **描述**: 检查服务器就绪状态（数据库、Redis连接）
- **认证**: 无需认证

**响应示例**:
```json
{
  "status": "ready",
  "timestamp": "2025-09-01T12:00:00Z",
  "services": {
    "database": "connected",
    "redis": "connected"
  }
}
```

### 3. 存活检查
- **URL**: `/api/live`
- **方法**: `GET`
- **描述**: 检查服务器存活状态
- **认证**: 无需认证

**响应示例**:
```json
{
  "status": "alive",
  "timestamp": "2025-09-01T12:00:00Z"
}
```

## 🔐 认证接口

### 1. 用户登录
- **URL**: `/api/v1/auth/login`
- **方法**: `POST`
- **描述**: 用户登录获取JWT令牌
- **认证**: 无需认证
- **限流**: 5次/分钟

**请求参数**:
```json
{
  "username": "用户名或邮箱",
  "password": "密码",
  "remember_me": false
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "登录成功",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 86400,
    "refresh_expires_in": 604800,
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "nickname": "管理员",
      "is_active": true,
      "last_login_at": "2025-09-01T12:00:00Z",
      "roles": ["admin"],
      "permissions": ["user:read", "user:write", "admin:all"]
    }
  }
}
```

**错误响应**:
```json
{
  "success": false,
  "error": "INVALID_CREDENTIALS",
  "message": "用户名或密码错误"
}
```

### 2. 获取登录表单
- **URL**: `/api/v1/auth/login`
- **方法**: `GET`
- **描述**: 获取登录表单页面（HTML）
- **认证**: 无需认证

### 3. 刷新令牌
- **URL**: `/api/v1/auth/refresh`
- **方法**: `POST`
- **描述**: 使用刷新令牌获取新的访问令牌
- **认证**: 无需认证

**请求参数**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 86400
  }
}
```

### 4. 从请求头刷新令牌
- **URL**: `/api/v1/auth/refresh-header`
- **方法**: `POST`
- **描述**: 从Authorization头刷新令牌
- **认证**: Bearer Token (Refresh Token)

**请求头**:
```
Authorization: Bearer <refresh_token>
```

### 5. 检查令牌过期时间
- **URL**: `/api/v1/auth/check-expiry`
- **方法**: `POST`
- **描述**: 检查令牌过期时间
- **认证**: 无需认证

**请求参数**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "expires_at": "2025-09-02T12:00:00Z",
    "expires_in": 86400,
    "is_expired": false
  }
}
```

## 🔒 用户认证接口（需要JWT认证）

> **注意**: 以下接口需要在请求头中包含有效的JWT令牌
> 
> **请求头格式**:
> ```
> Authorization: Bearer <access_token>
> ```

### 1. 用户登出
- **URL**: `/api/v1/auth/logout`
- **方法**: `POST`
- **描述**: 用户登出，使当前令牌失效
- **认证**: Bearer Token

**响应示例**:
```json
{
  "success": true,
  "message": "登出成功"
}
```

### 2. 用户全部登出
- **URL**: `/api/v1/auth/logout-all`
- **方法**: `POST`
- **描述**: 用户全部设备登出，使所有令牌失效
- **认证**: Bearer Token

**响应示例**:
```json
{
  "success": true,
  "message": "已从所有设备登出"
}
```

## 👤 用户信息接口（需要JWT认证）

### 1. 获取当前用户信息
- **URL**: `/api/v1/user/profile`
- **方法**: `GET`
- **描述**: 获取当前登录用户的详细信息
- **认证**: Bearer Token

**响应示例**:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "nickname": "管理员",
    "avatar": "",
    "phone": "",
    "remark": "系统管理员",
    "is_active": true,
    "password_version": 1,
    "last_login_at": "2025-09-01T12:00:00Z",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-09-01T12:00:00Z",
    "roles": [
      {
        "id": 1,
        "name": "admin",
        "display_name": "管理员",
        "description": "系统管理员角色"
      }
    ],
    "permissions": [
      {
        "id": 1,
        "name": "user:read",
        "display_name": "用户查看",
        "resource": "user",
        "action": "read"
      }
    ]
  }
}
```

### 2. 修改用户密码
- **URL**: `/api/v1/user/change-password`
- **方法**: `POST`
- **描述**: 修改当前用户密码
- **认证**: Bearer Token

**请求参数**:
```json
{
  "old_password": "旧密码",
  "new_password": "新密码",
  "confirm_password": "确认新密码"
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "密码修改成功",
  "data": {
    "password_version": 2,
    "updated_at": "2025-09-01T12:00:00Z"
  }
}
```

### 3. 更新用户信息
- **URL**: `/api/v1/user/profile`
- **方法**: `PUT`
- **描述**: 更新当前用户的基本信息
- **认证**: Bearer Token

**请求参数**:
```json
{
  "nickname": "新昵称",
  "email": "new@example.com",
  "phone": "13800138000",
  "avatar": "头像URL"
}
```

### 4. 获取用户权限
- **URL**: `/api/v1/user/permissions`
- **方法**: `GET`
- **描述**: 获取当前用户的权限列表
- **认证**: Bearer Token

**响应示例**:
```json
{
  "success": true,
  "data": {
    "permissions": [
      {
        "id": 1,
        "name": "user:read",
        "display_name": "用户查看",
        "description": "查看用户信息的权限",
        "resource": "user",
        "action": "read"
      }
    ]
  }
}
```

### 5. 获取用户角色
- **URL**: `/api/v1/user/roles`
- **方法**: `GET`
- **描述**: 获取当前用户的角色列表
- **认证**: Bearer Token

**响应示例**:
```json
{
  "success": true,
  "data": {
    "roles": [
      {
        "id": 1,
        "name": "admin",
        "display_name": "管理员",
        "description": "系统管理员角色",
        "created_at": "2025-01-01T00:00:00Z"
      }
    ]
  }
}
```

## 👨‍💼 管理员接口（需要管理员权限）

> **注意**: 以下接口需要管理员权限，请确保JWT令牌对应的用户具有管理员角色

### 用户管理

#### 1. 获取用户列表
- **URL**: `/api/v1/admin/users/list`
- **方法**: `GET`
- **描述**: 获取所有用户列表（支持分页和搜索）
- **认证**: Bearer Token (管理员)
- **权限**: `user:read`

**查询参数**:
- `page`: 页码（可选，默认1）
- `limit`: 每页数量（可选，默认10，最大100）
- `search`: 搜索关键词（可选，支持用户名、邮箱、昵称）
- `status`: 用户状态（可选，active/inactive）
- `role`: 角色筛选（可选）
- `sort`: 排序字段（可选，id/username/created_at）
- `order`: 排序方向（可选，asc/desc，默认desc）

**响应示例**:
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "username": "admin",
        "email": "admin@example.com",
        "nickname": "管理员",
        "is_active": true,
        "last_login_at": "2025-09-01T12:00:00Z",
        "created_at": "2025-01-01T00:00:00Z",
        "roles": ["admin"]
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 1,
      "pages": 1
    }
  }
}
```

#### 2. 创建用户
- **URL**: `/api/v1/admin/users/create`
- **方法**: `POST`
- **描述**: 创建新用户
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`

**请求参数**:
```json
{
  "username": "新用户名",
  "email": "user@example.com",
  "password": "用户密码",
  "nickname": "用户昵称",
  "phone": "手机号码",
  "remark": "备注信息",
  "is_active": true,
  "roles": ["user"]
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "用户创建成功",
  "data": {
    "id": 2,
    "username": "newuser",
    "email": "user@example.com",
    "created_at": "2025-09-01T12:00:00Z"
  }
}
```

#### 3. 获取用户详情
- **URL**: `/api/v1/admin/users/{id}`
- **方法**: `GET`
- **描述**: 获取指定用户的详细信息
- **认证**: Bearer Token (管理员)
- **权限**: `user:read`

**路径参数**:
- `id`: 用户ID

#### 4. 更新用户信息
- **URL**: `/api/v1/admin/users/{id}`
- **方法**: `PUT`
- **描述**: 更新指定用户的信息
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`

**请求参数**:
```json
{
  "email": "newemail@example.com",
  "nickname": "新昵称",
  "phone": "新手机号",
  "remark": "新备注",
  "is_active": true
}
```

#### 5. 删除用户
- **URL**: `/api/v1/admin/users/{id}`
- **方法**: `DELETE`
- **描述**: 删除指定用户（软删除）
- **认证**: Bearer Token (管理员)
- **权限**: `user:delete`

#### 6. 重置用户密码
- **URL**: `/api/v1/admin/users/{id}/reset-password`
- **方法**: `POST`
- **描述**: 重置指定用户的密码
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`

**请求参数**:
```json
{
  "new_password": "新密码"
}
```

#### 7. 管理用户角色
- **URL**: `/api/v1/admin/users/{id}/roles`
- **方法**: `PUT`
- **描述**: 更新用户的角色分配
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`, `role:assign`

**请求参数**:
```json
{
  "roles": ["user", "editor"]
}
```

### 角色管理

#### 1. 获取角色列表
- **URL**: `/api/v1/admin/roles/list`
- **方法**: `GET`
- **描述**: 获取所有角色列表
- **认证**: Bearer Token (管理员)
- **权限**: `role:read`

#### 2. 创建角色
- **URL**: `/api/v1/admin/roles/create`
- **方法**: `POST`
- **描述**: 创建新角色
- **认证**: Bearer Token (管理员)
- **权限**: `role:write`

**请求参数**:
```json
{
  "name": "editor",
  "display_name": "编辑员",
  "description": "内容编辑角色",
  "permissions": ["content:read", "content:write"]
}
```

### 权限管理

#### 1. 获取权限列表
- **URL**: `/api/v1/admin/permissions/list`
- **方法**: `GET`
- **描述**: 获取所有权限列表
- **认证**: Bearer Token (管理员)
- **权限**: `permission:read`

**响应示例**:
```json
{
  "success": true,
  "data": {
    "permissions": [
      {
        "id": 1,
        "name": "user:read",
        "display_name": "用户查看",
        "description": "查看用户信息的权限",
        "resource": "user",
        "action": "read",
        "status": "active"
      }
    ]
  }
}
```

## 📊 系统统计接口

### 1. 系统概览
- **URL**: `/api/v1/admin/dashboard/overview`
- **方法**: `GET`
- **描述**: 获取系统概览统计信息
- **认证**: Bearer Token (管理员)
- **权限**: `system:read`

**响应示例**:
```json
{
  "success": true,
  "data": {
    "users": {
      "total": 100,
      "active": 95,
      "new_today": 5
    },
    "system": {
      "uptime": "24h30m15s",
      "version": "v2.0",
      "database_status": "healthy",
      "redis_status": "healthy"
    }
  }
}
```

## 🚨 错误代码说明

| 错误代码 | HTTP状态码 | 描述 |
|---------|-----------|------|
| `INVALID_REQUEST` | 400 | 请求参数错误 |
| `UNAUTHORIZED` | 401 | 未授权访问 |
| `FORBIDDEN` | 403 | 权限不足 |
| `NOT_FOUND` | 404 | 资源不存在 |
| `CONFLICT` | 409 | 资源冲突 |
| `RATE_LIMITED` | 429 | 请求频率限制 |
| `INTERNAL_ERROR` | 500 | 服务器内部错误 |
| `INVALID_CREDENTIALS` | 401 | 用户名或密码错误 |
| `TOKEN_EXPIRED` | 401 | 令牌已过期 |
| `TOKEN_INVALID` | 401 | 令牌无效 |
| `USER_INACTIVE` | 403 | 用户已被禁用 |
| `PASSWORD_WEAK` | 400 | 密码强度不足 |
| `EMAIL_EXISTS` | 409 | 邮箱已存在 |
| `USERNAME_EXISTS` | 409 | 用户名已存在 |

## 🔧 开发者信息

### 数据库架构更新 (v2.0)

1. **统一主键设计**：所有表使用自增`id`作为主键
2. **优化关联表**：`user_roles`和`role_permissions`表结构优化
3. **索引优化**：添加复合索引提升查询性能
4. **字段标准化**：统一时间戳字段和状态字段

### 架构改进 (v2.0)

1. **服务层重构**：UserService功能合并到UserRepository
2. **测试覆盖**：完整的单元测试和集成测试
3. **错误处理**：统一的错误响应格式
4. **性能优化**：数据库连接池和查询优化
5. **安全增强**：密码版本控制和令牌管理

### 测试环境

- **单元测试**: `go test ./test -run TestUserRepository -v`
- **集成测试**: `go test ./test -run TestAPIIntegration -v`
- **完整测试**: `go test ./test -v`

### 部署要求

- **Go版本**: 1.19+
- **MySQL版本**: 8.0+
- **Redis版本**: 6.0+
- **内存要求**: 最小512MB
- **磁盘空间**: 最小1GB

---

**文档维护**: 本文档与代码同步更新，如有疑问请参考源码或联系开发团队。
**最后更新**: 2025-09-01
**文档版本**: v2.0
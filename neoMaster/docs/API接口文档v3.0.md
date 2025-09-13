# NeoScan Master API 接口文档 v3.0

## 📋 版本更新说明

**版本**: v3.0  
**更新日期**: 2025-09-13  
**主要变更**:
- 新增会话管理接口
- 完善用户管理接口
- 增强角色和权限管理功能
- 优化认证和令牌管理机制
- 完善健康检查和监控接口

## 🌐 服务器信息

- **基础URL**: `http://localhost:8123`
- **API版本**: v1
- **认证方式**: JWT Bearer Token
- **内容类型**: `application/json`
- **服务器版本**: NeoScan Master v3.0
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
  "timestamp": "2025-09-01T12:00:00Z"
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
  "timestamp": "2025-09-01T12:00:00Z"
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

### 1. 用户注册
- **URL**: `/api/v1/auth/register`
- **方法**: `POST`
- **描述**: 用户注册账户
- **认证**: 无需认证

**请求参数**:
```json
{
  "username": "用户名",
  "email": "邮箱地址",
  "password": "密码"
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "注册成功",
  "data": {
    "id": 1,
    "username": "newuser",
    "email": "user@example.com",
    "created_at": "2025-09-01T12:00:00Z"
  }
}
```

### 2. 用户登录
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

### 3. 获取用户权限
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

### 4. 获取用户角色
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
  "roles": [1, 2]
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
- **URL**: `/api/v1/admin/users/:id`
- **方法**: `GET`
- **描述**: 获取指定用户的详细信息
- **认证**: Bearer Token (管理员)
- **权限**: `user:read`

**路径参数**:
- `id`: 用户ID

#### 4. 获取用户详细信息
- **URL**: `/api/v1/admin/users/:id/info`
- **方法**: `GET`
- **描述**: 获取指定用户的详细信息（包括角色和权限）
- **认证**: Bearer Token (管理员)
- **权限**: `user:read`

#### 5. 更新用户信息
- **URL**: `/api/v1/admin/users/:id`
- **方法**: `POST`
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

#### 6. 删除用户
- **URL**: `/api/v1/admin/users/:id`
- **方法**: `DELETE`
- **描述**: 删除指定用户（软删除）
- **认证**: Bearer Token (管理员)
- **权限**: `user:delete`

#### 7. 激活用户
- **URL**: `/api/v1/admin/users/:id/activate`
- **方法**: `POST`
- **描述**: 激活指定用户
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`

#### 8. 禁用用户
- **URL**: `/api/v1/admin/users/:id/deactivate`
- **方法**: `POST`
- **描述**: 禁用指定用户
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`

#### 9. 重置用户密码
- **URL**: `/api/v1/admin/users/:id/reset-password`
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

### 角色管理

#### 1. 获取角色列表
- **URL**: `/api/v1/admin/roles/list`
- **方法**: `GET`
- **描述**: 获取所有角色列表
- **认证**: Bearer Token (管理员)
- **权限**: `role:read`

**响应示例**:
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "admin",
        "display_name": "管理员",
        "description": "系统管理员角色",
        "is_active": true,
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
      }
    ]
  }
}
```

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
  "is_active": true
}
```

#### 3. 获取角色详情
- **URL**: `/api/v1/admin/roles/:id`
- **方法**: `GET`
- **描述**: 获取指定角色的详细信息
- **认证**: Bearer Token (管理员)
- **权限**: `role:read`

#### 4. 更新角色
- **URL**: `/api/v1/admin/roles/:id`
- **方法**: `POST`
- **描述**: 更新指定角色的信息
- **认证**: Bearer Token (管理员)
- **权限**: `role:write`

**请求参数**:
```json
{
  "name": "editor",
  "display_name": "编辑员",
  "description": "内容编辑角色",
  "is_active": true
}
```

#### 5. 删除角色
- **URL**: `/api/v1/admin/roles/:id`
- **方法**: `DELETE`
- **描述**: 删除指定角色
- **认证**: Bearer Token (管理员)
- **权限**: `role:delete`

#### 6. 激活角色
- **URL**: `/api/v1/admin/roles/:id/activate`
- **方法**: `POST`
- **描述**: 激活指定角色
- **认证**: Bearer Token (管理员)
- **权限**: `role:write`

#### 7. 禁用角色
- **URL**: `/api/v1/admin/roles/:id/deactivate`
- **方法**: `POST`
- **描述**: 禁用指定角色
- **认证**: Bearer Token (管理员)
- **权限**: `role:write`

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
    "items": [
      {
        "id": 1,
        "name": "user:read",
        "display_name": "用户查看",
        "description": "查看用户信息的权限",
        "resource": "user",
        "action": "read",
        "is_active": true,
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
      }
    ]
  }
}
```

#### 2. 创建权限
- **URL**: `/api/v1/admin/permissions/create`
- **方法**: `POST`
- **描述**: 创建新权限
- **认证**: Bearer Token (管理员)
- **权限**: `permission:write`

**请求参数**:
```json
{
  "name": "content:read",
  "display_name": "内容查看",
  "description": "查看内容的权限",
  "resource": "content",
  "action": "read",
  "is_active": true
}
```

#### 3. 获取权限详情
- **URL**: `/api/v1/admin/permissions/:id`
- **方法**: `GET`
- **描述**: 获取指定权限的详细信息
- **认证**: Bearer Token (管理员)
- **权限**: `permission:read`

#### 4. 更新权限
- **URL**: `/api/v1/admin/permissions/:id`
- **方法**: `POST`
- **描述**: 更新指定权限的信息
- **认证**: Bearer Token (管理员)
- **权限**: `permission:write`

**请求参数**:
```json
{
  "name": "content:read",
  "display_name": "内容查看",
  "description": "查看内容的权限",
  "resource": "content",
  "action": "read",
  "is_active": true
}
```

#### 5. 删除权限
- **URL**: `/api/v1/admin/permissions/:id`
- **方法**: `DELETE`
- **描述**: 删除指定权限
- **认证**: Bearer Token (管理员)
- **权限**: `permission:delete`

### 会话管理

#### 1. 获取活跃会话列表
- **URL**: `/api/v1/admin/sessions/list`
- **方法**: `GET`
- **描述**: 获取指定用户的活跃会话列表
- **认证**: Bearer Token (管理员)
- **权限**: `session:read`

**查询参数**:
- `user_id`: 用户ID（必填）

**响应示例**:
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "token_id": "uuid-string",
        "user_id": 1,
        "ip_address": "192.168.1.100",
        "user_agent": "Mozilla/5.0...",
        "issued_at": "2025-09-01T12:00:00Z",
        "expires_at": "2025-09-02T12:00:00Z"
      }
    ]
  }
}
```

#### 2. 撤销用户会话
- **URL**: `/api/v1/admin/sessions/:userId/revoke`
- **方法**: `POST`
- **描述**: 撤销指定用户的特定会话
- **认证**: Bearer Token (管理员)
- **权限**: `session:revoke`

**请求参数**:
```json
{
  "token_id": "uuid-string"
}
```

#### 3. 撤销用户所有会话
- **URL**: `/api/v1/admin/sessions/user/:userId/revoke-all`
- **方法**: `POST`
- **描述**: 撤销指定用户的所有会话
- **认证**: Bearer Token (管理员)
- **权限**: `session:revoke-all`

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

### 架构改进 (v3.0)

1. **服务层优化**: 完善了各服务层的实现，包括UserService、RoleService、PermissionService等
2. **会话管理**: 新增SessionService用于管理用户会话和令牌黑名单
3. **RBAC权限控制**: 完善了基于角色的访问控制机制
4. **密码管理**: 增强了密码安全性和版本控制
5. **测试覆盖**: 完整的单元测试和集成测试

### 测试环境

- **单元测试**: `go test ./test -run TestUserService -v`
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
**最后更新**: 2025-09-13
**文档版本**: v3.0
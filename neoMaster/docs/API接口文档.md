# NeoScan Master API 接口文档

## 服务器信息

- **基础URL**: `http://localhost:8123`
- **API版本**: v1
- **认证方式**: JWT Bearer Token
- **内容类型**: `application/json`

## 通用响应格式

### 成功响应
```json
{
  "success": true,
  "message": "操作成功",
  "data": {}
}
```

### 错误响应
```json
{
  "success": false,
  "error": "错误代码",
  "message": "错误描述"
}
```

## 健康检查接口

### 1. 健康检查
- **URL**: `/api/health`
- **方法**: `GET`
- **描述**: 检查服务器健康状态
- **认证**: 无需认证

**响应示例**:
```json
{
  "status": "healthy",
  "timestamp": ""
}
```

### 2. 就绪检查
- **URL**: `/api/ready`
- **方法**: `GET`
- **描述**: 检查服务器就绪状态
- **认证**: 无需认证

**响应示例**:
```json
{
  "status": "ready",
  "timestamp": ""
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
  "timestamp": ""
}
```

## 认证接口

### 1. 用户登录
- **URL**: `/api/v1/auth/login`
- **方法**: `POST`
- **描述**: 用户登录获取JWT令牌
- **认证**: 无需认证

**请求参数**:
```json
{
  "username": "用户名",
  "password": "密码"
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
    "expires_in": 86400,
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "roles": ["admin"]
    }
  }
}
```

### 2. 获取登录表单
- **URL**: `/api/v1/auth/login`
- **方法**: `GET`
- **描述**: 获取登录表单页面
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

### 4. 从请求头刷新令牌
- **URL**: `/api/v1/auth/refresh-header`
- **方法**: `POST`
- **描述**: 从Authorization头刷新令牌
- **认证**: Bearer Token

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

## 用户认证接口（需要JWT认证）

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

### 2. 用户全部登出
- **URL**: `/api/v1/auth/logout-all`
- **方法**: `POST`
- **描述**: 用户全部设备登出，使所有令牌失效
- **认证**: Bearer Token

## 用户信息接口（需要JWT认证）

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
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "is_active": true,
    "roles": ["admin"],
    "permissions": ["user:read", "user:write", "admin:all"]
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
  "new_password": "新密码"
}
```

### 3. 获取用户权限
- **URL**: `/api/v1/user/permissions`
- **方法**: `GET`
- **描述**: 获取当前用户的权限列表
- **认证**: Bearer Token

### 4. 获取用户角色
- **URL**: `/api/v1/user/roles`
- **方法**: `GET`
- **描述**: 获取当前用户的角色列表
- **认证**: Bearer Token

## 管理员接口（需要管理员权限）

> **注意**: 以下接口需要管理员权限，请确保JWT令牌对应的用户具有管理员角色

### 用户管理

#### 1. 获取用户列表
- **URL**: `/api/v1/admin/users/list`
- **方法**: `GET`
- **描述**: 获取所有用户列表
- **认证**: Bearer Token (管理员)

**查询参数**:
- `page`: 页码（可选，默认1）
- `limit`: 每页数量（可选，默认10）
- `search`: 搜索关键词（可选）

#### 2. 创建用户
- **URL**: `/api/v1/admin/users/create`
- **方法**: `POST`
- **描述**: 创建新用户
- **认证**: Bearer Token (管理员)

**请求参数**:
```json
{
  "username": "新用户名",
  "email": "user@example.com",
  "password": "密码",
  "roles": ["user"]
}
```

#### 3. 获取用户详情
- **URL**: `/api/v1/admin/users/{id}`
- **方法**: `GET`
- **描述**: 根据ID获取用户详细信息
- **认证**: Bearer Token (管理员)

#### 4. 更新用户信息
- **URL**: `/api/v1/admin/users/{id}`
- **方法**: `PUT`
- **描述**: 更新用户信息
- **认证**: Bearer Token (管理员)

**请求参数**:
```json
{
  "username": "更新的用户名",
  "email": "updated@example.com",
  "roles": ["user", "moderator"]
}
```

#### 5. 删除用户
- **URL**: `/api/v1/admin/users/{id}`
- **方法**: `DELETE`
- **描述**: 删除用户
- **认证**: Bearer Token (管理员)

#### 6. 激活用户
- **URL**: `/api/v1/admin/users/{id}/activate`
- **方法**: `POST`
- **描述**: 激活用户账户
- **认证**: Bearer Token (管理员)

#### 7. 停用用户
- **URL**: `/api/v1/admin/users/{id}/deactivate`
- **方法**: `POST`
- **描述**: 停用用户账户
- **认证**: Bearer Token (管理员)

### 角色管理

#### 1. 获取角色列表
- **URL**: `/api/v1/admin/roles/list`
- **方法**: `GET`
- **描述**: 获取所有角色列表
- **认证**: Bearer Token (管理员)

#### 2. 创建角色
- **URL**: `/api/v1/admin/roles/create`
- **方法**: `POST`
- **描述**: 创建新角色
- **认证**: Bearer Token (管理员)

**请求参数**:
```json
{
  "name": "角色名称",
  "description": "角色描述",
  "permissions": ["permission1", "permission2"]
}
```

#### 3. 获取角色详情
- **URL**: `/api/v1/admin/roles/{id}`
- **方法**: `GET`
- **描述**: 根据ID获取角色详细信息
- **认证**: Bearer Token (管理员)

#### 4. 更新角色
- **URL**: `/api/v1/admin/roles/{id}`
- **方法**: `PUT`
- **描述**: 更新角色信息
- **认证**: Bearer Token (管理员)

#### 5. 删除角色
- **URL**: `/api/v1/admin/roles/{id}`
- **方法**: `DELETE`
- **描述**: 删除角色
- **认证**: Bearer Token (管理员)

### 权限管理

#### 1. 获取权限列表
- **URL**: `/api/v1/admin/permissions/list`
- **方法**: `GET`
- **描述**: 获取所有权限列表
- **认证**: Bearer Token (管理员)

#### 2. 创建权限
- **URL**: `/api/v1/admin/permissions/create`
- **方法**: `POST`
- **描述**: 创建新权限
- **认证**: Bearer Token (管理员)

**请求参数**:
```json
{
  "name": "权限名称",
  "description": "权限描述",
  "resource": "资源名称",
  "action": "操作类型"
}
```

#### 3. 获取权限详情
- **URL**: `/api/v1/admin/permissions/{id}`
- **方法**: `GET`
- **描述**: 根据ID获取权限详细信息
- **认证**: Bearer Token (管理员)

#### 4. 更新权限
- **URL**: `/api/v1/admin/permissions/{id}`
- **方法**: `PUT`
- **描述**: 更新权限信息
- **认证**: Bearer Token (管理员)

#### 5. 删除权限
- **URL**: `/api/v1/admin/permissions/{id}`
- **方法**: `DELETE`
- **描述**: 删除权限
- **认证**: Bearer Token (管理员)

### 会话管理

#### 1. 获取活跃会话列表
- **URL**: `/api/v1/admin/sessions/list`
- **方法**: `GET`
- **描述**: 获取所有活跃会话列表
- **认证**: Bearer Token (管理员)

#### 2. 撤销会话
- **URL**: `/api/v1/admin/sessions/{sessionId}/revoke`
- **方法**: `POST`
- **描述**: 撤销指定会话
- **认证**: Bearer Token (管理员)

#### 3. 撤销用户所有会话
- **URL**: `/api/v1/admin/sessions/user/{userId}/revoke-all`
- **方法**: `POST`
- **描述**: 撤销指定用户的所有会话
- **认证**: Bearer Token (管理员)

## 错误代码说明

| 错误代码 | HTTP状态码 | 描述 |
|---------|-----------|------|
| `UNAUTHORIZED` | 401 | 未授权，需要登录 |
| `FORBIDDEN` | 403 | 禁止访问，权限不足 |
| `NOT_FOUND` | 404 | 资源不存在 |
| `VALIDATION_ERROR` | 400 | 请求参数验证失败 |
| `INTERNAL_ERROR` | 500 | 服务器内部错误 |
| `DATABASE_ERROR` | 500 | 数据库连接错误 |
| `JWT_INVALID` | 401 | JWT令牌无效 |
| `JWT_EXPIRED` | 401 | JWT令牌已过期 |
| `PASSWORD_INVALID` | 400 | 密码格式无效 |
| `USER_NOT_FOUND` | 404 | 用户不存在 |
| `USER_INACTIVE` | 403 | 用户账户未激活 |

## 使用Apifox测试说明

### 1. 环境配置
在Apifox中创建环境变量：
- `base_url`: `http://localhost:8123`
- `access_token`: 登录后获取的访问令牌

### 2. 认证配置
对于需要认证的接口，在请求头中添加：
```
Authorization: Bearer {{access_token}}
```

### 3. 测试流程
1. 首先测试健康检查接口确认服务器运行正常
2. 使用登录接口获取JWT令牌
3. 将获取的令牌保存到环境变量中
4. 测试其他需要认证的接口

### 4. 注意事项
- 确保MySQL和Redis服务已启动
- JWT令牌有过期时间，需要定期刷新
- 管理员接口需要具有管理员权限的用户令牌
- 所有POST/PUT请求的Content-Type应设置为`application/json`

## 数据库依赖说明

当前服务器可以在没有数据库连接的情况下启动，但以下功能需要数据库支持：
- 用户登录验证
- 用户信息查询
- 权限验证
- 会话管理

如需完整测试所有功能，请确保：
1. MySQL服务已启动（默认端口3306）
2. Redis服务已启动（默认端口6379）
3. 数据库中已有初始用户数据

---

**文档版本**: v1.0  
**更新时间**: 2024年8月29日  
**服务器版本**: NeoScan Master v1.0
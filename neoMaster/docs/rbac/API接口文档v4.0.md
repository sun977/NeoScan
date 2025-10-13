# NeoScan Master API 接口文档 v4.0

## 📋 版本更新说明

**版本**: v4.0  
**更新日期**: 2025-09-25  
**主要变更**:
- 人工测试并修改了v3.0的bug

## 🌐 服务器信息

- **基础URL**: `http://localhost:8123`
- **API版本**: v1
- **认证方式**: JWT Bearer Token
- **内容类型**: `application/json`
- **服务器版本**: NeoScan Master v4.0

## 📊 通用响应格式

### 成功响应
```json
{
  "code": 200,
  "status": "success",
  "message": "操作成功",
  "data": {}
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
  "code": 200,
  "status": "success",
  "message": "操作成功",
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
  "code": 201,
  "status": "success",
  "message": "注册成功",
  "data": {
    "user": {
      "id": 1,
      "username": "newuser",
      "email": "user@example.com",
      "nickname": "Sun977",
      "avatar": "",
      "phone": "",
      "status": 1,
      "last_login_at": null,
      "created_at": "2025-09-20T18:35:56.431+08:00"
    },
    "message": "registration successful"
  }
}
```

### 2. 用户登录
- **URL**: `/api/v1/auth/login`
- **方法**: `POST`
- **描述**: 用户登录获取JWT令牌
- **认证**: 无需认证

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
  "code": 200,
  "status": "success",
  "message": "login successful",
  "data": {
    "user": {
      "id": 46,
      "username": "newuser",
      "email": "user@example.com",
      "nickname": "Sun977",
      "avatar": "",
      "phone": "",
      "socket_id": "",
      "remark": "",
      "status": 1,
      "last_login_at": "2025-09-15T15:48:11+08:00",
      "last_login_ip": "127.0.0.5",
      "created_at": "2025-09-15T15:38:21+08:00",
      "updated_at": "2025-09-15T15:48:11+08:00",
      "roles": []
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
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
  "code": 200,
  "status": "success",
  "message": "refresh token successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "token_type": "Bearer"
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

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "refresh token successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "token_type": "Bearer"
  }
}
```

### 5. 检查令牌过期时间
- **URL**: `/api/v1/auth/check-expiry`
- **方法**: `POST`
- **描述**: 检查令牌过期时间
- **认证**: Bearer Token

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "check token expiry successful",
  "data": {
    "is_expiring_soon": false,
    "remaining_seconds": 3384,
    "remaining_time": "56m24.8637136s"
  }
}
```

### 6. 用户登出(已弃用,统一使用7.用户全部登出)
- **URL**: `/api/v1/auth/logout`
- **方法**: `POST`
- **描述**: 用户登出，使当前accessToken令牌失效（accessToken令牌进入缓存redis黑名单）
- **认证**: Bearer Token

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "logout successful"
}
```

### 7. 用户全部登出
- **URL**: `/api/v1/auth/logout-all`
- **方法**: `POST`
- **描述**: 用户全部设备登出，使所有令牌失效（密码版本自增，所有令牌失效）
- **认证**: Bearer Token

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "logout all successful",
  "data": {
    "message": "All sessions have been terminated",
    "user_id": 46
  }
}
```

## 👤 用户信息接口（需要JWT认证）

### 1. 获取当前用户信息
- **URL**: `/api/v1/user/profile`
- **方法**: `GET`
- **描述**: 获取当前登录用户的详细信息（token在revoked状态不能获取）
- **认证**: Bearer Token

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "获取用户信息成功",
  "data": {
    "id": 46,
    "username": "newuser",
    "email": "user@example.com",
    "nickname": "Sun977",
    "avatar": "",
    "phone": "",
    "status": 1,
    "last_login_at": "2025-09-15T16:54:21+08:00",
    "created_at": "2025-09-15T15:38:21+08:00",
    "roles": [],
    "permissions": [],
    "remark": ""
  }
}
```

### 2. 更新用户信息
- **URL**: `/api/v1/user/update`
- **方法**: `POST`
- **描述**: 更新当前用户的基本信息
- **认证**: Bearer Token

**请求参数**:
```json
{
  "nickname": "新昵称",
  "email": "new@example.com",
  "phone": "13800138000",
  "avatar": "https://example.com/avatar.jpg"
}
```

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "user info updated success",
  "data": {
    "id": 2,
    "username": "newuser123",
    "email": "newem222ail22@example.com",
    "nickname": "新昵称1",
    "avatar": "ceshi2.png",
    "phone": "13800138002",
    "status": 1,
    "last_login_at": "2025-09-23T19:14:02+08:00",
    "created_at": "2025-09-15T19:22:06+08:00",
    "remark": "更新的备注2"
  }
}
```

### 3. 修改用户密码
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

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "密码修改成功，请重新登录"
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
  "code": 200,
  "status": "success",
  "message": "user permissions retrieved successfully",
  "data": {
    "permissions": [
      {
        "id": 2,
        "name": "user:create",
        "display_name": "创建用户",
        "description": "创建新用户的权限",
        "resource": "user",
        "action": "create",
        "created_at": "2025-09-01T19:20:34+08:00",
        "updated_at": "2025-09-01T19:20:34+08:00"
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
  "code": 200,
  "status": "success",
  "message": "user roles retrieved successfully",
  "data": {
    "roles": [
      {
        "id": 1,
        "name": "admin",
        "display_name": "系统管理员",
        "description": "拥有系统所有权限的超级管理员",
        "status": 1,
        "created_at": "2025-09-01T19:20:34+08:00",
        "updated_at": "2025-09-01T19:20:34+08:00",
        "permissions": null
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
- `offset`: 偏移量（可选，默认1）
- `limit`: 每页数量（可选，默认10，最大100）

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "user list retrieved successfully",
  "data": {
    "items": [
      {
        "id": 1,
        "username": "admin",
        "email": "admin@neoscan.com",
        "nickname": "系统管理员",
        "avatar": "",
        "phone": "",
        "socket_id": "",
        "remark": "系统用户",
        "status": 1,
        "last_login_at": "2025-09-15T17:38:26+08:00",
        "last_login_ip": "127.0.0.5",
        "created_at": "2025-09-01T19:20:34+08:00",
        "updated_at": "2025-09-15T17:38:26+08:00",
        "roles": null
      }
    ],
    "pagination": {
      "limit": 10,
      "page": 1,
      "pages": 2,
      "total": 12
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
  "code": 201,
  "status": "success",
  "message": "user created successfully",
  "data": {
    "user": {
      "created_at": "2025-09-15T17:53:13.04+08:00",
      "email": "user@qq.com",
      "id": 47,
      "nickname": "新用户",
      "phone": "",
      "status": 1,
      "username": "newuser2"
    }
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

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "user information retrieved successfully",
  "data": {
    "user": {
      "created_at": "2025-09-15T17:53:13+08:00",
      "email": "user@qq.com",
      "id": 47,
      "nickname": "新用户",
      "phone": "",
      "status": 1,
      "updated_at": "2025-09-15T17:53:13+08:00",
      "username": "newuser2"
    }
  }
}
```

#### 4. 获取用户详细信息
- **URL**: `/api/v1/admin/users/{id}/info`
- **方法**: `GET`
- **描述**: 获取指定用户的详细信息（包括角色和权限）
- **认证**: Bearer Token (管理员)
- **权限**: `user:read`

**路径参数**:
- `id`: 用户ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "获取用户信息成功",
  "data": {
    "id": 2,
    "username": "newuser",
    "email": "user@example.com",
    "nickname": "Sun977",
    "avatar": "",
    "phone": "",
    "status": 1,
    "last_login_at": null,
    "created_at": "2025-09-15T19:22:06+08:00",
    "roles": ["user"],
    "permissions": ["role:read", "permission:read", "user:read", "user:update"],
    "remark": ""
  }
}
```

#### 5. 更新用户信息(含角色信息更新)
- **URL**: `/api/v1/admin/users/{id}`
- **方法**: `POST`
- **描述**: 更新指定用户的信息(事务修改)，字段全部可选增加，不可修改用户角色
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`

**路径参数**:
- `id`: 用户ID

**请求参数**:
```json
{
  "username": "新用户名",
  "email": "newemail22@example.com",
  "nickname": "新昵称1",
  "phone": "13800138002",
  "remark": "更新的备注2",
  "status": 0,
  "avatar": "ceshi2.png",
  "password": "admin123"
}
```

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "user updated successfully",
  "data": {
    "id": 8,
    "username": "newname",
    "email": "newemail22@example.com",
    "nickname": "新昵称1",
    "avatar": "ceshi2.png",
    "phone": "13800138002",
    "status": 0,
    "last_login_at": "2025-09-15T20:24:54+08:00",
    "created_at": "2025-09-15T20:23:53+08:00",
    "roles": null,
    "permissions": null,
    "remark": "更新的备注2"
  }
}
```

#### 6. 删除用户
- **URL**: `/api/v1/admin/users/{id}`
- **方法**: `DELETE`
- **描述**: 删除指定用户（软删除）
- **认证**: Bearer Token (管理员)
- **权限**: `user:delete`

**路径参数**:
- `id`: 用户ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "用户删除成功"
}
```

#### 7. 激活用户
- **URL**: `/api/v1/admin/users/{id}/activate`
- **方法**: `POST`
- **描述**: 激活指定用户
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`

**路径参数**:
- `id`: 用户ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "用户激活成功",
  "data": {
    "status": "activated",
    "user_id": 8
  }
}
```

#### 8. 禁用用户
- **URL**: `/api/v1/admin/users/{id}/deactivate`
- **方法**: `POST`
- **描述**: 禁用指定用户
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`

**路径参数**:
- `id`: 用户ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "用户禁用成功",
  "data": {
    "status": "deactivated",
    "user_id": 8
  }
}
```

#### 9. 重置用户密码
- **URL**: `/api/v1/admin/users/{id}/reset-password`
- **方法**: `POST`
- **描述**: 重置指定用户的密码
- **认证**: Bearer Token (管理员)
- **权限**: `user:write`

**路径参数**:
- `id`: 用户ID

**请求参数**:
```json
{
  "new_password": "新密码"
}
```

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "重置密码成功"
}
```

### 角色管理

#### 1. 获取角色列表
- **URL**: `/api/v1/admin/roles/list`
- **方法**: `GET`
- **描述**: 获取所有角色列表
- **认证**: Bearer Token (管理员)
- **权限**: `role:read`

**查询参数**:
- `offset`: 偏移（可选）
- `limit`: 每页数量（可选）

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "获取角色列表成功",
  "data": {
    "roles": [
      {
        "id": 1,
        "name": "admin",
        "display_name": "系统管理员",
        "description": "拥有系统所有权限的超级管理员",
        "status": 1,
        "created_at": "2025-09-15T19:18:10+08:00",
        "updated_at": "2025-09-15T19:18:10+08:00",
        "permissions": null
      }
    ],
    "pagination": {
      "total": 4,
      "page": 1,
      "page_size": 2,
      "total_pages": 2,
      "has_next": true,
      "has_previous": false,
      "data": null
    }
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

**响应示例**:
```json
{
  "code": 201,
  "status": "success",
  "message": "角色创建成功",
  "data": {
    "id": 5,
    "name": "editor",
    "display_name": "编辑员",
    "description": "内容编辑角色",
    "status": 1,
    "created_at": "2025-09-16T16:36:24.709+08:00",
    "updated_at": "2025-09-16T16:36:24.709+08:00",
    "permissions": null
  }
}
```

#### 3. 获取角色详情
- **URL**: `/api/v1/admin/roles/{id}`
- **方法**: `GET`
- **描述**: 获取指定角色的详细信息
- **认证**: Bearer Token (管理员)
- **权限**: `role:read`

**路径参数**:
- `id`: 角色ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "获取角色信息成功",
  "data": {
    "id": 6,
    "name": "editor1",
    "display_name": "编辑员",
    "description": "内容编辑角色",
    "status": 1,
    "created_at": "2025-09-16T16:38:36+08:00",
    "updated_at": "2025-09-16T16:38:36+08:00",
    "permissions": null
  }
}
```

#### 4. 更新角色
- **URL**: `/api/v1/admin/roles/{id}`
- **方法**: `POST`
- **描述**: 更新指定角色的信息
- **认证**: Bearer Token (管理员)
- **权限**: `role:write`

**路径参数**:
- `id`: 角色ID

**请求参数**:
```json
{
  "name": "角色名称",
  "display_name": "角色显示名称",
  "description": "角色描述",
  "status": 1
}
```

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "角色更新成功",
  "data": {
    "id": 6,
    "name": "editor2",
    "display_name": "编辑员3",
    "description": "内容编辑员3",
    "status": 1,
    "created_at": "2025-09-16T16:38:36+08:00",
    "updated_at": "2025-09-16T18:02:53.605+08:00",
    "permissions": null
  }
}
```

#### 5. 删除角色
- **URL**: `/api/v1/admin/roles/{id}`
- **方法**: `DELETE`
- **描述**: 删除指定角色
- **认证**: Bearer Token (管理员)
- **权限**: `role:delete`

**路径参数**:
- `id`: 角色ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "角色删除成功"
}
```

#### 6. 激活角色
- **URL**: `/api/v1/admin/roles/{id}/activate`
- **方法**: `POST`
- **描述**: 激活指定角色
- **认证**: Bearer Token (管理员)
- **权限**: `role:write`

**路径参数**:
- `id`: 角色ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "角色激活成功"
}
```

#### 7. 禁用角色
- **URL**: `/api/v1/admin/roles/{id}/deactivate`
- **方法**: `POST`
- **描述**: 禁用指定角色
- **认证**: Bearer Token (管理员)
- **权限**: `role:write`

**路径参数**:
- `id`: 角色ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "角色禁用成功"
}
```

### 权限管理

#### 1. 获取权限列表
- **URL**: `/api/v1/admin/permissions/list`
- **方法**: `GET`
- **描述**: 获取所有权限列表
- **认证**: Bearer Token (管理员)
- **权限**: `permission:read`

**查询参数**:
- `offset`: 偏移量
- `limit`: 每页数量

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "获取权限列表成功",
  "data": {
    "permissions": [
      {
        "id": 1,
        "name": "system:admin",
        "display_name": "系统管理",
        "description": "系统管理权限",
        "resource": "system",
        "action": "admin",
        "created_at": "2025-09-15T19:18:10+08:00",
        "updated_at": "2025-09-15T19:18:10+08:00"
      }
    ],
    "pagination": {
      "total": 13,
      "page": 1,
      "page_size": 10,
      "total_pages": 2,
      "has_next": true,
      "has_previous": false,
      "data": null
    }
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

**响应示例**:
```json
{
  "code": 201,
  "status": "success",
  "message": "权限创建成功",
  "data": {
    "id": 15,
    "name": "content:read",
    "display_name": "内容查看",
    "description": "查看内容的权限",
    "resource": "content",
    "action": "read",
    "created_at": "2025-09-18T18:35:35.445+08:00",
    "updated_at": "2025-09-18T18:35:35.445+08:00"
  }
}
```

#### 3. 获取权限详情
- **URL**: `/api/v1/admin/permissions/{id}`
- **方法**: `GET`
- **描述**: 获取指定权限的详细信息
- **认证**: Bearer Token (管理员)
- **权限**: `permission:read`

**路径参数**:
- `id`: 权限ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "获取权限信息成功",
  "data": {
    "id": 1,
    "name": "system:admin",
    "display_name": "系统管理",
    "description": "系统管理权限",
    "resource": "system",
    "status": 1,
    "action": "admin",
    "created_at": "2025-09-15T19:18:10+08:00",
    "updated_at": "2025-09-15T19:18:10+08:00"
  }
}
```

#### 4. 更新权限
- **URL**: `/api/v1/admin/permissions/{id}`
- **方法**: `POST`
- **描述**: 更新指定权限的信息
- **认证**: Bearer Token (管理员)
- **权限**: `permission:write`

**路径参数**:
- `id`: 权限ID

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

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "权限更新成功",
  "data": {
    "id": 16,
    "name": "content:read",
    "display_name": "内容查看222",
    "description": "查看内容的权限222",
    "resource": "content22",
    "status": 1,
    "action": "read22",
    "created_at": "2025-09-18T18:49:06+08:00",
    "updated_at": "2025-09-18T19:34:10.876+08:00"
  }
}
```

#### 5. 删除权限
- **URL**: `/api/v1/admin/permissions/{id}`
- **方法**: `DELETE`
- **描述**: 删除指定权限
- **认证**: Bearer Token (管理员)
- **权限**: `permission:delete`

**路径参数**:
- `id`: 权限ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "权限删除成功"
}
```

### 会话管理

#### 1. 获取活跃会话列表
- **URL**: `/api/v1/admin/sessions/user/list`
- **方法**: `GET`
- **描述**: 获取指定用户的活跃会话列表(不区分用户合法性)
  - 用户ID合法且有会话 --- 返回会话信息
  - 用户ID合法且无会话 --- 返回data[]空
  - 用户ID不合法 --- 返回data[]空
- **认证**: Bearer Token (管理员)
- **权限**: `session:read`

**查询参数**:
- `userId`: 用户ID（必填）

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "获取活跃会话成功",
  "data": [
    {
      "user_id": 1,
      "username": "admin",
      "email": "admin@neoscan.com",
      "roles": ["admin"],
      "permissions": ["system:admin", "user:create", "user:read"],
      "login_time": "2025-09-19T15:53:06.9708233+08:00",
      "last_active": "2025-09-19T15:53:06.9708233+08:00",
      "client_ip": "127.0.0.5",
      "user_agent": "Apifox/1.0.0 (https://apifox.com)"
    }
  ]
}
```

#### 2. 撤销用户会话
- **URL**: `/api/v1/admin/sessions/user/{userId}/revoke`
- **方法**: `POST`
- **描述**: 撤销指定用户的特定会话
- **认证**: Bearer Token (管理员)
- **权限**: `session:revoke`

**路径参数**:
- `userId`: 用户ID

**请求参数**:
```json
{
  "token_id": "uuid-string"
}
```

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "撤销会话成功"
}
```

#### 3. 撤销用户所有会话
- **URL**: `/api/v1/admin/sessions/user/{userId}/revoke-all`
- **方法**: `POST`
- **描述**: 撤销指定用户的所有会话
- **认证**: Bearer Token (管理员)
- **权限**: `session:revoke-all`

**路径参数**:
- `userId`: 用户ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "撤销用户所有会话成功"
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

### 架构改进 (v4.0)

1. **Bug修复**: 修复了v3.0版本中发现的问题
2. **接口优化**: 优化了部分接口的响应格式和参数验证
3. **安全性增强**: 增强了会话管理和令牌验证机制

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
**最后更新**: 2025-09-25
**文档版本**: v4.0
# NeoScan Master API 测试指南 v4.0

## 📋 版本更新说明

**版本**: v4.0  
**更新日期**: 2025-09-25  
**主要变更**:
- 根据v4.0接口文档更新测试用例
- 优化测试流程和测试数据管理
- 增强会话管理测试覆盖
- 改进权限和角色管理测试
- 更新认证和令牌管理测试

## 🎯 测试概览

本指南提供了NeoScan Master API v4.0的完整测试方案，包括：
- **单元测试**: 模型层、仓库层、服务层的独立测试
- **集成测试**: API端点的完整流程测试
- **性能测试**: 接口响应时间和并发能力测试
- **安全测试**: 认证、授权和数据安全测试

## 🚀 快速开始

### 1. 环境准备

#### 系统要求
- **Go版本**: 1.19+
- **MySQL**: 8.0+ (测试数据库)
- **Redis**: 6.0+ (测试缓存)
- **内存**: 最小512MB
- **磁盘空间**: 最小500MB

#### 配置文件检查
确保以下配置文件正确设置：

**config.yaml**:
```yaml
database:
  host: localhost
  port: 3306
  username: root
  password: ROOT  # 实际密码，不使用环境变量
  database: neoscan_dev
  test_database: neoscan_test
  charset: utf8mb4
  
redis:
  host: localhost
  port: 6379
  password: ""
  database: 0
  test_database: 1
```

### 2. 数据库准备

#### 创建测试数据库
```bash
# 使用MySQL命令行创建测试数据库
& "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT -e "DROP DATABASE IF EXISTS neoscan_test; CREATE DATABASE neoscan_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 导入测试数据结构
Get-Content -Path "database_schema.sql" -Encoding UTF8 | & "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT neoscan_test
```

#### 验证数据库结构
```bash
& "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT -e "USE neoscan_test; SHOW TABLES; DESCRIBE users;"
```

### 3. 运行服务器

#### 开发模式启动
```bash
# 在项目根目录执行
go run main.go
```

#### 验证服务器状态
```bash
# 检查健康状态
curl http://localhost:8123/api/health

# 检查就绪状态
curl http://localhost:8123/api/ready
```

**预期响应**:
```json
{
  "status": "healthy",
  "timestamp": "2025-09-01T12:00:00Z"
}
```

## 🧪 单元测试

### 1. 用户模型测试

#### 运行用户模型测试
```bash
# 运行用户模型相关测试
go test ./test -run TestUserModel -v
```

#### 测试覆盖内容
- ✅ 用户创建和验证
- ✅ 密码哈希和验证
- ✅ 用户状态管理
- ✅ 角色分配和验证
- ✅ 密码版本控制
- ✅ 用户信息更新

#### 预期输出示例
```
=== RUN   TestUserModel
=== RUN   TestUserModel/TestCreateUser
=== RUN   TestUserModel/TestValidatePassword
=== RUN   TestUserModel/TestUserStatus
=== RUN   TestUserModel/TestUserRoles
=== RUN   TestUserModel/TestPasswordVersion
--- PASS: TestUserModel (0.01s)
    --- PASS: TestUserModel/TestCreateUser (0.00s)
    --- PASS: TestUserModel/TestValidatePassword (0.00s)
    --- PASS: TestUserModel/TestUserStatus (0.00s)
    --- PASS: TestUserModel/TestUserRoles (0.00s)
    --- PASS: TestUserModel/TestPasswordVersion (0.00s)
PASS
```

### 2. 用户仓库测试

#### 运行用户仓库测试
```bash
# 运行用户仓库相关测试
go test ./test -run TestUserRepository -v
```

#### 测试覆盖内容
- ✅ 用户创建 (Create)
- ✅ 用户查询 (GetByID, GetByUsername, GetByEmail)
- ✅ 用户更新 (Update)
- ✅ 用户删除 (Delete)
- ✅ 用户列表查询 (List with pagination)
- ✅ 数据库事务处理
- ✅ 错误处理和边界条件

#### 预期输出示例
```
=== RUN   TestUserRepository
=== RUN   TestUserRepository/TestCreateUser
=== RUN   TestUserRepository/TestGetUser
=== RUN   TestUserRepository/TestUpdateUser
=== RUN   TestUserRepository/TestDeleteUser
=== RUN   TestUserRepository/TestListUsers
--- PASS: TestUserRepository (0.15s)
    --- PASS: TestUserRepository/TestCreateUser (0.03s)
    --- PASS: TestUserRepository/TestGetUser (0.02s)
    --- PASS: TestUserRepository/TestUpdateUser (0.03s)
    --- PASS: TestUserRepository/TestDeleteUser (0.02s)
    --- PASS: TestUserRepository/TestListUsers (0.05s)
PASS
```

### 3. 认证服务测试

#### 运行认证服务测试
```bash
# 运行认证服务相关测试
go test ./test -run TestAuthService -v
```

#### 测试覆盖内容
- ✅ 用户登录验证
- ✅ JWT令牌生成和验证
- ✅ 令牌刷新机制
- ✅ 用户登出处理
- ✅ 权限验证
- ✅ 会话管理

## 🔗 集成测试

### 1. API集成测试

#### 运行完整API测试
```bash
# 运行所有API集成测试
go test ./test -run TestAPIIntegration -v
```

#### 测试覆盖内容
- ✅ 用户注册流程
- ✅ 用户登录流程
- ✅ 令牌刷新流程
- ✅ 用户信息获取
- ✅ 密码修改流程
- ✅ 权限验证
- ✅ 用户登出流程
- ✅ 管理员功能测试

#### 预期输出示例
```
=== RUN   TestAPIIntegration
=== RUN   TestAPIIntegration/TestUserRegistration
=== RUN   TestAPIIntegration/TestUserLogin
=== RUN   TestAPIIntegration/TestTokenRefresh
=== RUN   TestAPIIntegration/TestUserProfile
=== RUN   TestAPIIntegration/TestChangePassword
=== RUN   TestAPIIntegration/TestPermissionCheck
=== RUN   TestAPIIntegration/TestUserLogout
--- PASS: TestAPIIntegration (0.25s)
PASS
```

### 2. 完整用户流程测试

#### 测试场景
```bash
# 运行完整用户流程测试
go test ./test -run TestCompleteUserFlow -v
```

#### 流程步骤
1. **用户注册** → 创建新用户账户
2. **邮箱验证** → 验证用户邮箱（如果启用）
3. **用户登录** → 获取访问令牌
4. **获取用户信息** → 验证用户数据
5. **修改用户信息** → 更新用户资料
6. **修改密码** → 更新用户密码
7. **权限验证** → 测试用户权限
8. **令牌刷新** → 刷新访问令牌
9. **用户登出** → 清理用户会话

## 🛠️ 工具配置

### 1. Postman 配置

#### 环境变量设置
```json
{
  "base_url": "http://localhost:8123",
  "api_version": "v1",
  "access_token": "",
  "refresh_token": "",
  "user_id": ""
}
```

#### 预请求脚本（自动令牌管理）
```javascript
// 检查令牌是否存在且未过期
const token = pm.environment.get("access_token");
if (!token) {
    console.log("No access token found, please login first");
    return;
}

// 检查令牌过期时间
const tokenExpiry = pm.environment.get("token_expiry");
if (tokenExpiry && new Date() > new Date(tokenExpiry)) {
    console.log("Token expired, attempting refresh...");
    // 这里可以添加自动刷新令牌的逻辑
}
```

#### 测试脚本（响应验证）
```javascript
// 验证响应状态
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

// 验证响应格式
pm.test("Response has code field", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('code');
    pm.expect(jsonData.code).to.eql(200);
});

// 保存令牌（登录接口）
if (pm.response.json().data && pm.response.json().data.access_token) {
    pm.environment.set("access_token", pm.response.json().data.access_token);
    pm.environment.set("refresh_token", pm.response.json().data.refresh_token);
    
    // 计算令牌过期时间
    const expiresIn = pm.response.json().data.expires_in;
    const expiryTime = new Date(Date.now() + expiresIn * 1000);
    pm.environment.set("token_expiry", expiryTime.toISOString());
}
```

### 2. Apifox 配置

#### 项目设置
1. **基础URL**: `http://localhost:8123`
2. **全局参数**:
   - `Content-Type`: `application/json`
   - `Accept`: `application/json`

#### 认证配置
```json
{
  "type": "bearer",
  "token": "{{access_token}}"
}
```

#### 环境变量
```json
{
  "base_url": "http://localhost:8123",
  "access_token": "",
  "refresh_token": "",
  "test_username": "testuser",
  "test_password": "testpass123",
  "test_email": "test@example.com"
}
```

### 3. cURL 测试脚本

#### 健康检查
```bash
#!/bin/bash
# health_check.sh

BASE_URL="http://localhost:8123"

echo "=== 健康检查 ==="
curl -s "$BASE_URL/api/health" | jq .

echo -e "\n=== 就绪检查 ==="
curl -s "$BASE_URL/api/ready" | jq .

echo -e "\n=== 存活检查 ==="
curl -s "$BASE_URL/api/live" | jq .
```

#### 用户登录测试
```bash
#!/bin/bash
# login_test.sh

BASE_URL="http://localhost:8123"
USERNAME="admin"
PASSWORD="admin123"

echo "=== 用户登录 ==="
RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"$USERNAME\",
    \"password\": \"$PASSWORD\"
  }")

echo $RESPONSE | jq .

# 提取访问令牌
ACCESS_TOKEN=$(echo $RESPONSE | jq -r '.data.access_token')
echo "Access Token: $ACCESS_TOKEN"

# 保存令牌到文件
echo $ACCESS_TOKEN > .access_token
```

#### 用户信息获取
```bash
#!/bin/bash
# profile_test.sh

BASE_URL="http://localhost:8123"
ACCESS_TOKEN=$(cat .access_token)

echo "=== 获取用户信息 ==="
curl -s -X GET "$BASE_URL/api/v1/user/profile" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" | jq .
```

## 🔐 认证流程测试

### 1. 基础认证测试

#### 步骤1: 用户注册
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "newuser",
  "email": "user@example.com",
  "password": "userpass123"
}
```

**预期响应**:
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

#### 步骤2: 用户登录
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "newuser",
  "password": "userpass123"
}
```

**预期响应**:
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

#### 步骤3: 使用令牌访问受保护资源
```http
GET /api/v1/user/profile
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### 步骤4: 刷新令牌
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### 步骤5: 用户全部登出
```http
POST /api/v1/auth/logout-all
Authorization: Bearer eyJhbGciOiJIuI1NiIsInR5cCI6IkpXVCJ9...
```

### 2. 权限验证测试

#### 管理员权限测试
```http
# 需要管理员权限的接口
GET /api/v1/admin/users/list
Authorization: Bearer <admin_token>
```

#### 普通用户权限测试
```http
# 普通用户访问管理员接口（应该返回403）
GET /api/v1/admin/users/list
Authorization: Bearer <user_token>
```

**预期错误响应**:
```json
{
  "success": false,
  "error": "FORBIDDEN",
  "message": "权限不足"
}
```

## 👨‍💼 管理员功能测试

### 1. 用户管理测试

#### 获取用户列表
```http
GET /api/v1/admin/users/list?offset=1&limit=10
Authorization: Bearer <admin_token>
```

#### 创建用户
```http
POST /api/v1/admin/users/create
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "username": "testuser",
  "email": "test@example.com",
  "password": "testpass123",
  "nickname": "测试用户",
  "phone": "13800138000",
  "remark": "测试账户",
  "is_active": true
}
```

#### 获取用户详情
```http
GET /api/v1/admin/users/{id}
Authorization: Bearer <admin_token>
```

#### 更新用户信息
```http
POST /api/v1/admin/users/{id}
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "username": "updateduser",
  "email": "updated@example.com",
  "nickname": "更新用户",
  "phone": "13900139000",
  "remark": "更新账户",
  "status": 1
}
```

#### 删除用户
```http
DELETE /api/v1/admin/users/{id}
Authorization: Bearer <admin_token>
```

### 2. 角色管理测试

#### 获取角色列表
```http
GET /api/v1/admin/roles/list
Authorization: Bearer <admin_token>
```

#### 创建角色
```http
POST /api/v1/admin/roles/create
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "editor",
  "display_name": "编辑员",
  "description": "内容编辑角色"
}
```

#### 获取角色详情
```http
GET /api/v1/admin/roles/{id}
Authorization: Bearer <admin_token>
```

#### 更新角色
```http
POST /api/v1/admin/roles/{id}
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "editor",
  "display_name": "编辑员",
  "description": "内容编辑角色",
  "status": 1
}
```

#### 删除角色
```http
DELETE /api/v1/admin/roles/{id}
Authorization: Bearer <admin_token>
```

### 3. 权限管理测试

#### 获取权限列表
```http
GET /api/v1/admin/permissions/list
Authorization: Bearer <admin_token>
```

#### 创建权限
```http
POST /api/v1/admin/permissions/create
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "content:read",
  "display_name": "内容查看",
  "description": "查看内容的权限",
  "resource": "content",
  "action": "read"
}
```

#### 获取权限详情
```http
GET /api/v1/admin/permissions/{id}
Authorization: Bearer <admin_token>
```

#### 更新权限
```http
POST /api/v1/admin/permissions/{id}
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "content:read",
  "display_name": "内容查看",
  "description": "查看内容的权限",
  "resource": "content",
  "action": "read",
  "is_active": true
}
```

#### 删除权限
```http
DELETE /api/v1/admin/permissions/{id}
Authorization: Bearer <admin_token>
```

### 4. 会话管理测试

#### 获取活跃会话列表
```http
GET /api/v1/admin/sessions/user/list?userId=1
Authorization: Bearer <admin_token>
```

#### 撤销用户会话
```http
POST /api/v1/admin/sessions/user/{userId}/revoke
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "token_id": "uuid-string"
}
```

#### 撤销用户所有会话
```http
POST /api/v1/admin/sessions/user/{userId}/revoke-all
Authorization: Bearer <admin_token>
```

## 📊 性能测试

### 1. 响应时间测试

#### 使用Apache Bench (ab)
```bash
# 健康检查接口性能测试
ab -n 1000 -c 10 http://localhost:8123/api/health

# 登录接口性能测试
ab -n 100 -c 5 -p login_data.json -T application/json http://localhost:8123/api/v1/auth/login
```

#### login_data.json
```json
{
  "username": "admin",
  "password": "admin123"
}
```

#### 预期性能指标
- **健康检查**: < 10ms
- **用户登录**: < 100ms
- **用户信息获取**: < 50ms
- **数据库查询**: < 200ms

### 2. 并发测试

#### 使用wrk工具
```bash
# 安装wrk (Windows需要使用WSL或者下载编译版本)
# 并发测试健康检查接口
wrk -t12 -c400 -d30s http://localhost:8123/api/health

# 并发测试登录接口
wrk -t4 -c100 -d10s -s login_script.lua http://localhost:8123/api/v1/auth/login
```

#### login_script.lua
```lua
wrk.method = "POST"
wrk.body   = '{"username":"admin","password":"admin123"}'
wrk.headers["Content-Type"] = "application/json"
```

### 3. 内存和CPU监控

#### 监控脚本
```bash
#!/bin/bash
# monitor.sh

echo "=== 系统资源监控 ==="
echo "时间: $(date)"
echo "内存使用:"
free -h
echo "CPU使用:"
top -bn1 | grep "Cpu(s)"
echo "进程信息:"
ps aux | grep "main" | grep -v grep
```

## 🔍 错误处理测试

### 1. 输入验证测试

#### 无效登录凭据
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "invalid_user",
  "password": "wrong_password"
}
```

**预期响应**:
```json
{
  "success": false,
  "error": "INVALID_CREDENTIALS",
  "message": "用户名或密码错误"
}
```

#### 缺少必需参数
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin"
  // 缺少password字段
}
```

**预期响应**:
```json
{
  "success": false,
  "error": "INVALID_REQUEST",
  "message": "缺少必需参数: password"
}
```

### 2. 令牌验证测试

#### 过期令牌
```http
GET /api/v1/user/profile
Authorization: Bearer <expired_token>
```

**预期响应**:
```json
{
  "success": false,
  "error": "TOKEN_EXPIRED",
  "message": "令牌已过期"
}
```

#### 无效令牌
```http
GET /api/v1/user/profile
Authorization: Bearer invalid_token_string
```

**预期响应**:
```json
{
  "success": false,
  "error": "TOKEN_INVALID",
  "message": "令牌无效"
}
```

### 3. 数据库连接测试

#### 模拟数据库连接失败
```bash
# 停止MySQL服务进行测试
# 然后访问需要数据库的接口
curl http://localhost:8123/api/v1/user/profile \
  -H "Authorization: Bearer <valid_token>"
```

**预期响应**:
```json
{
  "success": false,
  "error": "INTERNAL_ERROR",
  "message": "数据库连接失败"
}
```

## 🧹 测试数据清理

### 1. 自动清理脚本

#### cleanup_test_data.sh
```bash
#!/bin/bash
# cleanup_test_data.sh

echo "=== 清理测试数据 ==="

# 清理测试数据库
& "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT -e "
  USE neoscan_test;
  DELETE FROM user_roles WHERE user_id > 1;
  DELETE FROM users WHERE id > 1;
  DELETE FROM role_permissions WHERE id > 3;
  DELETE FROM roles WHERE id > 1;
  DELETE FROM permissions WHERE id > 3;
"

# 清理Redis测试数据
redis-cli -n 1 FLUSHDB

echo "测试数据清理完成"
```

### 2. 测试环境重置

#### reset_test_env.sh
```bash
#!/bin/bash
# reset_test_env.sh

echo "=== 重置测试环境 ==="

# 重新创建测试数据库
& "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT -e "DROP DATABASE IF EXISTS neoscan_test; CREATE DATABASE neoscan_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 重新导入数据结构
Get-Content -Path "database_schema.sql" -Encoding UTF8 | & "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT neoscan_test

# 清理Redis
redis-cli -n 1 FLUSHDB

echo "测试环境重置完成"
```

## 📈 测试报告

### 1. 测试覆盖率报告

#### 生成覆盖率报告
```bash
# 生成测试覆盖率报告
go test ./test -coverprofile=coverage.out -v
go tool cover -html=coverage.out -o coverage.html

# 查看覆盖率统计
go tool cover -func=coverage.out
```

#### 预期覆盖率目标
- **总体覆盖率**: > 80%
- **核心业务逻辑**: > 90%
- **API接口**: > 85%
- **数据库操作**: > 95%

### 2. 性能测试报告

#### 基准测试
```bash
# 运行基准测试
go test ./test -bench=. -benchmem -v
```

#### 预期性能指标
```
BenchmarkUserLogin-8         1000    1000000 ns/op    1024 B/op    10 allocs/op
BenchmarkUserProfile-8       2000     500000 ns/op     512 B/op     5 allocs/op
BenchmarkTokenRefresh-8      1500     750000 ns/op     768 B/op     8 allocs/op
```

## 🚨 故障排除

### 1. 常见问题

#### 数据库连接失败
**问题**: `ERROR 1045 (28000): Access denied for user 'root'@'localhost'`

**解决方案**:
1. 检查MySQL服务是否运行
2. 验证用户名和密码
3. 确认数据库权限设置

```bash
# 检查MySQL服务状态
Get-Service MySQL80

# 重置MySQL密码（如果需要）
& "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -p
```

#### 测试数据库不存在
**问题**: `Error 1049: Unknown database 'neoscan_test'`

**解决方案**:
```bash
# 创建测试数据库
& "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT -e "CREATE DATABASE neoscan_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

#### Redis连接失败
**问题**: `dial tcp 127.0.0.1:6379: connect: connection refused`

**解决方案**:
1. 启动Redis服务
2. 检查Redis配置
3. 验证端口是否被占用

```bash
# 启动Redis服务
redis-server

# 测试Redis连接
redis-cli ping
```

### 2. 调试技巧

#### 启用详细日志
```bash
# 设置日志级别为DEBUG
export LOG_LEVEL=debug
go run main.go
```

#### 使用调试器
```bash
# 使用delve调试器
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug main.go
```

#### 查看测试详细输出
```bash
# 运行测试并显示详细输出
go test ./test -v -count=1

# 运行特定测试
go test ./test -run TestUserRepository/TestCreateUser -v
```

## 📚 最佳实践

### 1. 测试编写原则

- **独立性**: 每个测试应该独立运行，不依赖其他测试
- **可重复性**: 测试结果应该一致和可预测
- **清晰性**: 测试名称和逻辑应该清晰易懂
- **完整性**: 覆盖正常流程、边界条件和错误情况

### 2. 测试数据管理

- **使用事务**: 在测试中使用数据库事务，测试结束后回滚
- **数据隔离**: 每个测试使用独立的测试数据
- **清理机制**: 确保测试后清理所有测试数据

### 3. 性能测试建议

- **基线建立**: 建立性能基线，监控性能变化
- **渐进测试**: 从小并发开始，逐步增加负载
- **资源监控**: 同时监控CPU、内存、数据库等资源使用

### 4. 安全测试要点

- **输入验证**: 测试各种无效输入和边界值
- **权限验证**: 确保权限控制正确实施
- **令牌安全**: 测试令牌的生成、验证和过期机制

## 🔄 持续集成

### 1. GitHub Actions配置

#### .github/workflows/test.yml
```yaml
name: Test

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: ROOT
          MYSQL_DATABASE: neoscan_test
        ports:
          - 3306:3306
        options: --health-cmd="mysqladmin ping" --health-interval=10s --health-timeout=5s --health-retries=3
      
      redis:
        image: redis:6.0
        ports:
          - 6379:6379
        options: --health-cmd="redis-cli ping" --health-interval=10s --health-timeout=5s --health-retries=3
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.19
    
    - name: Install dependencies
      run: go mod download
    
    - name: Setup database
      run: |
        mysql -h 127.0.0.1 -u root -pROOT neoscan_test < database_schema.sql
    
    - name: Run tests
      run: go test ./test -v -coverprofile=coverage.out
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

### 2. 本地CI脚本

#### ci_test.sh
```bash
#!/bin/bash
# ci_test.sh - 本地持续集成测试脚本

set -e

echo "=== NeoScan Master CI 测试 ==="

# 检查环境
echo "检查Go版本..."
go version

echo "检查MySQL连接..."
& "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT -e "SELECT 1;"

echo "检查Redis连接..."
redis-cli ping

# 准备测试环境
echo "准备测试数据库..."
& "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT -e "DROP DATABASE IF EXISTS neoscan_test; CREATE DATABASE neoscan_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
Get-Content -Path "database_schema.sql" -Encoding UTF8 | & "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe" -u root -pROOT neoscan_test

# 运行测试
echo "运行单元测试..."
go test ./test -run TestUserModel -v

echo "运行仓库测试..."
go test ./test -run TestUserRepository -v

echo "运行集成测试..."
go test ./test -run TestAPIIntegration -v

# 生成覆盖率报告
echo "生成覆盖率报告..."
go test ./test -coverprofile=coverage.out -v
go tool cover -func=coverage.out

# 构建检查
echo "检查构建..."
go build ./...

echo "=== 所有测试通过 ==="
```

---

**文档维护**: 本测试指南与代码同步更新，确保测试用例覆盖所有功能点。  
**最后更新**: 2025-09-25  
**文档版本**: v4.0  
**测试覆盖率目标**: > 80%
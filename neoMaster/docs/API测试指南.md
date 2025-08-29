# NeoScan Master API 测试指南

## 📋 文档概览

本目录包含了完整的API测试文档和工具配置文件：

- **API接口文档.md** - 详细的API接口说明文档
- **openapi.yaml** - OpenAPI 3.0规范文件，可导入Apifox、Swagger等工具
- **NeoScan-API.postman_collection.json** - Postman集合文件
- **API测试指南.md** - 本文档，快速上手指南

## 🚀 快速开始

### 1. 确认服务器运行状态

首先确保NeoScan Master服务器正在运行：

```bash
# 在neoMaster目录下启动服务器
cd neoMaster
go run cmd/master/main.go
```

服务器启动后会显示：
```
Starting server on 0.0.0.0:8123
```

### 2. 测试基础连接

使用浏览器或curl测试健康检查接口：

```bash
# 健康检查
curl http://localhost:8123/api/health

# 预期响应
{"status":"healthy","timestamp":""}
```

## 🛠️ 工具配置

### Apifox 配置

1. **导入OpenAPI文档**：
   - 打开Apifox
   - 新建项目
   - 选择"导入数据" → "OpenAPI"
   - 上传 `openapi.yaml` 文件

2. **环境配置**：
   - 创建环境：`NeoScan Local`
   - 添加变量：
     ```
     base_url: http://localhost:8123
     access_token: (登录后自动填充)
     ```

3. **认证配置**：
   - 在需要认证的接口中添加Header：
     ```
     Authorization: Bearer {{access_token}}
     ```

### Postman 配置

1. **导入集合**：
   - 打开Postman
   - 点击"Import"
   - 选择 `NeoScan-API.postman_collection.json` 文件

2. **环境变量**：
   集合已包含以下变量：
   ```
   base_url: http://localhost:8123
   access_token: (登录后自动填充)
   refresh_token: (登录后自动填充)
   ```

3. **自动令牌管理**：
   登录接口包含自动脚本，会自动保存令牌到环境变量

### Swagger UI 配置

如果你想使用Swagger UI：

1. 在线版本：
   - 访问 https://editor.swagger.io/
   - 将 `openapi.yaml` 内容粘贴进去

2. 本地版本：
   ```bash
   # 使用Docker运行Swagger UI
   docker run -p 8080:8080 -e SWAGGER_JSON=/openapi.yaml -v $(pwd)/docs:/usr/share/nginx/html swaggerapi/swagger-ui
   ```

## 🔐 认证流程

### 基础认证测试

1. **测试登录接口**：
   ```bash
   curl -X POST http://localhost:8123/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username":"admin","password":"admin123"}'
   ```

2. **保存访问令牌**：
   从响应中提取 `access_token` 和 `refresh_token`

3. **测试认证接口**：
   ```bash
   curl -X GET http://localhost:8123/api/v1/user/profile \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
   ```

### 令牌管理

- **访问令牌有效期**: 24小时
- **刷新令牌有效期**: 7天
- **自动刷新**: 使用 `/api/v1/auth/refresh` 接口

## 📊 测试场景

### 场景1：基础功能测试

1. 健康检查 → 登录 → 获取用户信息 → 登出

### 场景2：管理员功能测试

1. 管理员登录 → 获取用户列表 → 创建用户 → 更新用户 → 删除用户

### 场景3：权限验证测试

1. 普通用户登录 → 尝试访问管理员接口（应返回403）

### 场景4：令牌生命周期测试

1. 登录 → 等待令牌过期 → 使用刷新令牌 → 继续访问

## ⚠️ 注意事项

### 数据库依赖

当前服务器可以在没有数据库的情况下启动，但以下功能需要数据库：

- ✅ 健康检查接口（无需数据库）
- ❌ 用户登录验证（需要MySQL）
- ❌ 用户信息查询（需要MySQL）
- ❌ 会话管理（需要Redis）

### 启动数据库服务

如需完整测试，请启动数据库服务：

```bash
# MySQL (Windows)
net start mysql

# Redis (Windows)
# 下载并启动Redis服务器
redis-server
```

### 初始数据

数据库启动后，需要：
1. 创建数据库表结构（参考 `sql/database_schema.sql`）
2. 插入初始管理员用户数据

## 🐛 常见问题

### Q: 登录返回"invalid hash format"错误
**A**: 数据库中的用户密码格式不正确，需要使用正确的bcrypt哈希格式

### Q: 管理员接口返回403权限不足
**A**: 确认用户具有管理员角色，检查JWT令牌中的角色信息

### Q: 连接被拒绝错误
**A**: 确认服务器正在运行，检查端口8123是否被占用

### Q: 数据库连接失败
**A**: 检查MySQL和Redis服务是否启动，配置文件中的连接参数是否正确

## 📞 技术支持

如遇到问题，请检查：

1. **服务器日志**: 查看控制台输出的错误信息
2. **配置文件**: 确认 `configs/config.yaml` 配置正确
3. **网络连接**: 确认防火墙没有阻止8123端口
4. **依赖服务**: 确认MySQL和Redis服务状态

---

**祝你测试愉快！** 🎉

如果你发现任何问题或有改进建议，欢迎反馈。
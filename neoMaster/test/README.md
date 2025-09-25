# NeoScan Master 测试套件使用说明

## 概述

本测试套件包含了 NeoScan Master 项目的完整测试集合，涵盖了从基础数据模型到API接口的全方位测试。所有测试文件均按照统一格式命名，并包含详细的测试说明。

## 测试文件命名规范

所有测试文件按照以下格式命名：
```
YYYYMMDD-功能模块-测试结果_test.go
```

例如：`20250925-RoleHandler-pass_test.go`

## 测试文件列表

| 文件名 | 测试内容 | 独立测试命令 |
|--------|---------|-------------|
| 20250925-ChangePasswordHandler-pass_test.go | 用户修改密码功能 | `go test -v -run TestChangePasswordHandler ./test` |
| 20250925-DataInitialization-pass_test.go | 测试数据初始化 | 基础框架文件，无独立测试 |
| 20250925-DeprecatedAPI-pass_test.go | 已弃用API接口 | `go test -v -run TestDeprecatedAPI ./test` |
| 20250925-JWTFunctionality-pass_test.go | JWT令牌功能 | `go test -v -run TestJWTService ./test` |
| 20250925-MiddlewareFunctionality-pass_test.go | 中间件功能 | `go test -v -run TestMiddlewareChaining ./test` |
| 20250925-MySQLConnection-pass_test.go | MySQL连接测试 | `go test -v -run TestMySQLConnection ./test` |
| 20250925-PermissionHandler-pass_test.go | 权限处理器功能 | `go test -v -run TestPermissionHandler ./test` |
| 20250925-RoleHandler-pass_test.go | 角色处理器功能 | `go test -v -run TestRoleHandler ./test` |
| 20250925-SessionAPI-pass_test.go | 会话管理API接口 | `go test -v -run TestSessionAPI ./test` |
| 20250925-SessionHandler-pass_test.go | 会话处理器功能 | `go test -v -run TestSessionHandler ./test` |
| 20250925-TestBaseFramework-pass_test.go | 测试基础框架 | 基础框架文件，无独立测试 |
| 20250925-UserFunctionality-pass_test.go | 用户功能（模型、仓库、服务） | `go test -v -run TestUserModel ./test` 或 `go test -v -run TestUserRepository ./test` 或 `go test -v -run TestUserService ./test` |

## 运行测试

### 运行所有测试
```bash
go test -v ./test
```

### 运行特定测试文件
```bash
go test -v ./test/20250925-RoleHandler-pass_test.go
```

### 运行特定测试函数
```bash
go test -v -run TestRoleHandler ./test
```

## 测试结果评估

### 通过标准
- 所有测试用例执行完成无错误
- 断言条件满足预期结果
- 数据库状态符合预期变化

### 失败情况处理
- 查看详细错误信息和日志输出
- 检查测试环境配置（数据库、Redis等）
- 确认代码变更是否影响了相关功能

## 日志输出

测试运行时会将结果输出到 `logs/test.log` 文件中，方便开发人员查看和分析测试结果。

```bash
go test -v ./test > logs/test.log 2>&1
```

## 注意事项

1. 运行测试前请确保测试数据库 `neoscan_test` 可用
2. 测试会自动清理和初始化测试数据，请勿在生产环境中运行
3. 部分测试可能因为令牌版本更新导致401错误，这是正常现象
4. 测试过程中会输出详细的执行日志，便于调试和问题定位
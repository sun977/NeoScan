# 认证与授权模块测试说明（当前阶段）

本说明覆盖当前已完成的认证与授权模块功能的测试用例与验证范围，旨在帮助快速理解与运行测试。本文不包含性能测试。

## 一、如何运行测试

1. 准备依赖服务（如需要）：
   - MySQL（测试库已在 `sql/test_schema.sql` 提供）
   - Redis
2. 设置必要的环境变量（示例）：
   - `APP_ENV=test`
   - `MYSQL_DSN` 指向测试数据库
   - `REDIS_ADDR` 指向测试 Redis
   - `JWT_SECRET` 测试用密钥
3. 在项目根目录执行：
```
go test ./neoMaster/test -v
```
或执行单个文件：
```
go test ./neoMaster/test -run ^TestName$ -v
```

## 二、测试套件概览与覆盖功能

- api_integration_test.go（集成）
  - 用户注册、登录、刷新令牌、登出完整流转
  - 受保护路由访问校验（JWT + 活跃用户）
  - 管理员路由访问校验（Admin 角色）
  - 用户、角色、权限的核心管理端接口联动

- auth_service_test.go（服务）
  - SessionService 登录、登出、刷新令牌、会话写入与校验
  - JWTService 令牌生成、校验、过期与剩余时间计算
  - RBACService 角色与权限判定（CheckRole / CheckPermission / Any/All）

- jwt_test.go（JWT 基础）
  - 访问令牌与刷新令牌生成与解析
  - 过期时间、签名与声明校验

- jwt_blacklist_test.go（令牌黑名单）
  - 令牌撤销（RevokeToken）与黑名单校验（IsTokenRevoked）

- middleware_test.go（中间件）
  - CORS、安全响应头、日志、限流中间件行为
  - JWT 鉴权与用户活跃状态校验
  - 管理员角色校验（Admin 访问控制）

- user_test.go（用户）
  - 用户创建、查询、分页与状态管理
  - 修改密码（ChangePassword）逻辑（版本号递增、旧 Token 失效）
  - 管理端重置密码（ResetUserPassword，默认 123456）
  - 用户角色与权限获取

- change_password_handler_test.go（Handler）
  - 修改密码接口的入参、鉴权与业务路径

- session_handler_test.go（Handler）
  - 会话管理接口的入参、鉴权与业务路径
  - 管理员列出用户活跃会话
  - 管理员撤销指定用户会话
  - 管理员撤销指定用户所有会话

- data_test.go（数据一致性）
  - 关键实体读写一致性与边界条件验证

- mysql_test.go（存储层）
  - MySQL 连接、基础 CRUD 与事务行为的最小验证

## 三、与近期迭代对齐的新增/变更点

- 权限管理（Permission）
  - Repository：权限 CRUD、分页、存在性检查、与角色关联读取与清理
  - Service：权限创建/更新/删除（事务清理关联）、列表与查询
  - Handler：管理员权限接口（list/create/get/update/delete）

- 角色管理（Role）
  - Service：状态变更通用函数（启用/禁用）、更新与删除均采用事务并记录审计日志
  - Repository：角色字段原子更新、权限关联增删

- 会话管理（Session）
  - Service：新增 `GetUserSessions`、`DeleteUserSession` 包装，以供管理端会话控制
  - Handler：管理员列出活跃会话、撤销指定用户会话、撤销某用户全部会话

- 用户密码管理（Password/User）
  - PasswordService：`ChangePassword` 原子更新密码并递增版本，缓存同步与会话清理
  - UserService：`ResetUserPassword` 管理端重置为 `123456`，原子更新版本并清理会话

以上变更均已在相关测试中覆盖或通过集成流转间接验证。

## 四、测试数据与前置条件

- 数据库表结构：见 `neoMaster/sql/test_schema.sql`
- 必要初始数据：可在 `base_test.go` 或各自测试的 `setup`/`teardown` 中构造
- 日志：所有关键路径均记录业务日志与错误日志，测试可通过断言响应与副作用验证

## 五、排除项

- 本阶段不包含性能与压测用例
- 不包含前端 UI 层面的 E2E 测试

## 六、常见问题

- 令牌相关测试失败：确认 `JWT_SECRET` 与时间相关断言（比如过期阈值）
- Redis/MySQL 连接失败：检查环境变量与服务是否启动

---

如需为新接口补充测试，请参考现有文件的组织方式与断言风格，保持日志与错误信息的一致性。
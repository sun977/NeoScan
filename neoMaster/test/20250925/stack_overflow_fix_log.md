# 栈溢出问题修复日志

## 问题诊断

### 错误现象
```
runtime: goroutine stack exceeds 1000000000-byte limit
runtime: sp=0xc02c9074b0 stack=[0xc02c906000, 0xc04c906000]
Detaching and terminating target process
dlv dap (92176) exited with code: 0
```

### 根本原因分析
通过代码分析发现，问题的根源是 **JWTService** 和 **SessionService** 之间存在循环依赖：

1. **SessionService** 需要 **JWTService** 来生成和验证令牌
2. **JWTService** 原本设计需要 **SessionService** 作为令牌黑名单服务
3. 在服务初始化时形成了循环引用，导致无限递归调用

### Linus式分析

**【核心判断】**
✅ 值得做：这是典型的设计缺陷，循环依赖违反了"好品味"原则

**【关键洞察】**
- 数据结构：服务依赖关系形成了环，这是设计的根本错误
- 复杂度：临时对象创建和重新赋值的复杂逻辑是糟糕设计的补丁
- 风险点：栈溢出会导致整个应用崩溃

**【Linus式方案】**
"把这个循环依赖消除掉" - 使用接口解耦，让依赖关系变成单向的

## 修复方案

### 1. 重构SessionService - 使用接口解耦

**修改文件**: `internal/service/auth/session.go`

**核心改动**:
```go
// 新增TokenGenerator接口
type TokenGenerator interface {
    GenerateTokens(ctx context.Context, user *model.User) (*auth.TokenPair, error)
    ValidateAccessToken(tokenString string) (*auth.JWTClaims, error)
    RefreshTokens(ctx context.Context, refreshToken string) (*auth.TokenPair, error)
    CheckTokenExpiry(tokenString string, threshold time.Duration) (bool, error)
    GetTokenRemainingTime(tokenString string) (time.Duration, error)
    ValidatePasswordVersion(ctx context.Context, tokenString string) (bool, error)
}

// SessionService结构体修改
type SessionService struct {
    userService     *UserService
    passwordManager *auth.PasswordManager
    tokenGenerator  TokenGenerator // 使用接口而不是具体实现
    rbacService     *RBACService
    sessionRepo     *redis.SessionRepository
}

// 新增SetTokenGenerator方法
func (s *SessionService) SetTokenGenerator(tokenGenerator TokenGenerator) {
    s.tokenGenerator = tokenGenerator
}
```

**替换所有jwtService调用**:
- `s.jwtService.GenerateTokens()` → `s.tokenGenerator.GenerateTokens()`
- `s.jwtService.ValidateAccessToken()` → `s.tokenGenerator.ValidateAccessToken()`
- 等等...

### 2. 更新服务初始化逻辑

**修改文件**: `internal/app/master/router.go`

**新的初始化顺序**:
```go
// 先创建SessionService（不传入JWTService）
sessionService := authService.NewSessionService(userService, passwordManager, rbacService, sessionRepo)

// 再创建JWTService
jwtService := authService.NewJWTService(jwtManager, userService, sessionRepo)

// 设置SessionService的TokenGenerator（解决循环依赖）
sessionService.SetTokenGenerator(jwtService)
```

**移除的复杂逻辑**:
- 删除了临时对象创建
- 删除了重新赋值逻辑
- 删除了注释掉的复杂代码

### 3. 修复测试文件

**修改文件**: `test/base_test.go`

应用相同的依赖注入模式，确保测试环境也使用正确的初始化顺序。

## 技术细节

### 依赖关系优化

**修复前**:
```
SessionService ←→ JWTService (循环依赖)
```

**修复后**:
```
SessionService → TokenGenerator ← JWTService (单向依赖)
```

### 符合Linus的"好品味"原则

1. **消除特殊情况**: 不再需要临时对象和重新赋值的特殊处理
2. **简化复杂度**: 从复杂的多步初始化简化为清晰的三步流程
3. **单一职责**: 每个服务只关心自己的核心功能
4. **接口隔离**: 通过接口解耦，降低耦合度

## 验证步骤

1. 编译检查：确保所有文件编译通过
2. 单元测试：运行现有测试用例
3. 集成测试：启动完整应用验证功能
4. 压力测试：确认栈溢出问题已解决

## 预期结果

- ✅ 消除栈溢出错误
- ✅ 保持所有现有功能不变
- ✅ 提高代码可维护性
- ✅ 降低系统复杂度

---

**修复时间**: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
**修复原则**: "Never break userspace" - 保持API兼容性
**代码品味**: 从复杂的循环依赖优化为简洁的接口解耦
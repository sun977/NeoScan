# 密码版本控制功能说明

## 概述

密码版本控制（Password Version Control）是一个安全功能，确保用户修改密码后，所有旧的JWT令牌立即失效。这可以防止已泄露的令牌在密码修改后继续被恶意使用。

## 实现原理

### 1. 数据模型层

在用户模型中添加了 `PasswordV` 字段：

```go
type User struct {
    ID        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
    Username  string `json:"username" gorm:"uniqueIndex;not null;size:50"`
    Email     string `json:"email" gorm:"uniqueIndex;not null;size:100"`
    Password  string `json:"-" gorm:"not null;size:255"`
    PasswordV int64  `json:"-" gorm:"default:1;comment:密码版本号,用于使旧token失效"` // 新增字段
    // ... 其他字段
}
```

### 2. JWT令牌结构

在JWT Claims中包含密码版本号：

```go
type JWTClaims struct {
    UserID    uint     `json:"user_id"`
    Username  string   `json:"username"`
    Email     string   `json:"email"`
    PasswordV int64    `json:"password_v"` // 密码版本号
    Roles     []string `json:"roles"`
    jwt.RegisteredClaims
}
```

### 3. 工作流程

#### 用户登录时：
1. 验证用户名和密码
2. 获取用户当前的密码版本号
3. 将密码版本号嵌入到JWT令牌中
4. 返回包含密码版本的令牌

#### 令牌验证时：
1. 解析JWT令牌获取其中的密码版本号
2. 从数据库或缓存获取用户当前的密码版本号
3. 比较两个版本号是否一致
4. 如果不一致，拒绝访问并要求重新登录

#### 修改密码时：
1. 验证旧密码
2. 原子性地更新密码和递增密码版本号
3. 更新缓存中的密码版本号
4. 删除用户所有会话（可选）

## 核心代码实现

### 1. 数据访问层

```go
// UpdatePasswordWithVersion 更新用户密码并递增密码版本号
func (r *UserRepository) UpdatePasswordWithVersion(ctx context.Context, userID uint, passwordHash string) error {
    query := `
        UPDATE users 
        SET password_hash = ?, password_v = password_v + 1, updated_at = ?
        WHERE id = ? AND deleted_at IS NULL
    `
    
    result, err := r.db.ExecContext(ctx, query, passwordHash, time.Now(), userID)
    if err != nil {
        return fmt.Errorf("failed to update password with version: %w", err)
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if rowsAffected == 0 {
        return fmt.Errorf("user not found or already deleted")
    }
    
    return nil
}
```

### 2. JWT服务层

```go
// ValidatePasswordVersion 验证令牌中的密码版本是否与用户当前密码版本匹配
func (s *JWTService) ValidatePasswordVersion(ctx context.Context, tokenString string) (bool, error) {
    claims, err := s.ValidateAccessToken(tokenString)
    if err != nil {
        return false, err
    }
    
    // 优先从缓存获取密码版本
    currentPasswordV, err := s.userRepo.GetUserPasswordVersion(ctx, uint(claims.UserID))
    if err != nil {
        return false, fmt.Errorf("failed to get user password version: %w", err)
    }
    
    // 检查密码版本是否匹配
    return claims.PasswordV == currentPasswordV, nil
}
```

### 3. 中间件层

```go
// JWT认证中间件中添加密码版本验证
func (m *MiddlewareManager) JWTAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ... 提取和验证令牌 ...
        
        // 验证密码版本（确保修改密码后旧token失效）
        validVersion, err := m.jwtService.ValidatePasswordVersion(r.Context(), accessToken)
        if err != nil {
            m.writeErrorResponse(w, http.StatusUnauthorized, "failed to validate token version", err)
            return
        }
        if !validVersion {
            m.writeErrorResponse(w, http.StatusUnauthorized, "token version mismatch, please login again", nil)
            return
        }
        
        // ... 继续处理请求 ...
    })
}
```

## 使用方式

### 1. 修改密码

当用户修改密码时，系统会：
- 验证旧密码
- 更新新密码
- 自动递增 `password_v` 字段
- 更新缓存中的密码版本
- 使所有旧令牌失效

```bash
POST /api/v1/auth/change-password
Authorization: Bearer <access_token>
Content-Type: application/json

{
    "old_password": "oldpassword123",
    "new_password": "newpassword456"
}
```

### 2. 令牌验证

每次API请求时，中间件会：
- 验证JWT令牌的有效性
- 检查令牌中的密码版本号
- 与数据库中的当前版本号比较
- 如果版本不匹配，拒绝请求

## 安全价值

### 1. 防止令牌重放攻击
- 即使攻击者获得了用户的JWT令牌，一旦用户修改密码，所有旧令牌立即失效
- 攻击者无法继续使用已泄露的令牌

### 2. 强制会话失效
- 用户修改密码后，所有设备上的登录会话都会失效
- 用户需要重新登录，确保账户安全

### 3. 审计和监控
- 可以通过密码版本号追踪用户的密码修改历史
- 便于安全审计和异常检测

## 性能优化

### 1. 缓存策略
- 密码版本号会缓存到Redis中，减少数据库查询
- 缓存过期时间可配置，平衡性能和一致性

### 2. 批量验证
- 对于高频API请求，可以考虑批量验证密码版本
- 使用布隆过滤器等数据结构优化验证性能

## 配置说明

### 环境变量
```bash
# JWT密钥
JWT_SECRET=your-secret-key

# 令牌过期时间
JWT_ACCESS_TOKEN_TTL=1h
JWT_REFRESH_TOKEN_TTL=24h

# 缓存过期时间
PASSWORD_VERSION_CACHE_TTL=1h
```

### 数据库迁移
```sql
-- 添加密码版本字段
ALTER TABLE users ADD COLUMN password_v BIGINT DEFAULT 1 COMMENT '密码版本号,用于使旧token失效';

-- 创建索引（可选，用于性能优化）
CREATE INDEX idx_users_password_v ON users(password_v);
```

## 测试用例

系统包含完整的测试用例，验证密码版本控制功能：

1. **TestPasswordVersionControl**: 测试完整的密码修改流程
2. **TestPasswordVersionInToken**: 测试JWT令牌中包含密码版本
3. **TestTokenValidationAfterPasswordChange**: 测试密码修改后令牌验证

运行测试：
```bash
go test ./test -v -run TestPasswordVersion
```

## 注意事项

1. **数据库事务**: 密码更新和版本号递增必须在同一个事务中执行
2. **缓存一致性**: 确保缓存中的密码版本与数据库保持一致
3. **性能影响**: 每次令牌验证都需要检查密码版本，会增加一定的性能开销
4. **用户体验**: 密码修改后用户需要重新登录，需要在UI中给出明确提示

## 扩展功能

### 1. 选择性失效
- 可以扩展为只让特定设备的令牌失效
- 通过设备ID或会话ID进行精确控制

### 2. 密码历史
- 记录密码版本的修改历史
- 防止用户重复使用近期密码

### 3. 安全事件通知
- 密码修改时发送邮件或短信通知
- 记录安全事件日志
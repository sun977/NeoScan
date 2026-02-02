# NeoAgent 注册与认证机制设计方案 (Simplified v2.2)

## 1. 核心设计原则

遵循 **KISS (Keep It Simple, Stupid)** 原则，移除冗余的中间状态。

- **单一凭证**: `agents` 表中的 `token` 字段即为 Agent 的唯一身份凭证 (API Key)。
- **两种接入方式**:
  1.  **预分发 (Manual)**: 管理员预先生成 Token，Agent 直接使用。
  2.  **自动注册 (Auto)**: 基于全局共享密钥 (Global Secret) 换取专属 Token。

---

## 2. 数据模型 (Database Schema)

**无需新增表**。复用现有 `agents` 表，确保以下字段定义清晰：

```sql
CREATE TABLE `agents` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `agent_id` varchar(100) NOT NULL COMMENT 'Agent逻辑ID',
  `token` varchar(128) NOT NULL COMMENT '核心认证凭据(API Key)',
  `hostname` varchar(255) DEFAULT NULL,
  `ip_address` varchar(50) DEFAULT NULL,
  `status` varchar(20) DEFAULT 'offline',
  `fingerprint` varchar(128) DEFAULT NULL COMMENT '硬件指纹(可选)',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_agent_id` (`agent_id`),
  UNIQUE KEY `idx_token` (`token`) -- 必须确保Token唯一，用于快速鉴权
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 3. 交互流程

### 3.1 场景 A：自动注册 (推荐用于批量部署)

Master 在配置文件中设置全局注册密钥。

**Master Config (`config.yaml`)**:
```yaml
security:
  # Agent 通信与数据安全配置
  agent:
    token_secret: "neo_scan_secret_key_2026"      # 全局注册暗号，用于验证 Agent 身份
```

**Step 1: Agent 发起注册**
Agent 启动时无 Token，使用暗号请求注册。

**API**: `POST /api/v1/agent/register`
**Request**:
```json
{
  "token_secret": "neo_scan_secret_key_2026",
  "hostname": "scanner-01",
  "version": "1.0.0"
}
```

**Step 2: Master 处理**
1. 比对 `token_secret` 是否与配置文件中的 `security.agent.token_secret` 一致。
2. 若一致，生成唯一 `agent_id` 和随机 `token` (e.g., `nk_7f8a9b...`)。
3. 写入 `agents` 表。
4. 返回 `token`。

**Response**:
```json
{
  "code": 200,
  "data": {
    "agent_id": "agent_scanner_01_uuid",
    "token": "nk_7f8a9b..." // <--- Agent 需落盘保存到 agent.yaml
  }
}
```

### 3.2 场景 B：手动/预配置 (适用于高安全环境)

1. **管理员**在数据库或管理后台手动 `INSERT INTO agents`，并生成一个 Token (e.g., `manual_token_123`)。
2. **管理员**将该 Token 写入 Agent 的配置文件 `agent.yaml`。
3. **Agent** 启动，直接使用该 Token 通信，跳过注册步骤。

---

## 4. 运行时鉴权 (Runtime Auth)

所有非注册接口（心跳、任务领取等）均采用统一鉴权逻辑。

**Header**: `Authorization: Bearer <token>`

**Middleware Logic (`GinAgentAuthMiddleware`)**:
```go
func GinAgentAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 提取 Token
        token := extractBearerToken(c)
        if token == "" {
            c.AbortWithStatus(401)
            return
        }

        // 2. 数据库/缓存查验 (建议对 Token 做 LRU 缓存)
        agent, err := agentRepo.GetAgentByToken(token)
        if err != nil || agent == nil {
             c.AbortWithStatusJSON(401, gin.H{"error": "Invalid Token"})
             return
        }

        // 3. 注入身份
        c.Set("agent_id", agent.AgentID)
        c.Next()
    }
}
```

---

## 5. 接口规范

### 5.1 注册接口
**Path**: `/api/v1/agent/register`
**Method**: `POST`
**Auth**: None (校验 Body 中的 `token_secret`)

### 5.2 心跳接口
**Path**: `/api/v1/agent/heartbeat`
**Method**: `POST`
**Auth**: Bearer Token

---

## 6. 安全性说明

- **Secret 保护**: 全局 `token_secret` 仅用于新节点接入，泄露后可修改 Master 配置并重启（不影响已注册 Agent）。
- **Token 隔离**: 每个 Agent 拥有独立 Token，单点泄露不影响全局，可随时重置特定 Agent 的 Token。
- **通信加密**: 必须强制使用 HTTPS，防止 Token 在传输层被嗅探。

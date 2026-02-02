# NeoAgent 注册与认证机制设计方案 (Refined v2.0)

## 1. 核心设计原则

为彻底分离**用户访问控制** (User RBAC) 与 **Agent 接入控制** (Machine-to-Machine Auth)，本方案采用双层令牌机制。

- **User Auth**: 使用 JWT (Access/Refresh Token)，面向人类管理员，用于操作 Master API。
- **Agent Auth**: 
  1.  **Join Token (准入令牌)**: 短期有效，由管理员生成，用于 Agent 首次注册（握手）。
  2.  **Agent Secret (通信凭证)**: 长期有效，由 Master 颁发，用于 Agent 后续通信。

---

## 2. 数据模型设计 (Database Schema)

### 2.1 Join Token 表 (`sys_join_tokens`)
用于管理允许 Agent 接入的凭证。

```sql
CREATE TABLE `sys_join_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `token` varchar(64) NOT NULL COMMENT '准入令牌(jt_开头)',
  `name` varchar(100) DEFAULT NULL COMMENT '令牌备注/名称',
  `usage_limit` int DEFAULT 1 COMMENT '最大使用次数(0为不限)',
  `usage_count` int DEFAULT 0 COMMENT '已使用次数',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `created_by` bigint unsigned DEFAULT 0 COMMENT '创建人ID',
  `initial_tags` json DEFAULT NULL COMMENT '自动绑定的标签ID列表',
  `is_active` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_token` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Agent准入令牌表';
```

### 2.2 Agent 表更新 (`agents`)
现有 `agents` 表需明确字段用途，确保 `token` 字段存储的是长期通信凭证。

```sql
-- 确认 agents 表包含以下关键认证字段
ALTER TABLE `agents` 
  MODIFY COLUMN `token` varchar(128) NOT NULL COMMENT 'Agent通信密钥(API Key)',
  ADD COLUMN `join_token_id` bigint unsigned DEFAULT 0 COMMENT '关联的准入令牌ID(审计用)',
  ADD COLUMN `fingerprint` varchar(128) DEFAULT NULL COMMENT 'Agent硬件指纹(防伪造)';
```

---

## 3. 详细交互流程

### 3.1 阶段一：管理员生成准入令牌 (Admin)

**API**: `POST /api/v1/orchestrator/agent-join-tokens`
**Auth**: User JWT (Admin Only)

**Request**:
```json
{
  "name": "Production Cluster Deployment",
  "usage_limit": 100,
  "ttl_seconds": 86400,
  "tags": ["prod", "linux"]
}
```

**Response**:
```json
{
  "code": 200,
  "data": {
    "token": "jt_7c4a8d09ca3762af", 
    "expires_at": "2026-02-03T10:00:00Z"
  }
}
```

### 3.2 阶段二：Agent 首次注册 (Handshake)

Agent 启动时，若无本地凭证，使用启动参数中的 Token 发起注册。

**Command**: 
```bash
./neoAgent join --master 10.0.0.1:8080 --token jt_7c4a8d09ca3762af
```

**API**: `POST /api/v1/agent/register` (注意：此接口**不需要** Agent认证，但需要 Join Token)

**Request**:
```json
{
  "join_token": "jt_7c4a8d09ca3762af",
  "hostname": "scanner-01",
  "version": "1.0.0",
  "fingerprint": "hw-id-cpu-serial-xyz", 
  "ip_address": "192.168.1.50"
}
```

**Master 处理逻辑**:
1. 校验 `join_token` 是否存在、未过期、`usage_count < usage_limit`。
2. 若校验通过：
   - 增加 `sys_join_tokens.usage_count`。
   - 创建/更新 `agents` 记录。
   - 生成长期 **API Key** (e.g., `ak_5f3b...`)。
   - 自动绑定 `initial_tags`。
3. 返回 API Key。

**Response**:
```json
{
  "code": 200,
  "data": {
    "agent_id": "agent_scanner_01_uuid",
    "api_key": "ak_5f3b2c9d...",  // <--- 长期凭证，Agent需落盘保存
    "master_ca": "-----BEGIN CERTIFICATE..."
  }
}
```

### 3.3 阶段三：Agent 日常通信 (Runtime)

Agent 获取 API Key 后，后续所有请求（心跳、任务获取）均使用该 Key。

**Auth Header**: 
`Authorization: Bearer ak_5f3b2c9d...`

**Middleware Logic (`GinAgentAuthMiddleware`)**:
1. 拦截 `/api/v1/agent/**` (注册接口除外)。
2. 提取 Bearer Token。
3. 查询 `agents` 表匹配 `token` 字段。
4. 校验 Agent 状态 (Active)。
5. 注入 `agent_id` 到上下文。

---

## 4. 接口规范修订

### 4.1 注册接口
**Path**: `/api/v1/agent/register`
**Method**: `POST`
**Auth**: None (Relies on Join Token in body)

### 4.2 心跳接口
**Path**: `/api/v1/agent/heartbeat`
**Method**: `POST`
**Auth**: Bearer API Key

**Request**:
```json
{
  "status": "idle",
  "load": 15
}
```

---

## 5. 错误码定义

| HTTP Code | Error Code | 说明 | 应对 |
| :--- | :--- | :--- | :--- |
| 401 | `AUTH_JOIN_TOKEN_INVALID` | 准入令牌不存在或已失效 | 提示用户获取新 Token |
| 401 | `AUTH_JOIN_TOKEN_LIMIT` | 准入令牌使用次数耗尽 | 提示用户获取新 Token |
| 403 | `AUTH_AGENT_FORBIDDEN` | API Key 无效或 Agent 被禁用 | Agent 应停止工作并报警 |
| 409 | `AGENT_CONFLICT` | 指纹冲突或重复注册 | 视策略覆盖或报错 |

---

## 6. 安全加固建议 (Future)
1. **API Key 轮换**: 提供 `/rotate-key` 接口，允许 Agent 定期更换密钥。
2. **mTLS**: 在 API Key 基础上，强制要求 Agent 使用 Master 签发的客户端证书进行 TLS 双向认证（最高安全级别）。

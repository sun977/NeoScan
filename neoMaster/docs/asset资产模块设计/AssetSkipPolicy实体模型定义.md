# AssetSkipPolicy实体模型定义

## 概述

AssetSkipPolicy实体用于定义统一的资产跳过策略机制，适用于所有扫描类型。该表设计旨在提供跨扫描类型的通用跳过策略功能，避免为每种扫描类型重复创建策略表。

## 模型结构

### 核心字段

| 字段名 | 类型 | 描述 |
|--------|------|------|
| `id` | uint | 自增主键 |
| `policy_name` | string | 策略名称 |
| `policy_type` | string | 策略类型 |
| `description` | string | 策略描述 |
| `condition_rules` | JSON | 条件规则 |
| `action_config` | JSON | 动作配置 |
| `scope` | JSON | 作用域配置 |
| `priority` | int | 优先级（数值越大优先级越高） |
| `enabled` | bool | 是否启用 |
| `created_by` | string | 创建人 |
| `tags` | JSON | 标签信息 |
| `valid_from` | timestamp | 生效开始时间 |
| `valid_to` | timestamp | 生效结束时间 |
| `created_at` | timestamp | 创建时间 |
| `updated_at` | timestamp | 更新时间 |

### 字段详解

#### 1. policy_type（策略类型）
定义跳过策略的应用类型：
- `global`: 全局跳过策略，适用于所有扫描类型
- `scan_type`: 特定扫描类型跳过策略
- `project`: 特定项目跳过策略
- `workflow`: 特定工作流跳过策略
- `stage`: 特定阶段跳过策略

#### 2. condition_rules（条件规则）
定义触发跳过策略的条件规则：

```json
{
  "conditions": [
    {
      "field": "device_type",
      "operator": "equals",
      "value": "honeypot"
    },
    {
      "field": "os",
      "operator": "contains",
      "value": "honeypot"
    },
    {
      "field": "port_count",
      "operator": "greater_than",
      "value": 1000
    }
  ],
  "logic_operator": "and"  // and/or
}
```

支持的操作符：
- `equals`: 等于
- `not_equals`: 不等于
- `contains`: 包含
- `not_contains`: 不包含
- `starts_with`: 以...开头
- `ends_with`: 以...结尾
- `greater_than`: 大于
- `less_than`: 小于
- `greater_than_or_equal`: 大于等于
- `less_than_or_equal`: 小于等于
- `in`: 在列表中
- `not_in`: 不在列表中
- `is_null`: 为空
- `is_not_null`: 不为空
- `regex`: 正则表达式匹配

#### 3. action_config（动作配置）
定义满足条件时执行的动作：

```json
{
  "action": "skip",  // skip/log/alert/tag
  "log_level": "info",
  "alert_channels": ["email", "slack"],
  "custom_tags": {
    "flag": "suspicious",
    "reason": "honeypot_detected"
  }
}
```

动作类型：
- `skip`: 跳过扫描
- `log`: 仅记录日志
- `alert`: 发送告警
- `tag`: 添加标记

#### 4. scope（作用域配置）
定义策略的作用范围：

```json
{
  "scan_types": ["ip_alive_scan", "port_scan", "vuln_scan"],  // 适用的扫描类型
  "projects": [1, 2, 3],                                      // 适用的项目ID
  "workflows": [10, 20],                                      // 适用的工作流ID
  "stages": [101, 102],                                       // 适用的阶段ID
  "agents": [1, 2],                                           // 适用的Agent ID
  "networks": ["192.168.0.0/16"]                             // 适用的网络范围
}
```

#### 5. tags（标签信息）
用于分类和标记跳过策略：

```json
{
  "business_unit": "security",
  "environment": "production",
  "category": "honeypot",
  "compliance": ["iso27001", "nist"]
}
```

## 设计原则

### 1. 统一性
- 单一表结构支持所有扫描类型的跳过策略需求
- 统一的条件规则和动作处理逻辑
- 一致的管理接口

### 2. 灵活性
- 支持复杂的条件组合（AND/OR逻辑）
- 可配置的动作类型
- 支持时间有效期控制

### 3. 可扩展性
- JSON字段支持未来扩展
- 标签系统便于分类管理
- 支持复杂的组合条件

## 使用场景

### 1. 全局跳过策略
适用于所有扫描任务的通用跳过策略：

```sql
INSERT INTO asset_skip_policies 
(policy_name, policy_type, condition_rules, action_config, enabled)
VALUES 
('Skip Honeypot Devices', 'global',
'{"conditions": [{"field": "device_type", "operator": "equals", "value": "honeypot"}], "logic_operator": "and"}',
'{"action": "skip"}',
true);
```

### 2. 特定扫描类型策略
仅对特定扫描类型生效的跳过策略：

```sql
INSERT INTO asset_skip_policies 
(policy_name, policy_type, condition_rules, action_config, scope, enabled)
VALUES 
('Skip High Port Count', 'scan_type',
'{"conditions": [{"field": "port_count", "operator": "greater_than", "value": 1000}], "logic_operator": "and"}',
'{"action": "skip"}',
'{"scan_types": ["port_scan", "service_scan"]}',
true);
```

### 3. 项目级跳过策略
特定项目中的跳过策略：

```sql
INSERT INTO asset_skip_policies 
(policy_name, policy_type, condition_rules, action_config, scope, enabled)
VALUES 
('Skip Internal Test Servers', 'project',
'{"conditions": [{"field": "hostname", "operator": "contains", "value": "test"}], "logic_operator": "and"}',
'{"action": "log", "log_level": "warn"}',
'{"projects": [101]}',
true);
```

## 条件匹配规则

### 1. 单一条件匹配
```text
条件: device_type = honeypot
匹配: {"device_type": "honeypot"}
不匹配: {"device_type": "router"}
```

### 2. 多条件AND匹配
```text
条件: device_type = honeypot AND os contains linux
匹配: {"device_type": "honeypot", "os": "linux ubuntu"}
不匹配: {"device_type": "honeypot", "os": "windows"}
```

### 3. 多条件OR匹配
```text
条件: device_type = honeypot OR device_type = router
匹配: {"device_type": "honeypot"} 或 {"device_type": "router"}
不匹配: {"device_type": "server"}
```

### 4. 范围条件匹配
```text
条件: port_count > 1000
匹配: {"port_count": 1500}
不匹配: {"port_count": 500}
```

### 5. 列表条件匹配
```text
条件: device_type in [honeypot, router]
匹配: {"device_type": "honeypot"} 或 {"device_type": "router"}
不匹配: {"device_type": "server"}
```

## 性能优化建议

### 1. 索引设计
```sql
-- 主键索引
CREATE INDEX idx_asset_skip_policies_id ON asset_skip_policies(id);

-- 类型和启用状态索引
CREATE INDEX idx_asset_skip_policies_type_enabled ON asset_skip_policies(policy_type, enabled);

-- 优先级索引
CREATE INDEX idx_asset_skip_policies_priority ON asset_skip_policies(priority);

-- 时间范围索引
CREATE INDEX idx_asset_skip_policies_valid ON asset_skip_policies(valid_from, valid_to);

-- 组合索引优化常见查询
CREATE INDEX idx_asset_skip_policies_query ON asset_skip_policies(policy_type, enabled, valid_from, valid_to);
```

### 2. 查询优化
```sql
-- 查询当前生效的全局跳过策略
SELECT * FROM asset_skip_policies 
WHERE policy_type = 'global' 
  AND enabled = true 
  AND (valid_from IS NULL OR valid_from <= NOW())
  AND (valid_to IS NULL OR valid_to >= NOW())
ORDER BY priority DESC;
```

## 与其他实体的关系

### 1. 与ScanStage的关系
- ScanStage中的跳过策略引用AssetSkipPolicy
- 通过scope字段关联特定的扫描阶段

### 2. 与Project的关系
- 项目可以定义专用的跳过策略
- 通过scope.projects字段关联

### 3. 与Asset的关系
- 跳过策略基于资产属性进行匹配
- 用于决定是否跳过特定资产的扫描

## 管理接口设计

### 1. 创建跳过策略
```http
POST /api/skip-policies
Content-Type: application/json

{
  "policy_name": "Skip Honeypot Devices",
  "policy_type": "global",
  "description": "Skip scanning known honeypot devices",
  "condition_rules": {
    "conditions": [
      {
        "field": "device_type",
        "operator": "equals",
        "value": "honeypot"
      }
    ],
    "logic_operator": "and"
  },
  "action_config": {
    "action": "skip"
  },
  "enabled": true,
  "priority": 100
}
```

### 2. 查询跳过策略
```http
GET /api/skip-policies?policy_type=global&enabled=true
```

### 3. 策略匹配检查
```http
POST /api/skip-policies/check
Content-Type: application/json

{
  "asset_data": {
    "device_type": "honeypot",
    "ip": "192.168.1.100",
    "hostname": "honeypot-01"
  },
  "context": {
    "scan_type": "port_scan",
    "project_id": 1,
    "workflow_id": 10
  }
}
```

## 最佳实践

### 1. 策略设计建议
- 合理设置优先级，确保重要策略优先执行
- 使用具体明确的条件，避免过于宽泛的匹配
- 定期审查和清理不再需要的策略

### 2. 性能优化
- 对频繁查询的字段建立合适的索引
- 避免在策略中使用过于复杂的条件组合
- 合理使用时间有效期避免过期策略影响性能

### 3. 安全考虑
- 严格控制策略的创建和修改权限
- 记录所有策略操作的审计日志
- 对影响范围大的策略进行额外审批

## 示例数据

```sql
-- 全局跳过策略
INSERT INTO asset_skip_policies (policy_name, policy_type, condition_rules, action_config, description, enabled, priority) VALUES
('Skip Honeypot Devices', 'global', 
'{"conditions": [{"field": "device_type", "operator": "equals", "value": "honeypot"}], "logic_operator": "and"}',
'{"action": "skip"}',
'Skip scanning known honeypot devices', true, 100),

('Skip High Port Count Devices', 'global',
'{"conditions": [{"field": "port_count", "operator": "greater_than", "value": 1000}], "logic_operator": "and"}',
'{"action": "log", "log_level": "warn"}',
'Log devices with unusually high port count', true, 90),

('Skip Test Environments', 'global',
'{"conditions": [{"field": "environment", "operator": "equals", "value": "test"}], "logic_operator": "or"}, 
  {"field": "hostname", "operator": "contains", "value": "test"}], "logic_operator": "or"}',
'{"action": "skip"}',
'Skip scanning test environments', true, 80);

-- 特定扫描类型策略
INSERT INTO asset_skip_policies (policy_name, policy_type, condition_rules, action_config, scope, description, enabled, priority) VALUES
('Vulnerability Scan Exclusions', 'scan_type',
'{"conditions": [{"field": "critical_system", "operator": "equals", "value": true}], "logic_operator": "and"}',
'{"action": "skip", "alert_channels": ["email"]}',
'{"scan_types": ["vuln_scan"]}',
'Exclude critical systems from vulnerability scanning', true, 100);
```
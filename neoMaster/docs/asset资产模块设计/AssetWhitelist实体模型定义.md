# AssetWhitelist实体模型定义

## 概述

AssetWhitelist实体用于定义统一的资产白名单机制，适用于所有扫描类型。该表设计旨在提供跨扫描类型的通用白名单功能，避免为每种扫描类型重复创建白名单表。

## 模型结构

### 核心字段

| 字段名 | 类型 | 描述 |
|--------|------|------|
| `id` | uint | 自增主键 |
| `whitelist_name` | string | 白名单名称 |
| `whitelist_type` | string | 白名单类型 |
| `target_type` | string | 目标类型 |
| `target_value` | string | 目标值 |
| `description` | string | 描述信息 |
| `valid_from` | timestamp | 生效开始时间 |
| `valid_to` | timestamp | 生效结束时间 |
| `created_by` | string | 创建人 |
| `tags` | JSON | 标签信息 |
| `scope` | JSON | 作用域配置 |
| `enabled` | bool | 是否启用 |
| `note` | string | 备注信息 |
| `created_at` | timestamp | 创建时间 |
| `updated_at` | timestamp | 更新时间 |

### 字段详解

#### 1. whitelist_type（白名单类型）
定义白名单的应用范围：
- `global`: 全局白名单，适用于所有扫描类型
- `scan_type`: 特定扫描类型白名单
- `project`: 特定项目白名单
- `workflow`: 特定工作流白名单
- `stage`: 特定阶段白名单

#### 2. target_type（目标类型）
定义白名单条目的目标类型：
- `ip`: IP地址 (如: 192.168.1.100)
- `ip_range`: IP范围 (如: 192.168.1.0/24)
- `domain`: 域名 (如: example.com)
- `domain_pattern`: 域名模式 (如: *.example.com)
- `url`: URL地址 (如: https://example.com)
- `cidr`: CIDR表示法 (如: 10.0.0.0/8)
- `mac`: MAC地址 (如: 00:11:22:33:44:55)
- `host`: 主机名 (如: server01)
- `asn`: 自治系统号 (如: AS12345)
- `country`: 国家代码 (如: CN, US)
- `custom`: 自定义类型

#### 3. target_value（目标值）
具体的白名单目标值，支持多种格式：
- 精确匹配: `192.168.1.100`
- 范围匹配: `192.168.1.0-192.168.1.255`
- CIDR表示: `192.168.1.0/24`
- 通配符: `*.example.com`
- 正则表达式: `^.*\.example\.com$`

#### 4. scope（作用域配置）
定义白名单的作用范围：

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
用于分类和标记白名单条目：

```json
{
  "business_unit": "finance",
  "environment": "production",
  "criticality": "high",
  "compliance": ["pci-dss", "sox"]
}
```

## 设计原则

### 1. 统一性
- 单一表结构支持所有扫描类型的白名单需求
- 统一的匹配规则和处理逻辑
- 一致的管理接口

### 2. 灵活性
- 支持多种目标类型和匹配模式
- 可配置的作用域范围
- 支持时间有效期控制

### 3. 可扩展性
- JSON字段支持未来扩展
- 标签系统便于分类管理
- 支持复杂的组合条件

## 使用场景

### 1. 全局白名单
适用于所有扫描任务的通用白名单：

```sql
INSERT INTO asset_whitelists 
(whitelist_name, whitelist_type, target_type, target_value, enabled)
VALUES 
('Critical Production Servers', 'global', 'ip_range', '192.168.10.0/24', true),
('Trusted Domains', 'global', 'domain', 'example.com', true);
```

### 2. 特定扫描类型白名单
仅对特定扫描类型生效的白名单：

```sql
INSERT INTO asset_whitelists 
(whitelist_name, whitelist_type, target_type, target_value, scope, enabled)
VALUES 
('Database Scan Exceptions', 'scan_type', 'ip', '192.168.1.100', 
 '{"scan_types": ["port_scan", "vuln_scan"]}', true);
```

### 3. 项目级白名单
特定项目中的白名单条目：

```sql
INSERT INTO asset_whitelists 
(whitelist_name, whitelist_type, target_type, target_value, scope, enabled)
VALUES 
('Project Alpha Exceptions', 'project', 'domain_pattern', '*.alpha.example.com',
 '{"projects": [101]}', true);
```

## 匹配规则

### 1. 精确匹配
```text
目标值: 192.168.1.100
匹配: 192.168.1.100
不匹配: 192.168.1.101, 192.168.1.0/24
```

### 2. 范围匹配
```text
目标值: 192.168.1.100-192.168.1.200
匹配: 192.168.1.150, 192.168.1.100, 192.168.1.200
不匹配: 192.168.1.99, 192.168.1.201
```

### 3. CIDR匹配
```text
目标值: 192.168.1.0/24
匹配: 192.168.1.1, 192.168.1.254
不匹配: 192.168.2.1, 192.168.0.1
```

### 4. 域名通配符匹配
```text
目标值: *.example.com
匹配: www.example.com, api.example.com
不匹配: example.com, www.example.org
```

### 5. 正则表达式匹配
```text
目标值: ^.*\.example\.com$
匹配: www.example.com, api.example.com
不匹配: example.com, www.example.org
```

## 性能优化建议

### 1. 索引设计
```sql
-- 主键索引
CREATE INDEX idx_asset_whitelists_id ON asset_whitelists(id);

-- 类型和目标类型索引
CREATE INDEX idx_asset_whitelists_types ON asset_whitelists(whitelist_type, target_type);

-- 目标值索引（前缀索引适用于字符串匹配）
CREATE INDEX idx_asset_whitelists_target ON asset_whitelists(target_value);

-- 时间范围索引
CREATE INDEX idx_asset_whitelists_valid ON asset_whitelists(valid_from, valid_to);

-- 启用状态索引
CREATE INDEX idx_asset_whitelists_enabled ON asset_whitelists(enabled);

-- 组合索引优化常见查询
CREATE INDEX idx_asset_whitelists_query ON asset_whitelists(whitelist_type, enabled, valid_from, valid_to);
```

### 2. 查询优化
```sql
-- 查询当前生效的全局白名单
SELECT * FROM asset_whitelists 
WHERE whitelist_type = 'global' 
  AND enabled = true 
  AND (valid_from IS NULL OR valid_from <= NOW())
  AND (valid_to IS NULL OR valid_to >= NOW());
```

## 与其他实体的关系

### 1. 与ScanStage的关系
- ScanStage中的白名单策略引用AssetWhitelist
- 通过scope字段关联特定的扫描阶段

### 2. 与Project的关系
- 项目可以定义专用的白名单条目
- 通过scope.projects字段关联

### 3. 与Asset的关系
- 白名单条目可能基于现有资产定义
- 用于排除特定资产的扫描

## 管理接口设计

### 1. 创建白名单
```http
POST /api/whitelists
Content-Type: application/json

{
  "whitelist_name": "Production DB Servers",
  "whitelist_type": "global",
  "target_type": "ip_range",
  "target_value": "192.168.10.0/24",
  "description": "Production database servers",
  "enabled": true,
  "tags": {
    "env": "production",
    "type": "database"
  }
}
```

### 2. 查询白名单
```http
GET /api/whitelists?whitelist_type=global&target_type=ip
```

### 3. 匹配检查
```http
POST /api/whitelists/check
Content-Type: application/json

{
  "target": "192.168.1.100",
  "target_type": "ip",
  "context": {
    "scan_type": "port_scan",
    "project_id": 1,
    "workflow_id": 10
  }
}
```

## 最佳实践

### 1. 白名单设计建议
- 优先使用精确匹配而非通配符匹配以提高性能
- 合理使用时间有效期避免过期条目堆积
- 使用标签系统对白名单进行分类管理
- 定期审查和清理不再需要的白名单条目

### 2. 性能优化
- 对频繁查询的字段建立合适的索引
- 避免在白名单中使用过于复杂的正则表达式
- 对大范围IP使用CIDR表示法而非多个条目

### 3. 安全考虑
- 严格控制白名单的创建和修改权限
- 记录所有白名单操作的审计日志
- 对敏感目标的白名单进行额外审批

## 示例数据

```sql
-- 全局IP白名单
INSERT INTO asset_whitelists (whitelist_name, whitelist_type, target_type, target_value, description, enabled) VALUES
('Localhost', 'global', 'ip', '127.0.0.1', 'Localhost address', true),
('Private Network', 'global', 'cidr', '192.168.0.0/16', 'Private network range', true),
('Documentation Range', 'global', 'cidr', '192.0.2.0/24', 'Documentation IP range', true);

-- 域名白名单
INSERT INTO asset_whitelists (whitelist_name, whitelist_type, target_type, target_value, description, enabled) VALUES
('Trusted Domain', 'global', 'domain', 'example.com', 'Main company domain', true),
('Wildcard Domain', 'global', 'domain_pattern', '*.example.com', 'All subdomains', true);

-- 特定扫描类型白名单
INSERT INTO asset_whitelists (whitelist_name, whitelist_type, target_type, target_value, scope, description, enabled) VALUES
('DB Port Scan Exception', 'scan_type', 'ip', '192.168.1.100', '{"scan_types": ["port_scan"]}', 'Database server port scan exception', true);
```
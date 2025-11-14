# Agent分组管理API说明文档

## 概述

Agent分组管理模块提供了对Agent进行分组管理的功能，支持创建、更新、删除分组，以及管理分组中的Agent成员。通过分组管理，可以更好地组织和批量管理Agent实例。

## API端点列表

### 1. 获取分组列表

**接口描述**: 获取Agent分组列表，支持分页、状态过滤、关键词搜索和标签过滤

**HTTP方法**: GET
**接口路径**: `/api/agents/groups`

**请求参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | int | 否 | 页码，默认为1 |
| page_size | int | 否 | 每页数量，默认为10 |
| status | int | 否 | 分组状态：1=激活，0=禁用，-1=全部 |
| keywords | string | 否 | 关键词搜索（分组名称、描述） |
| tags | []string | 否 | 标签过滤，支持多个标签 |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "ok",
  "data": {
    "groups": [
      {
        "group_id": "group_001",
        "name": "Web服务器分组",
        "description": "用于Web服务器的Agent分组",
        "status": 1,
        "tags": ["web", "server"],
        "created_at": "2025-11-14 10:00:00",
        "updated_at": "2025-11-14 11:00:00"
      }
    ],
    "pagination": {
      "total": 5,
      "page": 1,
      "page_size": 10,
      "total_pages": 1,
      "has_next": false,
      "has_previous": false
    }
  }
}
```

### 2. 创建分组

**接口描述**: 创建新的Agent分组

**HTTP方法**: POST
**接口路径**: `/api/agents/groups`

**请求体**:
```json
{
  "group_id": "group_001",
  "name": "Web服务器分组",
  "description": "用于Web服务器的Agent分组",
  "status": 1,
  "tags": ["web", "server"]
}
```

**字段说明**:
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| group_id | string | 是 | 分组唯一标识 |
| name | string | 是 | 分组名称 |
| description | string | 否 | 分组描述 |
| status | uint8 | 否 | 分组状态：1=激活，0=禁用，默认为1 |
| tags | []string | 否 | 分组标签列表 |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "ok",
  "data": {
    "group_id": "group_001",
    "name": "Web服务器分组",
    "description": "用于Web服务器的Agent分组",
    "status": 1,
    "tags": ["web", "server"],
    "created_at": "2025-11-14 10:00:00",
    "updated_at": "2025-11-14 10:00:00"
  }
}
```

### 3. 更新分组

**接口描述**: 更新指定分组的信息

**HTTP方法**: PUT
**接口路径**: `/api/agents/groups/{group_id}`

**路径参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| group_id | string | 是 | 分组ID |

**请求体**:
```json
{
  "name": "更新后的分组名称",
  "description": "更新后的描述",
  "status": 1,
  "tags": ["new_tag"]
}
```

**字段说明**:
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| name | string | 否 | 分组名称 |
| description | string | 否 | 分组描述 |
| status | uint8 | 否 | 分组状态 |
| tags | []string | 否 | 分组标签列表 |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "更新成功",
  "data": {
    "group_id": "group_001",
    "name": "更新后的分组名称",
    "description": "更新后的描述",
    "status": 1,
    "tags": ["new_tag"],
    "created_at": "2025-11-14 10:00:00",
    "updated_at": "2025-11-14 11:00:00"
  }
}
```

### 4. 删除分组

**接口描述**: 删除指定分组，分组中的Agent会被迁移到默认分组

**HTTP方法**: DELETE
**接口路径**: `/api/agents/groups/{group_id}`

**路径参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| group_id | string | 是 | 分组ID |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "删除分组成功,组成员已迁移至默认分组",
  "data": {
    "group_id": "group_001"
  }
}
```

### 5. 设置分组状态

**接口描述**: 设置分组的激活/禁用状态

**HTTP方法**: PUT
**接口路径**: `/api/agents/groups/{group_id}/status`

**路径参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| group_id | string | 是 | 分组ID |

**请求体**:
```json
{
  "status": 0
}
```

**字段说明**:
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| status | int | 是 | 分组状态：0=禁用，1=激活 |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "分组状态设置成功",
  "data": {
    "group_id": "group_001",
    "new_status": 0
  }
}
```

### 6. 添加Agent到分组

**接口描述**: 将指定Agent添加到指定分组

**HTTP方法**: POST
**接口路径**: `/api/agents/{id}/groups`

**路径参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | string | 是 | Agent业务ID |

**请求体**:
```json
{
  "group_id": "group_001"
}
```

**字段说明**:
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| group_id | string | 是 | 目标分组ID |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "添加Agent到分组成功",
  "data": {
    "agent_id": "agent_001",
    "group_id": "group_001"
  }
}
```

### 7. 从分组移除Agent

**接口描述**: 从指定分组中移除指定Agent

**HTTP方法**: DELETE
**接口路径**: `/api/agents/{id}/groups/{group_id}`

**路径参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | string | 是 | Agent业务ID |
| group_id | string | 是 | 分组ID |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "从分组移除Agent成功",
  "data": {
    "agent_id": "agent_001",
    "group_id": "group_001"
  }
}
```

### 8. 获取分组中的Agent列表

**接口描述**: 获取指定分组中的Agent成员列表，支持分页

**HTTP方法**: GET
**接口路径**: `/api/agents/group/members`

**请求参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| group_id | string | 是 | 分组ID |
| page | int | 否 | 页码，默认为1 |
| page_size | int | 否 | 每页数量，默认为10 |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "ok",
  "data": {
    "members": [
      {
        "agent_id": "agent_001",
        "hostname": "web-server-01",
        "ip_address": "192.168.1.100",
        "status": "online",
        "last_heartbeat": "2025-11-14 11:00:00"
      }
    ],
    "page": 1,
    "page_size": 10
  }
}
```

## 错误码说明

| HTTP状态码 | 错误码 | 说明 |
|------------|--------|------|
| 200 | 400 | 请求参数错误 |
| 200 | 404 | 资源不存在 |
| 200 | 500 | 服务器内部错误 |

## 使用示例

### 创建分组
```bash
curl -X POST "http://localhost:8080/api/agents/groups" \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": "web_servers",
    "name": "Web服务器分组",
    "description": "所有Web服务器的Agent",
    "status": 1,
    "tags": ["web", "production"]
  }'
```

### 添加Agent到分组
```bash
curl -X POST "http://localhost:8080/api/agents/agent_001/groups" \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": "web_servers"
  }'
```

### 获取分组列表
```bash
curl -X GET "http://localhost:8080/api/agents/groups?page=1&page_size=10&status=1&keywords=web"
```

## 注意事项

1. **分组ID唯一性**: group_id在系统中必须唯一，不能重复
2. **默认分组**: 系统会有一个默认分组，删除分组时成员会迁移到默认分组
3. **状态管理**: 禁用分组不会影响分组中Agent的正常运行，但可能影响批量操作
4. **权限控制**: 分组管理需要相应的权限，建议实施权限验证
5. **性能考虑**: 分组中Agent数量较多时，建议使用分页查询

## 数据库设计

### 分组表结构
```sql
CREATE TABLE agent_groups (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    group_id VARCHAR(64) UNIQUE NOT NULL COMMENT '分组唯一标识',
    name VARCHAR(128) NOT NULL COMMENT '分组名称',
    description TEXT COMMENT '分组描述',
    status TINYINT DEFAULT 1 COMMENT '分组状态：1=激活，0=禁用',
    tags JSON COMMENT '分组标签',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_group_id (group_id)
);
```

### 分组关系表
```sql
CREATE TABLE agent_group_members (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    agent_id VARCHAR(64) NOT NULL COMMENT 'Agent业务ID',
    group_id VARCHAR(64) NOT NULL COMMENT '分组ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_agent_group (agent_id, group_id),
    INDEX idx_group_id (group_id),
    INDEX idx_agent_id (agent_id)
);
```

## 扩展功能建议

1. **分组批量操作**: 支持对分组中的所有Agent进行批量配置更新
2. **分组权限**: 细粒度的分组权限管理
3. **分组统计**: 分组维度统计信息展示
4. **分组模板**: 支持分组配置模板，快速创建相似分组
5. **分组层级**: 支持多级分组结构
6. **分组标签管理**: 独立的标签管理接口
7. **分组导入导出**: 支持分组配置的导入导出功能
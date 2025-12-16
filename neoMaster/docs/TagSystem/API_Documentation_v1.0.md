# NeoScan Tag System API 接口文档 v1.0

## 📋 版本更新说明

**版本**: v1.0  
**更新日期**: 2025-12-16  
**主要变更**:
- 初始版本：包含标签管理和规则管理的核心接口。

## 🌐 服务器信息

- **基础URL**: `http://localhost:8123`
- **API版本**: v1
- **认证方式**: JWT Bearer Token
- **内容类型**: `application/json`

## 📊 通用响应格式

### 成功响应
```json
{
  "code": 200,
  "status": "success",
  "message": "操作成功",
  "data": {}
}
```

### 错误响应
```json
{
  "code": 400,
  "status": "error",
  "message": "错误描述"
}
```

### 分页响应
```json
{
  "code": 200,
  "status": "success",
  "message": "操作成功",
  "data": {
    "list": [],
    "total": 100,
    "page": 1,
    "page_size": 10
  }
}
```

## 🏷️ 标签管理接口

### 1. 创建标签
- **URL**: `/api/v1/tags`
- **方法**: `POST`
- **描述**: 创建一个新的标签
- **认证**: 需要

**请求参数 (Body)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| name | string | 是 | 标签名称 |
| parent_id | integer | 否 | 父标签ID (默认为0，即根标签) |
| color | string | 否 | 标签颜色 (HEX格式) |
| category | string | 否 | 业务分类 (如 asset, vul, user) |
| description | string | 否 | 描述信息 |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "Tag created successfully",
  "data": {
    "id": 1
  }
}
```

### 2. 获取标签列表
- **URL**: `/api/v1/tags`
- **方法**: `GET`
- **描述**: 获取标签列表，支持分页和筛选
- **认证**: 需要

**请求参数 (Query)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| parent_id | integer | 否 | 父标签ID筛选 |
| keyword | string | 否 | 搜索关键字 (名称或描述) |
| category | string | 否 | 业务分类筛选 |
| page | integer | 否 | 页码 (默认1) |
| page_size | integer | 否 | 每页数量 (默认10) |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "Tags retrieved successfully",
  "data": {
    "list": [
      {
        "id": 1,
        "name": "High Risk",
        "parent_id": 0,
        "path": "/1/",
        "level": 1,
        "color": "#FF0000",
        "category": "vul",
        "description": "High risk vulnerabilities",
        "created_at": "2025-12-16T10:00:00Z",
        "updated_at": "2025-12-16T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 10
  }
}
```

### 3. 获取标签详情
- **URL**: `/api/v1/tags/{id}`
- **方法**: `GET`
- **描述**: 获取单个标签的详细信息
- **认证**: 需要

**请求参数 (Path)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| id | integer | 是 | 标签ID |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "Tag retrieved successfully",
  "data": {
    "id": 1,
    "name": "High Risk",
    "parent_id": 0,
    "path": "/1/",
    "level": 1,
    "color": "#FF0000",
    "category": "vul",
    "description": "High risk vulnerabilities",
    "created_at": "2025-12-16T10:00:00Z",
    "updated_at": "2025-12-16T10:00:00Z"
  }
}
```

### 4. 更新标签
- **URL**: `/api/v1/tags/{id}`
- **方法**: `PUT`
- **描述**: 更新标签信息
- **认证**: 需要

**请求参数 (Path)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| id | integer | 是 | 标签ID |

**请求参数 (Body)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| name | string | 否 | 标签名称 |
| color | string | 否 | 标签颜色 |
| description | string | 否 | 描述信息 |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "Tag updated successfully",
  "data": null
}
```

### 5. 删除标签
- **URL**: `/api/v1/tags/{id}`
- **方法**: `DELETE`
- **描述**: 删除标签 (如果有子标签或关联规则可能无法删除)
- **认证**: 需要

**请求参数 (Path)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| id | integer | 是 | 标签ID |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "Tag deleted successfully",
  "data": null
}
```

## ⚙️ 规则管理接口

### 1. 创建规则
- **URL**: `/api/v1/tag-rules`
- **方法**: `POST`
- **描述**: 创建自动打标规则
- **认证**: 需要

**请求参数 (Body)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| tag_id | integer | 是 | 关联标签ID |
| name | string | 是 | 规则名称 |
| entity_type | string | 是 | 实体类型 (如 host, web_service) |
| priority | integer | 否 | 优先级 (数字越大优先级越高) |
| rule_json | object | 是 | 匹配规则JSON对象 |
| is_enabled | boolean | 否 | 是否启用 (默认true) |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "Rule created successfully",
  "data": {
    "id": 1
  }
}
```

### 2. 获取规则列表
- **URL**: `/api/v1/tag-rules`
- **方法**: `GET`
- **描述**: 获取规则列表，支持筛选
- **认证**: 需要

**请求参数 (Query)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| entity_type | string | 否 | 实体类型筛选 |
| tag_id | integer | 否 | 标签ID筛选 |
| keyword | string | 否 | 搜索关键字 (名称) |
| is_enabled | boolean | 否 | 启用状态筛选 |
| page | integer | 否 | 页码 |
| page_size | integer | 否 | 每页数量 |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "Rules retrieved successfully",
  "data": {
    "list": [
      {
        "id": 1,
        "tag_id": 1,
        "name": "Auto High Risk",
        "entity_type": "host",
        "priority": 10,
        "rule_json": {},
        "is_enabled": true,
        "created_at": "2025-12-16T10:00:00Z",
        "updated_at": "2025-12-16T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 10
  }
}
```

### 3. 更新规则
- **URL**: `/api/v1/tag-rules/{id}`
- **方法**: `PUT`
- **描述**: 更新规则信息
- **认证**: 需要

**请求参数 (Path)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| id | integer | 是 | 规则ID |

**请求参数 (Body)**:
| 字段名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| name | string | 否 | 规则名称 |
| priority | integer | 否 | 优先级 |
| rule_json | object | 否 | 匹配规则 |
| is_enabled | boolean | 否 | 是否启用 |

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "Rule updated successfully",
  "data": null
}
```

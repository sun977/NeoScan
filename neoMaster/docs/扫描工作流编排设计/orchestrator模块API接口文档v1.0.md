# NeoScan Orchestrator 模块 API 接口文档 v1.0

## 📋 版本更新说明

**版本**: v1.0  
**更新日期**: 2025-10-13  
**主要变更**:
- 初始版本发布
- 完整的扫描配置管理 API 接口
- 支持项目配置、工作流、扫描工具、扫描规则四大核心功能

## 🌐 服务器信息

- **基础URL**: `http://localhost:8123`
- **API版本**: v1
- **认证方式**: JWT Bearer Token
- **内容类型**: `application/json`
- **服务器版本**: NeoScan Master v1.0

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
  "message": "错误描述",
  "error": "详细错误信息"
}
```

### 分页响应
```json
{
  "code": 200,
  "status": "success",
  "message": "操作成功",
  "data": {
    "items": [],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 100,
      "pages": 10
    }
  }
}
```

## 🏗️ 项目配置管理 API

### 1. 创建项目配置
- **URL**: `/api/v1/orchestrator/projects`
- **方法**: `POST`
- **描述**: 创建新的扫描项目配置
- **认证**: 需要认证

**请求体**:
```json
{
  "name": "项目名称",
  "display_name": "项目显示名称",
  "description": "项目描述",
  "target_scope": "192.168.1.0/24,example.com",
  "exclude_list": "192.168.1.1,admin.example.com",
  "scan_frequency": 24,
  "max_concurrent": 10,
  "timeout_second": 300,
  "priority": 5,
  "notify_on_success": false,
  "notify_on_failure": true,
  "notify_emails": "admin@example.com,security@example.com",
  "tags": "web,security,production",
  "metadata": "{\"department\":\"security\",\"owner\":\"admin\"}"
}
```

**响应示例**:
```json
{
  "code": 201,
  "status": "success",
  "message": "项目配置创建成功",
  "data": {
    "id": 1,
    "name": "项目名称",
    "display_name": "项目显示名称",
    "description": "项目描述",
    "target_scope": "192.168.1.0/24,example.com",
    "exclude_list": "192.168.1.1,admin.example.com",
    "scan_frequency": 24,
    "max_concurrent": 10,
    "timeout_second": 300,
    "priority": 5,
    "notify_on_success": false,
    "notify_on_failure": true,
    "notify_emails": "admin@example.com,security@example.com",
    "status": 0,
    "is_enabled": true,
    "tags": "web,security,production",
    "metadata": "{\"department\":\"security\",\"owner\":\"admin\"}",
    "created_by": 1,
    "updated_by": 1,
    "created_at": "2025-01-11T10:00:00Z",
    "updated_at": "2025-01-11T10:00:00Z"
  }
}
```

### 2. 获取项目配置详情
- **URL**: `/api/v1/orchestrator/projects/{id}`
- **方法**: `GET`
- **描述**: 获取指定项目配置的详细信息
- **认证**: 需要认证

**路径参数**:
- `id` (integer): 项目配置ID

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "获取项目配置成功",
  "data": {
    "id": 1,
    "name": "项目名称",
    "display_name": "项目显示名称",
    "description": "项目描述",
    "target_scope": "192.168.1.0/24,example.com",
    "exclude_list": "192.168.1.1,admin.example.com",
    "scan_frequency": 24,
    "max_concurrent": 10,
    "timeout_second": 300,
    "priority": 5,
    "notify_on_success": false,
    "notify_on_failure": true,
    "notify_emails": "admin@example.com,security@example.com",
    "status": 1,
    "is_enabled": true,
    "tags": "web,security,production",
    "metadata": "{\"department\":\"security\",\"owner\":\"admin\"}",
    "created_by": 1,
    "updated_by": 1,
    "last_scan": "2025-01-11T09:00:00Z",
    "created_at": "2025-01-11T08:00:00Z",
    "updated_at": "2025-01-11T10:00:00Z",
    "workflows": []
  }
}
```

### 3. 更新项目配置
- **URL**: `/api/v1/orchestrator/projects/{id}`
- **方法**: `PUT`
- **描述**: 更新指定项目配置
- **认证**: 需要认证

**路径参数**:
- `id` (integer): 项目配置ID

**请求体** (所有字段可选):
```json
{
  "name": "更新的项目名称",
  "display_name": "更新的项目显示名称",
  "description": "更新的项目描述",
  "target_scope": "192.168.2.0/24,newexample.com",
  "exclude_list": "192.168.2.1",
  "scan_frequency": 12,
  "max_concurrent": 20,
  "timeout_second": 600,
  "priority": 8,
  "notify_on_success": true,
  "notify_on_failure": true,
  "notify_emails": "admin@example.com",
  "tags": "web,security,staging",
  "metadata": "{\"department\":\"devops\",\"owner\":\"admin\"}"
}
```

### 4. 删除项目配置
- **URL**: `/api/v1/orchestrator/projects/{id}`
- **方法**: `DELETE`
- **描述**: 删除指定项目配置
- **认证**: 需要认证

**路径参数**:
- `id` (integer): 项目配置ID

### 5. 获取项目配置列表
- **URL**: `/api/v1/orchestrator/projects`
- **方法**: `GET`
- **描述**: 获取项目配置列表，支持分页和过滤
- **认证**: 需要认证

**查询参数**:
- `page` (integer, 可选): 页码，默认1
- `limit` (integer, 可选): 每页数量，默认10，最大100
- `status` (string, 可选): 状态过滤 (inactive/active/archived)
- `keyword` (string, 可选): 关键词搜索

**响应示例**:
```json
{
  "code": 200,
  "status": "success",
  "message": "获取项目配置列表成功",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "项目名称",
        "display_name": "项目显示名称",
        "description": "项目描述",
        "status": 1,
        "is_enabled": true,
        "created_at": "2025-01-11T08:00:00Z",
        "updated_at": "2025-01-11T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 1,
      "pages": 1
    }
  }
}
```

### 6. 启用项目配置
- **URL**: `/api/v1/orchestrator/projects/{id}/enable`
- **方法**: `POST`
- **描述**: 启用指定项目配置
- **认证**: 需要认证

### 7. 禁用项目配置
- **URL**: `/api/v1/orchestrator/projects/{id}/disable`
- **方法**: `POST`
- **描述**: 禁用指定项目配置
- **认证**: 需要认证

### 8. 热重载项目配置
- **URL**: `/api/v1/orchestrator/projects/{id}/reload`
- **方法**: `POST`
- **描述**: 热重载指定项目配置
- **认证**: 需要认证

### 9. 同步项目配置
- **URL**: `/api/v1/orchestrator/projects/{id}/sync`
- **方法**: `POST`
- **描述**: 同步指定项目配置到执行节点
- **认证**: 需要认证

### 10. 获取系统配置
- **URL**: `/api/v1/orchestrator/system`
- **方法**: `GET`
- **描述**: 获取系统级配置信息
- **认证**: 需要认证

### 11. 更新系统配置
- **URL**: `/api/v1/orchestrator/system`
- **方法**: `PUT`
- **描述**: 更新系统级配置
- **认证**: 需要认证

## 🔧 扫描工具管理 API

### 1. 创建扫描工具
- **URL**: `/api/v1/orchestrator/tools`
- **方法**: `POST`
- **描述**: 创建新的扫描工具配置
- **认证**: 需要认证

**请求体**:
```json
{
  "name": "Nmap",
  "display_name": "网络映射器",
  "description": "网络发现和安全审计工具",
  "type": "port_scan",
  "version": "7.94",
  "executable_path": "/usr/bin/nmap",
  "config_template": "{\"default_args\":[\"-sS\",\"-O\"],\"timeout\":300}",
  "input_format": "json",
  "output_format": "xml",
  "supported_targets": ["ip", "domain", "cidr"],
  "max_concurrent": 5,
  "timeout_second": 600,
  "retry_count": 3,
  "status": "enabled",
  "is_built_in": true,
  "compatibility": "{\"os\":[\"linux\",\"windows\"],\"arch\":[\"x64\",\"arm64\"]}",
  "tags": "network,port,scan",
  "metadata": "{\"category\":\"network_security\",\"vendor\":\"nmap.org\"}"
}
```

### 2. 获取扫描工具详情
- **URL**: `/api/v1/orchestrator/tools/{id}`
- **方法**: `GET`
- **描述**: 获取指定扫描工具的详细信息
- **认证**: 需要认证

### 3. 更新扫描工具
- **URL**: `/api/v1/orchestrator/tools/{id}`
- **方法**: `PUT`
- **描述**: 更新指定扫描工具配置
- **认证**: 需要认证

### 4. 删除扫描工具
- **URL**: `/api/v1/orchestrator/tools/{id}`
- **方法**: `DELETE`
- **描述**: 删除指定扫描工具
- **认证**: 需要认证

### 5. 获取扫描工具列表
- **URL**: `/api/v1/orchestrator/tools`
- **方法**: `GET`
- **描述**: 获取扫描工具列表，支持分页和过滤
- **认证**: 需要认证

**查询参数**:
- `page` (integer, 可选): 页码，默认1
- `limit` (integer, 可选): 每页数量，默认10
- `type` (string, 可选): 工具类型过滤
- `status` (string, 可选): 状态过滤
- `keyword` (string, 可选): 关键词搜索

### 6. 启用扫描工具
- **URL**: `/api/v1/orchestrator/tools/{id}/enable`
- **方法**: `POST`
- **描述**: 启用指定扫描工具
- **认证**: 需要认证

### 7. 禁用扫描工具
- **URL**: `/api/v1/orchestrator/tools/{id}/disable`
- **方法**: `POST`
- **描述**: 禁用指定扫描工具
- **认证**: 需要认证

### 8. 扫描工具健康检查
- **URL**: `/api/v1/orchestrator/tools/{id}/health`
- **方法**: `GET`
- **描述**: 检查指定扫描工具的健康状态
- **认证**: 需要认证

### 9. 安装扫描工具
- **URL**: `/api/v1/orchestrator/tools/{id}/install`
- **方法**: `POST`
- **描述**: 安装指定扫描工具到执行节点
- **认证**: 需要认证

### 10. 卸载扫描工具
- **URL**: `/api/v1/orchestrator/tools/{id}/uninstall`
- **方法**: `POST`
- **描述**: 从执行节点卸载指定扫描工具
- **认证**: 需要认证

### 11. 获取工具指标
- **URL**: `/api/v1/orchestrator/tools/{id}/metrics`
- **方法**: `GET`
- **描述**: 获取指定扫描工具的性能指标
- **认证**: 需要认证

### 12. 获取可用工具列表
- **URL**: `/api/v1/orchestrator/tools/available`
- **方法**: `GET`
- **描述**: 获取系统中可用的扫描工具列表
- **认证**: 需要认证

### 13. 批量安装工具
- **URL**: `/api/v1/orchestrator/tools/batch-install`
- **方法**: `POST`
- **描述**: 批量安装多个扫描工具
- **认证**: 需要认证

### 14. 批量卸载工具
- **URL**: `/api/v1/orchestrator/tools/batch-uninstall`
- **方法**: `POST`
- **描述**: 批量卸载多个扫描工具
- **认证**: 需要认证

### 15. 获取系统状态
- **URL**: `/api/v1/orchestrator/tools/system-status`
- **方法**: `GET`
- **描述**: 获取扫描工具系统整体状态
- **认证**: 需要认证

### 16. 按类型获取工具
- **URL**: `/api/v1/orchestrator/tools/type/{type}`
- **方法**: `GET`
- **描述**: 获取指定类型的扫描工具列表
- **认证**: 需要认证

## 📋 扫描规则管理 API

### 1. 创建扫描规则
- **URL**: `/api/v1/orchestrator/rules`
- **方法**: `POST`
- **描述**: 创建新的扫描规则
- **认证**: 需要认证

**请求体**:
```json
{
  "name": "高危端口检测",
  "description": "检测高危端口开放情况",
  "type": "filter",
  "category": "port_security",
  "severity": "high",
  "config": {
    "enabled": true,
    "threshold": 5,
    "timeout": 30
  },
  "conditions": [
    {
      "field": "port",
      "operator": "in",
      "value": [22, 23, 3389, 5432, 3306],
      "logic": "and"
    }
  ],
  "actions": [
    {
      "type": "alert",
      "parameters": {
        "level": "high",
        "notify": true
      },
      "message": "发现高危端口开放"
    }
  ],
  "tags": ["security", "port", "high-risk"],
  "is_built_in": false,
  "priority": 80,
  "status": "enabled"
}
```

### 2. 获取扫描规则详情
- **URL**: `/api/v1/orchestrator/rules/{id}`
- **方法**: `GET`
- **描述**: 获取指定扫描规则的详细信息
- **认证**: 需要认证

### 3. 更新扫描规则
- **URL**: `/api/v1/orchestrator/rules/{id}`
- **方法**: `PUT`
- **描述**: 更新指定扫描规则
- **认证**: 需要认证

### 4. 删除扫描规则
- **URL**: `/api/v1/orchestrator/rules/{id}`
- **方法**: `DELETE`
- **描述**: 删除指定扫描规则
- **认证**: 需要认证

### 5. 获取扫描规则列表
- **URL**: `/api/v1/orchestrator/rules`
- **方法**: `GET`
- **描述**: 获取扫描规则列表，支持分页和过滤
- **认证**: 需要认证

**查询参数**:
- `page` (integer, 可选): 页码，默认1
- `limit` (integer, 可选): 每页数量，默认10
- `type` (string, 可选): 规则类型过滤
- `category` (string, 可选): 规则分类过滤
- `severity` (string, 可选): 严重程度过滤
- `status` (string, 可选): 状态过滤
- `keyword` (string, 可选): 关键词搜索

### 6. 启用扫描规则
- **URL**: `/api/v1/orchestrator/rules/{id}/enable`
- **方法**: `POST`
- **描述**: 启用指定扫描规则
- **认证**: 需要认证

### 7. 禁用扫描规则
- **URL**: `/api/v1/orchestrator/rules/{id}/disable`
- **方法**: `POST`
- **描述**: 禁用指定扫描规则
- **认证**: 需要认证

### 8. 匹配扫描规则
- **URL**: `/api/v1/orchestrator/rules/match`
- **方法**: `POST`
- **描述**: 根据条件匹配适用的扫描规则
- **认证**: 需要认证

**请求体**:
```json
{
  "target_type": "ip",
  "scan_phase": "port_scan",
  "scan_tool": "nmap",
  "rule_type": "filter",
  "target_data": {
    "ip": "192.168.1.100",
    "ports": [22, 80, 443, 3389]
  },
  "context": {
    "project_id": 1,
    "scan_id": "scan_123"
  },
  "max_rules": 10,
  "only_enabled": true
}
```

### 9. 导入扫描规则
- **URL**: `/api/v1/orchestrator/rules/import`
- **方法**: `POST`
- **描述**: 批量导入扫描规则
- **认证**: 需要认证

### 10. 导出扫描规则
- **URL**: `/api/v1/orchestrator/rules/export`
- **方法**: `GET`
- **描述**: 导出扫描规则配置
- **认证**: 需要认证

### 11. 测试扫描规则
- **URL**: `/api/v1/orchestrator/rules/{id}/test`
- **方法**: `POST`
- **描述**: 测试指定扫描规则的执行效果
- **认证**: 需要认证

### 12. 按类型获取规则
- **URL**: `/api/v1/orchestrator/rules/type/{type}`
- **方法**: `GET`
- **描述**: 获取指定类型的扫描规则列表
- **认证**: 需要认证

### 13. 按严重程度获取规则
- **URL**: `/api/v1/orchestrator/rules/severity/{severity}`
- **方法**: `GET`
- **描述**: 获取指定严重程度的扫描规则列表
- **认证**: 需要认证

### 14. 获取活跃规则
- **URL**: `/api/v1/orchestrator/rules/active`
- **方法**: `GET`
- **描述**: 获取当前活跃的扫描规则列表
- **认证**: 需要认证

### 15. 获取规则指标
- **URL**: `/api/v1/orchestrator/rules/{id}/metrics`
- **方法**: `GET`
- **描述**: 获取指定扫描规则的性能指标
- **认证**: 需要认证

## 🔄 工作流管理 API

### 1. 创建工作流
- **URL**: `/api/v1/orchestrator/workflows`
- **方法**: `POST`
- **描述**: 创建新的扫描工作流
- **认证**: 需要认证

**请求体**:
```json
{
  "name": "Web安全扫描流程",
  "description": "针对Web应用的完整安全扫描工作流",
  "project_id": 1,
  "config": {
    "max_concurrent": 3,
    "timeout": 3600,
    "retry_count": 2
  },
  "steps": [
    {
      "name": "端口扫描",
      "type": "tool_execution",
      "tool_id": 1,
      "config": {
        "args": ["-sS", "-O"],
        "timeout": 300
      },
      "order": 1,
      "depends_on": [],
      "conditions": []
    },
    {
      "name": "Web扫描",
      "type": "tool_execution", 
      "tool_id": 2,
      "config": {
        "depth": 3,
        "timeout": 1800
      },
      "order": 2,
      "depends_on": [1],
      "conditions": [
        {
          "field": "http_ports",
          "operator": "exists",
          "value": true
        }
      ]
    }
  ],
  "tags": ["web", "security", "automated"],
  "is_built_in": false,
  "status": "active"
}
```

### 2. 获取工作流详情
- **URL**: `/api/v1/orchestrator/workflows/{id}`
- **方法**: `GET`
- **描述**: 获取指定工作流的详细信息
- **认证**: 需要认证

### 3. 更新工作流
- **URL**: `/api/v1/orchestrator/workflows/{id}`
- **方法**: `PUT`
- **描述**: 更新指定工作流配置
- **认证**: 需要认证

### 4. 删除工作流
- **URL**: `/api/v1/orchestrator/workflows/{id}`
- **方法**: `DELETE`
- **描述**: 删除指定工作流
- **认证**: 需要认证

### 5. 获取工作流列表
- **URL**: `/api/v1/orchestrator/workflows`
- **方法**: `GET`
- **描述**: 获取工作流列表，支持分页和过滤
- **认证**: 需要认证

### 6. 执行工作流
- **URL**: `/api/v1/orchestrator/workflows/{id}/execute`
- **方法**: `POST`
- **描述**: 执行指定工作流
- **认证**: 需要认证

### 7. 停止工作流
- **URL**: `/api/v1/orchestrator/workflows/{id}/stop`
- **方法**: `POST`
- **描述**: 停止正在执行的工作流
- **认证**: 需要认证

### 8. 暂停工作流
- **URL**: `/api/v1/orchestrator/workflows/{id}/pause`
- **方法**: `POST`
- **描述**: 暂停正在执行的工作流
- **认证**: 需要认证

### 9. 恢复工作流
- **URL**: `/api/v1/orchestrator/workflows/{id}/resume`
- **方法**: `POST`
- **描述**: 恢复暂停的工作流
- **认证**: 需要认证

### 10. 重试工作流
- **URL**: `/api/v1/orchestrator/workflows/{id}/retry`
- **方法**: `POST`
- **描述**: 重试失败的工作流
- **认证**: 需要认证

### 11. 启用工作流
- **URL**: `/api/v1/orchestrator/workflows/{id}/enable`
- **方法**: `POST`
- **描述**: 启用指定工作流
- **认证**: 需要认证

### 12. 禁用工作流
- **URL**: `/api/v1/orchestrator/workflows/{id}/disable`
- **方法**: `POST`
- **描述**: 禁用指定工作流
- **认证**: 需要认证

### 13. 获取工作流状态
- **URL**: `/api/v1/orchestrator/workflows/{id}/status`
- **方法**: `GET`
- **描述**: 获取指定工作流的执行状态
- **认证**: 需要认证

### 14. 获取工作流日志
- **URL**: `/api/v1/orchestrator/workflows/{id}/logs`
- **方法**: `GET`
- **描述**: 获取指定工作流的执行日志
- **认证**: 需要认证

### 15. 获取工作流指标
- **URL**: `/api/v1/orchestrator/workflows/{id}/metrics`
- **方法**: `GET`
- **描述**: 获取指定工作流的性能指标
- **认证**: 需要认证

### 16. 按项目获取工作流
- **URL**: `/api/v1/orchestrator/workflows/project/{project_id}`
- **方法**: `GET`
- **描述**: 获取指定项目的工作流列表
- **认证**: 需要认证

### 17. 获取系统统计
- **URL**: `/api/v1/orchestrator/workflows/system-statistics`
- **方法**: `GET`
- **描述**: 获取工作流系统统计信息
- **认证**: 需要认证

### 18. 获取系统性能
- **URL**: `/api/v1/orchestrator/workflows/system-performance`
- **方法**: `GET`
- **描述**: 获取工作流系统性能指标
- **认证**: 需要认证

## 🤖 规则引擎 API

### 1. 执行规则
- **URL**: `/api/v1/orchestrator/rule-engine/rules/{id}/execute`
- **方法**: `POST`
- **描述**: 执行指定扫描规则
- **认证**: 需要认证

**请求体**:
```json
{
  "target_data": {
    "ip": "192.168.1.100",
    "ports": [22, 80, 443],
    "services": ["ssh", "http", "https"]
  },
  "context": {
    "project_id": 1,
    "scan_id": "scan_123",
    "phase": "port_analysis"
  }
}
```

### 2. 批量执行规则
- **URL**: `/api/v1/orchestrator/rule-engine/rules/batch-execute`
- **方法**: `POST`
- **描述**: 批量执行多个扫描规则
- **认证**: 需要认证

### 3. 获取规则引擎指标
- **URL**: `/api/v1/orchestrator/rule-engine/metrics`
- **方法**: `GET`
- **描述**: 获取规则引擎的性能指标
- **认证**: 需要认证

### 4. 清除规则缓存
- **URL**: `/api/v1/orchestrator/rule-engine/cache/clear`
- **方法**: `POST`
- **描述**: 清除规则引擎缓存
- **认证**: 需要认证

### 5. 验证规则
- **URL**: `/api/v1/orchestrator/rule-engine/rules/validate`
- **方法**: `POST`
- **描述**: 验证规则配置的正确性
- **认证**: 需要认证

### 6. 解析条件
- **URL**: `/api/v1/orchestrator/rule-engine/conditions/parse`
- **方法**: `POST`
- **描述**: 解析和验证规则条件表达式
- **认证**: 需要认证

## 📊 数据模型

### ProjectConfig 项目配置模型
```json
{
  "id": 1,
  "name": "项目名称",
  "display_name": "项目显示名称", 
  "description": "项目描述",
  "target_scope": "扫描目标范围",
  "exclude_list": "排除列表",
  "scan_frequency": 24,
  "max_concurrent": 10,
  "timeout_second": 300,
  "priority": 5,
  "notify_on_success": false,
  "notify_on_failure": true,
  "notify_emails": "通知邮箱列表",
  "status": 1,
  "is_enabled": true,
  "tags": "标签",
  "metadata": "扩展元数据",
  "created_by": 1,
  "updated_by": 1,
  "last_scan": "2025-01-11T09:00:00Z",
  "created_at": "2025-01-11T08:00:00Z",
  "updated_at": "2025-01-11T10:00:00Z"
}
```

### ScanTool 扫描工具模型
```json
{
  "id": 1,
  "name": "工具名称",
  "display_name": "工具显示名称",
  "description": "工具描述",
  "type": "port_scan",
  "version": "版本号",
  "executable_path": "可执行文件路径",
  "config_template": "配置模板",
  "input_format": "输入格式",
  "output_format": "输出格式",
  "supported_targets": ["支持的目标类型"],
  "max_concurrent": 5,
  "timeout_second": 600,
  "retry_count": 3,
  "status": "enabled",
  "is_built_in": true,
  "compatibility": "兼容性信息",
  "tags": "标签",
  "metadata": "扩展元数据"
}
```

### ScanRule 扫描规则模型
```json
{
  "id": 1,
  "name": "规则名称",
  "description": "规则描述",
  "type": "filter",
  "category": "规则分类",
  "severity": "high",
  "config": {},
  "conditions": [],
  "actions": [],
  "tags": ["标签"],
  "is_built_in": false,
  "priority": 80,
  "status": "enabled"
}
```

### WorkflowConfig 工作流模型
```json
{
  "id": 1,
  "name": "工作流名称",
  "description": "工作流描述",
  "project_id": 1,
  "config": {},
  "steps": [],
  "tags": ["标签"],
  "is_built_in": false,
  "status": "active"
}
```

## 🔒 状态码说明

### 项目配置状态
- `0`: 未激活 (inactive)
- `1`: 激活 (active)  
- `2`: 已归档 (archived)

### 扫描工具状态
- `enabled`: 启用
- `disabled`: 禁用
- `installing`: 安装中
- `error`: 错误

### 扫描规则状态
- `enabled`: 启用
- `disabled`: 禁用
- `testing`: 测试中

### 工作流状态
- `0`: 草稿 (draft)
- `1`: 激活 (active)
- `2`: 未激活 (inactive)
- `3`: 已归档 (archived)

## 🚨 错误码说明

### 通用错误码
- `400`: 请求参数错误
- `401`: 未授权访问
- `403`: 权限不足
- `404`: 资源不存在
- `409`: 资源冲突
- `500`: 服务器内部错误

### 业务错误码
- `10001`: 项目配置不存在
- `10002`: 扫描工具不存在
- `10003`: 扫描规则不存在
- `10004`: 工作流不存在
- `10005`: 规则执行失败
- `10006`: 工作流执行失败

## 📝 使用示例

### 创建完整的扫描项目
```bash
# 1. 创建项目配置
curl -X POST http://localhost:8123/api/v1/orchestrator/projects \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Web安全扫描项目",
    "target_scope": "example.com",
    "scan_frequency": 24
  }'

# 2. 创建扫描工具
curl -X POST http://localhost:8123/api/v1/orchestrator/tools \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Nmap",
    "type": "port_scan",
    "executable_path": "/usr/bin/nmap"
  }'

# 3. 创建扫描规则
curl -X POST http://localhost:8123/api/v1/orchestrator/rules \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高危端口检测",
    "type": "filter",
    "category": "port_security",
    "severity": "high"
  }'

# 4. 创建工作流
curl -X POST http://localhost:8123/api/v1/orchestrator/workflows \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "自动化扫描流程",
    "project_id": 1,
    "steps": []
  }'
```

## 📚 更新日志

### v1.0 (2025-01-11)
- 初始版本发布
- 完整的项目配置管理API
- 扫描工具管理API
- 扫描规则管理API  
- 工作流管理API
- 规则引擎API
- 统一的响应格式和错误处理

---

**文档维护**: NeoScan 开发团队  
**最后更新**: 2025-01-11  
**版本**: v1.0
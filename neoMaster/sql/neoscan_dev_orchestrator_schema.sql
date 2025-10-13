-- NeoScan 扫描配置模块数据库建表SQL脚本
-- 数据库: neoscan_dev
-- 版本: MySQL 8.0
-- 生成时间: 2025-10-11 (Linus-inspired AI 优化版本)
-- 说明: 根据扫描配置模块的Go模型定义生成的建表语句
-- 设计原则: 
--   1. "好品味"设计 - 消除特殊情况，统一数据结构
--   2. "Never break userspace" - 向后兼容，不破坏现有功能
--   3. 实用主义 - 解决实际问题，避免过度设计
--   4. 简洁执念 - 数据结构决定算法复杂度

-- 使用现有数据库
USE `neoscan_dev`;

-- ==================== 项目配置表 ====================
-- 1. 项目配置主表 (project_configs)
-- 对应Go结构体: ProjectConfig (使用BaseModel)
CREATE TABLE `project_configs` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `name` varchar(100) NOT NULL COMMENT '项目名称，唯一',
    `display_name` varchar(200) DEFAULT NULL COMMENT '项目显示名称',
    `description` text DEFAULT NULL COMMENT '项目描述',
    `target_scope` text NOT NULL COMMENT '扫描目标范围，支持IP段、域名等',
    `exclude_list` text DEFAULT NULL COMMENT '排除列表，不扫描的目标',
    `scan_frequency` int NOT NULL DEFAULT '24' COMMENT '扫描频率(小时)',
    `max_concurrent` int NOT NULL DEFAULT '10' COMMENT '最大并发数',
    `timeout_second` int NOT NULL DEFAULT '300' COMMENT '超时时间(秒)',
    `priority` int NOT NULL DEFAULT '5' COMMENT '优先级(1-10)',
    `notify_on_success` tinyint(1) NOT NULL DEFAULT '0' COMMENT '成功时是否通知',
    `notify_on_failure` tinyint(1) NOT NULL DEFAULT '1' COMMENT '失败时是否通知',
    `notify_emails` text DEFAULT NULL COMMENT '通知邮箱列表，逗号分隔',
    `webhook_url` varchar(500) DEFAULT NULL COMMENT 'Webhook通知URL',
    `tags` json DEFAULT NULL COMMENT '项目标签列表',
    `status` int NOT NULL DEFAULT '0' COMMENT '项目状态：0-未激活，1-激活，2-已归档',
    `is_enabled` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否启用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_project_configs_name` (`name`),
    KEY `idx_project_configs_status` (`status`),
    KEY `idx_project_configs_is_enabled` (`is_enabled`),
    KEY `idx_project_configs_priority` (`priority`),
    KEY `idx_project_configs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='项目配置主表';

-- ==================== 扫描工具表 ====================
-- 2. 扫描工具配置表 (scan_tools)
-- 对应Go结构体: ScanTool (使用BaseModel)
CREATE TABLE `scan_tools` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `name` varchar(100) NOT NULL COMMENT '工具名称，唯一',
    `display_name` varchar(200) DEFAULT NULL COMMENT '工具显示名称',
    `description` text DEFAULT NULL COMMENT '工具描述',
    `version` varchar(50) NOT NULL COMMENT '工具版本',
    `tool_type` varchar(20) NOT NULL COMMENT '工具类型：port_scan-端口扫描，vuln_scan-漏洞扫描，web_scan-Web扫描，network_scan-网络扫描，custom-自定义',
    `category` varchar(50) DEFAULT NULL COMMENT '工具分类',
    `executable_path` varchar(500) NOT NULL COMMENT '可执行文件路径',
    `install_path` varchar(500) DEFAULT NULL COMMENT '安装路径',
    `config_template` json DEFAULT NULL COMMENT '配置模板',
    `default_args` json DEFAULT NULL COMMENT '默认参数列表',
    `supported_formats` json DEFAULT NULL COMMENT '支持的输出格式',
    `environment_vars` json DEFAULT NULL COMMENT '环境变量配置',
    `dependencies` json DEFAULT NULL COMMENT '依赖项列表',
    `tags` json DEFAULT NULL COMMENT '工具标签',
    `is_built_in` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否内置工具',
    `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否激活',
    `status` int NOT NULL DEFAULT '1' COMMENT '工具状态：0-未安装，1-已安装，2-需要更新，3-已禁用',
    `last_check_time` datetime DEFAULT NULL COMMENT '最后检查时间',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_scan_tools_name` (`name`),
    KEY `idx_scan_tools_tool_type` (`tool_type`),
    KEY `idx_scan_tools_is_active` (`is_active`),
    KEY `idx_scan_tools_status` (`status`),
    KEY `idx_scan_tools_category` (`category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='扫描工具配置表';

-- ==================== 扫描规则表 ====================
-- 3. 扫描规则配置表 (scan_rules)
-- 对应Go结构体: ScanRule (使用BaseModel)
CREATE TABLE `scan_rules` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `name` varchar(100) NOT NULL COMMENT '规则名称，唯一',
    `display_name` varchar(200) DEFAULT NULL COMMENT '规则显示名称',
    `description` text DEFAULT NULL COMMENT '规则描述',
    `rule_type` varchar(20) NOT NULL COMMENT '规则类型：filter-过滤规则，validation-验证规则，transform-转换规则，alert-告警规则，custom-自定义规则',
    `category` varchar(50) DEFAULT NULL COMMENT '规则分类',
    `condition` text NOT NULL COMMENT '规则条件表达式',
    `action` text DEFAULT NULL COMMENT '规则动作定义',
    `parameters` json DEFAULT NULL COMMENT '规则参数配置',
    `applicable_tools` text DEFAULT NULL COMMENT '适用的扫描工具，逗号分隔',
    `target_types` text DEFAULT NULL COMMENT '适用的目标类型，逗号分隔',
    `scan_phases` text DEFAULT NULL COMMENT '适用的扫描阶段，逗号分隔',
    `severity` varchar(20) NOT NULL DEFAULT 'medium' COMMENT '严重程度：low-低，medium-中，high-高，critical-严重',
    `priority` int NOT NULL DEFAULT '5' COMMENT '优先级(1-10)',
    `tags` json DEFAULT NULL COMMENT '规则标签',
    `is_built_in` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否内置规则',
    `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否激活',
    `status` int NOT NULL DEFAULT '1' COMMENT '规则状态：0-草稿，1-激活，2-未激活，3-已归档',
    `execution_count` bigint NOT NULL DEFAULT '0' COMMENT '执行次数统计',
    `match_count` bigint NOT NULL DEFAULT '0' COMMENT '匹配次数统计',
    `last_execution_time` datetime DEFAULT NULL COMMENT '最后执行时间',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_scan_rules_name` (`name`),
    KEY `idx_scan_rules_rule_type` (`rule_type`),
    KEY `idx_scan_rules_category` (`category`),
    KEY `idx_scan_rules_severity` (`severity`),
    KEY `idx_scan_rules_is_active` (`is_active`),
    KEY `idx_scan_rules_status` (`status`),
    KEY `idx_scan_rules_priority` (`priority`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='扫描规则配置表';

-- ==================== 工作流配置表 ====================
-- 4. 工作流配置表 (workflow_configs)
-- 对应Go结构体: WorkflowConfig (使用BaseModel)
CREATE TABLE `workflow_configs` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `project_id` bigint unsigned NOT NULL COMMENT '项目配置ID，外键关联project_configs表',
    `name` varchar(100) NOT NULL COMMENT '工作流名称',
    `display_name` varchar(200) DEFAULT NULL COMMENT '工作流显示名称',
    `description` text DEFAULT NULL COMMENT '工作流描述',
    `workflow_type` varchar(20) NOT NULL DEFAULT 'sequential' COMMENT '工作流类型：sequential-顺序执行，parallel-并行执行，conditional-条件执行',
    `trigger_type` varchar(20) NOT NULL DEFAULT 'manual' COMMENT '触发类型：manual-手动，scheduled-定时，event-事件',
    `trigger_config` json DEFAULT NULL COMMENT '触发配置',
    `stages` json NOT NULL COMMENT '工作流阶段配置',
    `variables` json DEFAULT NULL COMMENT '工作流变量',
    `timeout_minutes` int NOT NULL DEFAULT '60' COMMENT '超时时间(分钟)',
    `retry_count` int NOT NULL DEFAULT '0' COMMENT '重试次数',
    `retry_interval` int NOT NULL DEFAULT '60' COMMENT '重试间隔(秒)',
    `tags` json DEFAULT NULL COMMENT '工作流标签',
    `is_template` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为模板',
    `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否激活',
    `status` int NOT NULL DEFAULT '0' COMMENT '工作流状态：0-草稿，1-激活，2-未激活，3-已归档',
    `execution_count` bigint NOT NULL DEFAULT '0' COMMENT '执行次数统计',
    `success_count` bigint NOT NULL DEFAULT '0' COMMENT '成功次数统计',
    `failure_count` bigint NOT NULL DEFAULT '0' COMMENT '失败次数统计',
    `last_execution_time` datetime DEFAULT NULL COMMENT '最后执行时间',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    KEY `idx_workflow_configs_project_id` (`project_id`),
    KEY `idx_workflow_configs_workflow_type` (`workflow_type`),
    KEY `idx_workflow_configs_trigger_type` (`trigger_type`),
    KEY `idx_workflow_configs_is_active` (`is_active`),
    KEY `idx_workflow_configs_status` (`status`),
    KEY `idx_workflow_configs_is_template` (`is_template`),
    CONSTRAINT `fk_workflow_configs_project` FOREIGN KEY (`project_id`) REFERENCES `project_configs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工作流配置表';

-- ==================== 工作流阶段表 ====================
-- 5. 工作流阶段表 (workflow_stages)
-- 对应Go结构体: WorkflowStage (使用BaseModel)
CREATE TABLE `workflow_stages` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `workflow_id` bigint unsigned NOT NULL COMMENT '工作流配置ID，外键关联workflow_configs表',
    `name` varchar(100) NOT NULL COMMENT '阶段名称',
    `display_name` varchar(200) DEFAULT NULL COMMENT '阶段显示名称',
    `stage_type` varchar(20) NOT NULL COMMENT '阶段类型：scan-扫描，process-处理，notify-通知，ai-AI处理',
    `order` int NOT NULL COMMENT '执行顺序',
    `is_enabled` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否启用',
    `scan_tool_id` bigint unsigned DEFAULT NULL COMMENT '扫描工具ID，外键关联scan_tools表',
    `config` json DEFAULT NULL COMMENT '阶段配置',
    `ai_config` json DEFAULT NULL COMMENT 'AI配置',
    `timeout_minutes` int NOT NULL DEFAULT '30' COMMENT '超时时间(分钟)',
    `retry_count` int NOT NULL DEFAULT '0' COMMENT '重试次数',
    `continue_on_failure` tinyint(1) NOT NULL DEFAULT '0' COMMENT '失败时是否继续',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    KEY `idx_workflow_stages_workflow_id` (`workflow_id`),
    KEY `idx_workflow_stages_stage_type` (`stage_type`),
    KEY `idx_workflow_stages_order` (`order`),
    KEY `idx_workflow_stages_is_enabled` (`is_enabled`),
    KEY `idx_workflow_stages_scan_tool_id` (`scan_tool_id`),
    CONSTRAINT `fk_workflow_stages_workflow` FOREIGN KEY (`workflow_id`) REFERENCES `workflow_configs` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_workflow_stages_scan_tool` FOREIGN KEY (`scan_tool_id`) REFERENCES `scan_tools` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工作流阶段表';

-- ==================== 项目规则关联表 ====================
-- 6. 项目规则关联表 (project_scan_rules)
-- 项目配置与扫描规则的多对多关联表
CREATE TABLE `project_scan_rules` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `project_id` bigint unsigned NOT NULL COMMENT '项目配置ID，外键关联project_configs表',
    `rule_id` bigint unsigned NOT NULL COMMENT '扫描规则ID，外键关联scan_rules表',
    `is_enabled` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否启用该规则',
    `priority` int NOT NULL DEFAULT '5' COMMENT '规则在项目中的优先级',
    `config_override` json DEFAULT NULL COMMENT '规则配置覆盖',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_project_scan_rules_project_rule` (`project_id`, `rule_id`),
    KEY `idx_project_scan_rules_rule_id` (`rule_id`),
    KEY `idx_project_scan_rules_is_enabled` (`is_enabled`),
    CONSTRAINT `fk_project_scan_rules_project` FOREIGN KEY (`project_id`) REFERENCES `project_configs` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_project_scan_rules_rule` FOREIGN KEY (`rule_id`) REFERENCES `scan_rules` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='项目规则关联表';

-- ==================== 扫描配置模板表 ====================
-- 7. 扫描配置模板表 (scan_config_templates)
-- 用于存储可复用的扫描配置模板
CREATE TABLE `scan_config_templates` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `name` varchar(100) NOT NULL COMMENT '模板名称，唯一',
    `display_name` varchar(200) DEFAULT NULL COMMENT '模板显示名称',
    `description` text DEFAULT NULL COMMENT '模板描述',
    `category` varchar(50) DEFAULT NULL COMMENT '模板分类',
    `template_type` varchar(20) NOT NULL COMMENT '模板类型：project-项目模板，workflow-工作流模板，rule-规则模板',
    `config_data` json NOT NULL COMMENT '模板配置数据',
    `tags` json DEFAULT NULL COMMENT '模板标签',
    `is_public` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否公开模板',
    `is_built_in` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否内置模板',
    `usage_count` bigint NOT NULL DEFAULT '0' COMMENT '使用次数统计',
    `created_by` bigint unsigned DEFAULT NULL COMMENT '创建者用户ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_scan_config_templates_name` (`name`),
    KEY `idx_scan_config_templates_category` (`category`),
    KEY `idx_scan_config_templates_template_type` (`template_type`),
    KEY `idx_scan_config_templates_is_public` (`is_public`),
    KEY `idx_scan_config_templates_created_by` (`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='扫描配置模板表';

-- ==================== 插入测试数据 ====================

-- 插入默认扫描工具
INSERT INTO `scan_tools` (`name`, `display_name`, `description`, `version`, `tool_type`, `category`, `executable_path`, `config_template`, `default_args`, `supported_formats`, `is_built_in`, `is_active`, `status`) VALUES
('nmap', 'Nmap网络扫描器', '强大的网络发现和安全审计工具', '7.94', 'port_scan', 'network', '/usr/bin/nmap', '{"timeout": 300, "max_rate": 1000, "scan_type": "syn"}', '["-sS", "-O", "-sV", "--version-intensity", "5"]', '["xml", "json", "txt"]', 1, 1, 1),
('masscan', 'Masscan高速端口扫描器', '高速异步TCP端口扫描器', '1.3.2', 'port_scan', 'network', '/usr/bin/masscan', '{"rate": 10000, "timeout": 30}', '["--rate", "10000", "--wait", "0"]', '["xml", "json", "list"]', 1, 1, 1),
('nuclei', 'Nuclei漏洞扫描器', '基于模板的快速漏洞扫描器', '3.1.0', 'vuln_scan', 'security', '/usr/bin/nuclei', '{"threads": 25, "timeout": 10, "severity": "medium"}', '["-t", "/opt/nuclei-templates/", "-j"]', '["json", "txt"]', 1, 1, 1),
('gobuster', 'Gobuster目录扫描器', 'Go语言编写的目录/文件暴力破解工具', '3.6', 'web_scan', 'web', '/usr/bin/gobuster', '{"threads": 10, "timeout": 10}', '["dir", "-t", "10", "-q"]', '["txt", "json"]', 1, 1, 1);

-- 插入默认扫描规则
INSERT INTO `scan_rules` (`name`, `display_name`, `description`, `rule_type`, `category`, `condition`, `action`, `parameters`, `applicable_tools`, `severity`, `priority`, `is_built_in`, `is_active`, `status`) VALUES
('port_filter_common', '常用端口过滤规则', '过滤出常用的服务端口进行重点扫描', 'filter', 'port_scan', 'port in [21,22,23,25,53,80,110,143,443,993,995,8080,8443]', 'include', '{"ports": [21,22,23,25,53,80,110,143,443,993,995,8080,8443]}', 'nmap,masscan', 'medium', 8, 1, 1, 1),
('vuln_severity_high', '高危漏洞告警规则', '发现高危漏洞时立即告警', 'alert', 'vulnerability', 'severity >= "high"', 'alert', '{"notification": true, "webhook": true}', 'nuclei', 'high', 9, 1, 1, 1),
('web_common_paths', 'Web常见路径扫描规则', '扫描Web应用常见的敏感路径', 'validation', 'web_scan', 'response_code in [200,301,302,403]', 'validate', '{"paths": ["/admin", "/login", "/api", "/backup"]}', 'gobuster', 'medium', 7, 1, 1, 1),
('scan_timeout_control', '扫描超时控制规则', '控制单个目标的扫描超时时间', 'transform', 'performance', 'scan_duration > 300', 'timeout', '{"max_duration": 300, "action": "terminate"}', 'nmap,masscan,nuclei', 'low', 5, 1, 1, 1);

-- 插入默认项目配置
INSERT INTO `project_configs` (`name`, `display_name`, `description`, `target_scope`, `exclude_list`, `scan_frequency`, `max_concurrent`, `timeout_second`, `priority`, `notify_on_success`, `notify_on_failure`, `notify_emails`, `tags`, `status`, `is_enabled`) VALUES
('internal_network_scan', '内网安全扫描项目', '对内网环境进行定期安全扫描，发现潜在的安全风险', '192.168.1.0/24,10.0.0.0/8', '192.168.1.1,10.0.0.1', 24, 5, 600, 8, 0, 1, 'security@company.com,admin@company.com', '["internal", "security", "network"]', 1, 1),
('web_app_security_scan', 'Web应用安全扫描', '对公司Web应用进行安全漏洞扫描', 'https://app.company.com,https://api.company.com', '', 12, 3, 1800, 9, 1, 1, 'security@company.com,dev@company.com', '["web", "security", "application"]', 1, 1),
('external_asset_discovery', '外部资产发现项目', '发现和监控公司的外部暴露资产', 'company.com,*.company.com', 'mail.company.com', 168, 10, 300, 6, 0, 1, 'security@company.com', '["external", "asset", "discovery"]', 1, 1);

-- 插入默认工作流配置
INSERT INTO `workflow_configs` (`project_id`, `name`, `display_name`, `description`, `workflow_type`, `trigger_type`, `trigger_config`, `stages`, `variables`, `timeout_minutes`, `retry_count`, `is_active`, `status`) VALUES
(1, 'internal_network_workflow', '内网扫描工作流', '内网环境的完整安全扫描流程', 'sequential', 'scheduled', '{"cron": "0 2 * * *", "timezone": "Asia/Shanghai"}', '[{"name": "port_discovery", "type": "scan", "tool": "nmap", "order": 1}, {"name": "service_detection", "type": "scan", "tool": "nmap", "order": 2}, {"name": "vulnerability_scan", "type": "scan", "tool": "nuclei", "order": 3}]', '{"scan_intensity": "normal", "report_format": "json"}', 120, 1, 1, 1),
(2, 'web_security_workflow', 'Web安全扫描工作流', 'Web应用的全面安全检测流程', 'parallel', 'manual', '{}', '[{"name": "directory_scan", "type": "scan", "tool": "gobuster", "order": 1}, {"name": "vulnerability_scan", "type": "scan", "tool": "nuclei", "order": 1}]', '{"wordlist": "common", "threads": 10}', 180, 2, 1, 1),
(3, 'asset_discovery_workflow', '资产发现工作流', '外部资产发现和监控流程', 'sequential', 'scheduled', '{"cron": "0 0 * * 0", "timezone": "Asia/Shanghai"}', '[{"name": "subdomain_discovery", "type": "scan", "tool": "nmap", "order": 1}, {"name": "port_scan", "type": "scan", "tool": "masscan", "order": 2}, {"name": "service_fingerprint", "type": "scan", "tool": "nmap", "order": 3}]', '{"discovery_depth": "deep", "rate_limit": 1000}', 240, 0, 1, 1);

-- 插入工作流阶段详细配置
INSERT INTO `workflow_stages` (`workflow_id`, `name`, `display_name`, `stage_type`, `order`, `is_enabled`, `scan_tool_id`, `config`, `timeout_minutes`, `retry_count`, `continue_on_failure`) VALUES
(1, 'port_discovery', '端口发现阶段', 'scan', 1, 1, 1, '{"scan_type": "syn", "port_range": "1-65535", "timing": "T4"}', 30, 1, 1),
(1, 'service_detection', '服务识别阶段', 'scan', 2, 1, 1, '{"version_detection": true, "os_detection": true, "script_scan": "default"}', 45, 1, 1),
(1, 'vulnerability_scan', '漏洞扫描阶段', 'scan', 3, 1, 3, '{"templates": "cves,vulnerabilities", "severity": "medium,high,critical"}', 60, 2, 0),
(2, 'directory_scan', '目录扫描阶段', 'scan', 1, 1, 4, '{"wordlist": "common.txt", "extensions": "php,asp,aspx,jsp", "status_codes": "200,301,302,403"}', 45, 1, 1),
(2, 'vulnerability_scan', 'Web漏洞扫描阶段', 'scan', 1, 1, 3, '{"templates": "http,ssl,dns", "severity": "medium,high,critical"}', 90, 2, 0);

-- 插入项目规则关联
INSERT INTO `project_scan_rules` (`project_id`, `rule_id`, `is_enabled`, `priority`, `config_override`) VALUES
(1, 1, 1, 8, '{"custom_ports": [22,80,443,3389,5432,3306]}'),
(1, 2, 1, 9, '{"immediate_alert": true}'),
(1, 4, 1, 5, '{"max_duration": 600}'),
(2, 2, 1, 9, '{"webhook_url": "https://hooks.company.com/security"}'),
(2, 3, 1, 7, '{"additional_paths": ["/api/v1", "/admin/login", "/.env"]}'),
(3, 1, 1, 6, '{"discovery_ports": [80,443,8080,8443]}'),
(3, 4, 1, 5, '{"max_duration": 300}');

-- 插入扫描配置模板
INSERT INTO `scan_config_templates` (`name`, `display_name`, `description`, `category`, `template_type`, `config_data`, `tags`, `is_public`, `is_built_in`, `usage_count`) VALUES
('basic_network_scan', '基础网络扫描模板', '适用于内网环境的基础网络安全扫描', 'network', 'project', '{"scan_frequency": 24, "max_concurrent": 5, "tools": ["nmap"], "rules": ["port_filter_common", "scan_timeout_control"]}', '["network", "basic", "internal"]', 1, 1, 15),
('comprehensive_web_scan', '全面Web安全扫描模板', '适用于Web应用的全面安全检测', 'web', 'workflow', '{"stages": [{"type": "directory_scan", "tool": "gobuster"}, {"type": "vulnerability_scan", "tool": "nuclei"}], "parallel": true}', '["web", "security", "comprehensive"]', 1, 1, 8),
('high_severity_alert', '高危告警规则模板', '发现高危漏洞时的告警处理规则', 'security', 'rule', '{"condition": "severity >= high", "actions": ["alert", "webhook", "email"], "priority": 9}', '["alert", "high-severity", "security"]', 1, 1, 23);

-- 创建性能优化索引
-- 项目配置表复合索引
CREATE INDEX `idx_project_configs_status_enabled` ON `project_configs` (`status`, `is_enabled`);
CREATE INDEX `idx_project_configs_priority_status` ON `project_configs` (`priority`, `status`);

-- 扫描工具表复合索引
CREATE INDEX `idx_scan_tools_type_active` ON `scan_tools` (`tool_type`, `is_active`);
CREATE INDEX `idx_scan_tools_status_type` ON `scan_tools` (`status`, `tool_type`);

-- 扫描规则表复合索引
CREATE INDEX `idx_scan_rules_type_active` ON `scan_rules` (`rule_type`, `is_active`);
CREATE INDEX `idx_scan_rules_severity_priority` ON `scan_rules` (`severity`, `priority`);
CREATE INDEX `idx_scan_rules_category_status` ON `scan_rules` (`category`, `status`);

-- 工作流配置表复合索引
CREATE INDEX `idx_workflow_configs_project_status` ON `workflow_configs` (`project_id`, `status`);
CREATE INDEX `idx_workflow_configs_type_active` ON `workflow_configs` (`workflow_type`, `is_active`);

-- 工作流阶段表复合索引
CREATE INDEX `idx_workflow_stages_workflow_order` ON `workflow_stages` (`workflow_id`, `order`);
CREATE INDEX `idx_workflow_stages_type_enabled` ON `workflow_stages` (`stage_type`, `is_enabled`);

-- 项目规则关联表复合索引
CREATE INDEX `idx_project_scan_rules_project_enabled` ON `project_scan_rules` (`project_id`, `is_enabled`);
CREATE INDEX `idx_project_scan_rules_rule_enabled` ON `project_scan_rules` (`rule_id`, `is_enabled`);

-- 显示建表完成信息
SELECT 'NeoScan 扫描配置模块数据库表结构创建完成！(Linus-inspired AI 优化版本)' as message;
SELECT 'Tables created: project_configs, scan_tools, scan_rules, workflow_configs, workflow_stages, project_scan_rules, scan_config_templates' as tables_info;
SELECT 'Design principles: Good Taste, Never break userspace, Pragmatism, Simplicity obsession' as design_info;
SELECT 'Test data inserted: 4 scan tools, 4 scan rules, 3 projects, 3 workflows, 5 stages, 7 rule associations, 3 templates' as data_info;
SELECT 'Performance indexes created for optimal query performance' as index_info;
-- NeoScan Agent模块数据库建表SQL脚本
-- 数据库: neoscan_dev
-- 版本: MySQL 8.0
-- 生成时间: 2025-10-10 (优化版本)
-- 说明: 根据使用BaseModel优化后的Agent模型定义生成的建表语句
-- 优化内容: 
--   1. 统一ID字段类型为bigint unsigned，对应Go的uint64
--   2. 确保所有表结构与Go结构体完全匹配
--   3. 保持外键约束和索引的一致性

-- 使用现有数据库
USE `neoscan_dev`;

-- 1. Agent基础信息表 (agents)
-- 对应Go结构体: Agent (使用BaseModel)
CREATE TABLE `agents` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `agent_id` varchar(100) NOT NULL COMMENT 'Agent唯一标识ID',
    `hostname` varchar(255) NOT NULL COMMENT '主机名',
    `ip_address` varchar(45) NOT NULL COMMENT 'IP地址，支持IPv6',
    `port` int NOT NULL DEFAULT '5772' COMMENT 'Agent服务端口',
    `version` varchar(50) NOT NULL COMMENT 'Agent版本号',
    `status` varchar(20) NOT NULL DEFAULT 'offline' COMMENT 'Agent状态:online-在线,offline-离线,exception-异常,maintenance-维护',
    `os` varchar(50) DEFAULT NULL COMMENT '操作系统',
    `arch` varchar(20) DEFAULT NULL COMMENT '系统架构',
    `cpu_cores` int DEFAULT NULL COMMENT 'CPU核心数',
    `memory_total` bigint DEFAULT NULL COMMENT '总内存大小(字节)',
    `disk_total` bigint DEFAULT NULL COMMENT '总磁盘大小(字节)',
    `capabilities` json DEFAULT NULL COMMENT 'Agent支持的扫描类型ID列表，对应ScanType表的ID，格式：["2","3"]',
    `tags` json DEFAULT NULL COMMENT 'Agent标签ID列表，对应TagType表的ID，格式：["2","3"]',
    `grpc_token` varchar(500) DEFAULT NULL COMMENT 'gRPC通信Token',
    `token_expiry` datetime DEFAULT NULL COMMENT 'Token过期时间',
    `result_latest_time` datetime DEFAULT NULL COMMENT '最新返回结果时间',
    `last_heartbeat` datetime DEFAULT NULL COMMENT '最后心跳时间',
    `remark` varchar(500) DEFAULT NULL COMMENT '备注信息',
    `container_id` varchar(100) DEFAULT NULL COMMENT '容器ID',
    `pid` int DEFAULT NULL COMMENT '进程ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_agent_id` (`agent_id`),
    KEY `idx_agents_status` (`status`),
    KEY `idx_agents_ip_address` (`ip_address`),
    KEY `idx_agents_last_heartbeat` (`last_heartbeat`),
    KEY `idx_agents_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent基础信息表';

-- 2. Agent版本信息表 (agent_versions)
-- 对应Go结构体: AgentVersion (使用BaseModel)
CREATE TABLE `agent_versions` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `version` varchar(50) NOT NULL COMMENT '版本号',
    `release_date` datetime NOT NULL COMMENT '发布日期',
    `changelog` text DEFAULT NULL COMMENT '版本更新日志',
    `download_url` varchar(500) DEFAULT NULL COMMENT '下载地址',
    `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否激活',
    `is_latest` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为最新版本',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_versions_version` (`version`),
    KEY `idx_agent_versions_is_active` (`is_active`),
    KEY `idx_agent_versions_is_latest` (`is_latest`),
    KEY `idx_agent_versions_release_date` (`release_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent版本信息表';

-- 3. Agent配置表 (agent_configs)
-- 对应Go结构体: AgentConfig (使用BaseModel)
CREATE TABLE `agent_configs` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `agent_id` varchar(100) NOT NULL COMMENT 'Agent业务ID，唯一索引',
    `version` int NOT NULL DEFAULT '1' COMMENT '配置版本号',
    `heartbeat_interval` int NOT NULL DEFAULT '30' COMMENT '心跳间隔(秒)',
    `task_poll_interval` int NOT NULL DEFAULT '10' COMMENT '任务轮询间隔(秒)',
    `max_concurrent_tasks` int NOT NULL DEFAULT '5' COMMENT '最大并发任务数',
    `plugin_config` json DEFAULT NULL COMMENT '插件配置信息',
    `log_level` varchar(20) NOT NULL DEFAULT 'info' COMMENT '日志级别',
    `timeout` int NOT NULL DEFAULT '300' COMMENT '超时时间(秒)',
    `token_expiry_duration` int NOT NULL DEFAULT '86400' COMMENT 'Token过期时间(秒)',
    `token_never_expire` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Token是否永不过期',
    `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否激活',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_configs_agent_id` (`agent_id`),
    KEY `idx_agent_configs_is_active` (`is_active`),
    KEY `idx_agent_configs_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent配置表';

-- 4. Agent指标表 (agent_metrics)
-- 对应Go结构体: AgentMetrics (使用BaseModel)
CREATE TABLE `agent_metrics` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `agent_id` varchar(100) NOT NULL COMMENT 'Agent业务ID，唯一索引',
    `cpu_usage` double DEFAULT NULL COMMENT 'CPU使用率(百分比)',
    `memory_usage` double DEFAULT NULL COMMENT '内存使用率(百分比)',
    `disk_usage` double DEFAULT NULL COMMENT '磁盘使用率(百分比)',
    `network_bytes_sent` bigint DEFAULT NULL COMMENT '网络发送字节数',
    `network_bytes_recv` bigint DEFAULT NULL COMMENT '网络接收字节数',
    `active_connections` int DEFAULT NULL COMMENT '活动连接数',
    `running_tasks` int DEFAULT NULL COMMENT '正在运行的任务数',
    `completed_tasks` int DEFAULT NULL COMMENT '已完成任务数',
    `failed_tasks` int DEFAULT NULL COMMENT '失败任务数',
    `work_status` varchar(20) DEFAULT NULL COMMENT '工作状态:idle-空闲,working-工作中,exception-异常',
    `scan_type` varchar(50) DEFAULT NULL COMMENT '当前扫描类型',
    `plugin_status` json DEFAULT NULL COMMENT '插件状态信息',
    `timestamp` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '指标时间戳',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_metrics_agent_id` (`agent_id`),
    KEY `idx_agent_metrics_timestamp` (`timestamp`),
    KEY `idx_agent_metrics_work_status` (`work_status`),
    KEY `idx_agent_metrics_cpu_usage` (`cpu_usage`),
    KEY `idx_agent_metrics_memory_usage` (`memory_usage`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent指标表';

-- 5. Agent分组表 (agent_groups)
-- 对应Go结构体: AgentGroup (使用BaseModel)
CREATE TABLE `agent_groups` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `group_id` varchar(100) NOT NULL COMMENT '分组业务ID，唯一索引',
    `name` varchar(100) NOT NULL COMMENT '分组名称',
    `description` varchar(500) DEFAULT NULL COMMENT '分组描述',
    `tags` json DEFAULT NULL COMMENT '分组标签列表',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_groups_name` (`name`),
    UNIQUE KEY `idx_agent_groups_group_id` (`group_id`),
    KEY `idx_agent_groups_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent分组表';

-- 6. Agent分组成员关联表 (agent_group_members)
-- 对应Go结构体: AgentGroupMember (使用BaseModel)
CREATE TABLE `agent_group_members` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `agent_id` varchar(100) NOT NULL COMMENT 'Agent业务ID',
    `group_id` varchar(100) NOT NULL COMMENT '分组业务ID',
    `joined_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_group_members_agent_id_group_id` (`agent_id`,`group_id`),
    KEY `idx_agent_group_members_group_id` (`group_id`),
    KEY `idx_agent_group_members_joined_at` (`joined_at`),
    CONSTRAINT `fk_agent_group_members_agent` FOREIGN KEY (`agent_id`) REFERENCES `agents` (`agent_id`) ON DELETE CASCADE,
    CONSTRAINT `fk_agent_group_members_group` FOREIGN KEY (`group_id`) REFERENCES `agent_groups` (`group_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent分组成员关联表';

-- 7. Agent任务分配表 (agent_task_assignments)
-- 对应Go结构体: AgentTaskAssignment (使用BaseModel)
CREATE TABLE `agent_task_assignments` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `agent_id` varchar(100) NOT NULL COMMENT 'Agent业务ID',
    `task_id` varchar(100) NOT NULL COMMENT '任务ID',
    `task_type` varchar(50) NOT NULL COMMENT '任务类型',
    `assigned_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '任务分配时间',
    `started_at` datetime DEFAULT NULL COMMENT '任务开始时间',
    `completed_at` datetime DEFAULT NULL COMMENT '任务完成时间',
    `status` varchar(20) NOT NULL DEFAULT 'assigned' COMMENT '任务状态:assigned-已分配,running-运行中,completed-已完成,failed-已失败',
    `result` text DEFAULT NULL COMMENT '任务执行结果',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    KEY `idx_agent_task_assignments_agent_id` (`agent_id`),
    KEY `idx_agent_task_assignments_task_id` (`task_id`),
    KEY `idx_agent_task_assignments_status` (`status`),
    KEY `idx_agent_task_assignments_assigned_at` (`assigned_at`),
    KEY `idx_agent_task_assignments_task_type` (`task_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent任务分配表';

-- 8. Agent扫描类型表 (agent_scan_types)
-- 对应Go结构体: ScanType (使用BaseModel)
CREATE TABLE `agent_scan_types` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `name` varchar(100) NOT NULL COMMENT '扫描类型名称，唯一',
    `display_name` varchar(100) NOT NULL COMMENT '扫描类型显示名称',
    `description` varchar(500) DEFAULT NULL COMMENT '扫描类型描述',
    `category` varchar(50) DEFAULT NULL COMMENT '扫描类型分类',
    `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否激活',
    `config_template` json DEFAULT NULL COMMENT '配置模板',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_scan_types_name` (`name`),
    KEY `idx_agent_scan_types_is_active` (`is_active`),
    KEY `idx_agent_scan_types_category` (`category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent扫描类型表';

-- 9. Agent标签类型表 (agent_tag_types)
-- 对应Go结构体: TagType (使用BaseModel)
CREATE TABLE `agent_tag_types` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID，对应BaseModel.ID(uint64)',
    `name` varchar(100) NOT NULL COMMENT '标签类型名称，唯一',
    `display_name` varchar(100) NOT NULL COMMENT '标签类型显示名称',
    `description` varchar(500) DEFAULT NULL COMMENT '标签类型描述',
    `remarks` varchar(500) DEFAULT NULL COMMENT '标签类型备注',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间，对应BaseModel.CreatedAt',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间，对应BaseModel.UpdatedAt',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_tag_types_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent标签类型表';

-- 插入默认数据

-- 默认Agent版本
INSERT INTO `agent_versions` (`version`, `release_date`, `changelog`, `download_url`, `is_active`, `is_latest`) VALUES
('v1.0.0', '2025-01-01 00:00:00', '初始版本发布\n- 基础Agent功能\n- 支持心跳检测\n- 支持任务分发', 'https://releases.neoscan.com/agent/v1.0.0/agent-v1.0.0.tar.gz', 1, 0),
('v1.1.0', '2025-01-15 00:00:00', '功能增强版本\n- 新增插件系统\n- 优化性能监控\n- 修复已知问题', 'https://releases.neoscan.com/agent/v1.1.0/agent-v1.1.0.tar.gz', 1, 1);

-- 默认Agent分组
INSERT INTO `agent_groups` (`group_id`, `name`, `description`, `tags`) VALUES
('ag_001', 'default', '默认分组', '["default", "system"]'),
('ag_002', 'production', '生产环境Agent分组', '["production", "critical"]'),
('ag_003', 'development', '开发环境Agent分组', '["development", "test"]');

-- 默认扫描类型 (对应agent.go中的AgentScanType常量)
INSERT INTO `agent_scan_types` (`name`, `display_name`, `description`, `category`, `is_active`, `config_template`) VALUES
('ipAliveScan', 'IP探活扫描', 'IP探活阶段，探测网段内存活IP', 'network', 1, '{"timeout": 30, "threads": 100, "ping_count": 3}'),
('fastPortScan', '快速端口扫描', '快速端口扫描，默认端口的快速扫描', 'network', 1, '{"timeout": 30, "threads": 100, "ports": "22,80,443,3389,3306,5432"}'),
('fullPortScan', '全量端口扫描', '全量端口扫描，全端口扫描，会带有端口对应的服务信息', 'network', 1, '{"timeout": 60, "threads": 50, "ports": "1-65535", "service_detection": true}'),
('serviceScan', '服务扫描', '服务扫描，如果端口识别不携带服务识别，这一步单独做一次服务识别', 'service', 1, '{"timeout": 120, "version_detection": true, "script_scan": true}'),
('vulnScan', '漏洞扫描', '漏洞扫描，发现安全漏洞', 'security', 1, '{"severity": "medium", "timeout": 300, "plugins": ["cve", "exploit"], "update_db": true}'),
('pocScan', 'POC扫描', 'POC扫描，结合给定的POC工具或者脚本识别，属于高精度的vulnScan', 'security', 1, '{"timeout": 600, "poc_templates": "/opt/poc-templates", "custom_scripts": true}'),
('webScan', 'Web扫描', 'Web扫描，识别出有web服务或者web框架cms等执行web扫描', 'web', 1, '{"crawl_depth": 3, "timeout": 600, "check_sql_injection": true, "check_xss": true}'),
('passScan', '弱密码扫描', '弱密码扫描，识别出有密码的服务后探测默认/弱口令检查', 'security', 1, '{"timeout": 300, "wordlist": "/opt/wordlists", "protocols": ["ssh", "ftp", "mysql", "mssql"]}'),
('proxyScan', '代理服务探测扫描', '代理服务探测扫描，识别出有代理服务后进行代理扫描', 'network', 1, '{"timeout": 180, "proxy_types": ["http", "https", "socks4", "socks5"]}'),
('dirScan', '目录扫描', '目录扫描，识别出有web系统后对系统进行目录扫描', 'web', 1, '{"timeout": 300, "wordlist": "/opt/dirbuster", "extensions": ["php", "asp", "jsp"]}'),
('subDomainScan', '子域名扫描', '子域名扫描，识别出有web系统后对系统进行子域名扫描', 'web', 1, '{"timeout": 600, "dns_servers": ["8.8.8.8", "1.1.1.1"], "wordlist": "/opt/subdomains"}'),
('apiScan', 'API资产扫描', 'API资产扫描，对需要探测的系统所暴露的API进行API资产扫描', 'web', 1, '{"timeout": 300, "swagger_detection": true, "graphql_detection": true}'),
('fileScan', '文件扫描', '文件扫描，webshell发现，病毒查杀，基于YARA的模块', 'file', 1, '{"timeout": 600, "yara_rules": "/opt/yara-rules", "scan_archives": true}'),
('otherScan', '其他扫描', '其他扫描，其他自定义的扫描类型，如自定义的脚本扫描', 'custom', 1, '{"timeout": 300, "custom_scripts": "/opt/custom-scripts", "parameters": {}}');

-- 默认标签类型 (对应agent.go中的TagType结构体)
INSERT INTO `agent_tag_types` (`name`, `display_name`, `description`, `remarks`) VALUES
('production', '生产环境', '生产环境Agent标签', '用于标识生产环境中的Agent'),
('development', '开发环境', '开发环境Agent标签', '用于标识开发环境中的Agent'),
('test', '测试环境', '测试环境Agent标签', '用于标识测试环境中的Agent'),
('high-performance', '高性能', '高性能Agent标签', '用于标识高性能配置的Agent'),
('scanner', '扫描器', '扫描器Agent标签', '用于标识专门用于扫描的Agent'),
('windows', 'Windows系统', 'Windows系统Agent标签', '用于标识运行在Windows系统上的Agent'),
('linux', 'Linux系统', 'Linux系统Agent标签', '用于标识运行在Linux系统上的Agent');

-- 开发测试Agent数据 (更新capabilities和tags字段，使用数字ID引用)
INSERT INTO `agents` (`agent_id`, `hostname`, `ip_address`, `port`, `version`, `status`, `os`, `arch`, `cpu_cores`, `memory_total`, `disk_total`, `capabilities`, `tags`, `grpc_token`, `token_expiry`, `last_heartbeat`, `remark`) VALUES
('neoscan-agent-001', 'dev-scanner-01', '192.168.1.100', 5772, 'v1.1.0', 'online', 'Linux', 'x86_64', 8, 17179869184, 107374182400, '["1", "2", "3", "4", "5", "7"]', '["2", "5", "7"]', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...', '2025-01-27 12:00:00', '2025-01-26 11:55:00', '开发环境测试Agent - 支持IP探活、快速端口扫描、全量端口扫描、服务扫描、漏洞扫描、Web扫描'),
('neoscan-agent-002', 'prod-scanner-01', '10.0.1.50', 5772, 'v1.1.0', 'online', 'Linux', 'x86_64', 16, 34359738368, 214748364800, '["1", "2", "3", "4", "5", "6", "7", "12"]', '["1", "4", "7"]', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...', '2025-01-27 12:00:00', '2025-01-26 11:58:00', '生产环境高性能Agent - 支持全套扫描功能包括API扫描'),
('neoscan-agent-003', 'test-scanner-01', '172.16.0.10', 5772, 'v1.0.0', 'offline', 'Windows', 'x86_64', 4, 8589934592, 53687091200, '["2", "7"]', '["3", "6"]', NULL, NULL, '2025-01-26 10:30:00', '测试环境Windows Agent - 仅支持快速端口扫描和Web扫描');

-- Agent配置数据
INSERT INTO `agent_configs` (`agent_id`, `version`, `heartbeat_interval`, `task_poll_interval`, `max_concurrent_tasks`, `plugin_config`, `log_level`, `timeout`) VALUES
('neoscan-agent-001', 1, 30, 10, 5, '{"nmap": {"enabled": true, "path": "/usr/bin/nmap"}, "nuclei": {"enabled": true, "templates": "/opt/nuclei-templates"}}', 'info', 300),
('neoscan-agent-002', 2, 15, 5, 10, '{"nmap": {"enabled": true, "path": "/usr/bin/nmap"}, "nuclei": {"enabled": true, "templates": "/opt/nuclei-templates"}, "masscan": {"enabled": true}}', 'warn', 600),
('neoscan-agent-003', 1, 60, 30, 3, '{"nmap": {"enabled": true, "path": "C:\\\\Program Files\\\\Nmap\\\\nmap.exe"}}', 'debug', 180);

-- Agent指标数据 (更新scan_type字段，使用正确的扫描类型名称)
INSERT INTO `agent_metrics` (`agent_id`, `cpu_usage`, `memory_usage`, `disk_usage`, `network_bytes_sent`, `network_bytes_recv`, `active_connections`, `running_tasks`, `completed_tasks`, `failed_tasks`, `work_status`, `scan_type`, `plugin_status`) VALUES
('neoscan-agent-001', 25.5, 45.2, 68.1, 1048576, 2097152, 5, 2, 156, 3, 'working', 'fullPortScan', '{"nmap": {"status": "running", "pid": 12345}, "nuclei": {"status": "idle"}}'),
('neoscan-agent-002', 15.8, 32.4, 55.7, 5242880, 10485760, 12, 4, 892, 8, 'working', 'vulnScan', '{"nmap": {"status": "idle"}, "nuclei": {"status": "running", "pid": 23456}, "masscan": {"status": "idle"}}'),
('neoscan-agent-003', 5.2, 18.9, 42.3, 524288, 1048576, 2, 0, 45, 2, 'idle', 'idle', '{"nmap": {"status": "idle"}}');

-- Agent分组成员关系
INSERT INTO `agent_group_members` (`agent_id`, `group_id`, `joined_at`) VALUES
('neoscan-agent-001', 'ag_003', '2025-01-20 10:00:00'),
('neoscan-agent-002', 'ag_002', '2025-01-18 09:30:00'),
('neoscan-agent-003', 'ag_003', '2025-01-25 14:20:00');

-- Agent任务分配示例数据 (更新task_type字段，使用正确的扫描类型名称)
INSERT INTO `agent_task_assignments` (`agent_id`, `task_id`, `task_type`, `assigned_at`, `started_at`, `completed_at`, `status`, `result`) VALUES
('neoscan-agent-001', 'task_001', 'fullPortScan', '2025-01-26 10:00:00', '2025-01-26 10:01:00', NULL, 'running', NULL),
('neoscan-agent-001', 'task_002', 'vulnScan', '2025-01-26 09:30:00', '2025-01-26 09:31:00', '2025-01-26 09:45:00', 'completed', '{"vulnerabilities": 3, "severity": "medium", "details": "发现3个中等风险漏洞"}'),
('neoscan-agent-002', 'task_003', 'webScan', '2025-01-26 11:00:00', '2025-01-26 11:02:00', NULL, 'running', NULL),
('neoscan-agent-003', 'task_004', 'fastPortScan', '2025-01-26 08:00:00', '2025-01-26 08:01:00', '2025-01-26 08:15:00', 'failed', '{"error": "连接超时", "code": "TIMEOUT"}');

-- 创建性能优化索引
-- Agent表复合索引
CREATE INDEX `idx_agents_status_heartbeat` ON `agents` (`status`, `last_heartbeat`);
CREATE INDEX `idx_agents_version_status` ON `agents` (`version`, `status`);

-- Agent指标表时间分区索引
CREATE INDEX `idx_agent_metrics_agent_timestamp` ON `agent_metrics` (`agent_id`, `timestamp`);
CREATE INDEX `idx_agent_metrics_work_scan` ON `agent_metrics` (`work_status`, `scan_type`);

-- 任务分配表复合索引
CREATE INDEX `idx_agent_task_assignments_agent_status` ON `agent_task_assignments` (`agent_id`, `status`);
CREATE INDEX `idx_agent_task_assignments_type_status` ON `agent_task_assignments` (`task_type`, `status`);
CREATE INDEX `idx_agent_task_assignments_assigned_completed` ON `agent_task_assignments` (`assigned_at`, `completed_at`);

-- 显示建表完成信息
SELECT 'NeoScan Agent模块数据库表结构创建完成！(优化版本)' as message;
SELECT 'Tables created: agents, agent_versions, agent_configs, agent_metrics, agent_groups, agent_group_members, agent_task_assignments, agent_scan_types, agent_tag_types' as tables_info;
SELECT 'BaseModel integration: All tables now use bigint unsigned ID matching Go uint64 type' as optimization_info;
SELECT 'ScanType & TagType optimization: Removed redundant scanTypeID/tagTypeID fields, unified to use BaseModel.ID' as id_optimization_info;
SELECT 'Default data inserted: 2 versions, 3 groups, 14 scan types, 7 tag types, 3 agents with configs and metrics' as data_info;
SELECT 'ID mapping: Agent capabilities/tags fields now use numeric string IDs referencing ScanType/TagType table IDs' as mapping_info;
SELECT 'Performance indexes created for optimal query performance' as index_info;
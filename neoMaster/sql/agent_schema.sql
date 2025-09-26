-- NeoScan Agent模块数据库建表SQL脚本
-- 数据库: neoscan_dev
-- 版本: MySQL 8.0
-- 生成时间: 2025-09-26
-- 说明: 根据Agent模型定义生成的建表语句

-- 使用现有数据库
USE `neoscan_dev`;

-- 1. Agent基础信息表 (agents)
CREATE TABLE `agents` (
    `id` varchar(50) NOT NULL COMMENT 'Agent唯一标识ID',
    `agent_id` varchar(100) NOT NULL COMMENT 'Agent业务ID',
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
    `capabilities` json DEFAULT NULL COMMENT 'Agent支持的功能模块列表',
    `tags` json DEFAULT NULL COMMENT 'Agent标签列表',
    `grpc_token` varchar(500) DEFAULT NULL COMMENT 'gRPC通信Token',
    `token_expiry` datetime DEFAULT NULL COMMENT 'Token过期时间',
    `result_latest_time` datetime DEFAULT NULL COMMENT '最新返回结果时间',
    `last_heartbeat` datetime DEFAULT NULL COMMENT '最后心跳时间',
    `registered_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '注册时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `remark` varchar(500) DEFAULT NULL COMMENT '备注信息',
    `container_id` varchar(100) DEFAULT NULL COMMENT '容器ID',
    `pid` int DEFAULT NULL COMMENT '进程ID',
    PRIMARY KEY (`id`),
    KEY `idx_agents_agent_id` (`agent_id`),
    KEY `idx_agents_status` (`status`),
    KEY `idx_agents_ip_address` (`ip_address`),
    KEY `idx_agents_last_heartbeat` (`last_heartbeat`),
    KEY `idx_agents_registered_at` (`registered_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent基础信息表';

-- 2. Agent版本信息表 (agent_versions)
CREATE TABLE `agent_versions` (
    `id` varchar(50) NOT NULL COMMENT '版本唯一标识ID',
    `version` varchar(50) NOT NULL COMMENT '版本号',
    `release_date` datetime NOT NULL COMMENT '发布日期',
    `changelog` text DEFAULT NULL COMMENT '版本更新日志',
    `download_url` varchar(500) DEFAULT NULL COMMENT '下载地址',
    `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否激活',
    `is_latest` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为最新版本',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_versions_version` (`version`),
    KEY `idx_agent_versions_is_active` (`is_active`),
    KEY `idx_agent_versions_is_latest` (`is_latest`),
    KEY `idx_agent_versions_release_date` (`release_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent版本信息表';

-- 3. Agent配置表 (agent_configs)
CREATE TABLE `agent_configs` (
    `id` varchar(50) NOT NULL COMMENT '配置唯一标识ID',
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
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_configs_agent_id` (`agent_id`),
    KEY `idx_agent_configs_is_active` (`is_active`),
    KEY `idx_agent_configs_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent配置表';

-- 4. Agent指标表 (agent_metrics)
CREATE TABLE `agent_metrics` (
    `id` varchar(50) NOT NULL COMMENT '指标唯一标识ID',
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
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_metrics_agent_id` (`agent_id`),
    KEY `idx_agent_metrics_timestamp` (`timestamp`),
    KEY `idx_agent_metrics_work_status` (`work_status`),
    KEY `idx_agent_metrics_cpu_usage` (`cpu_usage`),
    KEY `idx_agent_metrics_memory_usage` (`memory_usage`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent指标表';

-- 5. Agent分组表 (agent_groups)
CREATE TABLE `agent_groups` (
    `id` varchar(50) NOT NULL COMMENT '分组唯一标识ID',
    `name` varchar(100) NOT NULL COMMENT '分组名称',
    `description` varchar(500) DEFAULT NULL COMMENT '分组描述',
    `tags` json DEFAULT NULL COMMENT '分组标签列表',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_groups_name` (`name`),
    KEY `idx_agent_groups_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent分组表';

-- 6. Agent分组成员关联表 (agent_group_members)
CREATE TABLE `agent_group_members` (
    `agent_id` varchar(100) NOT NULL COMMENT 'Agent业务ID',
    `group_id` varchar(50) NOT NULL COMMENT '分组ID',
    `joined_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`agent_id`,`group_id`),
    KEY `idx_agent_group_members_group_id` (`group_id`),
    KEY `idx_agent_group_members_joined_at` (`joined_at`),
    CONSTRAINT `fk_agent_group_members_agent` FOREIGN KEY (`agent_id`) REFERENCES `agents` (`agent_id`) ON DELETE CASCADE,
    CONSTRAINT `fk_agent_group_members_group` FOREIGN KEY (`group_id`) REFERENCES `agent_groups` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent分组成员关联表';

-- 7. Agent任务分配表 (agent_task_assignments)
CREATE TABLE `agent_task_assignments` (
    `id` varchar(50) NOT NULL COMMENT '任务分配唯一标识ID',
    `agent_id` varchar(100) NOT NULL COMMENT 'Agent业务ID',
    `task_id` varchar(100) NOT NULL COMMENT '任务ID',
    `task_type` varchar(50) NOT NULL COMMENT '任务类型',
    `assigned_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '任务分配时间',
    `started_at` datetime DEFAULT NULL COMMENT '任务开始时间',
    `completed_at` datetime DEFAULT NULL COMMENT '任务完成时间',
    `status` varchar(20) NOT NULL DEFAULT 'assigned' COMMENT '任务状态:assigned-已分配,running-运行中,completed-已完成,failed-已失败',
    `result` text DEFAULT NULL COMMENT '任务执行结果',
    PRIMARY KEY (`id`),
    KEY `idx_agent_task_assignments_agent_id` (`agent_id`),
    KEY `idx_agent_task_assignments_task_id` (`task_id`),
    KEY `idx_agent_task_assignments_status` (`status`),
    KEY `idx_agent_task_assignments_assigned_at` (`assigned_at`),
    KEY `idx_agent_task_assignments_task_type` (`task_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent任务分配表';

-- 8. Agent扫描类型表 (agent_scan_types)
CREATE TABLE `agent_scan_types` (
    `id` varchar(50) NOT NULL COMMENT '扫描类型唯一标识ID',
    `name` varchar(100) NOT NULL COMMENT '扫描类型名称，唯一',
    `display_name` varchar(100) NOT NULL COMMENT '扫描类型显示名称',
    `description` varchar(500) DEFAULT NULL COMMENT '扫描类型描述',
    `category` varchar(50) DEFAULT NULL COMMENT '扫描类型分类',
    `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否激活',
    `config_template` json DEFAULT NULL COMMENT '配置模板',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_agent_scan_types_name` (`name`),
    KEY `idx_agent_scan_types_is_active` (`is_active`),
    KEY `idx_agent_scan_types_category` (`category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent扫描类型表';

-- 插入默认数据

-- 默认Agent版本
INSERT INTO `agent_versions` (`id`, `version`, `release_date`, `changelog`, `download_url`, `is_active`, `is_latest`) VALUES
('av_001', 'v1.0.0', '2025-01-01 00:00:00', '初始版本发布\n- 基础Agent功能\n- 支持心跳检测\n- 支持任务分发', 'https://releases.neoscan.com/agent/v1.0.0/agent-v1.0.0.tar.gz', 1, 0),
('av_002', 'v1.1.0', '2025-01-15 00:00:00', '功能增强版本\n- 新增插件系统\n- 优化性能监控\n- 修复已知问题', 'https://releases.neoscan.com/agent/v1.1.0/agent-v1.1.0.tar.gz', 1, 1);

-- 默认Agent分组
INSERT INTO `agent_groups` (`id`, `name`, `description`, `tags`) VALUES
('ag_001', 'default', '默认分组', '["default", "system"]'),
('ag_002', 'production', '生产环境Agent分组', '["production", "critical"]'),
('ag_003', 'development', '开发环境Agent分组', '["development", "test"]');

-- 默认扫描类型
INSERT INTO `agent_scan_types` (`id`, `name`, `display_name`, `description`, `category`, `config_template`) VALUES
('ast_001', 'port_scan', '端口扫描', '对目标主机进行端口扫描，发现开放的服务端口', 'network', '{"timeout": 30, "threads": 100, "ports": "1-65535"}'),
('ast_002', 'vuln_scan', '漏洞扫描', '对目标进行漏洞扫描，发现安全漏洞', 'security', '{"severity": "medium", "timeout": 300, "plugins": ["cve", "exploit"]}'),
('ast_003', 'web_scan', 'Web扫描', '对Web应用进行安全扫描', 'web', '{"crawl_depth": 3, "timeout": 600, "check_sql_injection": true}');

-- 开发测试Agent数据
INSERT INTO `agents` (`id`, `agent_id`, `hostname`, `ip_address`, `port`, `version`, `status`, `os`, `arch`, `cpu_cores`, `memory_total`, `disk_total`, `capabilities`, `tags`, `grpc_token`, `token_expiry`, `last_heartbeat`, `registered_at`, `remark`) VALUES
('agent_001', 'neoscan-agent-001', 'dev-scanner-01', '192.168.1.100', 5772, 'v1.1.0', 'online', 'Linux', 'x86_64', 8, 17179869184, 107374182400, '["port_scan", "vuln_scan", "web_scan"]', '["development", "scanner"]', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...', '2025-01-27 12:00:00', '2025-01-26 11:55:00', '2025-01-20 10:00:00', '开发环境测试Agent'),
('agent_002', 'neoscan-agent-002', 'prod-scanner-01', '10.0.1.50', 5772, 'v1.1.0', 'online', 'Linux', 'x86_64', 16, 34359738368, 214748364800, '["port_scan", "vuln_scan", "web_scan", "api_scan"]', '["production", "high-performance"]', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...', '2025-01-27 12:00:00', '2025-01-26 11:58:00', '2025-01-18 09:30:00', '生产环境高性能Agent'),
('agent_003', 'neoscan-agent-003', 'test-scanner-01', '172.16.0.10', 5772, 'v1.0.0', 'offline', 'Windows', 'x86_64', 4, 8589934592, 53687091200, '["port_scan", "web_scan"]', '["test", "windows"]', NULL, NULL, '2025-01-26 10:30:00', '2025-01-25 14:20:00', '测试环境Windows Agent');

-- Agent配置数据
INSERT INTO `agent_configs` (`id`, `agent_id`, `version`, `heartbeat_interval`, `task_poll_interval`, `max_concurrent_tasks`, `plugin_config`, `log_level`, `timeout`) VALUES
('ac_001', 'neoscan-agent-001', 1, 30, 10, 5, '{"nmap": {"enabled": true, "path": "/usr/bin/nmap"}, "nuclei": {"enabled": true, "templates": "/opt/nuclei-templates"}}', 'info', 300),
('ac_002', 'neoscan-agent-002', 2, 15, 5, 10, '{"nmap": {"enabled": true, "path": "/usr/bin/nmap"}, "nuclei": {"enabled": true, "templates": "/opt/nuclei-templates"}, "masscan": {"enabled": true}}', 'warn', 600),
('ac_003', 'neoscan-agent-003', 1, 60, 30, 3, '{"nmap": {"enabled": true, "path": "C:\\\\Program Files\\\\Nmap\\\\nmap.exe"}}', 'debug', 180);

-- Agent指标数据
INSERT INTO `agent_metrics` (`id`, `agent_id`, `cpu_usage`, `memory_usage`, `disk_usage`, `network_bytes_sent`, `network_bytes_recv`, `active_connections`, `running_tasks`, `completed_tasks`, `failed_tasks`, `work_status`, `scan_type`, `plugin_status`) VALUES
('am_001', 'neoscan-agent-001', 25.5, 45.2, 68.1, 1048576, 2097152, 5, 2, 156, 3, 'working', 'port_scan', '{"nmap": {"status": "running", "pid": 12345}, "nuclei": {"status": "idle"}}'),
('am_002', 'neoscan-agent-002', 15.8, 32.4, 55.7, 5242880, 10485760, 12, 4, 892, 8, 'working', 'vuln_scan', '{"nmap": {"status": "idle"}, "nuclei": {"status": "running", "pid": 23456}, "masscan": {"status": "idle"}}'),
('am_003', 'neoscan-agent-003', 5.2, 18.9, 42.3, 524288, 1048576, 2, 0, 45, 2, 'idle', 'idle', '{"nmap": {"status": "idle"}}');

-- Agent分组成员关系
INSERT INTO `agent_group_members` (`agent_id`, `group_id`, `joined_at`) VALUES
('neoscan-agent-001', 'ag_003', '2025-01-20 10:00:00'),
('neoscan-agent-002', 'ag_002', '2025-01-18 09:30:00'),
('neoscan-agent-003', 'ag_003', '2025-01-25 14:20:00');

-- Agent任务分配示例数据
INSERT INTO `agent_task_assignments` (`id`, `agent_id`, `task_id`, `task_type`, `assigned_at`, `started_at`, `completed_at`, `status`, `result`) VALUES
('ata_001', 'neoscan-agent-001', 'task_001', 'port_scan', '2025-01-26 10:00:00', '2025-01-26 10:01:00', NULL, 'running', NULL),
('ata_002', 'neoscan-agent-001', 'task_002', 'vuln_scan', '2025-01-26 09:30:00', '2025-01-26 09:31:00', '2025-01-26 09:45:00', 'completed', '{"vulnerabilities": 3, "severity": "medium", "details": "发现3个中等风险漏洞"}'),
('ata_003', 'neoscan-agent-002', 'task_003', 'web_scan', '2025-01-26 11:00:00', '2025-01-26 11:02:00', NULL, 'running', NULL),
('ata_004', 'neoscan-agent-003', 'task_004', 'port_scan', '2025-01-26 08:00:00', '2025-01-26 08:01:00', '2025-01-26 08:15:00', 'failed', '{"error": "连接超时", "code": "TIMEOUT"}');

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
SELECT 'NeoScan Agent模块数据库表结构创建完成！' as message;
SELECT 'Tables created: agents, agent_versions, agent_configs, agent_metrics, agent_groups, agent_group_members, agent_task_assignments, agent_scan_types' as tables_info;
SELECT 'Default data inserted: 2 versions, 3 groups, 3 scan types, 3 agents with configs and metrics' as data_info;
SELECT 'Performance indexes created for optimal query performance' as index_info;
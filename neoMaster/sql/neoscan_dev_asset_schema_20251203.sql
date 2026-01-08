-- ----------------------------
-- Table structure for asset_unified
-- ----------------------------
DROP TABLE IF EXISTS `asset_unified`;
CREATE TABLE `asset_unified` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `project_id` bigint unsigned NOT NULL COMMENT '所属项目ID',
  `ip` varchar(50) NOT NULL COMMENT 'IP地址',
  `port` int NOT NULL COMMENT '端口号',
  `host_name` varchar(200) DEFAULT NULL COMMENT '主机名',
  `os` varchar(100) DEFAULT NULL COMMENT '操作系统',
  `device_type` varchar(50) DEFAULT NULL COMMENT '设备类型',
  `mac_address` varchar(50) DEFAULT NULL COMMENT 'MAC地址',
  `location` varchar(100) DEFAULT NULL COMMENT '地理位置',
  `protocol` varchar(20) DEFAULT NULL COMMENT '协议(tcp/udp)',
  `service` varchar(100) DEFAULT NULL COMMENT '服务名称(http, ssh, mysql)',
  `product` varchar(100) DEFAULT NULL COMMENT '产品名称',
  `version` varchar(100) DEFAULT NULL COMMENT '版本号',
  `banner` varchar(2048) DEFAULT NULL COMMENT '服务横幅信息',
  `url` varchar(2048) DEFAULT NULL COMMENT 'Web服务入口URL',
  `title` varchar(255) DEFAULT NULL COMMENT '网页标题',
  `status_code` int DEFAULT NULL COMMENT 'HTTP状态码',
  `component` varchar(255) DEFAULT NULL COMMENT '关键组件/CMS识别结果',
  `tech_stack` json DEFAULT NULL COMMENT '详细技术栈(JSON)',
  `fingerprint` text COMMENT '关键指纹特征',
  `is_web` tinyint(1) DEFAULT '0' COMMENT '是否为Web资产',
  `source` varchar(50) DEFAULT NULL COMMENT '数据来源',
  `sync_time` datetime(3) DEFAULT NULL COMMENT '上次同步时间',
  PRIMARY KEY (`id`),
  KEY `idx_asset_unified_deleted_at` (`deleted_at`),
  KEY `idx_asset_unified_project_id` (`project_id`),
  KEY `idx_asset_unified_ip` (`ip`),
  KEY `idx_asset_unified_port` (`port`),
  KEY `idx_asset_unified_component` (`component`),
  KEY `idx_asset_unified_is_web` (`is_web`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='统一资产视图表(Read-Model)';


-- ----------------------------
-- Table structure for asset_hosts
-- ----------------------------
DROP TABLE IF EXISTS `asset_hosts`;
CREATE TABLE `asset_hosts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `ip` varchar(50) NOT NULL COMMENT 'IP地址',
  `hostname` varchar(200) DEFAULT NULL COMMENT '主机名',
  `os` varchar(100) DEFAULT NULL COMMENT '操作系统',
  `tags` json DEFAULT NULL COMMENT '标签(JSON)',
  `last_seen_at` datetime(3) DEFAULT NULL COMMENT '最后发现时间',
  `source_stage_ids` json DEFAULT NULL COMMENT '来源阶段ID列表(JSON)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_asset_hosts_ip` (`ip`),
  KEY `idx_asset_hosts_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='主机资产表';

-- ----------------------------
-- Table structure for asset_services
-- ----------------------------
DROP TABLE IF EXISTS `asset_services`;
CREATE TABLE `asset_services` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `host_id` bigint unsigned NOT NULL COMMENT '主机ID',
  `port` int NOT NULL COMMENT '端口号',
  `proto` varchar(10) DEFAULT 'tcp' COMMENT '协议(tcp/udp)',
  `name` varchar(100) DEFAULT NULL COMMENT '服务名称',
  `product` varchar(100) DEFAULT NULL COMMENT '产品名称',
  `version` varchar(100) DEFAULT NULL COMMENT '服务版本',
  `banner` varchar(2048) DEFAULT NULL COMMENT '服务横幅信息',
  `cpe` varchar(255) DEFAULT NULL COMMENT 'CPE标识',
  `fingerprint` json DEFAULT NULL COMMENT '指纹信息(JSON)',
  `asset_type` varchar(50) DEFAULT 'service' COMMENT '资产类型',
  `tags` json DEFAULT NULL COMMENT '标签(JSON)',
  `last_seen_at` datetime(3) DEFAULT NULL COMMENT '最后发现时间',
  PRIMARY KEY (`id`),
  KEY `idx_asset_services_host_id` (`host_id`),
  KEY `idx_asset_services_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='服务资产表';

-- ----------------------------
-- Table structure for asset_networks
-- ----------------------------
DROP TABLE IF EXISTS `asset_networks`;
CREATE TABLE `asset_networks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `network` varchar(50) NOT NULL COMMENT '原始网段',
  `cidr` varchar(50) NOT NULL COMMENT '拆分后的网段(CIDR格式)',
  `split_from_id` bigint unsigned DEFAULT '0' COMMENT '拆分来源ID',
  `split_order` int DEFAULT '0' COMMENT '拆分顺序',
  `round` int DEFAULT '0' COMMENT '扫描轮次',
  `network_type` varchar(20) DEFAULT 'internal' COMMENT '网络类型',
  `priority` int DEFAULT '0' COMMENT '调度优先级',
  `tags` json DEFAULT NULL COMMENT '标签(JSON)',
  `source_ref` varchar(100) DEFAULT NULL COMMENT '来源引用',
  `status` varchar(20) DEFAULT 'active' COMMENT '调度状态',
  `scan_status` varchar(20) DEFAULT 'idle' COMMENT '扫描状态',
  `last_scan_at` datetime(3) DEFAULT NULL COMMENT '最后扫描时间',
  `next_scan_at` datetime(3) DEFAULT NULL COMMENT '下次扫描时间',
  `note` varchar(255) DEFAULT NULL COMMENT '备注',
  `created_by` varchar(50) DEFAULT NULL COMMENT '创建人',
  PRIMARY KEY (`id`),
  KEY `idx_asset_networks_network` (`network`),
  KEY `idx_asset_networks_cidr` (`cidr`),
  KEY `idx_asset_networks_split_from_id` (`split_from_id`),
  KEY `idx_asset_networks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='正式网段资产表';

-- ----------------------------
-- Table structure for asset_whitelists
-- ----------------------------
DROP TABLE IF EXISTS `asset_whitelists`;
CREATE TABLE `asset_whitelists` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `whitelist_name` varchar(100) NOT NULL COMMENT '白名单名称',
  `whitelist_type` varchar(50) DEFAULT NULL COMMENT '白名单类型',
  `target_type` varchar(50) NOT NULL COMMENT '目标类型',
  `target_value` text NOT NULL COMMENT '目标值',
  `description` varchar(255) DEFAULT NULL COMMENT '描述信息',
  `valid_from` datetime(3) DEFAULT NULL COMMENT '生效开始时间',
  `valid_to` datetime(3) DEFAULT NULL COMMENT '生效结束时间',
  `created_by` varchar(100) DEFAULT NULL COMMENT '创建人',
  `tags` json DEFAULT NULL COMMENT '标签信息',
  `scope` json DEFAULT NULL COMMENT '作用域配置',
  `enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用',
  `note` text COMMENT '备注信息',
  PRIMARY KEY (`id`),
  KEY `idx_asset_whitelists_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='资产白名单表';

-- ----------------------------
-- Table structure for asset_skip_policies
-- ----------------------------
DROP TABLE IF EXISTS `asset_skip_policies`;
CREATE TABLE `asset_skip_policies` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `policy_name` varchar(100) NOT NULL COMMENT '策略名称',
  `policy_type` varchar(50) DEFAULT NULL COMMENT '策略类型',
  `description` varchar(255) DEFAULT NULL COMMENT '策略描述',
  `condition_rules` json DEFAULT NULL COMMENT '条件规则',
  `action_config` json DEFAULT NULL COMMENT '动作配置',
  `scope` json DEFAULT NULL COMMENT '作用域配置',
  `priority` int DEFAULT '0' COMMENT '优先级',
  `enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用',
  `created_by` varchar(100) DEFAULT NULL COMMENT '创建人',
  `tags` json DEFAULT NULL COMMENT '标签信息',
  `valid_from` datetime(3) DEFAULT NULL COMMENT '生效开始时间',
  `valid_to` datetime(3) DEFAULT NULL COMMENT '生效结束时间',
  `note` text COMMENT '备注信息',
  PRIMARY KEY (`id`),
  KEY `idx_asset_skip_policies_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='资产跳过策略表';

-- ----------------------------
-- Table structure for asset_network_scans
-- ----------------------------
DROP TABLE IF EXISTS `asset_network_scans`;
CREATE TABLE `asset_network_scans` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `network_id` bigint unsigned NOT NULL COMMENT '网段ID',
  `agent_id` bigint unsigned NOT NULL COMMENT '执行Agent ID',
  `scan_status` varchar(20) DEFAULT 'pending' COMMENT '扫描状态',
  `round` int DEFAULT '1' COMMENT '扫描轮次',
  `scan_tool` varchar(50) DEFAULT NULL COMMENT '扫描工具',
  `scan_config` json DEFAULT NULL COMMENT '扫描配置快照',
  `result_count` int DEFAULT '0' COMMENT '结果数量',
  `duration` int DEFAULT '0' COMMENT '扫描耗时',
  `error_message` text COMMENT '错误信息',
  `started_at` datetime(3) DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime(3) DEFAULT NULL COMMENT '完成时间',
  `assigned_at` datetime(3) DEFAULT NULL COMMENT '分配时间',
  `scan_result_ref` varchar(255) DEFAULT NULL COMMENT '结果引用',
  `note` varchar(255) DEFAULT NULL COMMENT '备注',
  `retry_count` int DEFAULT '0' COMMENT '重试次数',
  PRIMARY KEY (`id`),
  KEY `idx_asset_network_scans_network_id` (`network_id`),
  KEY `idx_asset_network_scans_agent_id` (`agent_id`),
  KEY `idx_asset_network_scans_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='网段扫描记录表';

-- ----------------------------
-- Table structure for asset_vulns
-- ----------------------------
DROP TABLE IF EXISTS `asset_vulns`;
CREATE TABLE `asset_vulns` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `target_type` varchar(50) NOT NULL COMMENT '目标类型',
  `target_ref_id` bigint unsigned NOT NULL COMMENT '指向对应实体的ID',
  `cve` varchar(50) DEFAULT NULL COMMENT 'CVE编号',
  `id_alias` varchar(100) DEFAULT NULL COMMENT '漏洞标识',
  `severity` varchar(20) DEFAULT 'medium' COMMENT '严重程度',
  `confidence` double DEFAULT '0' COMMENT '置信度',
  `evidence` json DEFAULT NULL COMMENT '原始证据(JSON)',
  `attributes` json DEFAULT NULL COMMENT '结构化属性(JSON)',
  `first_seen_at` datetime(3) DEFAULT NULL COMMENT '首次发现时间',
  `last_seen_at` datetime(3) DEFAULT NULL COMMENT '最后发现时间',
  `status` varchar(20) DEFAULT 'open' COMMENT '状态',
  `verify_status` varchar(20) DEFAULT 'not_verified' COMMENT '验证过程状态',
  `verified_by` varchar(100) DEFAULT NULL COMMENT '验证来源',
  `verified_at` datetime(3) DEFAULT NULL COMMENT '验证完成时间',
  `verify_result` text COMMENT '验证结果快照',
  PRIMARY KEY (`id`),
  KEY `idx_asset_vulns_target_type` (`target_type`),
  KEY `idx_asset_vulns_target_ref_id` (`target_ref_id`),
  KEY `idx_asset_vulns_cve` (`cve`),
  KEY `idx_asset_vulns_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='漏洞资产表';

-- ----------------------------
-- Table structure for asset_vuln_pocs
-- ----------------------------
DROP TABLE IF EXISTS `asset_vuln_pocs`;
CREATE TABLE `asset_vuln_pocs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `vuln_id` bigint unsigned NOT NULL COMMENT '关联漏洞ID',
  `poc_type` varchar(50) DEFAULT NULL COMMENT 'PoC类型',
  `name` varchar(100) DEFAULT NULL COMMENT 'PoC名称',
  `verify_url` varchar(2048) DEFAULT NULL COMMENT '验证目标URL',
  `content` longtext COMMENT 'PoC内容',
  `description` text COMMENT 'PoC详细描述',
  `source` varchar(100) DEFAULT NULL COMMENT '来源',
  `is_valid` tinyint(1) DEFAULT '1' COMMENT 'PoC本身是否有效',
  `priority` int DEFAULT '0' COMMENT '执行优先级',
  `author` varchar(50) DEFAULT NULL COMMENT '作者',
  `note` varchar(255) DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  KEY `idx_asset_vuln_pocs_vuln_id` (`vuln_id`),
  KEY `idx_asset_vuln_pocs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='漏洞验证/利用代码表';

-- ----------------------------
-- Table structure for asset_webs
-- ----------------------------
DROP TABLE IF EXISTS `asset_webs`;
CREATE TABLE `asset_webs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `host_id` bigint unsigned DEFAULT '0' COMMENT '主机ID',
  `domain` varchar(255) DEFAULT NULL COMMENT '域名',
  `url` varchar(2048) DEFAULT NULL COMMENT '完整的URL地址',
  `asset_type` varchar(50) DEFAULT 'web' COMMENT '资产类型',
  `tech_stack` json DEFAULT NULL COMMENT '技术栈信息(JSON)',
  `status` varchar(20) DEFAULT 'active' COMMENT '资产状态',
  `tags` json DEFAULT NULL COMMENT '标签信息(JSON)',
  `basic_info` json DEFAULT NULL COMMENT '基础Web信息(JSON)',
  `scan_level` int DEFAULT '0' COMMENT '扫描级别',
  `last_seen_at` datetime(3) DEFAULT NULL COMMENT '最后发现时间',
  PRIMARY KEY (`id`),
  KEY `idx_asset_webs_host_id` (`host_id`),
  KEY `idx_asset_webs_domain` (`domain`),
  KEY `idx_asset_webs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Web资产表';

-- ----------------------------
-- Table structure for asset_web_details
-- ----------------------------
DROP TABLE IF EXISTS `asset_web_details`;
CREATE TABLE `asset_web_details` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `asset_web_id` bigint unsigned NOT NULL COMMENT '关联AssetWeb表',
  `crawl_time` datetime(3) DEFAULT NULL COMMENT '爬取时间',
  `crawl_status` varchar(20) DEFAULT NULL COMMENT '爬取状态',
  `error_message` text COMMENT '错误信息',
  `content_details` json DEFAULT NULL COMMENT '详细内容信息(JSON)',
  `login_indicators` text COMMENT '登录相关标识',
  `cookies` text COMMENT 'Cookie信息',
  `screenshot` longtext COMMENT '页面截图',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_asset_web_details_asset_web_id` (`asset_web_id`),
  KEY `idx_asset_web_details_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Web详细信息表';

-- ----------------------------
-- Table structure for raw_assets
-- ----------------------------
DROP TABLE IF EXISTS `raw_assets`;
CREATE TABLE `raw_assets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `source_type` varchar(50) DEFAULT NULL COMMENT '数据来源类型',
  `source_name` varchar(100) DEFAULT NULL COMMENT '来源名称',
  `external_id` varchar(100) DEFAULT NULL COMMENT '外部ID',
  `payload` json DEFAULT NULL COMMENT '原始数据(JSON)',
  `checksum` varchar(64) DEFAULT NULL COMMENT '校验和',
  `import_batch_id` varchar(50) DEFAULT NULL COMMENT '导入批次标识',
  `priority` int DEFAULT '0' COMMENT '处理优先级',
  `asset_metadata` json DEFAULT NULL COMMENT '资产元数据(JSON)',
  `tags` json DEFAULT NULL COMMENT '标签(JSON)',
  `processing_config` json DEFAULT NULL COMMENT '处理配置(JSON)',
  `imported_at` datetime(3) DEFAULT NULL COMMENT '导入时间',
  `normalize_status` varchar(20) DEFAULT 'pending' COMMENT '规范化状态',
  `normalize_error` text COMMENT '规范化失败原因',
  PRIMARY KEY (`id`),
  KEY `idx_raw_assets_external_id` (`external_id`),
  KEY `idx_raw_assets_checksum` (`checksum`),
  KEY `idx_raw_assets_import_batch_id` (`import_batch_id`),
  KEY `idx_raw_assets_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='原始导入记录表';

-- ----------------------------
-- Table structure for raw_asset_networks
-- ----------------------------
DROP TABLE IF EXISTS `raw_asset_networks`;
CREATE TABLE `raw_asset_networks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `network` varchar(50) NOT NULL COMMENT '网段',
  `name` varchar(100) DEFAULT NULL COMMENT '资产名称',
  `description` varchar(255) DEFAULT NULL COMMENT '描述',
  `exclude_ip` text COMMENT '排除的IP',
  `location` varchar(100) DEFAULT NULL COMMENT '地理位置',
  `security_zone` varchar(50) DEFAULT NULL COMMENT '安全区域',
  `network_type` varchar(20) DEFAULT NULL COMMENT '网络类型',
  `priority` int DEFAULT '0' COMMENT '调度优先级',
  `tags` json DEFAULT NULL COMMENT '标签(JSON)',
  `source_type` varchar(50) DEFAULT NULL COMMENT '数据来源类型',
  `source_identifier` varchar(100) DEFAULT NULL COMMENT '来源标识',
  `status` varchar(20) DEFAULT 'pending' COMMENT '状态',
  `note` text COMMENT '备注',
  `created_by` varchar(100) DEFAULT NULL COMMENT '创建人',
  PRIMARY KEY (`id`),
  KEY `idx_raw_asset_networks_network` (`network`),
  KEY `idx_raw_asset_networks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='待确认网段表';

-- ----------------------------
-- Table structure for projects
-- ----------------------------
DROP TABLE IF EXISTS `projects`;
CREATE TABLE `projects` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(100) NOT NULL COMMENT '项目唯一标识名',
  `display_name` varchar(200) DEFAULT NULL COMMENT '显示名称',
  `description` text COMMENT '项目描述',
  `target_scope` text COMMENT '目标范围(CIDR/Domain列表)',
  `status` varchar(20) DEFAULT 'idle' COMMENT '运行状态',
  `enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用',
  `schedule_type` varchar(20) DEFAULT 'immediate' COMMENT '调度类型',
  `cron_expr` varchar(100) DEFAULT NULL COMMENT 'Cron表达式',
  `exec_mode` varchar(20) DEFAULT 'sequential' COMMENT '工作流执行模式',
  `notify_config` json DEFAULT NULL COMMENT '通知配置聚合(JSON)',
  `export_config` json DEFAULT NULL COMMENT '结果导出配置(JSON)',
  `extended_data` json DEFAULT NULL COMMENT '扩展数据(JSON)',
  `tags` json DEFAULT NULL COMMENT '标签列表(JSON)',
  `last_exec_time` datetime(3) DEFAULT NULL COMMENT '最后一次执行开始时间',
  `last_exec_id` varchar(100) DEFAULT NULL COMMENT '最后一次执行的任务ID',
  `created_by` bigint unsigned DEFAULT NULL COMMENT '创建者UserID',
  `updated_by` bigint unsigned DEFAULT NULL COMMENT '更新者UserID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_projects_name` (`name`),
  KEY `idx_projects_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='项目主表';

-- ----------------------------
-- Table structure for workflows
-- ----------------------------
DROP TABLE IF EXISTS `workflows`;
CREATE TABLE `workflows` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(100) NOT NULL COMMENT '工作流唯一标识名',
  `display_name` varchar(200) DEFAULT NULL COMMENT '显示名称',
  `version` varchar(20) DEFAULT '1.0.0' COMMENT '版本号',
  `description` text COMMENT '描述',
  `enabled` tinyint(1) DEFAULT '1' COMMENT '启用状态',
  `exec_mode` varchar(20) DEFAULT 'sequential' COMMENT '阶段执行模式',
  `global_vars` json DEFAULT NULL COMMENT '全局变量定义(JSON)',
  `policy_config` json DEFAULT NULL COMMENT '执行策略配置(JSON)',
  `tags` json DEFAULT NULL COMMENT '标签列表(JSON)',
  `created_by` bigint unsigned DEFAULT NULL COMMENT '创建者ID',
  `updated_by` bigint unsigned DEFAULT NULL COMMENT '更新者ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_workflows_name` (`name`),
  KEY `idx_workflows_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='工作流定义表';

-- ----------------------------
-- Table structure for project_workflows
-- ----------------------------
DROP TABLE IF EXISTS `project_workflows`;
CREATE TABLE `project_workflows` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `project_id` bigint unsigned NOT NULL COMMENT '项目ID',
  `workflow_id` bigint unsigned NOT NULL COMMENT '工作流ID',
  `sort_order` int DEFAULT '0' COMMENT '执行顺序',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_project_workflow` (`project_id`,`workflow_id`),
  KEY `idx_project_workflows_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='项目与工作流关联表';

-- ----------------------------
-- Table structure for workflow_stats
-- ----------------------------
DROP TABLE IF EXISTS `workflow_stats`;
CREATE TABLE `workflow_stats` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `workflow_id` bigint unsigned NOT NULL COMMENT '工作流ID',
  `total_execs` int DEFAULT '0' COMMENT '总执行次数',
  `success_execs` int DEFAULT '0' COMMENT '成功次数',
  `failed_execs` int DEFAULT '0' COMMENT '失败次数',
  `avg_duration_ms` int DEFAULT '0' COMMENT '平均执行耗时(ms)',
  `last_exec_id` varchar(100) DEFAULT NULL COMMENT '最后一次执行ID',
  `last_exec_status` varchar(20) DEFAULT NULL COMMENT '最后一次执行状态',
  `last_exec_time` datetime(3) DEFAULT NULL COMMENT '最后一次执行时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_workflow_stats_workflow_id` (`workflow_id`),
  KEY `idx_workflow_stats_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='工作流运行时统计表';

-- ----------------------------
-- Table structure for scan_stages
-- ----------------------------
DROP TABLE IF EXISTS `scan_stages`;
CREATE TABLE `scan_stages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `workflow_id` bigint unsigned NOT NULL COMMENT '所属工作流ID',
  `stage_order` int DEFAULT '0' COMMENT '阶段顺序',
  `stage_name` varchar(100) DEFAULT NULL COMMENT '阶段名称',
  `stage_type` varchar(50) DEFAULT NULL COMMENT '阶段类型枚举',
  `tool_name` varchar(100) DEFAULT NULL COMMENT '使用的扫描工具名称',
  `tool_params` text COMMENT '扫描工具参数',
  `target_policy` json DEFAULT NULL COMMENT '目标策略配置(JSON)',
  `execution_policy` json DEFAULT NULL COMMENT '执行策略配置(JSON)',
  `performance_settings` json DEFAULT NULL COMMENT '性能设置配置(JSON)',
  `output_config` json DEFAULT NULL COMMENT '输出配置(JSON)',
  `notify_config` json DEFAULT NULL COMMENT '通知配置(JSON)',
  `enabled` tinyint(1) DEFAULT '1' COMMENT '阶段是否启用',
  PRIMARY KEY (`id`),
  KEY `idx_scan_stages_workflow_id` (`workflow_id`),
  KEY `idx_scan_stages_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='扫描阶段定义表';

-- ----------------------------
-- Table structure for stage_results
-- ----------------------------
DROP TABLE IF EXISTS `stage_results`;
CREATE TABLE `stage_results` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `project_id` bigint unsigned NOT NULL COMMENT '所属项目ID',
  `workflow_id` bigint unsigned NOT NULL COMMENT '所属工作流ID',
  `stage_id` bigint unsigned NOT NULL COMMENT '阶段ID',
  `agent_id` varchar(100) NOT NULL COMMENT '执行扫描的AgentID',
  `result_type` varchar(50) DEFAULT NULL COMMENT '结果类型枚举',
  `target_type` varchar(50) DEFAULT NULL COMMENT '目标类型',
  `target_value` varchar(2048) DEFAULT NULL COMMENT '目标值',
  `attributes` json DEFAULT NULL COMMENT '结构化属性(JSON)',
  `evidence` json DEFAULT NULL COMMENT '原始证据(JSON)',
  `produced_at` datetime(3) DEFAULT NULL COMMENT '产生时间',
  `producer` varchar(100) DEFAULT NULL COMMENT '工具标识与版本',
  `output_config_hash` varchar(64) DEFAULT NULL COMMENT '输出配置指纹',
  `output_actions` json DEFAULT NULL COMMENT '实际执行的轻量动作摘要(JSON)',
  PRIMARY KEY (`id`),
  KEY `idx_stage_results_project_id` (`project_id`),
  KEY `idx_stage_results_workflow_id` (`workflow_id`),
  KEY `idx_stage_results_stage_id` (`stage_id`),
  KEY `idx_stage_results_agent_id` (`agent_id`),
  KEY `idx_stage_results_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='扫描结果表';

-- ----------------------------
-- Table structure for scan_tool_templates
-- ----------------------------
DROP TABLE IF EXISTS `scan_tool_templates`;
CREATE TABLE `scan_tool_templates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(100) NOT NULL COMMENT '模板名称',
  `tool_name` varchar(100) NOT NULL COMMENT '所属工具名称',
  `tool_params` text COMMENT '工具命令行参数模板',
  `description` varchar(255) DEFAULT NULL COMMENT '模板描述',
  `category` varchar(50) DEFAULT NULL COMMENT '分类标签',
  `is_public` tinyint(1) DEFAULT '0' COMMENT '是否公开',
  `created_by` varchar(50) DEFAULT NULL COMMENT '创建人',
  PRIMARY KEY (`id`),
  KEY `idx_scan_tool_templates_tool_name` (`tool_name`),
  KEY `idx_scan_tool_templates_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='扫描工具参数模板表';

-- ----------------------------
-- Table structure for agent_tasks
-- ----------------------------
DROP TABLE IF EXISTS `agent_tasks`;
CREATE TABLE `agent_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `task_id` varchar(100) NOT NULL COMMENT '任务唯一标识ID',
  `project_id` bigint unsigned NOT NULL COMMENT '所属项目ID',
  `workflow_id` bigint unsigned NOT NULL COMMENT '所属工作流ID',
  `stage_id` bigint unsigned NOT NULL COMMENT '所属阶段ID',
  `agent_id` varchar(100) DEFAULT NULL COMMENT '执行Agent的ID',
  `status` varchar(20) DEFAULT 'pending' COMMENT '任务状态(pending/assigned/running/completed/failed)',
  `priority` int DEFAULT '0' COMMENT '任务优先级',
  `task_type` varchar(20) DEFAULT 'tool' COMMENT '任务类型',
  `tool_name` varchar(100) DEFAULT NULL COMMENT '工具名称',
  `tool_params` text COMMENT '工具参数',
  `input_target` json DEFAULT NULL COMMENT '输入目标(JSON)',
  `required_tags` json DEFAULT NULL COMMENT '执行所需标签(JSON)',
  `output_result` json DEFAULT NULL COMMENT '输出结果摘要(JSON)',
  `error_msg` text COMMENT '错误信息',
  `assigned_at` datetime(3) DEFAULT NULL COMMENT '分配时间',
  `started_at` datetime(3) DEFAULT NULL COMMENT '开始执行时间',
  `finished_at` datetime(3) DEFAULT NULL COMMENT '完成时间',
  `timeout` int DEFAULT '3600' COMMENT '超时时间(秒)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_agent_tasks_task_id` (`task_id`),
  KEY `idx_agent_tasks_project_id` (`project_id`),
  KEY `idx_agent_tasks_workflow_id` (`workflow_id`),
  KEY `idx_agent_tasks_stage_id` (`stage_id`),
  KEY `idx_agent_tasks_agent_id` (`agent_id`),
  KEY `idx_agent_tasks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent任务表';

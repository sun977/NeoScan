-- Tag System Schema
-- Created at 2025-12-16

-- Table: sys_tags
DROP TABLE IF EXISTS `sys_tags`;
CREATE TABLE IF NOT EXISTS `sys_tags` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `name` varchar(100) NOT NULL COMMENT '标签名称',
  `parent_id` bigint(20) unsigned DEFAULT '0' COMMENT '父标签ID',
  `path` varchar(1000) DEFAULT NULL COMMENT '物理路径',
  `level` int(11) DEFAULT '0' COMMENT '层级深度',
  `color` varchar(7) DEFAULT NULL COMMENT '标签颜色',
  `category` varchar(50) DEFAULT NULL COMMENT '业务分类',
  `description` varchar(255) DEFAULT NULL COMMENT '描述',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_parent_name` (`parent_id`,`name`),
  KEY `idx_sys_tags_path` (`path`),
  KEY `idx_sys_tags_category` (`category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='标签定义表';

-- Table: sys_match_rules
DROP TABLE IF EXISTS `sys_match_rules`;
CREATE TABLE IF NOT EXISTS `sys_match_rules` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `tag_id` bigint(20) unsigned NOT NULL COMMENT '关联标签ID',
  `name` varchar(100) DEFAULT NULL COMMENT '规则名称',
  `entity_type` varchar(50) NOT NULL COMMENT '实体类型',
  `priority` int(11) DEFAULT '0' COMMENT '优先级',
  `rule_json` text NOT NULL COMMENT '匹配规则JSON',
  `is_enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用',
  PRIMARY KEY (`id`),
  KEY `idx_sys_match_rules_tag_id` (`tag_id`),
  KEY `idx_sys_match_rules_entity_type` (`entity_type`),
  KEY `idx_sys_match_rules_is_enabled` (`is_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='自动打标规则表';

-- Table: sys_entity_tags
DROP TABLE IF EXISTS `sys_entity_tags`;
CREATE TABLE IF NOT EXISTS `sys_entity_tags` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `entity_type` varchar(50) NOT NULL COMMENT '实体类型',
  `entity_id` varchar(100) NOT NULL COMMENT '实体ID',
  `tag_id` bigint(20) unsigned NOT NULL COMMENT '标签ID',
  `source` varchar(50) DEFAULT 'manual' COMMENT '来源',
  `rule_id` bigint(20) unsigned DEFAULT '0' COMMENT '命中规则ID',
  `created_at` bigint(20) DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_entity` (`entity_type`,`entity_id`),
  KEY `idx_sys_entity_tags_tag_id` (`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='实体-标签关联表';

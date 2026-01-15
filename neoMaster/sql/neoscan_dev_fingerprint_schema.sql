-- 数据库初始化脚本: 指纹识别模块
-- 对应 Go 模型: internal/model/asset/asset_cpe.go, internal/model/asset/asset_finger.go
-- 日期: 2026-01-07

USE neoscan_dev;

-- -----------------------------------------------------
-- Table structure for asset_cpe
-- CPE指纹表: 存储 Nmap 探针和正则映射规则
-- -----------------------------------------------------
DROP TABLE IF EXISTS `asset_cpe`;
CREATE TABLE `asset_cpe` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT '删除时间(软删除)',
  
  `name` varchar(255) DEFAULT '' COMMENT '指纹名称',
  `probe` varchar(255) DEFAULT '' COMMENT 'Nmap Probe 名称 (e.g. NULL, GenericLines)',
  `match_str` varchar(500) NOT NULL COMMENT '正则表达式',
  `vendor` varchar(255) DEFAULT '' COMMENT 'Vendor',
  `product` varchar(255) DEFAULT '' COMMENT 'Product',
  `version` varchar(255) DEFAULT '' COMMENT 'Version (或 Regex 占位符 $1)',
  `update` varchar(255) DEFAULT '' COMMENT 'Update',
  `edition` varchar(255) DEFAULT '' COMMENT 'Edition',
  `language` varchar(255) DEFAULT '' COMMENT 'Language',
  `part` varchar(1) DEFAULT 'a' COMMENT 'CPE类型 (a: Application, o: OS, h: Hardware)',
  `cpe` varchar(500) DEFAULT '' COMMENT '完整 CPE 模板',
  `enabled` boolean DEFAULT true COMMENT '是否启用',
  `source` varchar(20) DEFAULT 'system' COMMENT '来源(system/custom)',
  
  PRIMARY KEY (`id`),
  KEY `idx_asset_cpe_deleted_at` (`deleted_at`),
  KEY `idx_asset_cpe_product` (`product`),
  KEY `idx_asset_cpe_vendor` (`vendor`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='CPE指纹规则表';

-- -----------------------------------------------------
-- Table structure for asset_finger
-- Web指纹表: 存储 HTTP 特征指纹规则
-- -----------------------------------------------------
DROP TABLE IF EXISTS `asset_finger`;
CREATE TABLE `asset_finger` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT '删除时间(软删除)',
  
  `name` varchar(255) DEFAULT '' COMMENT '指纹名称',
  `status_code` varchar(50) DEFAULT '' COMMENT 'HTTP状态码',
  `url` varchar(500) DEFAULT '' COMMENT 'URL路径',
  `title` varchar(255) DEFAULT '' COMMENT '网页标题',
  `subtitle` varchar(255) DEFAULT '' COMMENT '网页副标题',
  `footer` varchar(255) DEFAULT '' COMMENT '网页页脚',
  `header` varchar(255) DEFAULT '' COMMENT 'HTTP响应头',
  `response` varchar(1000) DEFAULT '' COMMENT 'HTTP响应内容',
  `server` varchar(500) DEFAULT '' COMMENT 'Server头',
  `x_powered_by` varchar(255) DEFAULT '' COMMENT 'X-Powered-By头',
  `body` varchar(255) DEFAULT '' COMMENT 'HTTP响应体',
  `match` varchar(255) DEFAULT '' COMMENT '匹配模式(如正则)',
  `enabled` boolean DEFAULT true COMMENT '是否启用',
  `source` varchar(20) DEFAULT 'system' COMMENT '来源(system/custom)',
  
  PRIMARY KEY (`id`),
  KEY `idx_asset_finger_deleted_at` (`deleted_at`),
  KEY `idx_asset_finger_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Web指纹规则表';

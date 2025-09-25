-- NeoScan 测试数据库建表SQL脚本
-- 数据库: neoscan_test
-- 版本: MySQL 8.0
-- 生成时间: 2025-09-01
-- 说明: 用于测试环境的数据库表结构

-- 使用测试数据库
USE `neoscan_test`;

-- 1. 用户表 (users)
CREATE TABLE `users` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '用户唯一标识ID，主键自增',
    `username` varchar(50) NOT NULL COMMENT '用户名，唯一索引，3-50字符',
    `email` varchar(100) NOT NULL COMMENT '邮箱地址，唯一索引，必须符合邮箱格式',
    `password` varchar(255) NOT NULL COMMENT '用户密码，加密存储',
    `password_v` bigint NOT NULL DEFAULT '1' COMMENT '密码版本号,用于使旧token失效',
    `nickname` varchar(50) DEFAULT NULL COMMENT '用户昵称，最大50字符',
    `avatar` varchar(255) DEFAULT NULL COMMENT '用户头像URL，最大255字符',
    `phone` varchar(20) DEFAULT NULL COMMENT '手机号码，最大20字符',
    `socket_id` varchar(100) DEFAULT NULL COMMENT 'WebSocket连接ID',
    `remark` varchar(500) DEFAULT NULL COMMENT '管理员备注',
    `status` tinyint NOT NULL DEFAULT '1' COMMENT '用户状态:0-禁用,1-启用',
    `last_login_at` datetime DEFAULT NULL COMMENT '最后登录时间',
    `last_login_ip` varchar(45) DEFAULT NULL COMMENT '最后登录IP',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_users_username` (`username`),
    UNIQUE KEY `idx_users_email` (`email`),
    KEY `idx_users_deleted_at` (`deleted_at`),
    KEY `idx_users_status` (`status`),
    KEY `idx_users_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 2. 角色表 (roles)
CREATE TABLE `roles` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '角色唯一标识ID，主键自增',
    `name` varchar(50) NOT NULL COMMENT '角色名称，唯一索引，必填',
    `display_name` varchar(100) DEFAULT NULL COMMENT '角色显示名称，用于前端展示',
    `description` varchar(255) DEFAULT NULL COMMENT '角色描述信息，最大255字符',
    `status` tinyint NOT NULL DEFAULT '1' COMMENT '角色状态:0-禁用,1-启用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_roles_name` (`name`),
    KEY `idx_roles_deleted_at` (`deleted_at`),
    KEY `idx_roles_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色表';

-- 3. 权限表 (permissions)
CREATE TABLE `permissions` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '权限唯一标识ID，主键自增',
    `name` varchar(50) NOT NULL COMMENT '权限名称，唯一索引，必填',
    `display_name` varchar(100) DEFAULT NULL COMMENT '权限显示名称，用于前端展示',
    `description` varchar(255) DEFAULT NULL COMMENT '权限描述信息，最大255字符',
    `resource` varchar(100) DEFAULT NULL COMMENT '权限资源标识',
    `action` varchar(50) DEFAULT NULL COMMENT '权限操作类型',
    `status` tinyint NOT NULL DEFAULT '1' COMMENT '权限状态:0-禁用,1-启用',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_permissions_name` (`name`),
    KEY `idx_permissions_deleted_at` (`deleted_at`),
    KEY `idx_permissions_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='权限表';

-- 4. 用户角色关联表 (user_roles)
CREATE TABLE `user_roles` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '关联唯一标识ID，主键自增',
    `user_id` bigint unsigned NOT NULL COMMENT '用户ID，外键关联users表',
    `role_id` bigint unsigned NOT NULL COMMENT '角色ID，外键关联roles表',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_user_roles_user_role` (`user_id`,`role_id`),
    KEY `idx_user_roles_user_id` (`user_id`),
    KEY `idx_user_roles_role_id` (`role_id`),
    CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户角色关联表';

-- 5. 角色权限关联表 (role_permissions)
CREATE TABLE `role_permissions` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '关联唯一标识ID，主键自增',
    `role_id` bigint unsigned NOT NULL COMMENT '角色ID，外键关联roles表',
    `permission_id` bigint unsigned NOT NULL COMMENT '权限ID，外键关联permissions表',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_role_permissions_role_permission` (`role_id`,`permission_id`),
    KEY `idx_role_permissions_role_id` (`role_id`),
    KEY `idx_role_permissions_permission_id` (`permission_id`),
    CONSTRAINT `fk_role_permissions_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_role_permissions_permission` FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色权限关联表';
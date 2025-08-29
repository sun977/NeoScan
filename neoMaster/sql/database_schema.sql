-- NeoScan 数据库建表SQL脚本
-- 数据库: neoscan_dev
-- 版本: MySQL 8.0
-- 生成时间: 2025-08-29
-- 说明: 根据GORM模型定义生成的建表语句

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS `neoscan_dev` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `neoscan_dev`;

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
    `name` varchar(100) NOT NULL COMMENT '权限名称，唯一索引，必填',
    `display_name` varchar(100) DEFAULT NULL COMMENT '权限显示名称，用于前端展示',
    `description` varchar(255) DEFAULT NULL COMMENT '权限描述信息，最大255字符',
    `resource` varchar(100) DEFAULT NULL COMMENT '资源标识',
    `action` varchar(50) DEFAULT NULL COMMENT '操作标识',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_permissions_name` (`name`),
    KEY `idx_permissions_resource` (`resource`),
    KEY `idx_permissions_action` (`action`),
    KEY `idx_permissions_resource_action` (`resource`, `action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='权限表';

-- 4. 用户角色关联表 (user_roles)
CREATE TABLE `user_roles` (
    `user_id` bigint unsigned NOT NULL COMMENT '用户ID，联合主键',
    `role_id` bigint unsigned NOT NULL COMMENT '角色ID，联合主键',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '关联创建时间',
    PRIMARY KEY (`user_id`, `role_id`),
    KEY `idx_user_roles_user_id` (`user_id`),
    KEY `idx_user_roles_role_id` (`role_id`),
    CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户角色关联表';

-- 5. 角色权限关联表 (role_permissions)
CREATE TABLE `role_permissions` (
    `role_id` bigint unsigned NOT NULL COMMENT '角色ID，联合主键',
    `permission_id` bigint unsigned NOT NULL COMMENT '权限ID，联合主键',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '关联创建时间',
    PRIMARY KEY (`role_id`, `permission_id`),
    KEY `idx_role_permissions_role_id` (`role_id`),
    KEY `idx_role_permissions_permission_id` (`permission_id`),
    CONSTRAINT `fk_role_permissions_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_role_permissions_permission` FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色权限关联表';

-- 插入默认数据
-- 默认角色
INSERT INTO `roles` (`name`, `display_name`, `description`, `status`) VALUES
('admin', '系统管理员', '拥有系统所有权限的超级管理员', 1),
('user', '普通用户', '系统普通用户，拥有基础功能权限', 1),
('guest', '访客用户', '只读权限的访客用户', 1);

-- 默认权限
INSERT INTO `permissions` (`name`, `display_name`, `description`, `resource`, `action`) VALUES
('system:admin', '系统管理', '系统管理权限', 'system', 'admin'),
('user:create', '创建用户', '创建新用户的权限', 'user', 'create'),
('user:read', '查看用户', '查看用户信息的权限', 'user', 'read'),
('user:update', '更新用户', '更新用户信息的权限', 'user', 'update'),
('user:delete', '删除用户', '删除用户的权限', 'user', 'delete'),
('role:create', '创建角色', '创建新角色的权限', 'role', 'create'),
('role:read', '查看角色', '查看角色信息的权限', 'role', 'read'),
('role:update', '更新角色', '更新角色信息的权限', 'role', 'update'),
('role:delete', '删除角色', '删除角色的权限', 'role', 'delete'),
('permission:read', '查看权限', '查看权限信息的权限', 'permission', 'read');

-- 为管理员角色分配所有权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`) 
SELECT r.id, p.id 
FROM `roles` r, `permissions` p 
WHERE r.name = 'admin';

-- 为普通用户分配基础权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`) 
SELECT r.id, p.id 
FROM `roles` r, `permissions` p 
WHERE r.name = 'user' AND p.name IN ('user:read', 'role:read', 'permission:read');

-- 为访客用户分配只读权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`) 
SELECT r.id, p.id 
FROM `roles` r, `permissions` p 
WHERE r.name = 'guest' AND p.name IN ('user:read', 'role:read', 'permission:read');

-- 创建默认管理员用户（密码需要在应用中加密后更新）
INSERT INTO `users` (`username`, `email`, `password`, `nickname`, `status`) VALUES
('admin', 'admin@neoscan.com', 'temp_password_need_hash', '系统管理员', 1);

-- 为默认管理员分配管理员角色
INSERT INTO `user_roles` (`user_id`, `role_id`) 
SELECT u.id, r.id 
FROM `users` u, `roles` r 
WHERE u.username = 'admin' AND r.name = 'admin';

-- 创建性能优化索引
-- 用户表额外索引
CREATE INDEX `idx_users_last_login_at` ON `users` (`last_login_at`);
CREATE INDEX `idx_users_socket_id` ON `users` (`socket_id`);

-- 权限表复合索引优化
CREATE INDEX `idx_permissions_created_at` ON `permissions` (`created_at`);

-- 关联表时间索引
CREATE INDEX `idx_user_roles_created_at` ON `user_roles` (`created_at`);
CREATE INDEX `idx_role_permissions_created_at` ON `role_permissions` (`created_at`);

-- 显示建表完成信息
SELECT 'NeoScan数据库表结构创建完成！' as message;
SELECT 'Database: neoscan_dev' as database_info;
SELECT 'Tables created: users, roles, permissions, user_roles, role_permissions' as tables_info;
SELECT 'Default data inserted: admin role and user' as data_info;
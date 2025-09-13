-- NeoScan 测试数据插入脚本
-- 数据库: neoscan_dev
-- 版本: MySQL 8.0
-- 生成时间: 2025-09-13
-- 说明: 为测试目的向数据库中添加测试数据，每个表添加10条记录

-- 使用数据库
USE `neoscan_dev`;

-- 插入测试用户数据 (10条)
INSERT INTO `users` (`username`, `email`, `password`, `password_v`, `nickname`, `avatar`, `phone`, `remark`, `status`) VALUES
('testuser1', 'testuser1@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 1, '测试用户1', 'https://example.com/avatar1.jpg', '13800000001', '测试账户1', 1),
('testuser2', 'testuser2@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 1, '测试用户2', 'https://example.com/avatar2.jpg', '13800000002', '测试账户2', 1),
('testuser3', 'testuser3@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 1, '测试用户3', 'https://example.com/avatar3.jpg', '13800000003', '测试账户3', 1),
('testuser4', 'testuser4@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 1, '测试用户4', 'https://example.com/avatar4.jpg', '13800000004', '测试账户4', 1),
('testuser5', 'testuser5@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 1, '测试用户5', 'https://example.com/avatar5.jpg', '13800000005', '测试账户5', 1),
('testuser6', 'testuser6@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 2, '测试用户6', 'https://example.com/avatar6.jpg', '13800000006', '测试账户6', 1),
('testuser7', 'testuser7@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 1, '测试用户7', 'https://example.com/avatar7.jpg', '13800000007', '测试账户7', 0),
('testuser8', 'testuser8@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 1, '测试用户8', 'https://example.com/avatar8.jpg', '13800000008', '测试账户8', 1),
('testuser9', 'testuser9@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 3, '测试用户9', 'https://example.com/avatar9.jpg', '13800000009', '测试账户9', 1),
('testuser10', 'testuser10@example.com', '$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ', 1, '测试用户10', 'https://example.com/avatar10.jpg', '13800000010', '测试账户10', 1);

-- 插入测试角色数据 (10条)
INSERT INTO `roles` (`name`, `display_name`, `description`, `status`) VALUES
('developer', '开发者', '系统开发者角色', 1),
('tester', '测试员', '系统测试员角色', 1),
('operator', '操作员', '系统操作员角色', 1),
('auditor', '审计员', '系统审计员角色', 1),
('manager', '经理', '系统管理人员角色', 1),
('supervisor', '主管', '系统主管角色', 1),
('support', '技术支持', '技术支持人员角色', 1),
('analyst', '分析师', '数据分析员角色', 1),
('editor', '编辑', '内容编辑角色', 1),
('viewer', '查看员', '只读查看员角色', 0);

-- 插入测试权限数据 (10条)
INSERT INTO `permissions` (`name`, `display_name`, `description`, `resource`, `action`, `status`) VALUES
('user:import', '导入用户', '批量导入用户权限', 'user', 'import', 1),
('user:export', '导出用户', '导出用户数据权限', 'user', 'export', 1),
('role:assign', '分配角色', '为用户分配角色权限', 'role', 'assign', 1),
('permission:grant', '授予权限', '为角色授予权限权限', 'permission', 'grant', 1),
('log:read', '查看日志', '查看系统日志权限', 'log', 'read', 1),
('report:generate', '生成报告', '生成系统报告权限', 'report', 'generate', 1),
('config:manage', '配置管理', '系统配置管理权限', 'config', 'manage', 1),
('backup:create', '创建备份', '创建系统备份权限', 'backup', 'create', 1),
('backup:restore', '恢复备份', '恢复系统备份权限', 'backup', 'restore', 1),
('monitor:dashboard', '监控面板', '查看监控面板权限', 'monitor', 'dashboard', 1);

-- 插入测试用户角色关联数据 (10条)
-- 为测试用户分配不同角色
INSERT INTO `user_roles` (`user_id`, `role_id`) VALUES
(2, 4),  -- testuser1 -> auditor
(3, 5),  -- testuser2 -> manager
(4, 6),  -- testuser3 -> supervisor
(5, 7),  -- testuser4 -> support
(6, 8),  -- testuser5 -> analyst
(7, 9),  -- testuser6 -> editor
(8, 10), -- testuser7 -> viewer
(9, 2),  -- testuser8 -> tester
(10, 3), -- testuser9 -> operator
(11, 4); -- testuser10 -> auditor

-- 插入测试角色权限关联数据 (10条)
-- 为新角色分配相关权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`) VALUES
(4, 11),  -- auditor -> user:import
(5, 12),  -- manager -> user:export
(6, 13),  -- supervisor -> role:assign
(7, 14),  -- support -> permission:grant
(8, 15),  -- analyst -> log:read
(9, 16),  -- editor -> report:generate
(10, 17), -- viewer -> config:manage
(2, 18),  -- tester -> backup:create
(3, 19),  -- operator -> backup:restore
(4, 20);  -- auditor -> monitor:dashboard

-- 显示数据插入完成信息
SELECT 'NeoScan测试数据插入完成！' as message;
SELECT 'Database: neoscan_dev' as database_info;
SELECT 'Test data inserted: 10 users, 10 roles, 10 permissions, 10 user_roles, 10 role_permissions' as data_info;
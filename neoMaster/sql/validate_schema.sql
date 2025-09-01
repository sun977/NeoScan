-- 验证修改后的database_schema.sql语法
-- 这个文件用于快速检查主要的表结构修改

-- 检查permissions表结构
SELECT 'permissions表结构检查' as check_item;
DESC permissions;

-- 检查user_roles表结构
SELECT 'user_roles表结构检查' as check_item;
DESC user_roles;

-- 检查role_permissions表结构
SELECT 'role_permissions表结构检查' as check_item;
DESC role_permissions;

-- 检查索引
SELECT 'permissions表索引检查' as check_item;
SHOW INDEX FROM permissions;

SELECT 'user_roles表索引检查' as check_item;
SHOW INDEX FROM user_roles;

SELECT 'role_permissions表索引检查' as check_item;
SHOW INDEX FROM role_permissions;
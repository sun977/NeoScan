-- 修复agent_metrics表的唯一约束问题
-- 问题：agent_id字段被设置为唯一索引，导致每个Agent只能有一条性能指标记录
-- 解决：移除唯一约束，添加复合索引支持时间序列数据

USE `neoscan_dev`;

-- 1. 删除错误的唯一索引
ALTER TABLE `agent_metrics` DROP INDEX `idx_agent_metrics_agent_id`;

-- 2. 添加正确的复合索引，支持按Agent和时间查询
CREATE INDEX `idx_agent_metrics_agent_timestamp` ON `agent_metrics` (`agent_id`, `timestamp`);

-- 3. 添加按Agent查询的普通索引
CREATE INDEX `idx_agent_metrics_agent_id` ON `agent_metrics` (`agent_id`);

-- 4. 验证索引修改结果
SHOW INDEX FROM `agent_metrics` WHERE Key_name LIKE '%agent%';

SELECT 'agent_metrics表唯一约束修复完成！现在支持每个Agent存储多条性能指标历史记录' as message;
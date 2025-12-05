Resource Allocator (资源调度器)
职责: 管理 Agent 资源池，实现最优分配。
核心组件:
AgentSelector: 智能匹配。
基于 Capability (能力) 匹配: 只有安装了 Masscan 的 Agent 才能领 Masscan 任务。
基于 Tag (标签) 匹配: 只有 "Zone:Inside" 的 Agent 才能扫内网。
RateLimiter: 速率限制。
全局限速: 防止 Master 被大量心跳打挂。
目标限速: 防止把目标网段打挂。
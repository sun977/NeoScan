Policy Enforcer (策略执行器)
职责: 在任务下发前的最后一道防线，负责"安检"与合规。
核心组件:
WhitelistChecker: 强制阻断。检查目标是否命中 AssetWhitelist。
原则: 必须在 Master 端拦截，严禁将白名单目标下发给 Agent。
SkipLogicEvaluator: 动态跳过。执行 AssetSkipPolicy 逻辑。
场景: "如果是生产环境标签 && 当前时间是工作日 -> 跳过高风险扫描"。
ScopeValidator: 范围校验。确保扫描目标严格限制在 Project 定义的 TargetScope 内，防止意外扫描互联网。
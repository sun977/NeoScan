// Package ingestor 结果摄入模块
// 职责:
// 1. 接收 Agent 上报的 StageResult (HTTP)
// 2. 校验结果格式与签名 (ResultValidator)
// 3. 将结果推送到缓冲队列 (ResultQueue) 进行削峰填谷
// 4. 将原始证据 (Evidence) 归档到对象存储 (EvidenceArchiver)
//
// 架构定位:
// 位于 Orchestrator 域，是 Master 的"数据入口"，负责任务生命周期的闭环。
package ingestor

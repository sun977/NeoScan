// Package etl 资产清洗引擎 (Asset ETL Engine)
// 职责:
// 1. 消费 ResultQueue 中的 StageResult
// 2. 数据清洗与标准化 (Parser/Normalizer)
// 3. Web数据处理 (WebCrawlerDataHandler) - 处理截图、HTML等大体积数据
// 4. 资产合并与入库 (Merger) - 核心 Upsert 逻辑
//
// ResultIngestor -> Queue -> ETL Processor -> Database
// 架构定位:
// 位于 Asset 域，是资产数据的"生产车间"。
package etl

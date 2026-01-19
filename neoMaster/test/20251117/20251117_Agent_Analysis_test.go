/**
 * Agent高级统计与分析功能测试样例
 * 作者: AI Assistant
 * 日期: 2025-11-17
 * 说明: 覆盖四个聚合接口的基础功能，验证：
 * - 在线判定窗口与全量分布统计的口径
 * - 负载评分排序与TopN
 * - 性能聚合与Top列表
 * - 容量阈值与瓶颈统计、容量得分
 */
package agent_test

import (
	"context"
	"testing"
	"time"

	agentModel "neomaster/internal/model/agent"
	tagSystemModel "neomaster/internal/model/tag_system"
	agentRepository "neomaster/internal/repo/mysql/agent"
	agentService "neomaster/internal/service/agent"
)

// -------------------- 测试桩：假 TagService --------------------
type fakeTagService struct{}

func (s *fakeTagService) CreateTag(ctx context.Context, tag *tagSystemModel.SysTag) error { return nil }
func (s *fakeTagService) GetTag(ctx context.Context, id uint64) (*tagSystemModel.SysTag, error) {
	return nil, nil
}
func (s *fakeTagService) GetTagByName(ctx context.Context, name string) (*tagSystemModel.SysTag, error) {
	return nil, nil
}
func (s *fakeTagService) GetTagByNameAndParent(ctx context.Context, name string, parentID uint64) (*tagSystemModel.SysTag, error) {
	return nil, nil
}
func (s *fakeTagService) GetTagsByIDs(ctx context.Context, ids []uint64) ([]tagSystemModel.SysTag, error) {
	return nil, nil
}
func (s *fakeTagService) UpdateTag(ctx context.Context, tag *tagSystemModel.SysTag) error { return nil }
func (s *fakeTagService) MoveTag(ctx context.Context, id, targetParentID uint64) error    { return nil }
func (s *fakeTagService) DeleteTag(ctx context.Context, id uint64, force bool) error      { return nil }
func (s *fakeTagService) ListTags(ctx context.Context, req *tagSystemModel.ListTagsRequest) ([]tagSystemModel.SysTag, int64, error) {
	return nil, 0, nil
}
func (s *fakeTagService) CreateRule(ctx context.Context, rule *tagSystemModel.SysMatchRule) error {
	return nil
}
func (s *fakeTagService) UpdateRule(ctx context.Context, rule *tagSystemModel.SysMatchRule) error {
	return nil
}
func (s *fakeTagService) DeleteRule(ctx context.Context, id uint64) error { return nil }
func (s *fakeTagService) GetRule(ctx context.Context, id uint64) (*tagSystemModel.SysMatchRule, error) {
	return nil, nil
}
func (s *fakeTagService) ListRules(ctx context.Context, req *tagSystemModel.ListRulesRequest) ([]tagSystemModel.SysMatchRule, int64, error) {
	return nil, 0, nil
}
func (s *fakeTagService) ReloadMatchRules() error { return nil }
func (s *fakeTagService) AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error {
	return nil
}
func (s *fakeTagService) SubmitPropagationTask(ctx context.Context, ruleID uint64, action string) (string, error) {
	return "", nil
}
func (s *fakeTagService) SubmitEntityPropagationTask(ctx context.Context, entityType string, entityID uint64, tagIDs []uint64, action string) (string, error) {
	return "", nil
}
func (s *fakeTagService) SyncEntityTags(ctx context.Context, entityType string, entityID string, targetTagIDs []uint64, sourceScope string, ruleID uint64) error {
	return nil
}
func (s *fakeTagService) AddEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64, source string, ruleID uint64) error {
	return nil
}
func (s *fakeTagService) RemoveEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64) error {
	return nil
}
func (s *fakeTagService) GetEntityTags(ctx context.Context, entityType string, entityID string) ([]tagSystemModel.SysEntityTag, error) {
	return nil, nil
}
func (s *fakeTagService) GetEntityIDsByTagIDs(ctx context.Context, entityType string, tagIDs []uint64) ([]string, error) {
	// 简单模拟：如果包含 tagID=1，返回 A1, A3 (模拟之前的 G-1 分组)
	for _, id := range tagIDs {
		if id == 1 {
			return []string{"A1", "A3"}, nil
		}
	}
	return []string{}, nil
}

// -------------------- 测试桩：假 UpdateService --------------------
type fakeUpdateService struct{}

func (s *fakeUpdateService) GetSnapshotInfo(ctx context.Context, ruleType agentService.RuleType) (*agentService.RuleSnapshotInfo, error) {
	return nil, nil
}
func (s *fakeUpdateService) BuildSnapshot(ctx context.Context, ruleType agentService.RuleType) (*agentService.RuleSnapshot, error) {
	return nil, nil
}
func (s *fakeUpdateService) GetEncryptedSnapshot(ctx context.Context, ruleType agentService.RuleType) (*agentService.RuleSnapshot, error) {
	return nil, nil
}

// -------------------- 测试桩：假仓储 --------------------
// fakeRepo 实现 agentRepository.AgentRepository，用内存数据支撑分析逻辑
type fakeRepo struct {
	metrics []*agentModel.AgentMetrics // 全部快照（单快照模型）
}

// 构造测试数据：A1/A2/A3 三台
func newFakeRepoWithData(now time.Time) *fakeRepo {
	// 为了覆盖窗口在线判定：A3 设为 300s 前（离线），A1/A2 为窗口内
	m1 := &agentModel.AgentMetrics{ // A1: 高负载
		AgentID: "A1", WorkStatus: agentModel.AgentWorkStatusWorking, ScanType: string(agentModel.AgentScanTypeFullPortScan),
		CPUUsage: 90, MemoryUsage: 90, DiskUsage: 40,
		RunningTasks: 2, CompletedTasks: 10, FailedTasks: 1,
		NetworkBytesSent: 1000, NetworkBytesRecv: 500,
		Timestamp: now.Add(-60 * time.Second),
	}
	m2 := &agentModel.AgentMetrics{ // A2: 低负载
		AgentID: "A2", WorkStatus: agentModel.AgentWorkStatusIdle, ScanType: string(agentModel.AgentScanTypeFastPortScan),
		CPUUsage: 10, MemoryUsage: 10, DiskUsage: 20,
		RunningTasks: 0, CompletedTasks: 5, FailedTasks: 0,
		NetworkBytesSent: 200, NetworkBytesRecv: 100,
		Timestamp: now.Add(-10 * time.Second),
	}
	m3 := &agentModel.AgentMetrics{ // A3: 窗口外，模拟离线；性能用于全量聚合
		AgentID: "A3", WorkStatus: agentModel.AgentWorkStatusWorking, ScanType: string(agentModel.AgentScanTypeVulnScan),
		CPUUsage: 70, MemoryUsage: 30, DiskUsage: 60,
		RunningTasks: 5, CompletedTasks: 20, FailedTasks: 3,
		NetworkBytesSent: 800, NetworkBytesRecv: 200,
		Timestamp: now.Add(-300 * time.Second),
	}

	return &fakeRepo{
		metrics: []*agentModel.AgentMetrics{m1, m2, m3},
	}
}

// 仅实现分析所需的方法，其他方法为占位返回值满足接口
// ========== Metrics 分析支撑 ==========
func (f *fakeRepo) GetAllMetrics() ([]*agentModel.AgentMetrics, error) { return f.metrics, nil }
func (f *fakeRepo) GetMetricsSince(since time.Time) ([]*agentModel.AgentMetrics, error) {
	var res []*agentModel.AgentMetrics
	for _, m := range f.metrics {
		if m.Timestamp.After(since) {
			res = append(res, m)
		}
	}
	return res, nil
}
func (f *fakeRepo) GetMetricsByAgentIDs(agentIDs []string) ([]*agentModel.AgentMetrics, error) {
	var result []*agentModel.AgentMetrics
	idMap := make(map[string]bool)
	for _, id := range agentIDs {
		idMap[id] = true
	}
	for _, m := range f.metrics {
		if idMap[m.AgentID] {
			result = append(result, m)
		}
	}
	return result, nil
}
func (f *fakeRepo) GetMetricsByAgentIDsSince(agentIDs []string, since time.Time) ([]*agentModel.AgentMetrics, error) {
	var result []*agentModel.AgentMetrics
	idMap := make(map[string]bool)
	for _, id := range agentIDs {
		idMap[id] = true
	}
	for _, m := range f.metrics {
		if idMap[m.AgentID] && m.Timestamp.After(since) {
			result = append(result, m)
		}
	}
	return result, nil
}

// -------------------- 接口其余方法-占位实现 --------------------
// Agent 基础数据操作
func (f *fakeRepo) Create(*agentModel.Agent) error                              { return nil }
func (f *fakeRepo) GetByID(string) (*agentModel.Agent, error)                   { return nil, nil }
func (f *fakeRepo) GetByHostname(string) (*agentModel.Agent, error)             { return nil, nil }
func (f *fakeRepo) GetByHostnameAndPort(string, int) (*agentModel.Agent, error) { return nil, nil }
func (f *fakeRepo) Update(*agentModel.Agent) error                              { return nil }
func (f *fakeRepo) Delete(string) error                                         { return nil }
func (f *fakeRepo) GetList(int, int, *agentModel.AgentStatus, *string, []string, []string) ([]*agentModel.Agent, int64, error) {
	return []*agentModel.Agent{}, 0, nil
}
func (f *fakeRepo) GetByStatus(agentModel.AgentStatus) ([]*agentModel.Agent, error) {
	return []*agentModel.Agent{}, nil
}

// 状态和心跳
func (f *fakeRepo) UpdateStatus(string, agentModel.AgentStatus) error { return nil }
func (f *fakeRepo) UpdateLastHeartbeat(string) error                  { return nil }

// Metrics 管理
func (f *fakeRepo) CreateMetrics(*agentModel.AgentMetrics) error              { return nil }
func (f *fakeRepo) GetLatestMetrics(string) (*agentModel.AgentMetrics, error) { return nil, nil }
func (f *fakeRepo) UpdateAgentMetrics(string, *agentModel.AgentMetrics) error { return nil }
func (f *fakeRepo) GetMetricsList(int, int, *agentModel.AgentWorkStatus, *agentModel.AgentScanType, *string) ([]*agentModel.AgentMetrics, int64, error) {
	return []*agentModel.AgentMetrics{}, 0, nil
}

// 能力
func (f *fakeRepo) IsValidCapabilityId(string) bool       { return true }
func (f *fakeRepo) IsValidCapabilityByName(string) bool   { return true }
func (f *fakeRepo) AddCapability(string, string) error    { return nil }
func (f *fakeRepo) RemoveCapability(string, string) error { return nil }
func (f *fakeRepo) HasCapability(string, string) bool     { return false }
func (f *fakeRepo) GetCapabilities(string) []string       { return []string{} }

// 标签
func (f *fakeRepo) IsValidTagId(string) bool       { return true }
func (f *fakeRepo) IsValidTagByName(string) bool   { return true }
func (f *fakeRepo) AddTag(string, string) error    { return nil }
func (f *fakeRepo) RemoveTag(string, string) error { return nil }
func (f *fakeRepo) HasTag(string, string) bool     { return false }
func (f *fakeRepo) GetTags(string) []string        { return []string{} }
func (f *fakeRepo) GetAgentIDsByTagIDs(tagIDs []uint64) ([]string, error) {
	// 简单模拟：如果包含 tagID=1，返回 A1, A3 (模拟之前的 G-1 分组)
	for _, id := range tagIDs {
		if id == 1 {
			return []string{"A1", "A3"}, nil
		}
	}
	return []string{}, nil
}

// 任务支持
func (f *fakeRepo) IsValidTaskSupportId(string) bool                       { return true }
func (f *fakeRepo) IsValidTaskSupportByName(string) bool                   { return true }
func (f *fakeRepo) AddTaskSupport(string, string) error                    { return nil }
func (f *fakeRepo) RemoveTaskSupport(string, string) error                 { return nil }
func (f *fakeRepo) HasTaskSupport(string, string) bool                     { return false }
func (f *fakeRepo) GetTaskSupport(string) []string                         { return []string{} }
func (f *fakeRepo) GetTagIDsByTaskSupportNames([]string) ([]uint64, error) { return []uint64{}, nil }
func (f *fakeRepo) GetTagIDsByTaskSupportIDs([]string) ([]uint64, error)   { return []uint64{}, nil }
func (f *fakeRepo) GetAllScanTypes() ([]*agentModel.ScanType, error) {
	return []*agentModel.ScanType{}, nil
}
func (f *fakeRepo) UpdateScanType(*agentModel.ScanType) error { return nil }

var _ agentRepository.AgentRepository = (*fakeRepo)(nil)

// -------------------- 工具函数 --------------------
func almostEqual(a, b, eps float64) bool {
	if a > b {
		return a-b < eps
	}
	return b-a < eps
}

// -------------------- 用例1：统计 --------------------
func Test_GetAgentStatistics_WithAndWithoutTag(t *testing.T) {
	now := time.Now()
	repo := newFakeRepoWithData(now)
	svc := agentService.NewAgentMonitorService(repo, &fakeTagService{}, &fakeUpdateService{})

	// 无标签过滤：3台，总数=3；窗口180s内在线A1/A2=2，离线=1
	respAll, err := svc.GetAgentStatistics(180, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respAll.TotalAgents != 3 {
		t.Fatalf("TotalAgents=3 expected, got %d", respAll.TotalAgents)
	}
	if respAll.OnlineAgents != 2 {
		t.Fatalf("OnlineAgents=2 expected, got %d", respAll.OnlineAgents)
	}
	if respAll.OfflineAgents != 1 {
		t.Fatalf("OfflineAgents=1 expected, got %d", respAll.OfflineAgents)
	}
	// 聚合均值校验：CPUAvg=(90+10+70)/3=56.66...
	if !almostEqual(respAll.Performance.CPUAvg, (90+10+70)/3.0, 0.01) {
		t.Fatalf("CPUAvg mismatch, got %.2f", respAll.Performance.CPUAvg)
	}

	// 标签Tag-1（模拟G-1）：成员A1/A3，总数=2；在线=1（A1），离线=1（A3）
	respT1, err := svc.GetAgentStatistics(180, []uint64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respT1.TotalAgents != 2 {
		t.Fatalf("T1 TotalAgents=2 expected, got %d", respT1.TotalAgents)
	}
	if respT1.OnlineAgents != 1 {
		t.Fatalf("T1 OnlineAgents=1 expected, got %d", respT1.OnlineAgents)
	}
	if respT1.OfflineAgents != 1 {
		t.Fatalf("T1 OfflineAgents=1 expected, got %d", respT1.OfflineAgents)
	}
}

// -------------------- 用例2：负载均衡 --------------------
func Test_GetAgentLoadBalance_TagTopN(t *testing.T) {
	now := time.Now()
	repo := newFakeRepoWithData(now)
	svc := agentService.NewAgentMonitorService(repo, &fakeTagService{}, &fakeUpdateService{})

	// 负载评分 A1≈100, A3≈75, A2≈10；TopBusy取前2：A1,A3；TopIdle取前2(最小)：A2,A3
	resp, err := svc.GetAgentLoadBalance(310, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.TopBusyAgents) != 2 || resp.TopBusyAgents[0].AgentID != "A1" || resp.TopBusyAgents[1].AgentID != "A3" {
		t.Fatalf("TopBusyAgents expected [A1,A3], got: %+v", resp.TopBusyAgents)
	}
	if len(resp.TopIdleAgents) != 2 || resp.TopIdleAgents[0].AgentID != "A2" || resp.TopIdleAgents[1].AgentID != "A3" {
		t.Fatalf("TopIdleAgents expected [A2,A3], got: %+v", resp.TopIdleAgents)
	}

	// 标签Tag-1（A1,A3）：TopBusy/TopIdle长度应为2(若topN=2)，成员只来自A1/A3
	respT1, err := svc.GetAgentLoadBalance(310, 2, []uint64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(respT1.TopBusyAgents) != 2 || len(respT1.TopIdleAgents) != 2 {
		t.Fatalf("T1 Top lists length expected 2, got %d/%d", len(respT1.TopBusyAgents), len(respT1.TopIdleAgents))
	}
}

// -------------------- 用例3：性能分析 --------------------
func Test_GetAgentPerformanceAnalysis_TopLists(t *testing.T) {
	now := time.Now()
	repo := newFakeRepoWithData(now)
	svc := agentService.NewAgentMonitorService(repo, &fakeTagService{}, &fakeUpdateService{})

	resp, err := svc.GetAgentPerformanceAnalysis(310, 2, nil) // 窗口放宽至包含A3
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.TopCPU) != 2 || resp.TopCPU[0].AgentID != "A1" || resp.TopCPU[1].AgentID != "A3" {
		t.Fatalf("TopCPU expected [A1,A3], got: %+v", resp.TopCPU)
	}
	if len(resp.TopMemory) != 2 || resp.TopMemory[0].AgentID != "A1" {
		t.Fatalf("TopMemory first expected A1, got: %+v", resp.TopMemory)
	}
	if len(resp.TopNetwork) != 2 || resp.TopNetwork[0].AgentID != "A1" {
		t.Fatalf("TopNetwork first expected A1, got: %+v", resp.TopNetwork)
	}
	if len(resp.TopFailedTasks) != 2 || resp.TopFailedTasks[0].AgentID != "A3" {
		t.Fatalf("TopFailedTasks first expected A3, got: %+v", resp.TopFailedTasks)
	}

	// 标签Tag-1: A1, A3. 结果应仅包含这两个
	respTag, err := svc.GetAgentPerformanceAnalysis(310, 2, []uint64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(respTag.TopCPU) != 2 || respTag.TopCPU[0].AgentID != "A1" || respTag.TopCPU[1].AgentID != "A3" {
		t.Fatalf("Tag TopCPU expected [A1,A3], got: %+v", respTag.TopCPU)
	}
}

// -------------------- 用例4：容量分析 --------------------
func Test_GetAgentCapacityAnalysis_BottlenecksAndScore(t *testing.T) {
	now := time.Now()
	// 定制数据：让A3也处于窗口内，便于统计
	repo := newFakeRepoWithData(now)
	repo.metrics[2].Timestamp = now.Add(-30 * time.Second) // A3 调整为窗口内

	svc := agentService.NewAgentMonitorService(repo, &fakeTagService{}, &fakeUpdateService{})
	// 阈值：CPU/Mem/Disk=80
	// 标签Tag-1: A1(90/90/40), A3(70/30/60)
	resp, err := svc.GetAgentCapacityAnalysis(180, 80, 80, 80, []uint64{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 过载：A1(CPU/Mem>=80)，A3无过载 => Overloaded=1 或2? 逻辑：任一维度>=阈值即过载。
	// A1 过载；A3 各项<80，不过载。T1窗口内 online=2
	if resp.OnlineAgents != 2 {
		t.Fatalf("OnlineAgents=2 expected, got %d", resp.OnlineAgents)
	}
	if resp.Overloaded != 1 {
		t.Fatalf("Overloaded=1 expected, got %d", resp.Overloaded)
	}
	if resp.Bottlenecks["cpu"] < 1 && resp.Bottlenecks["memory"] < 1 {
		t.Fatalf("Bottlenecks should include cpu or memory for A1")
	}

	// 容量得分：按每台 max(cpu,mem,disk) 的余量均值
	// A1 max=90 -> headroom=10; A3 max=70 -> headroom=30; 平均=20
	if !almostEqual(resp.CapacityScore, 20.0, 0.01) {
		t.Fatalf("CapacityScore≈20 expected, got %.2f", resp.CapacityScore)
	}
}

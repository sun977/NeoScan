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
    "testing"
    "time"

    agentModel "neomaster/internal/model/agent"
    agentService "neomaster/internal/service/agent"
    agentRepository "neomaster/internal/repo/mysql/agent"
)

// -------------------- 测试桩：假仓储 --------------------
// fakeRepo 实现 agentRepository.AgentRepository，用内存数据支撑分析逻辑
type fakeRepo struct {
    metrics []*agentModel.AgentMetrics          // 全部快照（单快照模型）
    groups  map[string][]string                 // 分组成员映射：groupID -> []agentID
}

// 构造测试数据：A1/A2/A3 三台，分组 G-1:{A1,A3}, G-2:{A2}
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
        groups: map[string][]string{
            "G-1": {"A1", "A3"},
            "G-2": {"A2"},
        },
    }
}

// 仅实现分析所需的方法，其他方法为占位返回值满足接口
// ========== Metrics 分析支撑 ==========
func (f *fakeRepo) GetAllMetrics() ([]*agentModel.AgentMetrics, error) { return f.metrics, nil }
func (f *fakeRepo) GetMetricsSince(since time.Time) ([]*agentModel.AgentMetrics, error) {
    out := make([]*agentModel.AgentMetrics, 0)
    for _, m := range f.metrics { if m.Timestamp.After(since) || m.Timestamp.Equal(since) { out = append(out, m) } }
    return out, nil
}
func (f *fakeRepo) GetMetricsByAgentIDs(agentIDs []string) ([]*agentModel.AgentMetrics, error) {
    if len(agentIDs) == 0 { return []*agentModel.AgentMetrics{}, nil }
    set := make(map[string]struct{}, len(agentIDs))
    for _, id := range agentIDs { set[id] = struct{}{} }
    out := make([]*agentModel.AgentMetrics, 0)
    for _, m := range f.metrics { if _, ok := set[m.AgentID]; ok { out = append(out, m) } }
    return out, nil
}
func (f *fakeRepo) GetMetricsByAgentIDsSince(agentIDs []string, since time.Time) ([]*agentModel.AgentMetrics, error) {
    if len(agentIDs) == 0 { return []*agentModel.AgentMetrics{}, nil }
    set := make(map[string]struct{}, len(agentIDs))
    for _, id := range agentIDs { set[id] = struct{}{} }
    out := make([]*agentModel.AgentMetrics, 0)
    for _, m := range f.metrics {
        if _, ok := set[m.AgentID]; ok && (m.Timestamp.After(since) || m.Timestamp.Equal(since)) {
            out = append(out, m)
        }
    }
    return out, nil
}

// ========== 分组 ==========
func (f *fakeRepo) GetAgentIDsInGroup(groupID string) ([]string, error) { if ids, ok := f.groups[groupID]; ok { return ids, nil }; return []string{}, nil }

// -------------------- 接口其余方法-占位实现 --------------------
// Agent 基础数据操作
func (f *fakeRepo) Create(*agentModel.Agent) error { return nil }
func (f *fakeRepo) GetByID(string) (*agentModel.Agent, error) { return nil, nil }
func (f *fakeRepo) GetByHostname(string) (*agentModel.Agent, error) { return nil, nil }
func (f *fakeRepo) GetByHostnameAndPort(string, int) (*agentModel.Agent, error) { return nil, nil }
func (f *fakeRepo) Update(*agentModel.Agent) error { return nil }
func (f *fakeRepo) Delete(string) error { return nil }
func (f *fakeRepo) GetList(int, int, *agentModel.AgentStatus, *string, []string, []string) ([]*agentModel.Agent, int64, error) {
    return []*agentModel.Agent{}, 0, nil
}
func (f *fakeRepo) GetByStatus(agentModel.AgentStatus) ([]*agentModel.Agent, error) { return []*agentModel.Agent{}, nil }

// 状态和心跳
func (f *fakeRepo) UpdateStatus(string, agentModel.AgentStatus) error { return nil }
func (f *fakeRepo) UpdateLastHeartbeat(string) error { return nil }

// Metrics 管理
func (f *fakeRepo) CreateMetrics(*agentModel.AgentMetrics) error { return nil }
func (f *fakeRepo) GetLatestMetrics(string) (*agentModel.AgentMetrics, error) { return nil, nil }
func (f *fakeRepo) UpdateAgentMetrics(string, *agentModel.AgentMetrics) error { return nil }
func (f *fakeRepo) GetMetricsList(int, int, *agentModel.AgentWorkStatus, *agentModel.AgentScanType, *string) ([]*agentModel.AgentMetrics, int64, error) {
    return []*agentModel.AgentMetrics{}, 0, nil
}

// 能力
func (f *fakeRepo) IsValidCapabilityId(string) bool { return true }
func (f *fakeRepo) IsValidCapabilityByName(string) bool { return true }
func (f *fakeRepo) AddCapability(string, string) error { return nil }
func (f *fakeRepo) RemoveCapability(string, string) error { return nil }
func (f *fakeRepo) HasCapability(string, string) bool { return false }
func (f *fakeRepo) GetCapabilities(string) []string { return []string{} }

// 标签
func (f *fakeRepo) IsValidTagId(string) bool { return true }
func (f *fakeRepo) IsValidTagByName(string) bool { return true }
func (f *fakeRepo) AddTag(string, string) error { return nil }
func (f *fakeRepo) RemoveTag(string, string) error { return nil }
func (f *fakeRepo) HasTag(string, string) bool { return false }
func (f *fakeRepo) GetTags(string) []string { return []string{} }

// 分组（其余）
func (f *fakeRepo) IsValidGroupId(string) bool { return true }
func (f *fakeRepo) IsValidGroupName(string) bool { return true }
func (f *fakeRepo) IsAgentInGroup(string, string) bool { return false }
func (f *fakeRepo) GetGroupByGID(string) (*agentModel.AgentGroup, error) { return nil, nil }
func (f *fakeRepo) GetGroupList(int, int, []string, int, string) ([]*agentModel.AgentGroup, int64, error) { return []*agentModel.AgentGroup{}, 0, nil }
func (f *fakeRepo) CreateGroup(*agentModel.AgentGroup) error { return nil }
func (f *fakeRepo) UpdateGroup(string, *agentModel.AgentGroup) (*agentModel.AgentGroup, error) { return nil, nil }
func (f *fakeRepo) DeleteGroup(string) error { return nil }
func (f *fakeRepo) SetGroupStatus(string, int) error { return nil }
func (f *fakeRepo) AddAgentToGroup(string, string) (*agentModel.AgentGroupMember, error) { return nil, nil }
func (f *fakeRepo) RemoveAgentFromGroup(string, string) error { return nil }
func (f *fakeRepo) GetAgentsInGroup(int, int, string) ([]*agentModel.Agent, int64, error) { return []*agentModel.Agent{}, 0, nil }

var _ agentRepository.AgentRepository = (*fakeRepo)(nil)

// -------------------- 工具函数 --------------------
func almostEqual(a, b, eps float64) bool {
    if a > b { return a-b < eps }
    return b-a < eps
}

// -------------------- 用例1：统计 --------------------
func Test_GetAgentStatistics_WithAndWithoutGroup(t *testing.T) {
    now := time.Now()
    repo := newFakeRepoWithData(now)
    svc := agentService.NewAgentMonitorService(repo)

    // 无分组：3台，总数=3；窗口180s内在线A1/A2=2，离线=1
    respAll, err := svc.GetAgentStatistics("", 180)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if respAll.TotalAgents != 3 { t.Fatalf("TotalAgents=3 expected, got %d", respAll.TotalAgents) }
    if respAll.OnlineAgents != 2 { t.Fatalf("OnlineAgents=2 expected, got %d", respAll.OnlineAgents) }
    if respAll.OfflineAgents != 1 { t.Fatalf("OfflineAgents=1 expected, got %d", respAll.OfflineAgents) }
    // 聚合均值校验：CPUAvg=(90+10+70)/3=56.66...
    if !almostEqual(respAll.Performance.CPUAvg, (90+10+70)/3.0, 0.01) {
        t.Fatalf("CPUAvg mismatch, got %.2f", respAll.Performance.CPUAvg)
    }

    // 分组G-1：成员A1/A3，总数=2；在线=1（A1），离线=1（A3）
    respG1, err := svc.GetAgentStatistics("G-1", 180)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if respG1.TotalAgents != 2 { t.Fatalf("G1 TotalAgents=2 expected, got %d", respG1.TotalAgents) }
    if respG1.OnlineAgents != 1 { t.Fatalf("G1 OnlineAgents=1 expected, got %d", respG1.OnlineAgents) }
    if respG1.OfflineAgents != 1 { t.Fatalf("G1 OfflineAgents=1 expected, got %d", respG1.OfflineAgents) }
}

// -------------------- 用例2：负载均衡 --------------------
func Test_GetAgentLoadBalance_GroupTopN(t *testing.T) {
    now := time.Now()
    repo := newFakeRepoWithData(now)
    svc := agentService.NewAgentMonitorService(repo)

    // 负载评分 A1≈100, A3≈75, A2≈10；TopBusy取前2：A1,A3；TopIdle取前2(最小)：A2,A3
    resp, err := svc.GetAgentLoadBalance("", 300, 2)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if len(resp.TopBusyAgents) != 2 || resp.TopBusyAgents[0].AgentID != "A1" || resp.TopBusyAgents[1].AgentID != "A3" {
        t.Fatalf("TopBusyAgents expected [A1,A3], got: %+v", resp.TopBusyAgents)
    }
    if len(resp.TopIdleAgents) != 2 || resp.TopIdleAgents[0].AgentID != "A2" || resp.TopIdleAgents[1].AgentID != "A3" {
        t.Fatalf("TopIdleAgents expected [A2,A3], got: %+v", resp.TopIdleAgents)
    }

    // 分组G-1（A1,A3）：TopBusy/TopIdle长度应为2(若topN=2)，成员只来自A1/A3
    respG1, err := svc.GetAgentLoadBalance("G-1", 300, 2)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if len(respG1.TopBusyAgents) != 2 || len(respG1.TopIdleAgents) != 2 {
        t.Fatalf("G1 Top lists length expected 2, got %d/%d", len(respG1.TopBusyAgents), len(respG1.TopIdleAgents))
    }
}

// -------------------- 用例3：性能分析 --------------------
func Test_GetAgentPerformanceAnalysis_TopLists(t *testing.T) {
    now := time.Now()
    repo := newFakeRepoWithData(now)
    svc := agentService.NewAgentMonitorService(repo)

    resp, err := svc.GetAgentPerformanceAnalysis("", 300, 2) // 窗口放宽至包含A3
    if err != nil { t.Fatalf("unexpected error: %v", err) }
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
}

// -------------------- 用例4：容量分析 --------------------
func Test_GetAgentCapacityAnalysis_BottlenecksAndScore(t *testing.T) {
    now := time.Now()
    // 定制数据：让A3也处于窗口内，便于统计
    repo := newFakeRepoWithData(now)
    repo.metrics[2].Timestamp = now.Add(-30 * time.Second) // A3 调整为窗口内

    svc := agentService.NewAgentMonitorService(repo)
    // 阈值：CPU/Mem/Disk=80
    resp, err := svc.GetAgentCapacityAnalysis("G-1", 180, 80, 80, 80) // 分组G-1: A1(90/90/40), A3(70/30/60)
    if err != nil { t.Fatalf("unexpected error: %v", err) }

    // 过载：A1(CPU/Mem>=80)，A3无过载 => Overloaded=1 或2? 逻辑：任一维度>=阈值即过载。
    // A1 过载；A3 各项<80，不过载。G1窗口内 online=2
    if resp.OnlineAgents != 2 { t.Fatalf("OnlineAgents=2 expected, got %d", resp.OnlineAgents) }
    if resp.Overloaded != 1 { t.Fatalf("Overloaded=1 expected, got %d", resp.Overloaded) }
    if resp.Bottlenecks["cpu"] < 1 && resp.Bottlenecks["memory"] < 1 {
        t.Fatalf("Bottlenecks should include cpu or memory for A1")
    }

    // 容量得分：按每台 max(cpu,mem,disk) 的余量均值
    // A1 max=90 -> headroom=10; A3 max=70 -> headroom=30; 平均=20
    if !almostEqual(resp.CapacityScore, 20.0, 0.01) { t.Fatalf("CapacityScore≈20 expected, got %.2f", resp.CapacityScore) }
}
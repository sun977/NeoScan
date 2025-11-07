package agent_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"

    handlerAgent "neomaster/internal/handler/agent"
    agentModel "neomaster/internal/model/agent"
)

// fakeManagerService 是对 AgentManagerService 的桩实现，仅用于 Handler 层单元测试。
// 为了简化测试，我们只实现与标签相关的方法，其他方法使用空实现以满足接口。
type fakeManagerService struct {
    store map[string][]string // 模拟 agentID -> tags 的存储
}

// ---- 标签相关方法（测试关注点） ----
func (f *fakeManagerService) GetAgentTags(agentID string) ([]string, error) {
    if f.store == nil {
        f.store = make(map[string][]string)
    }
    tags, ok := f.store[agentID]
    if !ok {
        return []string{}, nil
    }
    // 返回副本，避免外部修改内部切片
    cp := make([]string, len(tags))
    copy(cp, tags)
    return cp, nil
}

func (f *fakeManagerService) UpdateAgentTags(agentID string, newTags []string) ([]string, []string, error) {
    if f.store == nil {
        f.store = make(map[string][]string)
    }
    old := f.store[agentID]
    // 去重保持输入顺序
    uniq := func(in []string) []string {
        m := make(map[string]struct{}, len(in))
        out := make([]string, 0, len(in))
        for _, t := range in {
            if t == "" {
                continue
            }
            if _, ok := m[t]; !ok {
                m[t] = struct{}{}
                out = append(out, t)
            }
        }
        return out
    }
    newU := uniq(newTags)
    f.store[agentID] = newU
    // 返回旧/新标签
    oldCp := make([]string, len(old))
    copy(oldCp, old)
    newCp := make([]string, len(newU))
    copy(newCp, newU)
    return oldCp, newCp, nil
}

// ---- 其他方法（空实现以满足接口编译） ----
func (f *fakeManagerService) RegisterAgent(req *agentModel.RegisterAgentRequest) (*agentModel.RegisterAgentResponse, error) {
    return nil, nil
}
func (f *fakeManagerService) GetAgentList(req *agentModel.GetAgentListRequest) (*agentModel.GetAgentListResponse, error) {
    return nil, nil
}
func (f *fakeManagerService) GetAgentInfo(agentID string) (*agentModel.AgentInfo, error) {
    return nil, nil
}
func (f *fakeManagerService) UpdateAgentStatus(agentID string, status agentModel.AgentStatus) error { return nil }
func (f *fakeManagerService) DeleteAgent(agentID string) error { return nil }
func (f *fakeManagerService) CreateAgentGroup(req *agentModel.AgentGroupCreateRequest) (*agentModel.AgentGroupResponse, error) {
    return nil, nil
}
func (f *fakeManagerService) GetAgentGroups() ([]*agentModel.AgentGroupResponse, error) { return nil, nil }
func (f *fakeManagerService) GetAgentGroup(groupID string) (*agentModel.AgentGroupResponse, error) { return nil, nil }
func (f *fakeManagerService) UpdateAgentGroup(groupID string, req *agentModel.AgentGroupCreateRequest) error { return nil }
func (f *fakeManagerService) DeleteAgentGroup(groupID string) error { return nil }
func (f *fakeManagerService) AddAgentToGroup(req *agentModel.AgentGroupMemberRequest) error { return nil }
func (f *fakeManagerService) RemoveAgentFromGroup(req *agentModel.AgentGroupMemberRequest) error { return nil }
func (f *fakeManagerService) GetGroupMembers(groupID string) ([]*agentModel.AgentInfo, error) { return nil, nil }
func (f *fakeManagerService) IsValidTagId(tag string) bool { return true }
func (f *fakeManagerService) IsValidTagByName(tag string) bool { return true }
func (f *fakeManagerService) AddAgentTag(req *agentModel.AgentTagRequest) error { return nil }
func (f *fakeManagerService) RemoveAgentTag(req *agentModel.AgentTagRequest) error { return nil }
func (f *fakeManagerService) IsValidCapabilityId(capability string) bool { return true }
func (f *fakeManagerService) IsValidCapabilityByName(capability string) bool { return true }
func (f *fakeManagerService) AddAgentCapability(req *agentModel.AgentCapabilityRequest) error { return nil }
func (f *fakeManagerService) RemoveAgentCapability(req *agentModel.AgentCapabilityRequest) error { return nil }
func (f *fakeManagerService) GetAgentCapabilities(agentID string) ([]string, error) { return nil, nil }

// updateTagsReq 是测试专用的请求体结构。
type updateTagsReq struct {
    Tags []string `json:"tags"`
}

// updateTagsResp 用于解析 Handler 返回的响应结构（只关心 Data 字段的内容）。
type updateTagsResp struct {
    Code    int    `json:"code"`
    Status  string `json:"status"`
    Message string `json:"message"`
    Data    struct {
        AgentID string   `json:"agent_id"`
        OldTags []string `json:"old_tags"`
        NewTags []string `json:"new_tags"`
    } `json:"data"`
    Error string `json:"error"`
}

// TestUpdateAgentTagsHandler_Success 验证成功路径：入参合法、服务层返回旧/新标签。
func TestUpdateAgentTagsHandler_Success(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // 构造伪服务和初始标签
    fakeSvc := &fakeManagerService{store: map[string][]string{"agent-1": {"web", "db"}}}
    h := handlerAgent.NewAgentHandler(fakeSvc, nil, nil, nil)

    rr := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(rr)

    body := updateTagsReq{Tags: []string{"db", "cache"}}
    bs, _ := json.Marshal(body)
    req, _ := http.NewRequest("PUT", "/api/v1/agents/agent-1/tags", bytes.NewBuffer(bs))
    req.Header.Set("Content-Type", "application/json")

    c.Request = req
    c.Params = gin.Params{{Key: "id", Value: "agent-1"}}

    h.UpdateAgentTags(c)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, rr.Code, rr.Body.String())
    }

    var resp updateTagsResp
    if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
        t.Fatalf("unmarshal response failed: %v, body=%s", err, rr.Body.String())
    }

    // 断言：agent_id、旧/新标签
    if resp.Data.AgentID != "agent-1" {
        t.Errorf("expected agent_id 'agent-1', got %s", resp.Data.AgentID)
    }
    // 旧标签应为 [web, db]
    wantOld := []string{"web", "db"}
    if len(resp.Data.OldTags) != len(wantOld) {
        t.Fatalf("unexpected old_tags length: want=%d got=%d", len(wantOld), len(resp.Data.OldTags))
    }
    for i := range wantOld {
        if resp.Data.OldTags[i] != wantOld[i] {
            t.Errorf("old_tags mismatch at %d: want=%s got=%s", i, wantOld[i], resp.Data.OldTags[i])
        }
    }
    // 新标签应为 [db, cache]
    wantNew := []string{"db", "cache"}
    if len(resp.Data.NewTags) != len(wantNew) {
        t.Fatalf("unexpected new_tags length: want=%d got=%d", len(wantNew), len(resp.Data.NewTags))
    }
    for i := range wantNew {
        if resp.Data.NewTags[i] != wantNew[i] {
            t.Errorf("new_tags mismatch at %d: want=%s got=%s", i, wantNew[i], resp.Data.NewTags[i])
        }
    }
}

// TestUpdateAgentTagsHandler_MissingAgentID 验证缺少路径参数 id 时返回 400。
func TestUpdateAgentTagsHandler_MissingAgentID(t *testing.T) {
    gin.SetMode(gin.TestMode)
    fakeSvc := &fakeManagerService{store: map[string][]string{}}
    h := handlerAgent.NewAgentHandler(fakeSvc, nil, nil, nil)

    rr := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(rr)

    body := updateTagsReq{Tags: []string{"a"}}
    bs, _ := json.Marshal(body)
    req, _ := http.NewRequest("PUT", "/api/v1/agents//tags", bytes.NewBuffer(bs))
    req.Header.Set("Content-Type", "application/json")

    c.Request = req
    // 注意：不设置 c.Params，以触发缺少 agentID 的逻辑

    h.UpdateAgentTags(c)

    if rr.Code != http.StatusBadRequest {
        t.Fatalf("expected status %d, got %d, body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
    }
}

// TestUpdateAgentTagsHandler_InvalidJSON 验证无效 JSON 时返回 400。
func TestUpdateAgentTagsHandler_InvalidJSON(t *testing.T) {
    gin.SetMode(gin.TestMode)
    fakeSvc := &fakeManagerService{store: map[string][]string{}}
    h := handlerAgent.NewAgentHandler(fakeSvc, nil, nil, nil)

    rr := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(rr)

    req, _ := http.NewRequest("PUT", "/api/v1/agents/agent-1/tags", bytes.NewBufferString("not-json"))
    req.Header.Set("Content-Type", "application/json")

    c.Request = req
    c.Params = gin.Params{{Key: "id", Value: "agent-1"}}

    h.UpdateAgentTags(c)

    if rr.Code != http.StatusBadRequest {
        t.Fatalf("expected status %d, got %d, body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
    }
}
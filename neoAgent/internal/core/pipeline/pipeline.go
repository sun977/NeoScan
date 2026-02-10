package pipeline

import (
	"sync"

	"neoagent/internal/core/model"
)

// PipelineContext 扫描上下文
// 在各个阶段之间传递的数据载体
// 每个 IP 对应一个 Context
type PipelineContext struct {
	IP string

	// 阶段 1: 存活信息
	Alive    bool
	Hostname string
	OSGuess  string // 基于 TTL 的 OS 猜测

	// 阶段 2: 端口开放信息
	// 仅存储 Open 端口
	OpenPorts []int

	// 阶段 3: 服务识别信息
	// Port -> Result 映射
	Services map[int]*model.PortServiceResult

	// 阶段 4: 操作系统精确识别
	OSInfo *model.OsInfo

	// 阶段 5 (Phase 2): 攻击面评估结果
	WebResults   []*model.WebResult
	VulnResults  []*model.VulnResult
	BruteResults []*model.BruteResult

	mu sync.RWMutex
}

func NewPipelineContext(ip string) *PipelineContext {
	return &PipelineContext{
		IP:           ip,
		Services:     make(map[int]*model.PortServiceResult),
		WebResults:   make([]*model.WebResult, 0),
		VulnResults:  make([]*model.VulnResult, 0),
		BruteResults: make([]*model.BruteResult, 0),
	}
}

// 线程安全的方法...

func (c *PipelineContext) SetAlive(alive bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Alive = alive
}

func (c *PipelineContext) AddOpenPort(port int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.OpenPorts = append(c.OpenPorts, port)
}

func (c *PipelineContext) SetService(port int, result *model.PortServiceResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Services[port] = result
}

func (c *PipelineContext) SetOS(info *model.OsInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.OSInfo = info
}

func (c *PipelineContext) AddWebResult(res *model.WebResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.WebResults = append(c.WebResults, res)
}

func (c *PipelineContext) AddVulnResult(res *model.VulnResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.VulnResults = append(c.VulnResults, res)
}

func (c *PipelineContext) AddBruteResult(res *model.BruteResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BruteResults = append(c.BruteResults, res)
}

// Helper methods for Dispatcher

// HasService checks if a service exists in the context
func (c *PipelineContext) HasService(svcName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, s := range c.Services {
		if s.Service == svcName {
			return true
		}
	}
	return false
}

// GetPortsByService returns all ports running a specific service
func (c *PipelineContext) GetPortsByService(svcName string) []int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var ports []int
	for port, s := range c.Services {
		if s.Service == svcName {
			ports = append(ports, port)
		}
	}
	return ports
}

// GetAllServices returns a copy of all identified services
func (c *PipelineContext) GetAllServices() map[int]*model.PortServiceResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	copy := make(map[int]*model.PortServiceResult)
	for k, v := range c.Services {
		copy[k] = v
	}
	return copy
}

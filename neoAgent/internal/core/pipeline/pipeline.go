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

	// 后续阶段在这里添加

	mu sync.RWMutex
}

func NewPipelineContext(ip string) *PipelineContext {
	return &PipelineContext{
		IP:       ip,
		Services: make(map[int]*model.PortServiceResult),
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

/**
 * 任务模型定义 (Core Domain)
 * @author: Sun977
 * @date: 2026.01.21
 * @description: 核心任务模型，解耦了 HTTP 依赖。这是连接 CLI 和 Cluster 模式的通用语言。
 */

package model

import (
	"time"
)

// TaskType 定义任务类型
type TaskType string

// 定义支持的 7 种扫描任务类型和 2 种非扫描任务类型
const (
	TaskTypeIpAliveScan TaskType = "ip_alive_scan" // IP存活扫描 (ICMP/ARP等)
	TaskTypePortScan    TaskType = "port_scan"     // 端口扫描 (独立)
	TaskTypeServiceScan TaskType = "service_scan"  // 服务扫描 (深度识别) + CPE指纹识别
	TaskTypeWebScan     TaskType = "web_scan"      // Web 综合扫描 + web指纹识别
	TaskTypeDirScan     TaskType = "dir_scan"      // 目录扫描
	TaskTypeVulnScan    TaskType = "vuln_scan"     // 漏洞扫描 (Nuclei)
	TaskTypeSubdomain   TaskType = "subdomain"     // 子域名扫描
	TaskTypeProxy       TaskType = "proxy"         // 代理服务 (Socks5/HTTP/Forward)
	TaskTypeRawCmd      TaskType = "raw_cmd"       // 原始命令执行
)

// TaskStatus 定义任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task 核心任务结构体
// 无论任务来自 CLI 还是 Master，最终都必须转换为此结构体
type Task struct {
	ID        string                 `json:"id"`
	Type      TaskType               `json:"type"`
	Target    string                 `json:"target"`               // 扫描目标 (IP/Domain/CIDR)
	PortRange string                 `json:"port_range,omitempty"` // 端口范围 (e.g. "80,443,1000-2000")
	Params    map[string]interface{} `json:"params,omitempty"`     // 任务特定参数
	Timeout   time.Duration          `json:"timeout"`
	Priority  int                    `json:"priority"`
	CreatedAt time.Time              `json:"created_at"`
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID    string      `json:"task_id"`
	Status    TaskStatus  `json:"status"`
	Data      interface{} `json:"data"` // 具体的扫描结果 (强类型结构体或 Map)
	Error     string      `json:"error,omitempty"`
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
}

// NewTask 创建一个新任务
func NewTask(taskType TaskType, target string) *Task {
	return &Task{
		Type:      taskType,
		Target:    target,
		CreatedAt: time.Now(),
		Params:    make(map[string]interface{}),
	}
}

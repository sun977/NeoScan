package runner

import (
	"context"
	"fmt"
	"sync"

	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/brute"
	"neoagent/internal/core/scanner/brute/protocol"
)

// RunnerManager 管理所有的 Runner
type RunnerManager struct {
	runners map[model.TaskType]Runner
	mu      sync.RWMutex
}

func NewRunnerManager() *RunnerManager {
	m := &RunnerManager{
		runners: make(map[model.TaskType]Runner),
	}

	// 初始化并注册 BruteScanner
	bs := brute.NewBruteScanner()
	bs.RegisterCracker(protocol.NewSSHCracker())           // 注册 SSH 爆破器
	bs.RegisterCracker(protocol.NewMySQLCracker())         // 注册 MySQL 爆破器
	bs.RegisterCracker(protocol.NewRedisCracker())         // 注册 Redis 爆破器
	bs.RegisterCracker(protocol.NewPostgresCracker())      // 注册 Postgres 爆破器
	bs.RegisterCracker(protocol.NewFTPCracker())           // 注册 FTP 爆破器
	bs.RegisterCracker(protocol.NewMongoCracker())         // 注册 MongoDB 爆破器
	bs.RegisterCracker(protocol.NewClickHouseCracker())    // 注册 ClickHouse 爆破器
	bs.RegisterCracker(protocol.NewSMBCracker())           // 注册 SMB 爆破器
	bs.RegisterCracker(protocol.NewMSSQLCracker())         // 注册 MSSQL 爆破器
	bs.RegisterCracker(protocol.NewOracleCracker())        // 注册 Oracle 爆破器
	bs.RegisterCracker(protocol.NewOracleSIDCracker())     // 注册 Oracle SID 爆破器
	bs.RegisterCracker(protocol.NewTelnetCracker())        // 注册 Telnet 爆破器
	bs.RegisterCracker(protocol.NewElasticsearchCracker()) // 注册 Elasticsearch 爆破器
	bs.RegisterCracker(protocol.NewSNMPCracker())          // 注册 SNMP 爆破器
	m.Register(bs)

	return m
}

// Register 注册一个 Runner
func (m *RunnerManager) Register(runner Runner) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runners[runner.Name()] = runner
}

// Get 获取指定类型的 Runner
func (m *RunnerManager) Get(taskType model.TaskType) (Runner, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if runner, ok := m.runners[taskType]; ok {
		return runner, nil
	}
	return nil, fmt.Errorf("no runner found for task type: %s", taskType)
}

// Execute 执行任务
func (m *RunnerManager) Execute(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	runner, err := m.Get(task.Type)
	if err != nil {
		return nil, err
	}

	return runner.Run(ctx, task)
}

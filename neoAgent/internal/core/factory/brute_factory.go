package factory

import (
	"neoagent/internal/core/scanner/brute"
	"neoagent/internal/core/scanner/brute/protocol"
)

// NewFullBruteScanner 创建一个注册了所有支持协议的完整扫描器
// 这是一个工厂方法，确保所有消费者（AutoRunner, RunnerManager）获得一致的能力集
// 避免在不同入口处重复注册 Cracker，导致能力不一致的问题
func NewFullBruteScanner() *brute.BruteScanner {
	s := brute.NewBruteScanner()

	// 注册所有支持的爆破协议
	s.RegisterCracker(protocol.NewSSHCracker())
	s.RegisterCracker(protocol.NewMySQLCracker())
	s.RegisterCracker(protocol.NewRedisCracker())
	s.RegisterCracker(protocol.NewPostgresCracker())
	s.RegisterCracker(protocol.NewFTPCracker())
	s.RegisterCracker(protocol.NewMongoCracker())
	s.RegisterCracker(protocol.NewClickHouseCracker())
	s.RegisterCracker(protocol.NewSMBCracker())
	s.RegisterCracker(protocol.NewMSSQLCracker())
	s.RegisterCracker(protocol.NewOracleCracker())
	s.RegisterCracker(protocol.NewOracleSIDCracker())
	s.RegisterCracker(protocol.NewTelnetCracker())
	s.RegisterCracker(protocol.NewElasticsearchCracker())
	s.RegisterCracker(protocol.NewSNMPCracker())
	s.RegisterCracker(protocol.NewRDPCracker())

	return s
}

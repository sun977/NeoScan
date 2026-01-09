package agent

import (
	"context"

	"neomaster/internal/config"
	"neomaster/internal/service/agent_update"
)

type AgentUpdateService interface {
	GetFingerprintSnapshotInfo(ctx context.Context) (*agent_update.FingerprintSnapshotInfo, error)
	BuildFingerprintSnapshot(ctx context.Context) (*agent_update.FingerprintSnapshot, error)
}

type agentUpdateService struct {
	cfg *config.Config
}

func NewAgentUpdateService(cfg *config.Config) AgentUpdateService {
	return &agentUpdateService{cfg: cfg}
}

func (s *agentUpdateService) GetFingerprintSnapshotInfo(ctx context.Context) (*agent_update.FingerprintSnapshotInfo, error) {
	rulePath := ""
	if s.cfg != nil {
		rulePath = s.cfg.GetFingerprintRulePath()
	}
	return agent_update.GetFingerprintSnapshotInfo(ctx, rulePath)
}

func (s *agentUpdateService) BuildFingerprintSnapshot(ctx context.Context) (*agent_update.FingerprintSnapshot, error) {
	rulePath := ""
	if s.cfg != nil {
		rulePath = s.cfg.GetFingerprintRulePath()
	}
	return agent_update.BuildFingerprintSnapshot(ctx, rulePath)
}

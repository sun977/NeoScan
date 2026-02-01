package setup

import (
	"fmt"
	"neoagent/internal/config"
	"neoagent/internal/service/client"
)

// SetupClient 初始化客户端通信模块
func SetupClient(cfg *config.Config) *ClientModule {
	// 初始化Master服务
	var masterSvc client.MasterService
	if cfg.Master != nil {
		masterURL := fmt.Sprintf("%s://%s:%d", cfg.Master.Protocol, cfg.Master.Address, cfg.Master.Port)
		masterSvc = client.NewMasterService(masterURL)
	}

	return &ClientModule{
		MasterService: masterSvc,
	}
}

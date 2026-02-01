package setup

import (
	"net/http"

	"neoagent/internal/app/agent/router"
	"neoagent/internal/core/runner"
	"neoagent/internal/service/client"
)

// CoreModule 核心扫描模块
type CoreModule struct {
	RunnerManager *runner.RunnerManager
}

// ClientModule 客户端通信模块
type ClientModule struct {
	MasterService client.MasterService
}

// ServerModule 服务器模块
type ServerModule struct {
	Router     *router.Router
	HTTPServer *http.Server
}

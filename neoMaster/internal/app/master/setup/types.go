/**
 * 初始化
 * @author: sun977
 * @date: 2025.11.05
 * @description: 包含master程序初始化相关的类型定义
 * @func: Handler 本身包含 Service,但是Servicer本身又重新暴露一遍,方便调用（后续看是否需要修改）
 */
package setup

import (
	agentHandler "neomaster/internal/handler/agent"
	assetHandler "neomaster/internal/handler/asset"
	authHandler "neomaster/internal/handler/auth"
	systemHandler "neomaster/internal/handler/system"
	agentService "neomaster/internal/service/agent"
	assetService "neomaster/internal/service/asset"
	authService "neomaster/internal/service/auth"
)

// AuthModule 是认证模块的聚合输出
// 设计目的：
// - 将认证相关的 Handler 与 Service 作为一个整体进行初始化与对外暴露，便于 router_manager 进行路由与中间件装配。
// - 保持与现有工程的层级约束（Handler → Service → Repository），setup 层仅负责“依赖装配”，不侵入业务逻辑。
// - 最小化对现有代码的影响：router_manager 通过 AuthModule 取用需要的组件，不改变原有 Handler/Service 的签名。
//
// 字段说明：
// - LoginHandler/LogoutHandler/RefreshHandler/RegisterHandler：认证相关的路由处理器
// - SessionService/JWTService/PasswordService：认证模块对外需要暴露给其他模块（如中间件、System.UserHandler）的服务实例
type AuthModule struct {
	// Handlers（认证相关处理器）
	LoginHandler    *authHandler.LoginHandler
	LogoutHandler   *authHandler.LogoutHandler
	RefreshHandler  *authHandler.RefreshHandler
	RegisterHandler *authHandler.RegisterHandler

	// Services（对外暴露以供 router_manager 及其他模块使用）
	SessionService  *authService.SessionService
	JWTService      *authService.JWTService
	PasswordService *authService.PasswordService
	UserService     *authService.UserService
	RBACService     *authService.RBACService
}

// SystemRBACModule 是系统层面的 RBAC 管理模块聚合输出
// 设计目的：
// - 将“系统角色与权限（Role/Permission）”相关的 Handler 与 Service 作为一个独立模块进行初始化与对外暴露。
// - 与认证模块（AuthModule）分割，避免在 router_manager 中散落初始化逻辑，提升可维护性与可测试性。
// - setup 层仅负责依赖装配（Handler → Service → Repository），不侵入业务逻辑实现。
//
// 字段说明：
// - RoleHandler/PermissionHandler：系统角色与权限管理的路由处理器
// - RoleService/PermissionService：对应的业务服务实例，便于必要时外部模块复用
type SystemRBACModule struct {
	// Handlers（系统RBAC相关处理器）
	RoleHandler       *systemHandler.RoleHandler
	PermissionHandler *systemHandler.PermissionHandler

	// Services（对外暴露以供 router_manager 或其他模块使用）
	RoleService       *authService.RoleService
	PermissionService *authService.PermissionService
}

// AgentModule 是 Agent 管理模块的聚合输出
// 设计目的：
// - 将 Agent 管理域（Manager/Monitor/Config/Task）的 Service 与聚合后的 Handler 作为一个整体对外暴露，减少 router_manager 中的初始化样板代码。
// - 保持分层约束与模块边界：setup 层仅负责依赖装配（Repository → Service → Handler），不侵入具体业务实现。
// - 便于后续扩展：若其他模块需要复用某个 Agent Service，可直接从该 Module 获取。
//
// 字段说明：
// - AgentHandler：对外用于路由注册的统一处理器入口（内部组合了所有 Agent 相关服务）。
// - ManagerService/MonitorService/ConfigService/TaskService：便于在必要时复用具体服务或编写独立测试。
type AgentModule struct {
	// Handler（对外路由处理器）
	AgentHandler *agentHandler.AgentHandler

	// Services（对外暴露以供 router_manager 或其他模块使用）
	ManagerService agentService.AgentManagerService
	MonitorService agentService.AgentMonitorService
	ConfigService  agentService.AgentConfigService
	TaskService    agentService.AgentTaskService
}

// OrchestratorModule 是扫描编排器（项目配置/工作流/工具/规则/规则引擎）模块的聚合输出
// 设计目的：
// - 将扫描配置相关的 Service 与 Handler 作为一个整体进行初始化与对外暴露，路由层只做“装配与注册”。
// - 与 Agent、Auth、System RBAC 模块保持同一风格，统一在 setup 层进行依赖装配，遵循 Handler → Service → Repository 的层级约束。
// - 便于后续测试与扩展：Router 可直接使用该模块暴露的 Handler；需要复用某个 Service 时也可从该模块获取。
//
// 字段说明：
// - ProjectConfigHandler/WorkflowHandler/ScanToolHandler/ScanRuleHandler/RuleEngineHandler：对外用于路由注册的处理器。
// - ProjectConfigService/WorkflowService/ScanToolService/ScanRuleService：对应的业务服务实例，便于必要时复用或编写独立测试。
// type OrchestratorModule struct {
// 	// Handlers（扫描编排器相关处理器）
// 	ProjectConfigHandler *orchestratorHandler.ProjectConfigHandler
// 	WorkflowHandler      *orchestratorHandler.WorkflowHandler
// 	ScanToolHandler      *orchestratorHandler.ScanToolHandler
// 	ScanRuleHandler      *orchestratorHandler.ScanRuleHandler
// 	RuleEngineHandler    *orchestratorHandler.RuleEngineHandler

// 	// Services（对外暴露以供 router_manager 或其他模块使用）
// 	ProjectConfigService *orchestratorService.ProjectConfigService
// 	WorkflowService      *orchestratorService.WorkflowService
// 	ScanToolService      *orchestratorService.ScanToolService
// 	ScanRuleService      *orchestratorService.ScanRuleService
// }

// AssetModule 是资产管理模块的聚合输出
// 设计目的：
// - 将资产管理相关的 Service 与 Handler 作为一个整体进行初始化与对外暴露。
// - 保持分层约束与模块边界：setup 层仅负责依赖装配（Repository → Service → Handler）。
//
// 字段说明：
// - AssetHostHandler：对外用于路由注册的处理器。
// - AssetHostService：对应的业务服务实例。
type AssetModule struct {
	// Handlers
	AssetRawHandler     *assetHandler.RawAssetHandler
	AssetHostHandler    *assetHandler.AssetHostHandler
	AssetNetworkHandler *assetHandler.AssetNetworkHandler
	AssetPolicyHandler  *assetHandler.AssetPolicyHandler
	AssetWebHandler     *assetHandler.AssetWebHandler

	// Services
	AssetRawService     *assetService.RawAssetService
	AssetHostService    *assetService.AssetHostService
	AssetNetworkService *assetService.AssetNetworkService
	AssetPolicyService  *assetService.AssetPolicyService
	AssetWebService     *assetService.AssetWebService
}

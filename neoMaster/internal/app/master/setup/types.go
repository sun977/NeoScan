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
	orchestratorHandler "neomaster/internal/handler/orchestrator"
	systemHandler "neomaster/internal/handler/system"
	tagHandler "neomaster/internal/handler/tag_system"
	agentService "neomaster/internal/service/agent"
	assetService "neomaster/internal/service/asset"
	"neomaster/internal/service/asset/enrichment" // 引入资产富化 enrichment
	"neomaster/internal/service/asset/etl"        // 引入ETL
	authService "neomaster/internal/service/auth"
	"neomaster/internal/service/fingerprint" // 引入 fingerprint
	orchestratorService "neomaster/internal/service/orchestrator"
	"neomaster/internal/service/orchestrator/core/scheduler"
	"neomaster/internal/service/orchestrator/ingestor" // 引入ingestor
	"neomaster/internal/service/orchestrator/local_agent"
	tagService "neomaster/internal/service/tag_system"
)

// TagModule 是标签系统模块的聚合输出
type TagModule struct {
	// Handlers
	TagHandler *tagHandler.TagHandler

	// Services
	TagService tagService.TagService
}

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
// - 将 Agent 管理域（Manager/Monitor/Config）的 Service 与聚合后的 Handler 作为一个整体对外暴露。
// - 保持分层约束与模块边界：setup 层仅负责依赖装配（Repository → Service → Handler），不侵入具体业务实现。
// - 便于后续扩展：若其他模块需要复用某个 Agent Service，可直接从该 Module 获取。
//
// 字段说明：
// - AgentHandler：对外用于路由注册的统一处理器入口（内部组合了所有 Agent 相关服务）。
// - ManagerService/MonitorService/ConfigService：便于在必要时复用具体服务或编写独立测试。
type AgentModule struct {
	// Handler（对外路由处理器）
	AgentHandler *agentHandler.AgentHandler

	// Services（对外暴露以供 router_manager 或其他模块使用）
	ManagerService agentService.AgentManagerService
	MonitorService agentService.AgentMonitorService
	ConfigService  agentService.AgentConfigService
	UpdateService  agentService.AgentUpdateService
	// TaskService 移至 OrchestratorModule
}

// OrchestratorModule 是扫描编排器（项目配置/工作流/工具/规则/规则引擎/任务分发）模块的聚合输出
// 设计目的：
// - 将扫描配置相关的 Service 与 Handler 作为一个整体进行初始化与对外暴露，路由层只做“装配与注册”。
// - 与 Agent、Auth、System RBAC 模块保持同一风格，统一在 setup 层进行依赖装配，遵循 Handler → Service → Repository 的层级约束。
// - 便于后续测试与扩展：Router 可直接使用该模块暴露的 Handler；需要复用某个 Service 时也可从该模块获取。
//
// 字段说明：
// - ProjectHandler/WorkflowHandler/ScanStageHandler/ScanToolTemplateHandler/AgentTaskHandler：对外用于路由注册的处理器。
// - ProjectService/WorkflowService/ScanStageService/ScanToolTemplateService/AgentTaskService：对应的业务服务实例。
type OrchestratorModule struct {
	// Handlers（扫描编排器相关处理器）
	ProjectHandler          *orchestratorHandler.ProjectHandler
	WorkflowHandler         *orchestratorHandler.WorkflowHandler
	ScanStageHandler        *orchestratorHandler.ScanStageHandler
	ScanToolTemplateHandler *orchestratorHandler.ScanToolTemplateHandler
	AgentTaskHandler        *orchestratorHandler.AgentTaskHandler // 新增

	// Services（对外暴露以供 router_manager 或其他模块使用）
	ProjectService          *orchestratorService.ProjectService
	WorkflowService         *orchestratorService.WorkflowService
	ScanStageService        *orchestratorService.ScanStageService
	ScanToolTemplateService *orchestratorService.ScanToolTemplateService
	AgentTaskService        orchestratorService.AgentTaskService // 新增 (interface type)

	// Core Components (核心组件)
	TaskDispatcher   orchestratorService.TaskDispatcher
	SchedulerService scheduler.SchedulerService
	LocalAgent       *local_agent.LocalAgent // 本地Agent (原系统任务执行器)
	ResultIngestor   ingestor.ResultIngestor // 结果摄入服务
	ETLProcessor     etl.ResultProcessor     // ETL 结果处理器
}

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
	AssetRawHandler           *assetHandler.RawAssetHandler        // 原始资产处理器
	AssetHostHandler          *assetHandler.AssetHostHandler       // 主机资产处理器
	AssetNetworkHandler       *assetHandler.AssetNetworkHandler    // 网络资产处理器
	AssetPolicyHandler        *assetHandler.AssetPolicyHandler     // 策略执行处理器
	AssetFingerCmsHandler     *assetHandler.AssetFingerHandler     // CMS指纹资产处理器
	AssetFingerServiceHandler *assetHandler.AssetCPEHandler        // CPE指纹资产处理器
	AssetWebHandler           *assetHandler.AssetWebHandler        // Web资产处理器
	AssetVulnHandler          *assetHandler.AssetVulnHandler       // 漏洞资产处理器
	AssetUnifiedHandler       *assetHandler.AssetUnifiedHandler    // 统一资产视图处理器
	AssetScanHandler          *assetHandler.AssetScanHandler       // 扫描记录处理器
	FingerprintRuleHandler    *assetHandler.FingerprintRuleHandler // 指纹规则处理器 - 规则指纹供Agent使用
	ETLErrorHandler           *assetHandler.ETLErrorHandler        // ETL资产清洗错误处理器 - 用于处理ETL过程中出现的错误资产(dB充当"死信队列")

	// Services
	AssetRawService           *assetService.RawAssetService     // 原始资产服务
	AssetHostService          *assetService.AssetHostService    // 主机资产服务
	AssetNetworkService       *assetService.AssetNetworkService // 网络资产服务
	AssetPolicyService        *assetService.AssetPolicyService  // 策略执行服务
	AssetFingerCmsService     *assetService.AssetFingerService  // CMS指纹资产服务
	AssetFingerServiceService *assetService.AssetCPEService     // CPE指纹资产服务
	AssetWebService           *assetService.AssetWebService     // Web资产服务
	AssetVulnService          *assetService.AssetVulnService    // 漏洞资产服务
	AssetUnifiedService       *assetService.AssetUnifiedService // 统一资产视图服务
	AssetScanService          *assetService.AssetScanService    // 扫描记录服务
	FingerprintRuleManager    *fingerprint.RuleManager          // 指纹规则管理器 - 用于管理指纹规则
	AssetETLErrorService      assetService.AssetETLErrorService // ETL资产清洗错误服务 - 用于处理ETL过程中出现的错误资产(dB充当"死信队列")
	FingerprintGovernance     *enrichment.FingerprintMatcher    // 资产富化 - 指纹治理服务(用于Master端离线二次指纹识别)
}

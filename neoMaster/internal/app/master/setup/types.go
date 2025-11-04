package setup

import (
	authHandler "neomaster/internal/handler/auth"
	systemHandler "neomaster/internal/handler/system"
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

package setup

import (
    systemHandler "neomaster/internal/handler/system"
    systemRepo "neomaster/internal/repo/mysql/system"
    "neomaster/internal/pkg/logger"
    authService "neomaster/internal/service/auth"

    "gorm.io/gorm"
)

// BuildSystemRBACModule 构建系统RBAC模块（角色与权限管理）
// 责任边界：
// - 初始化系统角色与权限相关的仓库与服务（RoleService、PermissionService）
// - 初始化对应路由处理器（RoleHandler、PermissionHandler）
// - 与认证模块（AuthModule）分离，只聚合“系统RBAC域”的组件，供 router_manager 进行路由装配
//
// 参数说明：
// - db：MySQL连接（gorm.DB），用于构建系统角色与权限仓库
//
// 返回：
// - *SystemRBACModule：聚合后的系统RBAC模块输出（Handlers 与 Services）
func BuildSystemRBACModule(db *gorm.DB) *SystemRBACModule {
    // 结构化日志：记录模块化初始化的关键步骤
    logger.WithFields(map[string]interface{}{
        "path":      "internal.app.master.setup.rbac.BuildSystemRBACModule",
        "operation": "setup",
        "option":    "setup.rbac.begin",
        "func_name": "setup.rbac.BuildSystemRBACModule",
    }).Info("开始构建系统RBAC模块")

    // 1) 初始化仓库
    roleRepo := systemRepo.NewRoleRepository(db)
    permissionRepo := systemRepo.NewPermissionRepository(db)

    // 2) 初始化服务
    roleService := authService.NewRoleService(roleRepo)
    permissionService := authService.NewPermissionService(permissionRepo)

    // 3) 初始化处理器
    roleHandler := systemHandler.NewRoleHandler(roleService)
    permissionHandler := systemHandler.NewPermissionHandler(permissionService)

    // 4) 聚合输出
    module := &SystemRBACModule{
        RoleHandler:       roleHandler,
        PermissionHandler: permissionHandler,
        RoleService:       roleService,
        PermissionService: permissionService,
    }

    logger.WithFields(map[string]interface{}{
        "path":      "internal.app.master.setup.rbac.BuildSystemRBACModule",
        "operation": "setup",
        "option":    "setup.rbac.done",
        "func_name": "setup.rbac.BuildSystemRBACModule",
    }).Info("系统RBAC模块构建完成")

    return module
}
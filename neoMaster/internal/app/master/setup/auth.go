package setup

import (
	"time"

	"neomaster/internal/config"
	authHandler "neomaster/internal/handler/auth"
	authPkg "neomaster/internal/pkg/auth"
	pkgDatabase "neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"
	systemRepo "neomaster/internal/repo/mysql/system"
	redisRepo "neomaster/internal/repo/redis"
	authService "neomaster/internal/service/auth"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// BuildAuthModule 构建认证模块（Auth）
// 责任边界：
// - 初始化认证相关的工具、仓库与服务（JWT、Password、Session、JWTService、SessionService、PasswordService、UserService、RBACService）
// - 初始化认证相关的处理器（Login/Logout/Refresh/Register）
// - 仅聚合“认证域”的组件，并将其作为模块输出，供 router_manager 进行路由与中间件装配
//
// 参数说明：
// - db：MySQL连接（gorm.DB），用于构建系统用户仓库
// - redisClient：Redis 客户端；如为 nil，则按配置 cfg.Database.Redis 自动建立连接
// - cfg：全局配置；用于初始化 JWT 与安全相关参数
//
// 返回：
// - *AuthModule：聚合后的认证模块输出（Handlers 与 Services）
// - error：初始化过程中出现的错误（例如Redis连接失败）
func BuildAuthModule(
	db *gorm.DB,
	redisClient *redis.Client,
	cfg *config.Config,
) (*AuthModule, error) {
	// 结构化日志：记录模块化初始化的关键步骤
	logger.WithFields(map[string]interface{}{
		"path":      "internal.app.master.setup.auth.BuildAuthModule",
		"operation": "setup",
		"option":    "setup.auth.begin",
		"func_name": "setup.auth.BuildAuthModule",
	}).Info("开始构建认证模块")

	// 1) 初始化工具：JWTManager 与 PasswordManager（从配置读取TTL）
	jwtCfg := cfg.Security.JWT
	jwtManager := authPkg.NewJWTManager(jwtCfg.Secret, jwtCfg.AccessTokenExpire, jwtCfg.RefreshTokenExpire)

	// PasswordManager 的参数目前未配置化，沿用项目内既有初始化常量（与原 router_manager.go 一致）
	passwordConfig := &authPkg.PasswordConfig{
		Memory:      64 * 1024, // 64MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  32,
		KeyLength:   32,
	}
	passwordManager := authPkg.NewPasswordManager(passwordConfig)

	// 2) 初始化会话存储仓库（统一使用Redis实现，符合现有服务层依赖类型）
	// 说明：当前服务层依赖的是 *redis.SessionRepository 的具体类型，故此处不使用内存实现以避免类型不匹配。
	var redisCli *redis.Client
	if redisClient != nil {
		redisCli = redisClient
	} else {
		// 按配置建立Redis连接（兜底）
		cli, err := pkgDatabase.NewRedisConnection(&cfg.Database.Redis)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"path":      "internal.app.master.setup.auth.BuildAuthModule",
				"operation": "setup",
				"option":    "setup.auth.repo.session.redis.connect_error",
				"func_name": "setup.auth.BuildAuthModule",
				"error":     err.Error(),
			}).Error("Redis连接失败")
			return nil, err
		}
		redisCli = cli
	}
	sessionRepo := redisRepo.NewSessionRepository(redisCli)
	logger.WithFields(map[string]interface{}{
		"path":      "internal.app.master.setup.auth.BuildAuthModule",
		"operation": "setup",
		"option":    "setup.auth.repo.session.redis",
		"func_name": "setup.auth.BuildAuthModule",
	}).Info("会话存储使用 Redis 实现")

	// 3) 初始化系统用户仓库与服务
	userRepo := systemRepo.NewUserRepository(db)
	userService := authService.NewUserService(userRepo, sessionRepo, passwordManager, jwtManager)

	// 4) 初始化RBAC服务 (运行时鉴权使用,并非系统RBAC管理使用)
	rbacService := authService.NewRBACService(userService)

	// 5) 初始化会话与JWT服务（解决循环依赖）
	sessionService := authService.NewSessionService(userService, passwordManager, rbacService, sessionRepo)
	jwtService := authService.NewJWTService(jwtManager, userService, sessionRepo)
	sessionService.SetTokenGenerator(jwtService)

	// 6) 初始化密码服务
	passwordService := authService.NewPasswordService(userService, sessionService, passwordManager, time.Hour*24)

	// 7) 初始化处理器（认证相关）
	loginHandler := authHandler.NewLoginHandler(sessionService)
	logoutHandler := authHandler.NewLogoutHandler(sessionService)
	refreshHandler := authHandler.NewRefreshHandler(sessionService)
	registerHandler := authHandler.NewRegisterHandler(userService)

	// 8) 聚合输出
	module := &AuthModule{
		LoginHandler:    loginHandler,
		LogoutHandler:   logoutHandler,
		RefreshHandler:  refreshHandler,
		RegisterHandler: registerHandler,
		SessionService:  sessionService,
		JWTService:      jwtService,
		PasswordService: passwordService,
		UserService:     userService,
		RBACService:     rbacService,
	}

	logger.WithFields(map[string]interface{}{
		"path":      "internal.app.master.setup.auth.BuildAuthModule",
		"operation": "setup",
		"option":    "setup.auth.done",
		"func_name": "setup.auth.BuildAuthModule",
	}).Info("认证模块构建完成")

	return module, nil
}

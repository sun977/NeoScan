package setup

import (
	"gorm.io/gorm"
	tagHandler "neomaster/internal/handler/tag_system"
	"neomaster/internal/pkg/logger"
	tagRepo "neomaster/internal/repo/mysql/tag_system"
	tagService "neomaster/internal/service/tag_system"
)

// BuildTagSystemModule 构建标签系统模块
// 责任边界：
// - 初始化 TagSystem 相关的仓库与服务。
// - 聚合为统一的 TagHandler，供 router_manager 进行路由注册。
//
// 参数：
// - db：数据库连接（gorm.DB）。
//
// 返回：
// - *TagModule：聚合后的 Tag 模块输出。
func BuildTagSystemModule(db *gorm.DB) *TagModule {
	logger.WithFields(map[string]interface{}{
		"path":      "internal.app.master.setup.tag_system.BuildTagSystemModule",
		"operation": "setup",
		"option":    "setup.tag_system.begin",
		"func_name": "setup.tag_system.BuildTagSystemModule",
	}).Info("开始构建标签系统模块")

	// 1) 初始化仓库
	repo := tagRepo.NewTagRepository(db)

	// 2) 初始化服务
	service := tagService.NewTagService(repo, db)

	// 3) 初始化处理器
	handler := tagHandler.NewTagHandler(service)

	// 4) 聚合输出
	module := &TagModule{
		TagHandler: handler,
		TagService: service,
	}

	logger.WithFields(map[string]interface{}{
		"path":      "internal.app.master.setup.tag_system.BuildTagSystemModule",
		"operation": "setup",
		"option":    "setup.tag_system.done",
		"func_name": "setup.tag_system.BuildTagSystemModule",
	}).Info("标签系统模块构建完成")

	return module
}

package auth

import (
	"context"
	"errors"
	"fmt"
	"neomaster/internal/model/system"
	systemrepo "neomaster/internal/repo/mysql/system"
	"time"

	"neomaster/internal/pkg/logger"
)

// PermissionService 权限服务
// 仅处理权限自身的增删改查，不与角色分配、RBAC、用户授权等逻辑重叠
type PermissionService struct {
	permissionRepo *systemrepo.PermissionRepository
}

// NewPermissionService 创建权限服务
func NewPermissionService(permissionRepo *systemrepo.PermissionRepository) *PermissionService {
	return &PermissionService{permissionRepo: permissionRepo}
}

// CreatePermission 创建权限
func (s *PermissionService) CreatePermission(ctx context.Context, req *system.CreatePermissionRequest) (*system.Permission, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	type clientIPKeyType struct{}
	clientIP, _ := ctx.Value(clientIPKeyType{}).(string)

	if req == nil {
		logger.LogError(errors.New("request is nil"), "", 0, clientIP, "permission_create", "POST", map[string]interface{}{
			"operation": "create_permission",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("创建权限请求不能为空")
	}

	if req.Name == "" {
		logger.LogError(errors.New("permission name is empty"), "", 0, clientIP, "permission_create", "POST", map[string]interface{}{
			"operation": "create_permission",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("权限名称不能为空")
	}

	// 重名检查
	existing, err := s.permissionRepo.GetPermissionByName(ctx, req.Name)
	if err == nil && existing != nil {
		logger.LogError(errors.New("permission name already exists"), "", 0, clientIP, "permission_create", "POST", map[string]interface{}{
			"operation": "create_permission",
			"name":      req.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("权限名称已存在")
	}

	permission := &system.Permission{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Status:      system.PermissionStatusEnabled,
		Resource:    req.Resource,
		Action:      req.Action,
	}

	if err := s.permissionRepo.CreatePermission(ctx, permission); err != nil {
		logger.LogError(err, "", 0, clientIP, "permission_create", "POST", map[string]interface{}{
			"operation": "create_permission_db",
			"name":      req.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("创建权限失败: %w", err)
	}

	logger.LogBusinessOperation("create_permission", permission.ID, permission.Name, "", clientIP, "success", "Permission created successfully", map[string]interface{}{
		"name":         permission.Name,
		"display_name": permission.DisplayName,
		"resource":     permission.Resource,
		"action":       permission.Action,
		"timestamp":    logger.NowFormatted(),
	})

	return permission, nil
}

// GetPermissionByID 根据ID获取权限
func (s *PermissionService) GetPermissionByID(ctx context.Context, permissionID uint) (*system.Permission, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	type clientIPKeyType struct{}
	clientIP, _ := ctx.Value(clientIPKeyType{}).(string)
	if permissionID == 0 {
		logger.LogError(errors.New("invalid permission ID: cannot be zero"), "", 0, clientIP, "get_permission_by_id", "SERVICE", map[string]interface{}{
			"operation":     "parameter_validation",
			"permission_id": permissionID,
			"timestamp":     logger.NowFormatted(),
		})
		return nil, errors.New("权限ID不能为0")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	permission, err := s.permissionRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		logger.LogError(err, "", 0, clientIP, "get_permission_by_id", "SERVICE", map[string]interface{}{
			"operation":     "database_query",
			"permission_id": permissionID,
			"timestamp":     logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取权限失败: %w", err)
	}
	if permission == nil {
		logger.LogError(errors.New("permission not found"), "", 0, clientIP, "get_permission_by_id", "SERVICE", map[string]interface{}{
			"operation":     "permission_not_found",
			"permission_id": permissionID,
			"timestamp":     logger.NowFormatted(),
		})
		return nil, errors.New("权限不存在")
	}

	logger.LogBusinessOperation("get_permission_by_id", 0, permission.Name, clientIP, "", "success", "权限信息获取成功", map[string]interface{}{
		"permission_id": permissionID,
		"name":          permission.Name,
		"resource":      permission.Resource,
		"action":        permission.Action,
		"timestamp":     logger.NowFormatted(),
	})

	return permission, nil
}

// GetPermissionByName 根据名称获取权限
func (s *PermissionService) GetPermissionByName(ctx context.Context, name string) (*system.Permission, error) {
	if name == "" {
		return nil, errors.New("权限名称不能为空")
	}
	permission, err := s.permissionRepo.GetPermissionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, errors.New("权限不存在")
	}
	return permission, nil
}

// GetPermissionList 获取权限列表（分页）
func (s *PermissionService) GetPermissionList(ctx context.Context, offset, limit int) ([]*system.Permission, int64, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	type clientIPKeyType struct{}
	clientIP, _ := ctx.Value(clientIPKeyType{}).(string)

	originalOffset := offset
	originalLimit := limit
	if offset < 0 {
		logger.LogError(fmt.Errorf("invalid offset parameter: %d", offset), "", 0, clientIP, "get_permission_list", "SERVICE", map[string]interface{}{
			"operation": "get_permission_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if originalLimit != limit || originalOffset != offset {
		logger.LogBusinessOperation("get_permission_list", 0, "system", clientIP, "", "parameter_corrected", "分页参数已自动修正", map[string]interface{}{
			"operation":        "get_permission_list",
			"original_offset":  originalOffset,
			"original_limit":   originalLimit,
			"corrected_offset": offset,
			"corrected_limit":  limit,
			"timestamp":        logger.NowFormatted(),
		})
	}

	select {
	case <-ctx.Done():
		return nil, 0, fmt.Errorf("request cancelled: %w", ctx.Err())
	default:
	}

	permissions, total, err := s.permissionRepo.GetPermissionList(ctx, offset, limit)
	if err != nil {
		logger.LogError(err, "", 0, clientIP, "get_permission_list", "SERVICE", map[string]interface{}{
			"operation": "get_permission_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, fmt.Errorf("failed to get permission list from repo: %w", err)
	}
	if permissions == nil {
		permissions = make([]*system.Permission, 0)
	}

	logger.LogBusinessOperation("get_permission_list", 0, "system", clientIP, "", "success", "获取权限列表成功", map[string]interface{}{
		"operation":    "get_permission_list",
		"offset":       offset,
		"limit":        limit,
		"total":        total,
		"result_count": len(permissions),
		"timestamp":    logger.NowFormatted(),
	})

	return permissions, total, nil
}

// UpdatePermissionByID 更新权限
func (s *PermissionService) UpdatePermissionByID(ctx context.Context, permissionID uint, req *system.UpdatePermissionRequest) (*system.Permission, error) {
	if err := s.validateUpdatePermissionParams(ctx, permissionID, req); err != nil {
		return nil, err
	}
	permission, err := s.validatePermissionForUpdate(ctx, permissionID, req)
	if err != nil {
		return nil, err
	}
	return s.executePermissionUpdate(ctx, permission, req)
}

func (s *PermissionService) validateUpdatePermissionParams(ctx context.Context, permissionID uint, req *system.UpdatePermissionRequest) error {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	type clientIPKeyType struct{}
	clientIP, _ := ctx.Value(clientIPKeyType{}).(string)

	if permissionID == 0 {
		logger.LogError(errors.New("invalid permission ID for update"), "", 0, clientIP, "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "parameter_validation",
			"permission_id": permissionID,
			"error":         "permission_id_zero",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("权限ID不能为0")
	}
	if req == nil {
		logger.LogError(errors.New("update request is nil"), "", 0, clientIP, "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "parameter_validation",
			"permission_id": permissionID,
			"error":         "request_nil",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("更新权限请求不能为空")
	}
	return nil
}

func (s *PermissionService) validatePermissionForUpdate(ctx context.Context, permissionID uint, req *system.UpdatePermissionRequest) (*system.Permission, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	type clientIPKeyType struct{}
	clientIP, _ := ctx.Value(clientIPKeyType{}).(string)
	// 权限存在校验(permission_id有效性)
	permission, err := s.permissionRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		logger.LogError(err, "", 0, clientIP, "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_existence_check",
			"permission_id": permissionID,
			"error":         "database_query_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取权限失败: %w", err)
	}
	if permission == nil {
		logger.LogError(errors.New("permission not found for update"), "", 0, clientIP, "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_existence_check",
			"permission_id": permissionID,
			"error":         "permission_not_found",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, errors.New("权限不存在")
	}

	// 系统权限保护机制
	if permission.ID == 1 {
		logger.LogError(errors.New("system permission cannot be updated"), "", 0, clientIP, "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "system_permission_protection",
			"permission_id": permissionID,
			"error":         "system_permission_update_prohibited",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, errors.New("系统权限不能被修改")
	}

	// 权限名称冲突校验,不能重复[请求中的新权限名称不能和数据库中已有的权限名称重复]
	// 如果数据库本身就有权限名称唯一索引,则这部分校验逻辑可以省略,报错数据库唯一性错误,如下
	// Error 1062 (23000): Duplicate entry 'permission:delete' for key 'permissions.idx_permissions_name'
	// 如果数据库本身没有权限名称唯一索引,则下面代码会生效(保证权限名称唯一性)
	exists, err := s.permissionRepo.PermissionExists(ctx, req.Name)
	if err != nil {
		logger.LogError(err, "", 0, clientIP, "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_name_conflict_check",
			"permission_id": permissionID,
			"error":         "database_query_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, fmt.Errorf("检查权限名称冲突失败: %w", err)
	}
	if exists {
		// 请求中的新名字与数据库中的名字相同，则不进行更新[只修改其他属性可以不携带name字段]
		logger.LogError(errors.New("permission name already exists"), "", 0, clientIP, "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_name_conflict_check",
			"permission_id": permissionID,
			"error":         "permission_name_conflict",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, errors.New("权限名称冲突")
	}

	return permission, nil
}

func (s *PermissionService) executePermissionUpdate(ctx context.Context, permission *system.Permission, req *system.UpdatePermissionRequest) (*system.Permission, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	type clientIPKeyType struct{}
	clientIP, _ := ctx.Value(clientIPKeyType{}).(string)

	if req.Name != "" && req.Name != permission.Name {
		permission.Name = req.Name
	}
	if req.DisplayName != "" && req.DisplayName != permission.DisplayName {
		permission.DisplayName = req.DisplayName
	}
	if req.Description != "" && req.Description != permission.Description {
		permission.Description = req.Description
	}
	if req.Resource != "" && req.Resource != permission.Resource {
		permission.Resource = req.Resource
	}
	if req.Action != "" && req.Action != permission.Action {
		permission.Action = req.Action
	}
	if req.Status != nil && *req.Status != permission.Status {
		permission.Status = *req.Status
	}

	if err := s.permissionRepo.UpdatePermission(ctx, permission); err != nil {
		logger.LogError(err, "", 0, clientIP, "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "database_update",
			"permission_id": permission.ID,
			"error":         "update_permission_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, fmt.Errorf("更新权限失败: %w", err)
	}

	logger.LogBusinessOperation("update_permission", permission.ID, permission.Name, clientIP, "", "success", "权限更新成功", map[string]interface{}{
		"permission_id": permission.ID,
		"name":          permission.Name,
		"updated_at":    time.Now().Format(time.RFC3339),
		"timestamp":     logger.NowFormatted(),
	})

	return permission, nil
}

// DeletePermission 删除权限(权限表记录删除+权限角色关联表记录删除,不修改角色表记录)
// 优化版本：将所有数据库操作都放在事务内部，避免锁等待超时问题
func (s *PermissionService) DeletePermission(ctx context.Context, permissionID uint) error {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	type clientIPKeyType struct{}
	clientIP, _ := ctx.Value(clientIPKeyType{}).(string)

	// 参数验证
	if permissionID == 0 {
		logger.LogError(errors.New("invalid permission ID for deletion"), "", 0, clientIP, "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "parameter_validation",
			"permission_id": permissionID,
			"error":         "permission_id_zero",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("权限ID不能为0")
	}

	// 开始事务 - 将所有操作都放在事务内部，避免事务外查询造成的锁冲突
	tx := s.permissionRepo.BeginTx(ctx)
	if tx == nil {
		logger.LogError(errors.New("failed to begin transaction"), "", 0, clientIP, "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "transaction_begin",
			"permission_id": permissionID,
			"error":         "transaction_begin_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("开始事务失败")
	}

	// 使用 defer 确保事务回滚和 panic 恢复
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.LogError(fmt.Errorf("panic during permission deletion: %v", r), "", 0, clientIP, "delete_permission", "SERVICE", map[string]interface{}{
				"operation":     "panic_recovery",
				"permission_id": permissionID,
				"error":         "panic_occurred",
				"panic":         r,
				"timestamp":     logger.NowFormatted(),
			})
		}
	}()

	// 1. 在事务内检查权限是否存在（避免事务外查询造成的锁冲突）
	// 不再调用 permissionRepo.PermissionExistsByID 方法
	// 改用事务内的查询来检查权限是否存在
	var count int64
	if err := tx.WithContext(ctx).Model(&system.Permission{}).Where("id = ?", permissionID).Count(&count).Error; err != nil {
		tx.Rollback()
		logger.LogError(err, "", 0, clientIP, "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_existence_check",
			"permission_id": permissionID,
			"error":         "database_query_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return fmt.Errorf("检查权限存在性失败: %w", err)
	}
	if count == 0 {
		tx.Rollback()
		logger.LogError(errors.New("permission not found for deletion"), "", 0, clientIP, "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_existence_check",
			"permission_id": permissionID,
			"error":         "permission_not_found",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("权限不存在")
	}

	// 2. 先删除权限角色关联记录 - role_permissions（子表）- 确保删除顺序正确
	if err := s.permissionRepo.DeleteRolePermissionsByPermissionID(ctx, tx, permissionID); err != nil {
		tx.Rollback()
		logger.LogError(err, "", 0, clientIP, "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "cascade_delete_role_permissions",
			"permission_id": permissionID,
			"error":         "delete_role_permissions_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return fmt.Errorf("删除权限与角色关联失败: %w", err)
	}

	// 3. 再删除权限记录 - permissions（父表）- 使用事务内的删除操作
	if err := tx.WithContext(ctx).Delete(&system.Permission{}, permissionID).Error; err != nil {
		tx.Rollback()
		logger.LogError(err, "", 0, clientIP, "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "delete_permission",
			"permission_id": permissionID,
			"error":         "delete_permission_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return fmt.Errorf("删除权限失败: %w", err)
	}

	// 4. 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.LogError(err, "", 0, clientIP, "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "transaction_commit",
			"permission_id": permissionID,
			"error":         "transaction_commit_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 记录成功日志
	logger.LogBusinessOperation("delete_permission", 0, "", clientIP, "", "success", "权限删除成功", map[string]interface{}{
		"operation":     "permission_deletion_success",
		"permission_id": permissionID,
		"deleted_at":    logger.NowFormatted(),
		"timestamp":     logger.NowFormatted(),
	})
	return nil
}

// GetPermissionWithRoles 获取权限及其关联角色（只读）
func (s *PermissionService) GetPermissionWithRoles(ctx context.Context, permissionID uint) (*system.Permission, error) {
	if permissionID == 0 {
		return nil, errors.New("权限ID不能为0")
	}
	return s.permissionRepo.GetPermissionWithRoles(ctx, permissionID)
}

// GetPermissionRoles 获取权限关联的角色（只读）
func (s *PermissionService) GetPermissionRoles(ctx context.Context, permissionID uint) ([]*system.Role, error) {
	if permissionID == 0 {
		return nil, errors.New("权限ID不能为0")
	}
	return s.permissionRepo.GetPermissionRoles(ctx, permissionID)
}

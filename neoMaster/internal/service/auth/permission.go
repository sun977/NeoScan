package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/repository/mysql"
)

// PermissionService 权限服务
// 仅处理权限自身的增删改查，不与角色分配、RBAC、用户授权等逻辑重叠
type PermissionService struct {
	permissionRepo *mysql.PermissionRepository
}

// NewPermissionService 创建权限服务
func NewPermissionService(permissionRepo *mysql.PermissionRepository) *PermissionService {
	return &PermissionService{permissionRepo: permissionRepo}
}

// CreatePermission 创建权限
func (s *PermissionService) CreatePermission(ctx context.Context, req *model.CreatePermissionRequest) (*model.Permission, error) {
	if req == nil {
		logger.LogError(errors.New("request is nil"), "", 0, "", "permission_create", "POST", map[string]interface{}{
			"operation": "create_permission",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("创建权限请求不能为空")
	}

	if req.Name == "" {
		logger.LogError(errors.New("permission name is empty"), "", 0, "", "permission_create", "POST", map[string]interface{}{
			"operation": "create_permission",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("权限名称不能为空")
	}

	// 重名检查
	existing, err := s.permissionRepo.GetPermissionByName(ctx, req.Name)
	if err == nil && existing != nil {
		logger.LogError(errors.New("permission name already exists"), "", 0, "", "permission_create", "POST", map[string]interface{}{
			"operation": "create_permission",
			"name":      req.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("权限名称已存在")
	}

	permission := &model.Permission{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Status:      model.PermissionStatusEnabled,
		Resource:    req.Resource,
		Action:      req.Action,
	}

	if err := s.permissionRepo.CreatePermission(ctx, permission); err != nil {
		logger.LogError(err, "", 0, "", "permission_create", "POST", map[string]interface{}{
			"operation": "create_permission_db",
			"name":      req.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("创建权限失败: %w", err)
	}

	logger.LogBusinessOperation("create_permission", permission.ID, permission.Name, "", "", "success", "Permission created successfully", map[string]interface{}{
		"name":         permission.Name,
		"display_name": permission.DisplayName,
		"resource":     permission.Resource,
		"action":       permission.Action,
		"timestamp":    logger.NowFormatted(),
	})

	return permission, nil
}

// GetPermissionByID 根据ID获取权限
func (s *PermissionService) GetPermissionByID(ctx context.Context, permissionID uint) (*model.Permission, error) {
	if permissionID == 0 {
		logger.LogError(errors.New("invalid permission ID: cannot be zero"), "", 0, "", "get_permission_by_id", "SERVICE", map[string]interface{}{
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
		logger.LogError(err, "", permissionID, "", "get_permission_by_id", "SERVICE", map[string]interface{}{
			"operation":     "database_query",
			"permission_id": permissionID,
			"timestamp":     logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取权限失败: %w", err)
	}
	if permission == nil {
		logger.LogError(errors.New("permission not found"), "", permissionID, "", "get_permission_by_id", "SERVICE", map[string]interface{}{
			"operation":     "permission_not_found",
			"permission_id": permissionID,
			"timestamp":     logger.NowFormatted(),
		})
		return nil, errors.New("权限不存在")
	}

	logger.LogBusinessOperation("get_permission_by_id", permissionID, permission.Name, "", "", "success", "权限信息获取成功", map[string]interface{}{
		"permission_id": permissionID,
		"name":          permission.Name,
		"resource":      permission.Resource,
		"action":        permission.Action,
		"timestamp":     logger.NowFormatted(),
	})

	return permission, nil
}

// GetPermissionByName 根据名称获取权限
func (s *PermissionService) GetPermissionByName(ctx context.Context, name string) (*model.Permission, error) {
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
func (s *PermissionService) GetPermissionList(ctx context.Context, offset, limit int) ([]*model.Permission, int64, error) {
	originalOffset := offset
	originalLimit := limit
	if offset < 0 {
		logger.LogError(fmt.Errorf("invalid offset parameter: %d", offset), "", 0, "", "get_permission_list", "SERVICE", map[string]interface{}{
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
		logger.LogBusinessOperation("get_permission_list", 0, "system", "", "", "parameter_corrected", "分页参数已自动修正", map[string]interface{}{
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
		logger.LogError(err, "", 0, "", "get_permission_list", "SERVICE", map[string]interface{}{
			"operation": "get_permission_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, fmt.Errorf("failed to get permission list from repository: %w", err)
	}
	if permissions == nil {
		permissions = make([]*model.Permission, 0)
	}

	logger.LogBusinessOperation("get_permission_list", 0, "system", "", "", "success", "获取权限列表成功", map[string]interface{}{
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
func (s *PermissionService) UpdatePermissionByID(ctx context.Context, permissionID uint, req *model.UpdatePermissionRequest) (*model.Permission, error) {
	if err := s.validateUpdatePermissionParams(permissionID, req); err != nil {
		return nil, err
	}
	permission, err := s.validatePermissionForUpdate(ctx, permissionID, req)
	if err != nil {
		return nil, err
	}
	return s.executePermissionUpdate(ctx, permission, req)
}

func (s *PermissionService) validateUpdatePermissionParams(permissionID uint, req *model.UpdatePermissionRequest) error {
	if permissionID == 0 {
		logger.LogError(errors.New("invalid permission ID for update"), "", 0, "", "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "parameter_validation",
			"permission_id": permissionID,
			"error":         "permission_id_zero",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("权限ID不能为0")
	}
	if req == nil {
		logger.LogError(errors.New("update request is nil"), "", 0, "", "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "parameter_validation",
			"permission_id": permissionID,
			"error":         "request_nil",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("更新权限请求不能为空")
	}
	return nil
}

func (s *PermissionService) validatePermissionForUpdate(ctx context.Context, permissionID uint, req *model.UpdatePermissionRequest) (*model.Permission, error) {
	// 权限存在校验(permission_id有效性)
	permission, err := s.permissionRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		logger.LogError(err, "", 0, "", "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_existence_check",
			"permission_id": permissionID,
			"error":         "database_query_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取权限失败: %w", err)
	}
	if permission == nil {
		logger.LogError(errors.New("permission not found for update"), "", 0, "", "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_existence_check",
			"permission_id": permissionID,
			"error":         "permission_not_found",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, errors.New("权限不存在")
	}

	// 系统权限保护机制
	if permission.ID == 1 {
		logger.LogError(errors.New("system permission cannot be updated"), "", 0, "", "update_permission", "SERVICE", map[string]interface{}{
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
		logger.LogError(err, "", 0, "", "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_name_conflict_check",
			"permission_id": permissionID,
			"error":         "database_query_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, fmt.Errorf("检查权限名称冲突失败: %w", err)
	}
	if exists {
		// 请求中的新名字与数据库中的名字相同，则不进行更新[只修改其他属性可以不携带name字段]
		logger.LogError(errors.New("permission name already exists"), "", 0, "", "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_name_conflict_check",
			"permission_id": permissionID,
			"error":         "permission_name_conflict",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, errors.New("权限名称冲突")
	}

	return permission, nil
}

func (s *PermissionService) executePermissionUpdate(ctx context.Context, permission *model.Permission, req *model.UpdatePermissionRequest) (*model.Permission, error) {
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
		logger.LogError(err, "", 0, "", "update_permission", "SERVICE", map[string]interface{}{
			"operation":     "database_update",
			"permission_id": permission.ID,
			"error":         "update_permission_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return nil, fmt.Errorf("更新权限失败: %w", err)
	}

	logger.LogBusinessOperation("update_permission", permission.ID, permission.Name, "", "", "success", "权限更新成功", map[string]interface{}{
		"permission_id": permission.ID,
		"name":          permission.Name,
		"updated_at":    time.Now().Format(time.RFC3339),
		"timestamp":     logger.NowFormatted(),
	})

	return permission, nil
}

// DeletePermission 删除权限（含级联清理关联，但不做角色业务逻辑）
func (s *PermissionService) DeletePermission(ctx context.Context, permissionID uint) error {
	if permissionID == 0 {
		logger.LogError(errors.New("invalid permission ID for deletion"), "", 0, "", "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "parameter_validation",
			"permission_id": permissionID,
			"error":         "permission_id_zero",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("权限ID不能为0")
	}

	permission, err := s.permissionRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_existence_check",
			"permission_id": permissionID,
			"error":         "database_query_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return fmt.Errorf("获取权限失败: %w", err)
	}
	if permission == nil {
		logger.LogError(errors.New("permission not found for deletion"), "", 0, "", "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "permission_existence_check",
			"permission_id": permissionID,
			"error":         "permission_not_found",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("权限不存在")
	}

	// 事务：删除权限关联，再硬删除权限
	tx := s.permissionRepo.BeginTx(ctx)
	if tx == nil {
		logger.LogError(errors.New("failed to begin transaction"), "", 0, "", "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "transaction_begin",
			"permission_id": permissionID,
			"error":         "transaction_begin_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return errors.New("开始事务失败")
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.LogError(fmt.Errorf("panic during permission deletion: %v", r), "", 0, "", "delete_permission", "SERVICE", map[string]interface{}{
				"operation":     "panic_recovery",
				"permission_id": permissionID,
				"error":         "panic_occurred",
				"panic":         r,
				"timestamp":     logger.NowFormatted(),
			})
		}
	}()

	if err := s.permissionRepo.DeleteRolePermissionsByPermissionID(ctx, tx, permissionID); err != nil {
		tx.Rollback()
		logger.LogError(err, "", 0, "", "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "cascade_delete_role_permissions",
			"permission_id": permissionID,
			"error":         "delete_role_permissions_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return fmt.Errorf("删除权限与角色关联失败: %w", err)
	}

	if err := s.permissionRepo.DeletePermission(ctx, permissionID); err != nil {
		tx.Rollback()
		logger.LogError(err, "", 0, "", "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "soft_delete_permission",
			"permission_id": permissionID,
			"error":         "delete_permission_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return fmt.Errorf("删除权限失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		logger.LogError(err, "", 0, "", "delete_permission", "SERVICE", map[string]interface{}{
			"operation":     "transaction_commit",
			"permission_id": permissionID,
			"error":         "transaction_commit_failed",
			"timestamp":     logger.NowFormatted(),
		})
		return fmt.Errorf("提交事务失败: %w", err)
	}

	logger.LogBusinessOperation("delete_permission", permission.ID, permission.Name, "", "", "success", "权限删除成功", map[string]interface{}{
		"operation":     "permission_deletion_success",
		"permission_id": permission.ID,
		"name":          permission.Name,
		"deleted_at":    logger.NowFormatted(),
		"timestamp":     logger.NowFormatted(),
	})
	return nil
}

// GetPermissionWithRoles 获取权限及其关联角色（只读）
func (s *PermissionService) GetPermissionWithRoles(ctx context.Context, permissionID uint) (*model.Permission, error) {
	if permissionID == 0 {
		return nil, errors.New("权限ID不能为0")
	}
	return s.permissionRepo.GetPermissionWithRoles(ctx, permissionID)
}

// GetPermissionRoles 获取权限关联的角色（只读）
func (s *PermissionService) GetPermissionRoles(ctx context.Context, permissionID uint) ([]*model.Role, error) {
	if permissionID == 0 {
		return nil, errors.New("权限ID不能为0")
	}
	return s.permissionRepo.GetPermissionRoles(ctx, permissionID)
}

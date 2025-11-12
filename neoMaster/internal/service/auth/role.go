/*
 * @author: sun977
 * @date: 2025.09.11
 * @description: 角色务业务逻辑(角色自身的增删改查)
 * @func:
 * 1.创建角色
 * 2.更新角色
 * 3.删除角色
 * 4.角色状态变更等
 */

//  角色管理:
//  	CreateRole - 创建角色（包含权限分配）
//  	GetRoleByID - 根据ID获取角色
//  	GetRoleByName - 根据角色名获取角色
//  	GetRoleList - 分页获取角色列表
//  	UpdateRoleByID - 更新角色信息（包含权限更新）
//  	DeleteRole - 删除角色（包含级联删除）
//  状态管理:
//  	UpdateRoleStatus - 通用状态更新函数
//  	ActivateRole - 激活角色
//  	DeactivateRole - 禁用角色
//  权限管理:
//  	GetRoleWithPermissions - 获取角色及其权限
//  	GetRolePermissions - 获取角色权限
//  	AssignPermissionToRole - 为角色分配权限
//  	RemovePermissionFromRole - 移除角色权限

package auth

import (
	"context"
	"errors"
	"fmt"
	"neomaster/internal/model/system"
	systemrepo "neomaster/internal/repo/mysql/system"
	"time"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
)

// RoleService 角色服务
// 负责角色相关的业务逻辑，包括角色创建、获取角色信息等
type RoleService struct {
	roleRepo *systemrepo.RoleRepository // 角色数据仓库
}

// NewRoleService 创建新的角色服务实例
func NewRoleService(roleRepo *systemrepo.RoleRepository) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
	}
}

// CreateRole 创建角色
// 处理角色创建的完整流程，包括参数验证、重复检查、权限分配等
func (s *RoleService) CreateRole(ctx context.Context, req *system.CreateRoleRequest) (*system.Role, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	// 参数验证
	if req == nil {
		logger.LogBusinessError(errors.New("request is nil"), "", 0, clientIP, "role_create", "POST", map[string]interface{}{
			"operation": "create_role",
			"error":     "request is nil",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("创建角色请求不能为空")
	}

	if req.Name == "" {
		logger.LogBusinessError(errors.New("role name is empty"), "", 0, clientIP, "role_create", "POST", map[string]interface{}{
			"operation": "create_role",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("角色名称不能为空")
	}

	// 检查角色名是否已存在
	existingRole, err := s.roleRepo.GetRoleByName(ctx, req.Name)
	if err == nil && existingRole != nil {
		logger.LogBusinessError(errors.New("role name already exists"), "", 0, clientIP, "role_create", "POST", map[string]interface{}{
			"operation":        "create_role",
			"name":             req.Name,
			"existing_role_id": existingRole.ID,
			"timestamp":        logger.NowFormatted(),
		})
		return nil, errors.New("角色名称已存在")
	}

	// 创建角色模型
	role := &system.Role{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Status:      system.RoleStatusEnabled, // 默认启用状态
	}

	// 存储到数据库
	err = s.roleRepo.CreateRole(ctx, role)
	if err != nil {
		logger.LogBusinessError(err, "", 0, clientIP, "role_create", "POST", map[string]interface{}{
			"operation": "create_role_db",
			"name":      req.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}

	// 分配权限（如果有指定权限）
	if len(req.PermissionIDs) > 0 {
		for _, permissionID := range req.PermissionIDs {
			if err := s.roleRepo.AssignPermissionToRole(ctx, role.ID, permissionID); err != nil {
				// 记录权限分配失败日志，但不影响角色创建
				logger.LogBusinessError(err, "", 0, clientIP, "role_create", "POST", map[string]interface{}{
					"operation":     "assign_permission_to_role",
					"role_id":       role.ID,
					"permission_id": permissionID,
					"timestamp":     logger.NowFormatted(),
				})
				// return nil, errors.New("权限分配失败")
			}
		}
	}

	// 记录成功创建角色的业务日志
	logger.LogBusinessOperation("create_role", 0, "system", clientIP, "", "success", "Role created successfully", map[string]interface{}{
		"name":             role.Name,
		"role_id":          role.ID,
		"display_name":     role.DisplayName,
		"status":           role.Status,
		"permission_count": len(req.PermissionIDs),
		"timestamp":        logger.NowFormatted(),
	})

	return role, nil
}

// GetRoleByID 根据角色ID获取角色
// 完整的业务逻辑包括：参数验证、上下文检查、数据获取、状态验证、日志记录
func (s *RoleService) GetRoleByID(ctx context.Context, roleID uint) (*system.Role, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	// 参数验证：角色ID必须有效
	if roleID == 0 {
		logger.LogBusinessError(errors.New("invalid role ID: cannot be zero"), "", 0, clientIP, "get_role_by_id", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("角色ID不能为0")
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 从数据库获取角色信息
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		// 记录数据库查询失败日志
		logger.LogBusinessError(err, "", roleID, clientIP, "get_role_by_id", "SERVICE", map[string]interface{}{
			"operation": "database_query",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取角色信息失败: %w", err)
	}

	// 检查角色是否存在
	if role == nil {
		// 记录角色不存在日志
		logger.LogBusinessError(errors.New("role not found"), "", 0, clientIP, "get_role_by_id", "SERVICE", map[string]interface{}{
			"operation": "role_not_found",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("角色不存在")
	}

	// 记录成功获取角色信息的业务日志
	logger.LogBusinessOperation("get_role_by_id", 0, "system", clientIP, "", "success", "角色信息获取成功", map[string]interface{}{
		"operation":   "get_role_success",
		"role_id":     roleID,
		"name":        role.Name,
		"role_status": role.Status,
		"timestamp":   logger.NowFormatted(),
	})

	return role, nil
}

// GetRoleByName 根据角色名获取角色
func (s *RoleService) GetRoleByName(ctx context.Context, name string) (*system.Role, error) {
	if name == "" {
		return nil, errors.New("角色名称不能为空")
	}

	role, err := s.roleRepo.GetRoleByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("角色不存在")
	}

	return role, nil
}

// GetRoleList 获取角色列表
// 提供分页查询功能，包含完整的参数验证和错误处理
func (s *RoleService) GetRoleList(ctx context.Context, offset, limit int) ([]*system.Role, int64, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	// 保存原始参数值用于日志记录
	originalOffset := offset
	originalLimit := limit

	// 参数验证：偏移量不能为负数
	if offset < 0 {
		logger.LogBusinessError(fmt.Errorf("invalid offset parameter: %d", offset), "", 0, clientIP, "get_role_list", "SERVICE", map[string]interface{}{
			"operation": "get_role_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		offset = 0 // 自动修正为0
	}

	// 参数验证：限制每页数量的合理范围
	if limit <= 0 {
		limit = 20 // 默认每页20条
	} else if limit > 100 {
		limit = 100 // 最大每页100条，防止查询过大数据集
	}

	// 记录参数修正日志（如果发生了修正）
	if originalLimit != limit || originalOffset != offset {
		logger.LogBusinessOperation("get_role_list", 0, "system", clientIP, "", "parameter_corrected", "分页参数已自动修正", map[string]interface{}{
			"operation":        "get_role_list",
			"original_offset":  originalOffset,
			"original_limit":   originalLimit,
			"corrected_offset": offset,
			"corrected_limit":  limit,
			"timestamp":        logger.NowFormatted(),
		})
	}

	// 上下文检查：确保请求未被取消
	select {
	case <-ctx.Done():
		return nil, 0, fmt.Errorf("request cancelled: %w", ctx.Err())
	default:
		// 继续执行
	}

	// 调用repository层获取数据
	roles, total, err := s.roleRepo.GetRoleList(ctx, offset, limit)
	if err != nil {
		// 记录数据库查询错误
		logger.LogBusinessError(err, "", 0, clientIP, "get_role_list", "SERVICE", map[string]interface{}{
			"operation": "get_role_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, fmt.Errorf("failed to get role list from repo: %w", err)
	}

	// 数据完整性检查
	if roles == nil {
		roles = make([]*system.Role, 0) // 确保返回空切片而不是nil
	}

	// 记录成功操作日志
	logger.LogBusinessOperation("get_role_list", 0, "system", clientIP, "", "success", "获取角色列表成功", map[string]interface{}{
		"operation":    "get_role_list",
		"offset":       offset,
		"limit":        limit,
		"total":        total,
		"result_count": len(roles),
		"timestamp":    logger.NowFormatted(),
	})

	return roles, total, nil
}

// UpdateRoleByID 更新角色信息
// 处理角色更新的完整流程，包括参数验证、重复检查、权限更新、事务处理等
func (s *RoleService) UpdateRoleByID(ctx context.Context, roleID uint, req *system.UpdateRoleRequest) (*system.Role, error) {
	// 第一层：参数验证层
	if err := s.validateUpdateRoleParams(ctx, roleID, req); err != nil {
		// roleID 不能为0
		// 请求包 req 不能为空
		// 角色状态验证(0-禁用,1-启用)
		return nil, err
	}

	// 第二层：业务规则验证层
	role, err := s.validateRoleForUpdate(ctx, roleID, req)
	if err != nil {
		// 角色是否存在
		// 角色软删除状态不能更新(未启用)
		// 业务规则1：角色保护,系统管理员角色不能被更新
		// 角色名称冲突校验,不能重复
		// 权限id的有效性校验
		return nil, err
	}

	// 第三层：事务处理层
	return s.executeRoleUpdate(ctx, role, req)
}

// validateUpdateRoleParams 验证更新角色的参数
func (s *RoleService) validateUpdateRoleParams(ctx context.Context, roleID uint, req *system.UpdateRoleRequest) error {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)

	if roleID == 0 {
		logger.LogBusinessError(errors.New("invalid role ID for update"), "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"role_id":   roleID,
			"error":     "role_id_zero",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("role ID cannot be zero")
	}

	if req == nil {
		logger.LogBusinessError(errors.New("update request is nil"), "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"role_id":   roleID,
			"error":     "request_nil",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("update role request is nil")
	}

	// 验证状态值
	if req.Status != nil {
		if *req.Status < 0 || *req.Status > 1 {
			logger.LogBusinessError(errors.New("invalid status value"), "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
				"operation": "parameter_validation",
				"role_id":   roleID,
				"status":    *req.Status,
				"error":     "invalid_status_value",
				"timestamp": logger.NowFormatted(),
			})
			return errors.New("role status value invalid, must be 0 (disabled) or 1 (enabled)")
		}
	}

	return nil
}

// validateRoleForUpdate 验证角色是否可以更新
func (s *RoleService) validateRoleForUpdate(ctx context.Context, roleID uint, req *system.UpdateRoleRequest) (*system.Role, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	// 检查角色是否存在
	// role, err := s.roleRepo.GetRoleByID(ctx, roleID) [GetRoleByID 只能获取角色的基本信息,不包含权限信息]
	// [数据库中role表本身不带权限信息,角色权限关联信息在role_permissions表里,需要通过GetRoleWithPermissions方法获取角色及其关联的权限]
	// model.Role 模型中带有 permissions 字段列表
	roleWithPermissions, err := s.roleRepo.GetRoleWithPermissions(ctx, roleID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "role_existence_check",
			"role_id":   roleID,
			"error":     "database_query_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("failed to get role with permissions: %w", err)
	}

	if roleWithPermissions == nil {
		logger.LogBusinessError(errors.New("role not found for update"), "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "role_existence_check",
			"role_id":   roleID,
			"error":     "role_not_found",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("role id not found")
	}

	// 检查角色状态 - 已删除的角色不能更新[GetRoleWithPermissions 方法没有返回角色的软删除状态,所以此方法不可用]
	// if role.DeletedAt != nil {
	// 	logger.LogError(errors.New("role already deleted"), "", 0, "", "update_role", "SERVICE", map[string]interface{}{
	// 		"operation": "role_status_check",
	// 		"role_id":   roleID,
	// 		"error":     "role_already_deleted",
	// 		"timestamp": logger.NowFormatted(),
	// 	})
	// 	return nil, errors.New("角色已被删除，无法更新")
	// }

	// 角色名冲突校验
	roleNameConflict, err := s.roleRepo.GetRoleByName(ctx, req.Name)
	if err != nil {
		logger.LogBusinessError(err, "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "role_name_conflict_check",
			"role_id":   roleID,
			"error":     "database_query_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}
	if roleNameConflict != nil && roleNameConflict.ID != roleID {
		logger.LogBusinessError(errors.New("role name already exists"), "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "role_name_conflict_check",
			"role_id":   roleID,
			"error":     "role_name_conflict",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("role name already exists")
	}

	// 权限id的有效性校验[应该校验权限是否存在]
	if req.PermissionIDs != nil {
		for _, permissionID := range req.PermissionIDs {
			permissionExists, err := s.roleRepo.RolePermissionExists(ctx, permissionID) // 检查角色关联的权限是否存在
			if err != nil {
				logger.LogBusinessError(err, "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
					"operation": "permission_existence_check",
					"role_id":   roleID,
					"error":     "database_query_failed",
					"timestamp": logger.NowFormatted(),
				})
				return nil, fmt.Errorf("failed to check permission existence: %w", err)
			}
			if !permissionExists {
				logger.LogBusinessError(errors.New("permission not found"), "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
					"operation": "permission_existence_check",
					"role_id":   roleID,
					"error":     "permission_not_found",
					"timestamp": logger.NowFormatted(),
				})
				return nil, fmt.Errorf("permission not found: %d", permissionID)
			}
		}
	}

	// 业务规则：系统角色保护机制（可以根据需要添加）
	// 例如：某些系统内置角色不能被修改(角色1为系统管理员角色)
	if roleID == 1 {
		logger.LogBusinessError(errors.New("system role cannot be updated"), "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "business_rule_check",
			"role_id":   roleID,
			"error":     "system_role_update_forbidden",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("system role cannot be updated")
	}

	return roleWithPermissions, nil
}

// executeRoleUpdate 执行角色更新操作（包含事务处理）
func (s *RoleService) executeRoleUpdate(ctx context.Context, role *system.Role, req *system.UpdateRoleRequest) (*system.Role, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	// 开始事务
	tx := s.roleRepo.BeginTx(ctx)
	if tx == nil {
		logger.LogBusinessError(errors.New("failed to begin transaction"), "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "transaction_begin",
			"role_id":   role.ID,
			"error":     "transaction_begin_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("开始事务失败")
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.LogBusinessError(fmt.Errorf("panic during role update: %v", r), "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
				"operation": "panic_recovery",
				"role_id":   role.ID,
				"error":     "panic_occurred",
				"panic":     r,
				"timestamp": logger.NowFormatted(),
			})
		}
	}()

	// 记录更新前的状态
	oldStatus := role.Status
	permissionsChanged := false

	// 应用更新
	if req.Name != "" && req.Name != role.Name {
		role.Name = req.Name
	}
	if req.DisplayName != "" && req.DisplayName != role.DisplayName {
		role.DisplayName = req.DisplayName
	}
	if req.Description != "" && req.Description != role.Description {
		role.Description = req.Description
	}
	if req.Status != nil && *req.Status != role.Status {
		role.Status = *req.Status
	}

	// 更新权限（如果有指定）
	if req.PermissionIDs != nil {
		// 获取请求中权限信息到角色结构体 (不能直接将req的[]uint 赋给role的[]model.Permission，需要转换)
		permissions := make([]system.Permission, len(req.PermissionIDs))
		for i, id := range req.PermissionIDs {
			permissions[i] = system.Permission{ID: id}
		}
		// 手动创建permission对象(结构体)，并将req.PermissionIDs中的uint转换为Permission结构体，最后赋值给role.Permissions
		role.Permissions = permissions

		// 删除现有权限关联
		if err := s.roleRepo.DeleteRolePermissionsByRoleID(ctx, tx, role.ID); err != nil {
			tx.Rollback()
			logger.LogBusinessError(err, "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
				"operation": "delete_role_permissions",
				"role_id":   role.ID,
				"error":     "delete_permissions_failed",
				"timestamp": logger.NowFormatted(),
			})
			return nil, fmt.Errorf("failed to delete role permissions: %w", err)
		}

		// 然后创建新的role_permissions关联表记录(后续操作 UpdateRoleWithTx 会创建新的关联 借助GORM的特性实现的)
		// bug 请求中携带没有的permissionID时，会创建role_permissions关联表记录，同时在的permissions表中创建新的permissionID记录
		permissionsChanged = true
	}

	// 更新到数据库
	if err := s.roleRepo.UpdateRoleWithTx(ctx, tx, role); err != nil {
		tx.Rollback()
		logger.LogBusinessError(err, "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "database_update",
			"role_id":   role.ID,
			"error":     "update_role_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.LogBusinessError(err, "", 0, clientIP, "update_role", "SERVICE", map[string]interface{}{
			"operation": "transaction_commit",
			"role_id":   role.ID,
			"error":     "transaction_commit_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 记录成功更新日志
	changes := make(map[string]interface{})
	if req.Status != nil && *req.Status != oldStatus {
		changes["status_changed"] = map[string]int{"from": int(oldStatus), "to": int(*req.Status)}
	}
	if permissionsChanged {
		changes["permissions_changed"] = true
		changes["permission_count"] = len(req.PermissionIDs)
	}

	logger.LogBusinessOperation("update_role", 0, "", clientIP, "", "success", "角色更新成功", map[string]interface{}{
		"operation":  "role_update_success",
		"role_id":    role.ID,
		"role_name":  role.Name,
		"status":     role.Status,
		"changes":    changes,
		"updated_at": logger.NowFormatted(),
		"timestamp":  logger.NowFormatted(),
	})

	return role, nil
}

// DeleteRole 删除角色
// 完整的业务逻辑包括：参数验证、业务规则检查、级联删除、事务处理、审计日志
func (s *RoleService) DeleteRole(ctx context.Context, roleID uint) error {
	// 第一层：参数验证层
	if err := s.validateDeleteRoleParams(ctx, roleID); err != nil {
		// roleID 不为 0
		return err
	}

	// 第二层：业务规则验证层
	role, err := s.validateRoleForDeletion(ctx, roleID)
	if err != nil {
		// 检查角色是否存在
		// 检查角色状态 - 已删除的角色不能再次删除
		// 系统管理员角色不能删除
		return err
	}

	// 第三层：事务处理层
	return s.executeRoleDeletion(ctx, role)
}

// validateDeleteRoleParams 验证删除角色的参数
func (s *RoleService) validateDeleteRoleParams(ctx context.Context, roleID uint) error {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	if roleID == 0 {
		logger.LogBusinessError(errors.New("invalid role ID for deletion"), "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"role_id":   roleID,
			"error":     "role_id_zero",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("角色ID不能为0")
	}
	return nil
}

// validateRoleForDeletion 验证角色是否可以删除
func (s *RoleService) validateRoleForDeletion(ctx context.Context, roleID uint) (*system.Role, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	// 检查角色是否存在
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
			"operation": "role_existence_check",
			"role_id":   roleID,
			"error":     "database_query_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取角色失败: %w", err)
	}

	if role == nil {
		logger.LogBusinessError(errors.New("role not found for deletion"), "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
			"operation": "role_existence_check",
			"role_id":   roleID,
			"error":     "role_not_found",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("角色不存在")
	}

	// 检查角色状态 - 已删除的角色不能再次删除
	if role.DeletedAt != nil {
		logger.LogBusinessError(errors.New("role already deleted"), "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
			"operation": "role_status_check",
			"role_id":   roleID,
			"error":     "role_already_deleted",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("角色已被删除")
	}

	// 业务规则：系统角色保护机制（可以根据需要添加）
	// 例如：某些系统内置角色不能被删除
	if roleID == 1 {
		logger.LogBusinessError(errors.New("system role cannot be deleted"), "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
			"operation": "business_rule_check",
			"role_id":   roleID,
			"error":     "system_role_delete_forbidden",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("系统角色不能被删除")
	}

	return role, nil
}

// executeRoleDeletion 执行角色删除操作（包含事务处理）
func (s *RoleService) executeRoleDeletion(ctx context.Context, role *system.Role) error {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	// 开始事务
	tx := s.roleRepo.BeginTx(ctx)
	if tx == nil {
		logger.LogBusinessError(errors.New("failed to begin transaction"), "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
			"operation": "transaction_begin",
			"role_id":   role.ID,
			"error":     "transaction_begin_failed",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("开始事务失败")
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.LogBusinessError(fmt.Errorf("panic during role deletion: %v", r), "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
				"operation": "panic_recovery",
				"role_id":   role.ID,
				"error":     "panic_occurred",
				"panic":     r,
				"timestamp": logger.NowFormatted(),
			})
		}
	}()

	// 1. 删除角色权限关联[硬删除]
	if err := s.roleRepo.DeleteRolePermissionsByRoleID(ctx, tx, role.ID); err != nil {
		tx.Rollback()
		logger.LogBusinessError(err, "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
			"operation": "cascade_delete_role_permissions",
			"role_id":   role.ID,
			"error":     "delete_role_permissions_failed",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("删除角色权限关联失败: %w", err)
	}

	// 2. 硬删除角色
	if err := s.roleRepo.DeleteRoleWithTx(ctx, tx, role.ID); err != nil {
		tx.Rollback()
		logger.LogBusinessError(err, "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
			"operation": "soft_delete_role",
			"role_id":   role.ID,
			"error":     "delete_role_failed",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("删除角色失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.LogBusinessError(err, "", 0, clientIP, "delete_role", "SERVICE", map[string]interface{}{
			"operation": "transaction_commit",
			"role_id":   role.ID,
			"error":     "transaction_commit_failed",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 记录成功删除日志
	logger.LogBusinessOperation("delete_role", 0, "", clientIP, "", "success", "角色删除成功", map[string]interface{}{
		"operation":  "role_deletion_success",
		"role_id":    role.ID,
		"role_name":  role.Name,
		"deleted_at": logger.NowFormatted(),
		"timestamp":  logger.NowFormatted(),
	})

	return nil
}

// UpdateRoleStatus 更新角色状态 - 通用状态管理函数
// 将指定角色的状态设置为启用或禁用状态，消除重复代码，体现"好品味"原则
func (s *RoleService) UpdateRoleStatus(ctx context.Context, roleID uint, status system.RoleStatus) error {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	// 参数验证层 - 消除特殊情况
	if roleID == 0 {
		logger.LogBusinessError(errors.New("invalid role ID"), "", 0, clientIP, "update_role_status", "SERVICE", map[string]interface{}{
			"operation": "update_role_status",
			"error":     "invalid_role_id",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("角色ID不能为0")
	}

	// 验证状态值有效性 - 严格的参数检查
	if status != system.RoleStatusEnabled && status != system.RoleStatusDisabled {
		logger.LogBusinessError(errors.New("invalid status value"), "", 0, clientIP, "update_role_status", "SERVICE", map[string]interface{}{
			"operation": "update_role_status",
			"error":     "invalid_status_value",
			"role_id":   roleID,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("角色状态值无效,必须为0(禁用)或1(启用)")
	}

	// 业务规则验证层 - 检查角色是否存在
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		logger.LogBusinessError(err, "", roleID, clientIP, "update_role_status", "SERVICE", map[string]interface{}{
			"operation": "update_role_status",
			"error":     "get_role_failed",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取角色信息失败: %w", err)
	}

	if role == nil {
		logger.LogBusinessError(errors.New("role not found"), "", roleID, clientIP, "update_role_status", "SERVICE", map[string]interface{}{
			"operation": "update_role_status",
			"error":     "role_not_found",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("角色不存在")
	}

	// 幂等性检查 - 避免无意义操作
	if role.Status == status {
		statusText := "禁用"
		if status == system.RoleStatusEnabled {
			statusText = "启用"
		}

		logger.LogBusinessOperation("update_role_status", roleID, role.Name, clientIP, "", "success",
			fmt.Sprintf("角色已处于%s状态", statusText), map[string]interface{}{
				"operation":      "update_role_status",
				"role_id":        roleID,
				"name":           role.Name,
				"current_status": status,
				"target_status":  status,
				"timestamp":      logger.NowFormatted(),
			})
		return nil
	}

	// 数据操作层 - 执行状态更新
	// 使用 UpdateRoleFields 进行原子更新操作
	updateFields := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	err = s.roleRepo.UpdateRoleFields(ctx, roleID, updateFields)
	if err != nil {
		statusText := "禁用"
		if status == system.RoleStatusEnabled {
			statusText = "启用"
		}

		logger.LogBusinessError(err, "", roleID, clientIP, "update_role_status", "SERVICE", map[string]interface{}{
			"operation": "update_role_status",
			"error":     fmt.Sprintf("%s_failed", statusText),
			"role_id":   roleID,
			"name":      role.Name,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("%s角色失败: %w", statusText, err)
	}

	// 审计日志层 - 记录成功操作
	statusText := "禁用"
	statusTextOpposite := "启用"
	if status == system.RoleStatusEnabled {
		statusText = "启用"
		statusTextOpposite = "禁用"
	}

	logger.LogBusinessOperation("update_role_status", 0, "", clientIP, "", "success",
		fmt.Sprintf("角色%s成功", statusText), map[string]interface{}{
			"operation":       "update_role_status",
			"role_id":         roleID,
			"role_name":       role.Name,
			"previous_status": statusTextOpposite,
			"new_status":      statusText,
			"target_status":   int(status),
			"timestamp":       logger.NowFormatted(),
		})

	return nil
}

// ActivateRole 激活角色 - 语义化包装函数，保持向后兼容
// 将指定角色的状态设置为启用状态
func (s *RoleService) ActivateRole(ctx context.Context, roleID uint) error {
	// 调用通用状态更新函数，体现"好品味"原则：消除特殊情况
	return s.UpdateRoleStatus(ctx, roleID, system.RoleStatusEnabled)
}

// DeactivateRole 禁用角色 - 语义化包装函数
// 将指定角色的状态设置为禁用状态
func (s *RoleService) DeactivateRole(ctx context.Context, roleID uint) error {
	// 调用通用状态更新函数，体现"好品味"原则：消除特殊情况

	// 系统角色禁止禁用
	if roleID == 1 {
		return errors.New("系统角色禁止禁用")
	}

	return s.UpdateRoleStatus(ctx, roleID, system.RoleStatusDisabled)
}

// GetRoleWithPermissions 获取角色及其权限
func (s *RoleService) GetRoleWithPermissions(ctx context.Context, roleID uint) (*system.Role, error) {
	// 从标准上下文中 context 获取必要的信息[已在中间件中做过标准化处理]
	clientIP := utils.GetClientIPFromContext(ctx)
	if roleID == 0 {
		return nil, errors.New("角色ID不能为0")
	}

	// 从数据库获取角色信息
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		// 记录数据库查询失败日志
		logger.LogBusinessError(err, "", roleID, clientIP, "get_role_by_id", "SERVICE", map[string]interface{}{
			"operation": "database_query",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取角色信息失败: %w", err)
	}

	// 检查角色是否存在
	if role == nil {
		// 记录角色不存在日志
		logger.LogBusinessError(errors.New("role not found"), "", roleID, clientIP, "get_role_by_id", "SERVICE", map[string]interface{}{
			"operation": "role_not_found",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("角色不存在")
	}

	return s.roleRepo.GetRoleWithPermissions(ctx, roleID)
}

// GetRolePermissions 获取角色权限
func (s *RoleService) GetRolePermissions(ctx context.Context, roleID uint) ([]*system.Permission, error) {
	if roleID == 0 {
		return nil, errors.New("角色ID不能为0")
	}

	return s.roleRepo.GetRolePermissions(ctx, roleID)
}

// AssignPermissionToRole 为角色分配权限
func (s *RoleService) AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error {
	// 参数验证
	if roleID == 0 {
		return errors.New("角色ID不能为0")
	}
	if permissionID == 0 {
		return errors.New("权限ID不能为0")
	}

	// 调用数据访问层分配权限
	return s.roleRepo.AssignPermissionToRole(ctx, roleID, permissionID)
}

// RemovePermissionFromRole 移除角色权限
func (s *RoleService) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error {
	// 参数验证
	if roleID == 0 {
		return errors.New("角色ID不能为0")
	}
	if permissionID == 0 {
		return errors.New("权限ID不能为0")
	}

	// 调用数据访问层移除权限
	return s.roleRepo.RemovePermissionFromRole(ctx, roleID, permissionID)
}

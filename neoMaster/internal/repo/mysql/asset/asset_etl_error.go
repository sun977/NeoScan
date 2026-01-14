package asset

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	assetModel "neomaster/internal/model/asset"
)

// ETLErrorRepository ETL 错误日志仓库接口
type ETLErrorRepository interface {
	Create(ctx context.Context, errLog *assetModel.AssetETLError) error
	UpdateStatus(ctx context.Context, id uint64, status string) error
	GetByID(ctx context.Context, id uint64) (*assetModel.AssetETLError, error)
	// FindPending 获取待处理的错误 (status=new/retrying)
	FindPending(ctx context.Context, limit int) ([]*assetModel.AssetETLError, error)
	// GetByStatus 根据状态获取错误
	GetByStatus(ctx context.Context, status string, limit int) ([]*assetModel.AssetETLError, error)
	// List 获取错误列表 (带筛选)
	List(ctx context.Context, filter ETLErrorFilter) ([]*assetModel.AssetETLError, int64, error)
}

// ETLErrorFilter 错误筛选条件
type ETLErrorFilter struct {
	Page       int
	PageSize   int
	TaskID     string
	ResultType string
	Status     string
	ErrorStage string
	StartTime  string // 格式: 2006-01-02 15:04:05
	EndTime    string // 格式: 2006-01-02 15:04:05
}

type etlErrorRepository struct {
	db *gorm.DB
}

func NewETLErrorRepository(db *gorm.DB) ETLErrorRepository {
	return &etlErrorRepository{db: db}
}

func (r *etlErrorRepository) Create(ctx context.Context, errLog *assetModel.AssetETLError) error {
	if errLog == nil {
		return fmt.Errorf("error log cannot be nil")
	}
	return r.db.WithContext(ctx).Create(errLog).Error
}

func (r *etlErrorRepository) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return r.db.WithContext(ctx).Model(&assetModel.AssetETLError{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status": status,
		}).Error
}

func (r *etlErrorRepository) GetByID(ctx context.Context, id uint64) (*assetModel.AssetETLError, error) {
	var errLog assetModel.AssetETLError
	err := r.db.WithContext(ctx).First(&errLog, id).Error
	if err != nil {
		return nil, err
	}
	return &errLog, nil
}

func (r *etlErrorRepository) FindPending(ctx context.Context, limit int) ([]*assetModel.AssetETLError, error) {
	var logs []*assetModel.AssetETLError
	err := r.db.WithContext(ctx).
		Where("status IN ?", []string{"new", "retrying"}).
		Order("created_at ASC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (r *etlErrorRepository) GetByStatus(ctx context.Context, status string, limit int) ([]*assetModel.AssetETLError, error) {
	var logs []*assetModel.AssetETLError
	err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at ASC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (r *etlErrorRepository) List(ctx context.Context, filter ETLErrorFilter) ([]*assetModel.AssetETLError, int64, error) {
	var logs []*assetModel.AssetETLError
	var total int64

	query := r.db.WithContext(ctx).Model(&assetModel.AssetETLError{})

	if filter.TaskID != "" {
		query = query.Where("task_id = ?", filter.TaskID)
	}
	if filter.ResultType != "" {
		query = query.Where("result_type = ?", filter.ResultType)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ErrorStage != "" {
		query = query.Where("error_stage = ?", filter.ErrorStage)
	}
	if filter.StartTime != "" {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if filter.EndTime != "" {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	if offset < 0 {
		offset = 0
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	}

	err := query.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&logs).Error
	return logs, total, err
}

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

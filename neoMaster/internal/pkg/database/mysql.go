package database

import (
	"fmt"
	"time"

	"neomaster/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewMySQLConnection 创建MySQL数据库连接
func NewMySQLConnection(cfg *config.MySQLConfig) (*gorm.DB, error) {
	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
		cfg.ParseTime,
		cfg.Loc,
	)

	// 配置GORM日志级别
	var logLevel logger.LogLevel
	switch cfg.LogLevel {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Info
	}

	// 打开数据库连接
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// 获取底层的sql.DB对象来配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime))
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime))

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	return db, nil
}
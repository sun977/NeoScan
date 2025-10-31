// 日志管理器
package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"neomaster/internal/config"

	"github.com/sirupsen/logrus"
)

// LoggerManager 日志管理器
type LoggerManager struct {
	logger *logrus.Logger
	config *config.LogConfig
}

// LoggerInstance 全局日志实例
var LoggerInstance *LoggerManager

// InitLogger 初始化日志管理器
// 根据配置文件初始化logrus实例，支持多种输出方式和格式
func InitLogger(cfg *config.LogConfig) (*LoggerManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("log config cannot be nil")
	}

	// 创建logrus实例
	logger := logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		// 如果解析失败，默认使用info级别
		level = logrus.InfoLevel
		logger.Warnf("Invalid log level '%s', using 'info' as default", cfg.Level)
	}
	logger.SetLevel(level)

	// 设置日志格式
	if err := setLogFormatter(logger, cfg); err != nil {
		return nil, fmt.Errorf("failed to set log formatter: %w", err)
	}

	// 设置日志输出
	if err := setLogOutput(logger, cfg); err != nil {
		// 默认让hook机制处理所有日志 [neoMaster\internal\pkg\logger\hooks.go]
		return nil, fmt.Errorf("failed to set log output: %w", err)
	}

	// 添加FileHook以支持不同类型的日志输出到不同文件
	logger.AddHook(NewFileHook(cfg))

	// 设置调用者信息
	logger.SetReportCaller(cfg.Caller)

	// 创建日志管理器实例
	lm := &LoggerManager{
		logger: logger,
		config: cfg,
	}

	// 设置全局实例
	LoggerInstance = lm

	return lm, nil
}

// setLogFormatter 设置日志格式化器
func setLogFormatter(logger *logrus.Logger, cfg *config.LogConfig) error {
	// 定义合理精度的时间戳格式（精确到毫秒，不显示时区，使用空格分隔日期和时间|给管理器使用的时间戳格式）
	timestampFormat := "2006-01-02 15:04:05.000"

	switch strings.ToLower(cfg.Format) {
	case "json":
		// JSON格式化器，适合生产环境和日志分析
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: timestampFormat, // 使用毫秒精度时间戳
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
				logrus.FieldKeyFile:  "file",
			},
		})
	case "text":
		// 文本格式化器，适合开发环境和控制台输出
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: timestampFormat, // 使用毫秒精度时间戳
			FullTimestamp:   true,
			ForceColors:     true,
		})
	default:
		return fmt.Errorf("unsupported log format: %s", cfg.Format)
	}
	return nil
}

// setLogOutput 设置日志输出目标
// 使用Hook机制实现日志分离功能，将日志输出设置为io.Discard
// 实际的日志输出将由FileHook处理
func setLogOutput(logger *logrus.Logger, cfg *config.LogConfig) error {
	// 在调试模式下，同时输出到控制台和Hook机制
	// 在生产模式下，只使用Hook机制
	if strings.ToLower(cfg.Level) == "debug" {
		// 调试模式下创建一个MultiWriter，同时写入控制台和io.Discard
		// io.Discard确保日志主要通过Hook处理，控制台输出作为辅助
		multiWriter := io.MultiWriter(os.Stdout, io.Discard)
		logger.SetOutput(multiWriter)
	} else {
		// 非调试模式下只使用io.Discard，让Hook机制处理所有日志
		logger.SetOutput(io.Discard)
	}
	return nil
}

// GetLogger 获取logrus实例
func (lm *LoggerManager) GetLogger() *logrus.Logger {
	return lm.logger
}

// GetConfig 获取日志配置
func (lm *LoggerManager) GetConfig() *config.LogConfig {
	return lm.config
}

// UpdateConfig 更新日志配置
// 支持运行时动态更新日志配置
func (lm *LoggerManager) UpdateConfig(newCfg *config.LogConfig) error {
	if newCfg == nil {
		return fmt.Errorf("new config cannot be nil")
	}

	// 更新日志级别
	if newCfg.Level != lm.config.Level {
		level, err := logrus.ParseLevel(newCfg.Level)
		if err != nil {
			return fmt.Errorf("invalid log level: %w", err)
		}
		lm.logger.SetLevel(level)
		lm.logger.Infof("Log level updated from %s to %s", lm.config.Level, newCfg.Level)
	}

	// 更新日志格式
	if newCfg.Format != lm.config.Format {
		if err := setLogFormatter(lm.logger, newCfg); err != nil {
			return fmt.Errorf("failed to update log formatter: %w", err)
		}
		lm.logger.Infof("Log format updated from %s to %s", lm.config.Format, newCfg.Format)
	}

	// 更新日志输出
	if newCfg.Output != lm.config.Output || newCfg.FilePath != lm.config.FilePath {
		if err := setLogOutput(lm.logger, newCfg); err != nil {
			return fmt.Errorf("failed to update log output: %w", err)
		}
		lm.logger.Infof("Log output updated from %s to %s", lm.config.Output, newCfg.Output)
	}

	// 更新调用者信息
	if newCfg.Caller != lm.config.Caller {
		lm.logger.SetReportCaller(newCfg.Caller)
		lm.logger.Infof("Log caller reporting updated to %t", newCfg.Caller)
	}

	// 保存新配置
	lm.config = newCfg

	return nil
}

// 便捷方法：获取全局日志实例

// Debug 记录调试日志
func Debug(args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Debug(args...)
	}
}

// Debugf 记录格式化调试日志
func Debugf(format string, args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Debugf(format, args...)
	}
}

// Info 记录信息日志
func Info(args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Info(args...)
	}
}

// Infof 记录格式化信息日志
func Infof(format string, args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Infof(format, args...)
	}
}

// Warn 记录警告日志
func Warn(args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Warn(args...)
	}
}

// Warnf 记录格式化警告日志
func Warnf(format string, args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Warnf(format, args...)
	}
}

// Error 记录错误日志
func Error(args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Error(args...)
	}
}

// Errorf 记录格式化错误日志
func Errorf(format string, args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Errorf(format, args...)
	}
}

// Fatal 记录致命错误日志并退出程序
func Fatal(args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Fatal(args...)
	}
}

// Fatalf 记录格式化致命错误日志并退出程序
func Fatalf(format string, args ...interface{}) {
	if LoggerInstance != nil {
		LoggerInstance.logger.Fatalf(format, args...)
	}
}

// WithField 添加单个字段
func WithField(key string, value interface{}) *logrus.Entry {
	if LoggerInstance != nil {
		return LoggerInstance.logger.WithField(key, value)
	}
	return logrus.NewEntry(logrus.StandardLogger())
}

// WithFields 添加多个字段
func WithFields(fields logrus.Fields) *logrus.Entry {
	if LoggerInstance != nil {
		return LoggerInstance.logger.WithFields(fields)
	}
	return logrus.NewEntry(logrus.StandardLogger())
}

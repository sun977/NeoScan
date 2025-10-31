package logger

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"neomaster/internal/config"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// FileHook 是一个自定义Hook，用于将不同类型的日志写入不同的文件
type FileHook struct {
	logConfig *config.LogConfig
	writers   map[string]io.Writer
	formatter logrus.Formatter
	mutex     sync.Mutex
}

// NewFileHook 创建一个新的FileHook实例
func NewFileHook(logConfig *config.LogConfig) *FileHook {
	hook := &FileHook{
		logConfig: logConfig,
		writers:   make(map[string]io.Writer),
		formatter: &logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
				logrus.FieldKeyFile:  "file",
			},
		},
	}

	// 初始化默认writer
	hook.initDefaultWriter()

	return hook
}

// initDefaultWriter 初始化默认writer（主日志文件）
func (hook *FileHook) initDefaultWriter() {
	if hook.logConfig.Output == "file" && hook.logConfig.FilePath != "" {
		// 确保日志目录存在
		_ = os.MkdirAll(filepath.Dir(hook.logConfig.FilePath), 0755)
		hook.writers["default"] = &lumberjack.Logger{
			Filename:   hook.logConfig.FilePath,
			MaxSize:    hook.logConfig.MaxSize,
			MaxBackups: hook.logConfig.MaxBackups,
			MaxAge:     hook.logConfig.MaxAge,
			Compress:   hook.logConfig.Compress,
		}
	}
}

// Levels 返回此Hook关心的所有日志级别
func (hook *FileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire 在日志触发时执行
func (hook *FileHook) Fire(entry *logrus.Entry) error {
	// 获取日志类型，默认为default
	logType := "default"
	if lt, ok := entry.Data["type"]; ok {
		if t, ok := lt.(LogType); ok {
			// 如果是LogType类型，转换为字符串
			logType = string(t)
		} else if t, ok := lt.(string); ok {
			// 如果已经是字符串类型
			logType = t
		}
	}

	// 获取对应类型的writer
	writer := hook.getWriter(logType)
	if writer == nil {
		// 如果没有找到对应类型的writer，使用默认writer
		writer = hook.getWriter("default")
		if writer == nil {
			return nil
		}
	}

	// 格式化日志
	formatted, err := hook.formatter.Format(entry)
	if err != nil {
		return err
	}

	// 写入到对应文件
	hook.mutex.Lock()
	defer hook.mutex.Unlock()
	_, err = writer.Write(formatted)
	return err
}

// getWriter 获取指定类型的writer，如果不存在则创建
func (hook *FileHook) getWriter(logType string) io.Writer {
	hook.mutex.Lock()
	defer hook.mutex.Unlock()

	// 如果已经存在，直接返回
	if writer, exists := hook.writers[logType]; exists {
		return writer
	}

	// 根据日志类型创建对应的文件writer
	var filename string
	// 使用配置中的file_path获取日志目录
	logDir := filepath.Dir(hook.logConfig.FilePath)

	switch logType {
	case "access":
		filename = filepath.Join(logDir, "access.log")
	case "business":
		filename = filepath.Join(logDir, "business.log")
	case "error":
		filename = filepath.Join(logDir, "error.log")
	case "system":
		filename = filepath.Join(logDir, "system.log")
	case "audit":
		filename = filepath.Join(logDir, "audit.log")
	case "debug":
		filename = filepath.Join(logDir, "debug.log")
	default:
		// 对于未知类型，使用默认writer
		return hook.writers["default"]
	}

	// 确保日志目录存在
	_ = os.MkdirAll(filepath.Dir(filename), 0755)

	// 创建新的lumberjack logger
	writer := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    hook.logConfig.MaxSize,
		MaxBackups: hook.logConfig.MaxBackups,
		MaxAge:     hook.logConfig.MaxAge,
		Compress:   hook.logConfig.Compress,
	}

	// 保存到writers map中
	hook.writers[logType] = writer

	return writer
}

package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Level 日志级别
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
}

// Logger Agent 日志记录器
type Logger struct {
	writer io.Writer
	level  Level
	mu     sync.Mutex
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Init 初始化日志
func Init(logFile string, maxSize, maxBackups int, level string) error {
	var err error
	once.Do(func() {
		// 确保日志目录存在
		logDir := filepath.Dir(logFile)
		if err = os.MkdirAll(logDir, 0755); err != nil {
			return
		}

		// 配置日志轮转
		lumberjackLogger := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    maxSize,    // MB
			MaxBackups: maxBackups, // 保留的旧日志文件数量
			MaxAge:     30,         // 天
			Compress:   true,       // 压缩旧日志
			LocalTime:  true,       // 使用本地时间
		}

		// 同时输出到文件和标准输出
		multiWriter := io.MultiWriter(lumberjackLogger, os.Stdout)

		defaultLogger = &Logger{
			writer: multiWriter,
			level:  parseLevel(level),
		}
	})

	return err
}

// parseLevel 解析日志级别
func parseLevel(level string) Level {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// log 记录日志
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)

	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, levelName, message)
	l.writer.Write([]byte(logLine))
}

// Debug 记录 debug 日志
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(DebugLevel, format, args...)
	}
}

// Info 记录 info 日志
func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(InfoLevel, format, args...)
	}
}

// Warn 记录 warn 日志
func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(WarnLevel, format, args...)
	}
}

// Error 记录 error 日志
func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(ErrorLevel, format, args...)
	}
}

// Fatal 记录 fatal 日志并退出
func Fatal(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(ErrorLevel, format, args...)
	}
	os.Exit(1)
}

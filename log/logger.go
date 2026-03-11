package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Level 日志级别
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// Logger 日志器
type Logger struct {
	mu      sync.Mutex
	out     io.Writer
	errOut  io.Writer
	level   Level
	prefix  string
}

// 全局日志器
var defaultLogger *Logger

func init() {
	defaultLogger = &Logger{
		out:    os.Stdout,
		errOut: os.Stderr,
		level:  InfoLevel,
	}
}

// SetLevel 设置日志级别
func SetLevel(level Level) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
}

// SetOutput 设置输出
func SetOutput(w io.Writer) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.out = w
}

// 获取调用者信息
func callerInfo() (string, int, string) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "???", 0, ""
	}
	_, filename := filepath.Split(file)
	funcName := "???"
	if pc, _, _, ok := runtime.Caller(2); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			name := fn.Name()
			// 简化函数名
			if idx := filepath.Separator; idx != 0 {
				if lastSlash := filepath.Base(name); lastSlash != "" {
					funcName = lastSlash
				}
			}
		}
	}
	return filename, line, funcName
}

// format 格式化日志
func (l *Logger) format(level Level, format string, args ...interface{}) string {
	now := time.Now().Format("2006-01-02 15:04:05.000")
	file, line, _ := callerInfo()

	levelStr := ""
	switch level {
	case DebugLevel:
		levelStr = "DEBUG"
	case InfoLevel:
		levelStr = "INFO "
	case WarnLevel:
		levelStr = "WARN "
	case ErrorLevel:
		levelStr = "ERROR"
	case FatalLevel:
		levelStr = "FATAL"
	}

	msg := fmt.Sprintf(format, args...)
	return fmt.Sprintf("%s [%s] [%s:%d] %s\n", now, levelStr, file, line, msg)
}

// Debug 调试日志
func Debug(format string, args ...interface{}) {
	if defaultLogger.level <= DebugLevel {
		defaultLogger.mu.Lock()
		defer defaultLogger.mu.Unlock()
		fmt.Fprint(defaultLogger.out, defaultLogger.format(DebugLevel, format, args...))
	}
}

// Debugf 调试日志（带字段）
func Debugf(format string, args ...interface{}) {
	Debug(format, args...)
}

// Info 信息日志
func Info(format string, args ...interface{}) {
	if defaultLogger.level <= InfoLevel {
		defaultLogger.mu.Lock()
		defer defaultLogger.mu.Unlock()
		fmt.Fprint(defaultLogger.out, defaultLogger.format(InfoLevel, format, args...))
	}
}

// Infof 信息日志（带字段）
func Infof(format string, args ...interface{}) {
	Info(format, args...)
}

// Warn 警告日志
func Warn(format string, args ...interface{}) {
	if defaultLogger.level <= WarnLevel {
		defaultLogger.mu.Lock()
		defer defaultLogger.mu.Unlock()
		fmt.Fprint(defaultLogger.errOut, defaultLogger.format(WarnLevel, format, args...))
	}
}

// Warnf 警告日志（带字段）
func Warnf(format string, args ...interface{}) {
	Warn(format, args...)
}

// Error 错误日志
func Error(format string, args ...interface{}) {
	if defaultLogger.level <= ErrorLevel {
		defaultLogger.mu.Lock()
		defer defaultLogger.mu.Unlock()
		fmt.Fprint(defaultLogger.errOut, defaultLogger.format(ErrorLevel, format, args...))
	}
}

// Errorf 错误日志（带字段）
func Errorf(format string, args ...interface{}) {
	Error(format, args...)
}

// Fatal 致命日志
func Fatal(format string, args ...interface{}) {
	if defaultLogger.level <= FatalLevel {
		defaultLogger.mu.Lock()
		defer defaultLogger.mu.Unlock()
		fmt.Fprint(defaultLogger.errOut, defaultLogger.format(FatalLevel, format, args...))
		os.Exit(1)
	}
}

// Fatalf 致命日志（带字段）
func Fatalf(format string, args ...interface{}) {
	Fatal(format, args...)
}

// WithFields 带字段的日志（简化版）
func WithFields(fields map[string]interface{}) *FieldLogger {
	return &FieldLogger{fields: fields}
}

// FieldLogger 带字段的日志器
type FieldLogger struct {
	fields map[string]interface{}
}

// Info 信息日志
func (fl *FieldLogger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	for k, v := range fl.fields {
		msg += fmt.Sprintf(" %s=%v", k, v)
	}
	Info(msg)
}

// Error 错误日志
func (fl *FieldLogger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	for k, v := range fl.fields {
		msg += fmt.Sprintf(" %s=%v", k, v)
	}
	Error(msg)
}

// Errorf 错误日志
func (fl *FieldLogger) Errorf(format string, args ...interface{}) {
	fl.Error(format, args...)
}

// Debug 调试日志
func (fl *FieldLogger) Debug(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	for k, v := range fl.fields {
		msg += fmt.Sprintf(" %s=%v", k, v)
	}
	Debug(msg)
}

// Warn 警告日志
func (fl *FieldLogger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	for k, v := range fl.fields {
		msg += fmt.Sprintf(" %s=%v", k, v)
	}
	Warn(msg)
}

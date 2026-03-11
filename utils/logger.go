package utils

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Logger 自定义日志器
type Logger struct {
	infoLog  *log.Logger
	warnLog  *log.Logger
	errorLog *log.Logger
	debugLog *log.Logger
}

var defaultLogger *Logger

// InitLogger 初始化日志器
func InitLogger(logDir string) {
	if logDir != "" {
		os.MkdirAll(logDir, 0755)
	}

	logFile := filepath.Join(logDir, "server.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open log file, using stdout: %v", err)
		file = os.Stdout
	}

	defaultLogger = &Logger{
		infoLog:  log.New(file, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLog:  log.New(file, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLog: log.New(file, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLog: log.New(file, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// Info 记录信息日志
func Info(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.infoLog.Printf(format, v...)
	}
}

// Warn 记录警告日志
func Warn(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.warnLog.Printf(format, v...)
	}
}

// Error 记录错误日志
func Error(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.errorLog.Printf(format, v...)
	}
}

// Debug 记录调试日志
func Debug(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.debugLog.Printf(format, v...)
	}
}

// GetCaller 获取调用者信息
func GetCaller() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	return filepath.Base(file) + ":" + string(rune(line))
}

// GetTimestamp 获取时间戳
func GetTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

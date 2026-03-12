package log

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// 日志级别
const (
	DebugLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var (
	currentLevel = InfoLevel
	logChan      chan string
	wg           sync.WaitGroup
	instance     *AsyncLogger
	once         sync.Once
)

// AsyncLogger 异步日志器
type AsyncLogger struct {
	queue    chan *LogEntry
	file     *os.File
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// LogEntry 日志条目
type LogEntry struct {
	Level   int
	Message string
	Time    time.Time
}

// GetLogger 获取异步日志器实例
func GetLogger() *AsyncLogger {
	once.Do(func() {
		instance = NewAsyncLogger()
	})
	return instance
}

// NewAsyncLogger 创建异步日志器
func NewAsyncLogger() *AsyncLogger {
	logger := &AsyncLogger{
		queue:    make(chan *LogEntry, 10000),
		stopChan: make(chan struct{}),
		file:     os.Stdout,
	}

	// 启动异步写入协程
	logger.wg.Add(1)
	go logger.writeLoop()

	return logger
}

// writeLoop 异步写入循环
func (l *AsyncLogger) writeLoop() {
	defer l.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var batch []*LogEntry

	for {
		select {
		case entry := <-l.queue:
			batch = append(batch, entry)

			// 达到批量大小，立即写入
			if len(batch) >= 100 {
				l.flushBatch(batch)
				batch = nil
			}

		case <-ticker.C:
			// 定时写入
			if len(batch) > 0 {
				l.flushBatch(batch)
				batch = nil
			}

		case <-l.stopChan:
			// 停止前写入剩余数据
			if len(batch) > 0 {
				l.flushBatch(batch)
			}
			return
		}
	}
}

// flushBatch 批量写入日志
func (l *AsyncLogger) flushBatch(batch []*LogEntry) {
	for _, entry := range batch {
		timestamp := entry.Time.Format("2006-01-02 15:04:05.000")
		levelStr := getLevelString(entry.Level)
		fmt.Fprintf(l.file, "%s [%s] %s\n", timestamp, levelStr, entry.Message)
	}
}

func getLevelString(level int) string {
	switch level {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO "
	case WarnLevel:
		return "WARN "
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "INFO "
	}
}

// log 添加日志到队列
func (l *AsyncLogger) log(level int, format string, args ...interface{}) {
	if level < currentLevel {
		return
	}

	message := fmt.Sprintf(format, args...)
	entry := &LogEntry{
		Level:   level,
		Message: message,
		Time:    time.Now(),
	}

	select {
	case l.queue <- entry:
		// 成功加入队列
	default:
		// 队列已满，直接写入（降级）
		l.flushBatch([]*LogEntry{entry})
	}
}

// Stop 停止日志器
func (l *AsyncLogger) Stop() {
	close(l.stopChan)
	l.wg.Wait()
}

// 全局日志函数
func SetLevel(level int) {
	currentLevel = level
}

func Debug(format string, args ...interface{}) {
	GetLogger().log(DebugLevel, format, args...)
}

func Info(format string, args ...interface{}) {
	GetLogger().log(InfoLevel, format, args...)
}

func Warn(format string, args ...interface{}) {
	GetLogger().log(WarnLevel, format, args...)
}

func Error(format string, args ...interface{}) {
	GetLogger().log(ErrorLevel, format, args...)
}

func Fatal(format string, args ...interface{}) {
	GetLogger().log(FatalLevel, format, args...)
	os.Exit(1)
}

func Debugf(format string, args ...interface{}) {
	Debug(format, args...)
}

func Infof(format string, args ...interface{}) {
	Info(format, args...)
}

func Warnf(format string, args ...interface{}) {
	Warn(format, args...)
}

func Errorf(format string, args ...interface{}) {
	Error(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	Fatal(format, args...)
}

// WithFields 支持字段日志（简化版）
func WithFields(fields map[string]interface{}) Logger {
	return &fieldLogger{fields: fields}
}

type Logger interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

type fieldLogger struct {
	fields map[string]interface{}
}

func (l *fieldLogger) Info(format string, args ...interface{}) {
	Info(format+" "+fieldsToString(l.fields), args...)
}

func (l *fieldLogger) Warn(format string, args ...interface{}) {
	Warn(format+" "+fieldsToString(l.fields), args...)
}

func (l *fieldLogger) Error(format string, args ...interface{}) {
	Error(format+" "+fieldsToString(l.fields), args...)
}

func fieldsToString(fields map[string]interface{}) string {
	result := ""
	for k, v := range fields {
		result += fmt.Sprintf("%s=%v ", k, v)
	}
	return result
}

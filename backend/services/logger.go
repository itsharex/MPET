package services

import (
	"fmt"
	"sync"
	"time"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// Logger 全局日志收集器
type Logger struct {
	logs     []LogEntry
	mu       sync.RWMutex
	maxLogs  int
}

var globalLogger *Logger
var loggerOnce sync.Once

// GetLogger 获取全局日志实例
func GetLogger() *Logger {
	loggerOnce.Do(func() {
		globalLogger = &Logger{
			logs:    make([]LogEntry, 0),
			maxLogs: 1000, // 最多保存 1000 条日志
		}
	})
	return globalLogger
}

// Log 记录日志
func (l *Logger) Log(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	l.logs = append(l.logs, entry)

	// 如果超过最大数量，删除最旧的日志
	if len(l.logs) > l.maxLogs {
		l.logs = l.logs[len(l.logs)-l.maxLogs:]
	}
}

// Info 记录 INFO 级别日志
func (l *Logger) Info(message string) {
	l.Log("INFO", message)
}

// Warn 记录 WARN 级别日志
func (l *Logger) Warn(message string) {
	l.Log("WARN", message)
}

// Error 记录 ERROR 级别日志
func (l *Logger) Error(message string) {
	l.Log("ERROR", message)
}

// Debug 记录 DEBUG 级别日志
func (l *Logger) Debug(message string) {
	l.Log("DEBUG", message)
}

// GetLogs 获取所有日志
func (l *Logger) GetLogs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]string, len(l.logs))
	for i, entry := range l.logs {
		result[i] = fmt.Sprintf("[%s] [%s] %s",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Level,
			entry.Message,
		)
	}
	return result
}

// Clear 清空日志
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = make([]LogEntry, 0)
}

// GetRecentLogs 获取最近 N 条日志
func (l *Logger) GetRecentLogs(n int) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	start := 0
	if len(l.logs) > n {
		start = len(l.logs) - n
	}

	logs := l.logs[start:]
	result := make([]string, len(logs))
	for i, entry := range logs {
		result[i] = fmt.Sprintf("[%s] [%s] %s",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Level,
			entry.Message,
		)
	}
	return result
}

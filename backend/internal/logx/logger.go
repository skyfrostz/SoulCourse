package logx

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Field struct {
	Key   string
	Value any
}

type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	minLevel Level
	colorful bool
}

func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
		colorful: detectColorSupport(out),
	}
}

func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

func (l *Logger) Log(level Level, module string, action string, fields ...Field) {
	if level < l.minLevel {
		return
	}

	line := buildLine(level, module, action, fields...)
	if l.colorful {
		line = levelColor(level) + line + "\033[0m"
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(l.out, line)
}

func (l *Logger) Debug(module string, action string, fields ...Field) {
	l.Log(LevelDebug, module, action, fields...)
}

func (l *Logger) Info(module string, action string, fields ...Field) {
	l.Log(LevelInfo, module, action, fields...)
}

func (l *Logger) Warn(module string, action string, fields ...Field) {
	l.Log(LevelWarn, module, action, fields...)
}

func (l *Logger) Error(module string, action string, fields ...Field) {
	l.Log(LevelError, module, action, fields...)
}

func buildLine(level Level, module string, action string, fields ...Field) string {
	var builder strings.Builder
	builder.WriteString("[时间]")
	builder.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	builder.WriteString(" [级别]")
	builder.WriteString(levelLabel(level))
	builder.WriteString(" [模块]")
	builder.WriteString(fallback(module, "系统"))
	builder.WriteString(" [操作]")
	builder.WriteString(fallback(action, "执行日志"))

	for _, field := range fields {
		if strings.TrimSpace(field.Key) == "" || field.Value == nil {
			continue
		}
		builder.WriteString(" [")
		builder.WriteString(strings.TrimSpace(field.Key))
		builder.WriteString("]")
		builder.WriteString(formatValue(field.Value))
	}

	return builder.String()
}

func levelLabel(level Level) string {
	switch level {
	case LevelDebug:
		return "调试"
	case LevelWarn:
		return "警告"
	case LevelError:
		return "错误"
	default:
		return "信息"
	}
}

func levelColor(level Level) string {
	switch level {
	case LevelDebug:
		return "\033[36m"
	case LevelWarn:
		return "\033[33m"
	case LevelError:
		return "\033[31m"
	default:
		return "\033[32m"
	}
}

func formatValue(value any) string {
	switch typed := value.(type) {
	case time.Duration:
		return typed.Round(time.Millisecond).String()
	case error:
		return typed.Error()
	default:
		return fmt.Sprint(value)
	}
}

func fallback(value string, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}

func detectColorSupport(out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return true
}

package logx

import (
	"encoding/json"
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
	json     bool
}

func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
		colorful: detectColorSupport(out),
	}
}

func NewJSON(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
		json:     true,
	}
}

func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

func (l *Logger) Log(level Level, module string, action string, fields ...Field) {
	if level < l.minLevel {
		return
	}

	line := buildLine(level, module, action, l.json, fields...)
	if l.colorful && !l.json {
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

func buildLine(level Level, module string, action string, jsonFormat bool, fields ...Field) string {
	if jsonFormat {
		return buildJSONLine(level, module, action, fields...)
	}

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

func buildJSONLine(level Level, module string, action string, fields ...Field) string {
	entry := map[string]any{
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
		"level":  levelName(level),
		"module": fallback(module, "system"),
		"action": fallback(action, "log"),
	}
	for _, field := range fields {
		key := strings.TrimSpace(field.Key)
		if key == "" || field.Value == nil {
			continue
		}
		entry[key] = sanitizeLogValue(key, field.Value)
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return fmt.Sprintf(`{"time":%q,"level":"error","module":"logger","action":"encode_failed","error":%q}`, time.Now().UTC().Format(time.RFC3339Nano), err.Error())
	}
	return string(encoded)
}

func levelName(level Level) string {
	switch level {
	case LevelDebug:
		return "debug"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "info"
	}
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

func sanitizeLogValue(key string, value any) any {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	if strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "cookie") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "authorization") {
		return "[REDACTED]"
	}
	switch typed := value.(type) {
	case time.Duration:
		return typed.Round(time.Millisecond).String()
	case error:
		return typed.Error()
	default:
		return value
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

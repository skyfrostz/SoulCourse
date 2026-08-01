package logx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestJSONLoggerRedactsSensitiveFields(t *testing.T) {
	var output bytes.Buffer
	logger := NewJSON(&output, LevelInfo)

	logger.Info("auth", "login_failed",
		F("requestId", "req-public-beta-001"),
		F("password", "plain-text-password"),
		F("admin_token", "secret-token"),
		F("userId", int64(42)),
	)

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v line=%s", err, output.String())
	}
	if entry["password"] != "[REDACTED]" || entry["admin_token"] != "[REDACTED]" {
		t.Fatalf("sensitive fields were not redacted: %#v", entry)
	}
	if strings.Contains(output.String(), "plain-text-password") || strings.Contains(output.String(), "secret-token") {
		t.Fatalf("log leaked sensitive values: %s", output.String())
	}
	if entry["requestId"] != "req-public-beta-001" || entry["level"] != "info" {
		t.Fatalf("unexpected structured fields: %#v", entry)
	}
}

func TestJSONLoggerRedactsSensitiveKeyVariants(t *testing.T) {
	sensitive := []string{
		"Password", "password_hash", "new-password", "accessToken", "refresh_token",
		"session-cookie", "client_secret", "Authorization", "proxy_authorization",
	}
	fields := []Field{F("safe", "visible")}
	for _, key := range sensitive {
		fields = append(fields, F(key, "must-not-leak-"+key))
	}

	line := buildJSONLine(LevelWarn, " auth ", " login ", fields...)
	var entry map[string]any
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatal(err)
	}
	for _, key := range sensitive {
		if entry[key] != "[REDACTED]" {
			t.Errorf("%s = %v, want redacted", key, entry[key])
		}
		if strings.Contains(line, "must-not-leak-"+key) {
			t.Errorf("serialized log leaked %s", key)
		}
	}
	if entry["safe"] != "visible" || entry["module"] != "auth" || entry["action"] != "login" {
		t.Fatalf("unexpected non-sensitive fields: %#v", entry)
	}
}

func TestJSONLoggerFilteringFormattingAndFallbacks(t *testing.T) {
	var output bytes.Buffer
	logger := NewJSON(&output, LevelWarn)
	logger.Debug("module", "debug")
	logger.Info("module", "info")
	logger.Warn("", "", F("duration", 1500*time.Microsecond), F("error", errors.New("failed")), F("", "ignored"), F("nil", nil))
	logger.Error("module", "error")

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2: %q", len(lines), output.String())
	}
	var warning map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &warning); err != nil {
		t.Fatal(err)
	}
	if warning["level"] != "warn" || warning["module"] != "system" || warning["action"] != "log" {
		t.Fatalf("unexpected warning metadata: %#v", warning)
	}
	if warning["duration"] != "2ms" || warning["error"] != "failed" {
		t.Fatalf("unexpected formatted values: %#v", warning)
	}
}

func TestJSONLoggerEncodeFailureDoesNotLeakValue(t *testing.T) {
	secret := "channel-secret-value"
	line := buildJSONLine(LevelInfo, "test", "encode", F("ordinary", make(chan string)), F("password", secret))
	if !strings.Contains(line, `"action":"encode_failed"`) || !strings.Contains(line, `"module":"logger"`) {
		t.Fatalf("unexpected fallback line: %s", line)
	}
	if strings.Contains(line, secret) {
		t.Fatalf("fallback log leaked secret: %s", line)
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("fallback is not valid JSON: %v", err)
	}
}

func TestTextLoggerLevelsValuesAndConcurrentWrites(t *testing.T) {
	var output bytes.Buffer
	logger := New(&output, LevelDebug)
	logger.Debug("m", "debug", F("duration", 1500*time.Microsecond))
	logger.Info("m", "info", F("error", errors.New("boom")))
	logger.Warn("m", "warn")
	logger.Error("m", "error")
	for _, label := range []string{"调试", "信息", "警告", "错误", "2ms", "boom"} {
		if !strings.Contains(output.String(), label) {
			t.Errorf("text output missing %q: %s", label, output.String())
		}
	}

	output.Reset()
	jsonLogger := NewJSON(&output, LevelInfo)
	const workers = 32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			jsonLogger.Info("race", "write", F("id", id))
		}(i)
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != workers {
		t.Fatalf("concurrent lines = %d, want %d", len(lines), workers)
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatalf("interleaved JSON log line: %q", line)
		}
	}
}

func TestTextLoggerColorFormattingAndWriterFailures(t *testing.T) {
	colors := map[Level]string{
		LevelDebug: "\033[36m",
		LevelInfo:  "\033[32m",
		LevelWarn:  "\033[33m",
		LevelError: "\033[31m",
	}
	for level, want := range colors {
		if got := levelColor(level); got != want {
			t.Errorf("levelColor(%d) = %q, want %q", level, got, want)
		}
	}
	if got := formatValue(42); got != "42" {
		t.Fatalf("formatValue = %q, want 42", got)
	}
	if detectColorSupport(&bytes.Buffer{}) {
		t.Fatal("buffer unexpectedly supports terminal color")
	}
	t.Setenv("NO_COLOR", "1")
	if detectColorSupport(os.Stdout) {
		t.Fatal("NO_COLOR must disable terminal color")
	}

	logger := New(failingWriter{}, LevelDebug)
	logger.Debug("test", "write-failure", F("value", 1))
	jsonLogger := NewJSON(failingWriter{}, LevelDebug)
	jsonLogger.Error("test", "write-failure", F("value", 1))
}

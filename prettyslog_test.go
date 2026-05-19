package prettyslog

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

type captureStream struct {
	lines [][]byte
}

func (cs *captureStream) Write(bytes []byte) (int, error) {
	cs.lines = append(cs.lines, bytes)
	return len(bytes), nil
}

type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func Test_WritesToProvidedStream(t *testing.T) {
	cs := &captureStream{}
	handler := New(nil, WithDestinationWriter(cs), WithOutputEmptyAttrs())
	logger := slog.New(handler)

	logger.Info("testing logger")
	if len(cs.lines) != 1 {
		t.Errorf("expected 1 lines logged, got: %d", len(cs.lines))
	}

	lineMatcher := regexp.MustCompile(`\[\d{2}:\d{2}:\d{2}\.\d{3}\] INFO: testing logger {}`)
	line := string(cs.lines[0])
	if lineMatcher.MatchString(line) == false {
		t.Errorf("expected `testing logger` but found `%s`", line)
	}
	if !strings.HasSuffix(line, "\n") {
		t.Errorf("exected line to be terminated with `\\n` but found `%s`", line[len(line)-1:])
	}
}

func Test_SkipEmptyAttributes(t *testing.T) {
	cs := &captureStream{}
	handler := New(nil, WithDestinationWriter(cs))
	logger := slog.New(handler)

	logger.Info("testing logger")
	if len(cs.lines) != 1 {
		t.Errorf("expected 1 lines logged, got: %d", len(cs.lines))
	}

	lineMatcher := regexp.MustCompile(`\[\d{2}:\d{2}:\d{2}\.\d{3}\] INFO: testing logger`)
	line := string(cs.lines[0])
	if lineMatcher.MatchString(line) == false {
		t.Errorf("expected `testing logger` but found `%s`", line)
	}
	if !strings.HasSuffix(line, "\n") {
		t.Errorf("exected line to be terminated with `\\n` but found `%s`", line[len(line)-1:])
	}
}

func Test_Colorizer(t *testing.T) {
	got := colorizer(red, "hello")
	want := "\033[31mhello\033[0m"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func Test_WithColor_ProducesAnsiEscapes(t *testing.T) {
	cs := &captureStream{}
	handler := New(nil, WithDestinationWriter(cs), WithColor())
	logger := slog.New(handler)

	logger.Info("colorized")
	if len(cs.lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(cs.lines))
	}
	line := string(cs.lines[0])
	if !strings.Contains(line, "\033[") {
		t.Errorf("expected ANSI escape codes in colorized output, got %q", line)
	}
	if !strings.Contains(line, reset) {
		t.Errorf("expected reset code in colorized output, got %q", line)
	}
}

func Test_WithAttrs_IncludesAttrsInOutput(t *testing.T) {
	cs := &captureStream{}
	handler := New(nil, WithDestinationWriter(cs))
	logger := slog.New(handler).With("user", "alice", "id", 42)

	logger.Info("hello")
	if len(cs.lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(cs.lines))
	}
	line := string(cs.lines[0])
	if !strings.Contains(line, `"user": "alice"`) {
		t.Errorf("expected attr user=alice in output, got %q", line)
	}
	if !strings.Contains(line, `"id": 42`) {
		t.Errorf("expected attr id=42 in output, got %q", line)
	}
}

func Test_WithGroup_NestsAttrs(t *testing.T) {
	cs := &captureStream{}
	handler := New(nil, WithDestinationWriter(cs))
	logger := slog.New(handler).WithGroup("req").With("method", "GET")

	logger.Info("request")
	if len(cs.lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(cs.lines))
	}
	line := string(cs.lines[0])
	if !strings.Contains(line, `"req"`) {
		t.Errorf("expected group `req` in output, got %q", line)
	}
	if !strings.Contains(line, `"method": "GET"`) {
		t.Errorf("expected method=GET in output, got %q", line)
	}
}

func Test_NewHandler_DefaultsApplied(t *testing.T) {
	h := NewHandler(nil)
	if h.writer == nil {
		t.Error("expected writer to be set by NewHandler")
	}
	if !h.colorize {
		t.Error("expected colorize to be enabled by NewHandler")
	}
	if !h.outputEmptyAttrs {
		t.Error("expected outputEmptyAttrs to be enabled by NewHandler")
	}
}

func Test_Handle_AllLevels(t *testing.T) {
	cases := []struct {
		name  string
		level slog.Level
		log   func(l *slog.Logger)
	}{
		{"debug", slog.LevelDebug, func(l *slog.Logger) { l.Debug("d") }},
		{"info", slog.LevelInfo, func(l *slog.Logger) { l.Info("i") }},
		{"warn", slog.LevelWarn, func(l *slog.Logger) { l.Warn("w") }},
		{"error", slog.LevelError, func(l *slog.Logger) { l.Error("e") }},
		{"fatal", slog.LevelError + 4, func(l *slog.Logger) { l.Log(nil, slog.LevelError+4, "f") }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cs := &captureStream{}
			handler := New(&slog.HandlerOptions{Level: slog.LevelDebug}, WithDestinationWriter(cs), WithColor())
			logger := slog.New(handler)
			c.log(logger)
			if len(cs.lines) != 1 {
				t.Fatalf("expected 1 line, got %d", len(cs.lines))
			}
			if !strings.Contains(string(cs.lines[0]), "\033[") {
				t.Errorf("expected ANSI color in line, got %q", string(cs.lines[0]))
			}
		})
	}
}

func Test_Enabled_RespectsLevel(t *testing.T) {
	cs := &captureStream{}
	handler := New(&slog.HandlerOptions{Level: slog.LevelWarn}, WithDestinationWriter(cs))
	logger := slog.New(handler)

	logger.Info("should be skipped")
	if len(cs.lines) != 0 {
		t.Errorf("expected no lines below threshold, got %d", len(cs.lines))
	}

	logger.Warn("should appear")
	if len(cs.lines) != 1 {
		t.Errorf("expected 1 line at/above threshold, got %d", len(cs.lines))
	}
}

func Test_ReplaceAttr_AppliedToCustomAndDefaultKeys(t *testing.T) {
	cs := &captureStream{}
	replaceCalls := map[string]int{}
	opts := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			replaceCalls[a.Key]++
			if a.Key == "secret" {
				return slog.String("secret", "REDACTED")
			}
			return a
		},
	}
	handler := New(opts, WithDestinationWriter(cs))
	logger := slog.New(handler)

	logger.Info("hi", "secret", "p@ss")
	if len(cs.lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(cs.lines))
	}
	line := string(cs.lines[0])
	if !strings.Contains(line, "REDACTED") {
		t.Errorf("expected REDACTED in output, got %q", line)
	}
	if strings.Contains(line, "p@ss") {
		t.Errorf("expected raw secret to be replaced, got %q", line)
	}
	for _, key := range []string{slog.TimeKey, slog.LevelKey, slog.MessageKey, "secret"} {
		if replaceCalls[key] == 0 {
			t.Errorf("expected ReplaceAttr to be called for key %q", key)
		}
	}
}

func Test_ReplaceAttr_SuppressingDefaultsOmitsFields(t *testing.T) {
	cs := &captureStream{}
	opts := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || a.Key == slog.LevelKey || a.Key == slog.MessageKey {
				return slog.Attr{}
			}
			return a
		},
	}
	handler := New(opts, WithDestinationWriter(cs))
	logger := slog.New(handler)

	logger.Info("should be hidden", "k", "v")
	if len(cs.lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(cs.lines))
	}
	line := string(cs.lines[0])
	if strings.Contains(line, "INFO") {
		t.Errorf("expected level to be suppressed, got %q", line)
	}
	if strings.Contains(line, "should be hidden") {
		t.Errorf("expected message to be suppressed, got %q", line)
	}
	if !strings.Contains(line, `"k": "v"`) {
		t.Errorf("expected user attr to remain, got %q", line)
	}
}

func Test_Handle_WriterErrorPropagates(t *testing.T) {
	handler := New(nil, WithDestinationWriter(failingWriter{}))
	logger := slog.New(handler)

	defer func() {
		_ = recover()
	}()
	logger.Info("boom")
}

func Test_New_NilOptionsDoesNotPanic(t *testing.T) {
	cs := &captureStream{}
	handler := New(nil, WithDestinationWriter(cs))
	logger := slog.New(handler)
	logger.Info("ok")
	if len(cs.lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(cs.lines))
	}
}

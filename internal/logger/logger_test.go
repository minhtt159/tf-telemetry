package logger

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew_CreatesLoggerWithLevel(t *testing.T) {
	logger, err := New("info")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNew_InvalidLevel(t *testing.T) {
	_, err := New("invalid-level")
	if err == nil {
		t.Fatal("expected error for invalid level")
	}
}

func TestNewWithConfig_JSONEncoding(t *testing.T) {
	cfg := Config{
		Level:            "debug",
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := NewWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewWithConfig_ConsoleEncoding(t *testing.T) {
	cfg := Config{
		Level:            "warn",
		Encoding:         "console",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := NewWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewWithConfig_CustomPaths(t *testing.T) {
	cfg := Config{
		Level:            "info",
		Encoding:         "json",
		OutputPaths:      []string{"/dev/null"},
		ErrorOutputPaths: []string{"/dev/null"},
	}

	logger, err := NewWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Test that logger can write without error
	logger.Info("test message", zap.String("key", "value"))
}

func TestNewWithConfig_AllLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := Config{
				Level:            level,
				Encoding:         "json",
				OutputPaths:      []string{"stdout"},
				ErrorOutputPaths: []string{"stderr"},
			}

			logger, err := NewWithConfig(cfg)
			if err != nil {
				t.Fatalf("NewWithConfig() error = %v for level %s", err, level)
			}
			if logger == nil {
				t.Fatalf("expected non-nil logger for level %s", level)
			}
		})
	}
}

func TestNewWithConfig_DefaultsWhenEmpty(t *testing.T) {
	cfg := Config{
		Level: "info",
	}

	logger, err := NewWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewWithConfig_EncoderConfig(t *testing.T) {
	cfg := Config{
		Level:            "info",
		Encoding:         "console",
		OutputPaths:      []string{"/dev/null"},
		ErrorOutputPaths: []string{"/dev/null"},
	}

	logger, err := NewWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}

	// Verify logger can be used
	logger.Info("test", zap.String("field", "value"))

	// Test with JSON encoding
	cfg.Encoding = "json"
	logger, err = NewWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
	logger.Info("test", zap.String("field", "value"))
}

func TestNew_BackwardCompatibility(t *testing.T) {
	// Ensure old API still works
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		logger, err := New(level)
		if err != nil {
			t.Fatalf("New(%s) error = %v", level, err)
		}
		if logger == nil {
			t.Fatalf("expected non-nil logger for level %s", level)
		}
	}
}

func TestNewWithConfig_UsesLevelCorrectly(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		shouldLogInfo bool
		shouldLogWarn bool
	}{
		{"debug level logs all", "debug", true, true},
		{"info level logs info and above", "info", true, true},
		{"warn level logs warn and above", "warn", false, true},
		{"error level logs only error", "error", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Level:            tt.level,
				Encoding:         "json",
				OutputPaths:      []string{"/dev/null"},
				ErrorOutputPaths: []string{"/dev/null"},
			}

			logger, err := NewWithConfig(cfg)
			if err != nil {
				t.Fatalf("NewWithConfig() error = %v", err)
			}

			core := logger.Core()

			if tt.shouldLogInfo {
				if !core.Enabled(zapcore.InfoLevel) {
					t.Errorf("expected Info level to be enabled for level %s", tt.level)
				}
			} else {
				if core.Enabled(zapcore.InfoLevel) {
					t.Errorf("expected Info level to be disabled for level %s", tt.level)
				}
			}

			if tt.shouldLogWarn {
				if !core.Enabled(zapcore.WarnLevel) {
					t.Errorf("expected Warn level to be enabled for level %s", tt.level)
				}
			} else {
				if core.Enabled(zapcore.WarnLevel) {
					t.Errorf("expected Warn level to be disabled for level %s", tt.level)
				}
			}
		})
	}
}

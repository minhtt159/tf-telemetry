// Package logger provides a configured zap logger instance.
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds logger configuration options.
type Config struct {
	Level            string
	Encoding         string
	OutputPaths      []string
	ErrorOutputPaths []string
}

func New(level string) (*zap.Logger, error) {
	return NewWithConfig(Config{
		Level:            level,
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	})
}

func NewWithConfig(cfg Config) (*zap.Logger, error) {
	loggerConfig := zap.NewProductionConfig()

	if err := loggerConfig.Level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// Set encoding
	if cfg.Encoding != "" {
		loggerConfig.Encoding = cfg.Encoding
	}

	// Set output paths
	if len(cfg.OutputPaths) > 0 {
		loggerConfig.OutputPaths = cfg.OutputPaths
	}

	// Set error output paths
	if len(cfg.ErrorOutputPaths) > 0 {
		loggerConfig.ErrorOutputPaths = cfg.ErrorOutputPaths
	}

	// Configure encoder for better readability in console mode
	if cfg.Encoding == "console" {
		loggerConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	return loggerConfig.Build()
}

// Package logger provides a configured zap logger instance.
package logger

import (
	"fmt"

	"go.uber.org/zap"
)

func New(level string) (*zap.Logger, error) {
	loggerConfig := zap.NewProductionConfig()
	if err := loggerConfig.Level.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	return loggerConfig.Build()
}

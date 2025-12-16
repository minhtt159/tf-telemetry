// Package config initializes and loads the application configuration.
package config

import (
	"fmt"
	"math"

	"github.com/spf13/viper"
)

type BasicAuthConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type RateLimitConfig struct {
	Enabled           bool    `mapstructure:"enabled"`
	RequestsPerSecond float64 `mapstructure:"requests_per_second"`
	Burst             int     `mapstructure:"burst"`
}

type Config struct {
	Server struct {
		BindAddress string          `mapstructure:"bind_address"`
		GRPCPort    int             `mapstructure:"grpc_port"`
		HTTPPort    int             `mapstructure:"http_port"`
		BasicAuth   BasicAuthConfig `mapstructure:"basic_auth"`
		RateLimit   RateLimitConfig `mapstructure:"rate_limit"`
	} `mapstructure:"server"`
	Elastic struct {
		Addresses     []string `mapstructure:"addresses"`
		Username      string   `mapstructure:"username"`
		Password      string   `mapstructure:"password"`
		IndexMetrics  string   `mapstructure:"index_metrics"`
		IndexLogs     string   `mapstructure:"index_logs"`
		BatchSize     int      `mapstructure:"batch_size"`
		FlushInterval int      `mapstructure:"flush_interval_seconds"`
	} `mapstructure:"elasticsearch"`
	Logging struct {
		Level            string   `mapstructure:"level"`
		Encoding         string   `mapstructure:"encoding"`
		OutputPaths      []string `mapstructure:"output_paths"`
		ErrorOutputPaths []string `mapstructure:"error_output_paths"`
		MaxContextAttrs  int      `mapstructure:"max_context_attributes"`
	} `mapstructure:"logging"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fatal error config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if cfg.Server.RateLimit.Enabled {
		if cfg.Server.RateLimit.RequestsPerSecond <= 0 {
			return nil, fmt.Errorf("rate limit enabled but requests_per_second not set")
		}
		if cfg.Server.RateLimit.Burst == 0 {
			// Default burst to a single second worth of requests to align with limiter tokens.
			cfg.Server.RateLimit.Burst = int(math.Ceil(cfg.Server.RateLimit.RequestsPerSecond))
		}
	}

	return &cfg, nil
}

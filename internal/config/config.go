package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	ListenAddr             string   `yaml:"listen_addr"`
	DatabaseURL            string   `yaml:"database_url"`
	DatabaseSchema         string   `yaml:"database_schema"`
	LogBufferSize          int      `yaml:"log_buffer_size"`
	ManagementBootstrapKey string   `yaml:"management_bootstrap_key"`
	CORSOrigins            []string `yaml:"cors_origins"`
	EncryptionKey          string   `yaml:"encryption_key"`
	LogRetentionDays       int      `yaml:"log_retention_days"`
	RateLimitRPS           float64  `yaml:"rate_limit_rps"`
	RateLimitBurst         int      `yaml:"rate_limit_burst"`
	CBFailureThreshold     int      `yaml:"cb_failure_threshold"`
	CBTimeoutSeconds       int      `yaml:"cb_timeout_seconds"`
	RetryMaxAttempts       int      `yaml:"retry_max_attempts"`
	RetryBaseDelayMS       int      `yaml:"retry_base_delay_ms"`
	MaxDBConns             int32    `yaml:"max_db_conns"`
	MinDBConns             int32    `yaml:"min_db_conns"`
	MetricsEnabled         bool     `yaml:"metrics_enabled"`
	LogFormat              string   `yaml:"log_format"`
}

// Load reads configuration from config.yaml and overrides with environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:         ":8080",
		DatabaseSchema:     "public",
		LogBufferSize:      10000,
		LogRetentionDays:   7,
		CBFailureThreshold: 5,
		CBTimeoutSeconds:   30,
		RetryMaxAttempts:   3,
		RetryBaseDelayMS:   100,
		MaxDBConns:         25,
		MinDBConns:         5,
		LogFormat:          "json",
	}

	configPath := os.Getenv("PXBIN_CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	overrideFromEnv(cfg)
	return cfg, nil
}

func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("PXBIN_LISTEN_ADDR"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("PXBIN_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("PXBIN_DATABASE_SCHEMA"); v != "" {
		cfg.DatabaseSchema = v
	}
	if v := os.Getenv("PXBIN_LOG_BUFFER_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LogBufferSize = n
		}
	}
	if v := os.Getenv("PXBIN_MANAGEMENT_BOOTSTRAP_KEY"); v != "" {
		cfg.ManagementBootstrapKey = v
	}
	if v := os.Getenv("PXBIN_CORS_ORIGINS"); v != "" {
		cfg.CORSOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("PXBIN_ENCRYPTION_KEY"); v != "" {
		cfg.EncryptionKey = v
	}
	if v := os.Getenv("PXBIN_LOG_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LogRetentionDays = n
		}
	}
	if v := os.Getenv("PXBIN_RATE_LIMIT_RPS"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.RateLimitRPS = f
		}
	}
	if v := os.Getenv("PXBIN_RATE_LIMIT_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimitBurst = n
		}
	}
	if v := os.Getenv("PXBIN_CB_FAILURE_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.CBFailureThreshold = n
		}
	}
	if v := os.Getenv("PXBIN_CB_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.CBTimeoutSeconds = n
		}
	}
	if v := os.Getenv("PXBIN_RETRY_MAX_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RetryMaxAttempts = n
		}
	}
	if v := os.Getenv("PXBIN_RETRY_BASE_DELAY_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RetryBaseDelayMS = n
		}
	}
	if v := os.Getenv("PXBIN_MAX_DB_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxDBConns = int32(n)
		}
	}
	if v := os.Getenv("PXBIN_MIN_DB_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MinDBConns = int32(n)
		}
	}
	if v := os.Getenv("PXBIN_METRICS_ENABLED"); v != "" {
		cfg.MetricsEnabled = v == "true" || v == "1"
	}
	if v := os.Getenv("PXBIN_LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
}

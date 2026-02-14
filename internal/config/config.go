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
	LogBufferSize          int      `yaml:"log_buffer_size"`
	ManagementBootstrapKey string   `yaml:"management_bootstrap_key"`
	CORSOrigins            []string `yaml:"cors_origins"`
	EncryptionKey          string   `yaml:"encryption_key"`
}

// Load reads configuration from config.yaml and overrides with environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:    ":8080",
		LogBufferSize: 10000,
	}

	data, err := os.ReadFile("config.yaml")
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
}

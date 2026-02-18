package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var schemaNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// Validate checks the config for invalid or missing values. Returns a
// multi-error with all problems found.
func Validate(cfg *Config) error {
	var errs []string

	if cfg.ListenAddr == "" {
		errs = append(errs, "listen_addr is required")
	}
	if cfg.DatabaseURL == "" {
		errs = append(errs, "database_url is required")
	}
	if cfg.DatabaseSchema != "" && !schemaNamePattern.MatchString(cfg.DatabaseSchema) {
		errs = append(errs, "database_schema must match ^[a-z_][a-z0-9_]*$")
	}
	if cfg.EncryptionKey != "" && len(cfg.EncryptionKey) < 16 {
		errs = append(errs, "encryption_key must be at least 16 characters")
	}
	if cfg.RateLimitRPS < 0 {
		errs = append(errs, "rate_limit_rps must be >= 0")
	}
	if cfg.RateLimitBurst < 0 {
		errs = append(errs, "rate_limit_burst must be >= 0")
	}
	if cfg.MaxDBConns > 0 && cfg.MinDBConns > 0 && cfg.MaxDBConns <= cfg.MinDBConns {
		errs = append(errs, fmt.Sprintf("max_db_conns (%d) must be greater than min_db_conns (%d)", cfg.MaxDBConns, cfg.MinDBConns))
	}
	if cfg.CBFailureThreshold < 0 {
		errs = append(errs, "cb_failure_threshold must be >= 0")
	}
	if cfg.CBTimeoutSeconds < 0 {
		errs = append(errs, "cb_timeout_seconds must be >= 0")
	}
	if cfg.RetryMaxAttempts < 0 {
		errs = append(errs, "retry_max_attempts must be >= 0")
	}

	if len(errs) > 0 {
		return errors.New("config validation failed: " + strings.Join(errs, "; "))
	}
	return nil
}

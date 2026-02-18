package config

import (
	"strings"
	"testing"
)

func TestValidateValidConfig(t *testing.T) {
	cfg := &Config{
		ListenAddr:     ":8080",
		DatabaseURL:    "postgres://user:pass@localhost:5432/db",
		DatabaseSchema: "pxbin",
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}
}

func TestValidateMissingDatabaseURL(t *testing.T) {
	cfg := &Config{
		ListenAddr: ":8080",
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing database_url")
	}
	if !strings.Contains(err.Error(), "database_url") {
		t.Fatalf("expected database_url error, got: %v", err)
	}
}

func TestValidateInvalidDatabaseSchema(t *testing.T) {
	cfg := &Config{
		ListenAddr:     ":8080",
		DatabaseURL:    "postgres://localhost/db",
		DatabaseSchema: "Bad-Schema",
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid database_schema")
	}
	if !strings.Contains(err.Error(), "database_schema") {
		t.Fatalf("expected database_schema error, got: %v", err)
	}
}

func TestValidateShortEncryptionKey(t *testing.T) {
	cfg := &Config{
		ListenAddr:    ":8080",
		DatabaseURL:   "postgres://localhost/db",
		EncryptionKey: "short",
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for short encryption_key")
	}
	if !strings.Contains(err.Error(), "encryption_key") {
		t.Fatalf("expected encryption_key error, got: %v", err)
	}
}

func TestValidateNegativeRateLimitRPS(t *testing.T) {
	cfg := &Config{
		ListenAddr:   ":8080",
		DatabaseURL:  "postgres://localhost/db",
		RateLimitRPS: -1,
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for negative rate_limit_rps")
	}
}

func TestValidateMaxDBConnsLessThanMin(t *testing.T) {
	cfg := &Config{
		ListenAddr:  ":8080",
		DatabaseURL: "postgres://localhost/db",
		MaxDBConns:  5,
		MinDBConns:  10,
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for max_db_conns <= min_db_conns")
	}
}

func TestValidateMissingListenAddr(t *testing.T) {
	cfg := &Config{
		DatabaseURL: "postgres://localhost/db",
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing listen_addr")
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	cfg := &Config{} // missing both listen_addr and database_url
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected errors")
	}
	if !strings.Contains(err.Error(), "listen_addr") || !strings.Contains(err.Error(), "database_url") {
		t.Fatalf("expected both errors, got: %v", err)
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TC-CFG-01
// Contract: the shipped config.yaml parses and carries the values the server and
// the bootstrap seeding depend on. A drift here silently changes the admin
// password or the database target.
func TestLoadConfig_ShippedFile(t *testing.T) {
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig(config.yaml) failed: %v", err)
	}

	if cfg.Server.Port != 8081 {
		t.Errorf("server.port = %d, want 8081", cfg.Server.Port)
	}
	if cfg.Database.Host != "127.0.0.1" || cfg.Database.Port != 3307 {
		t.Errorf("database endpoint = %s:%d, want 127.0.0.1:3307",
			cfg.Database.Host, cfg.Database.Port)
	}
	if cfg.Database.DBName != "aegis" {
		t.Errorf("database.dbname = %q, want %q", cfg.Database.DBName, "aegis")
	}
	if cfg.Database.Charset != "utf8mb4" {
		t.Errorf("database.charset = %q, want utf8mb4", cfg.Database.Charset)
	}
	if !cfg.Database.ParseTime {
		t.Error("database.parseTime = false; timestamp handling depends on it being true")
	}
	if cfg.Auth.JWTSecret == "" {
		t.Error("auth.jwt_secret is empty")
	}
	if cfg.Auth.TokenExpireHours != 72 {
		t.Errorf("auth.token_expire_hours = %d, want 72", cfg.Auth.TokenExpireHours)
	}
	if cfg.Security.DefaultAdminPassword != "trueone" {
		t.Errorf("security.default_admin_password = %q, want %q",
			cfg.Security.DefaultAdminPassword, "trueone")
	}
}

// TC-CFG-02
// Contract: LoadConfig publishes the parsed value to the package global that
// InitDB and every service reads.
func TestLoadConfig_PublishesGlobal(t *testing.T) {
	previous := GlobalConfig
	t.Cleanup(func() { GlobalConfig = previous })

	if _, err := LoadConfig("config.yaml"); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if GlobalConfig == nil {
		t.Fatal("GlobalConfig is nil after a successful load")
	}
	if GlobalConfig.Database.DBName != "aegis" {
		t.Errorf("GlobalConfig.Database.DBName = %q, want aegis", GlobalConfig.Database.DBName)
	}
}

// TC-CFG-03
// Contract: a missing file is a hard, described error rather than a zero config,
// because a zero config would silently point the server at an empty database.
func TestLoadConfig_MissingFileFails(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("LoadConfig on a missing file returned nil error")
	}
}

// TC-CFG-04
// Contract: a type-mismatched YAML document is rejected instead of being
// partially applied.
func TestLoadConfig_MalformedYAMLFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: [1, 2, 3]\n"), 0o600); err != nil {
		t.Fatalf("cannot write fixture: %v", err)
	}

	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig accepted a sequence where an int was required")
	}
}

// TC-CFG-05
// Contract: every field maps from its documented YAML key. A renamed key would
// otherwise fall back to the zero value without any error.
func TestLoadConfig_MapsAllKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "full.yaml")
	document := `server:
  port: 9999
database:
  host: db.internal
  port: 3306
  user: svc
  password: secret
  dbname: anubis
  charset: utf8
  parseTime: false
  loc: UTC
auth:
  jwt_secret: unit-test-secret
  token_expire_hours: 5
security:
  default_admin_password: pw-fixture
`
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatalf("cannot write fixture: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("server.port = %d, want 9999", cfg.Server.Port)
	}
	if cfg.Database.Host != "db.internal" || cfg.Database.Port != 3306 {
		t.Errorf("database endpoint = %s:%d, want db.internal:3306", cfg.Database.Host, cfg.Database.Port)
	}
	if cfg.Database.User != "svc" || cfg.Database.Password != "secret" {
		t.Errorf("database credentials = %s/%s, want svc/secret", cfg.Database.User, cfg.Database.Password)
	}
	if cfg.Database.DBName != "anubis" {
		t.Errorf("database.dbname = %q, want anubis", cfg.Database.DBName)
	}
	if cfg.Database.ParseTime {
		t.Error("database.parseTime = true, want false")
	}
	if cfg.Database.Loc != "UTC" {
		t.Errorf("database.loc = %q, want UTC", cfg.Database.Loc)
	}
	if cfg.Auth.JWTSecret != "unit-test-secret" || cfg.Auth.TokenExpireHours != 5 {
		t.Errorf("auth = %q/%d, want unit-test-secret/5", cfg.Auth.JWTSecret, cfg.Auth.TokenExpireHours)
	}
	if cfg.Security.DefaultAdminPassword != "pw-fixture" {
		t.Errorf("security.default_admin_password = %q, want pw-fixture", cfg.Security.DefaultAdminPassword)
	}
}

// TC-CFG-06
// Contract: InitDB refuses to run before LoadConfig, so the server cannot start
// against an undefined DSN.
func TestInitDB_RequiresLoadedConfig(t *testing.T) {
	previous := GlobalConfig
	t.Cleanup(func() { GlobalConfig = previous })
	GlobalConfig = nil

	db, err := InitDB()
	if err == nil {
		t.Fatal("InitDB succeeded with a nil GlobalConfig")
	}
	if db != nil {
		t.Error("InitDB returned a non-nil handle alongside an error")
	}
}

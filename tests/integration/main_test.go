// Package integration exercises the Anubis HTTP surface against the real MySQL
// schema, so that every assertion can be backed by row-level database evidence.
//
// The suite is opt-in because it talks to a shared database:
//
//	ANUBIS_IT=1 go test ./tests/integration/ -v
//
// Every fixture it creates is namespaced with the TESTARCH prefix and removed by
// t.Cleanup, so a run leaves no residual rows behind.
package integration

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/logger"
	"trueone-anubis/config"
)

const (
	// SkipReason explains why the suite did not run.
	SkipReason = "integration tests are opt-in; set ANUBIS_IT=1 to run them against MySQL"

	defaultOrgID     = "100001"
	defaultProjectID = "100001100001"
	adminUserID      = "admin"
	adminPassword    = "trueone"
)

var dbReady bool

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	if os.Getenv("ANUBIS_IT") != "1" {
		os.Exit(m.Run())
	}

	cfg, err := config.LoadConfig("../../config/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration setup: cannot load config: %v\n", err)
		os.Exit(1)
	}

	if _, err := config.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "integration setup: cannot connect to %s:%d/%s: %v\n",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, err)
		os.Exit(1)
	}

	// The service logs every statement at Info level; keep the test output readable.
	config.DB.Logger = logger.Default.LogMode(logger.Silent)

	if sqlDB, err := config.DB.DB(); err == nil {
		sqlDB.SetMaxOpenConns(8)
		sqlDB.SetMaxIdleConns(4)
	}

	dbReady = true
	os.Exit(m.Run())
}

// requireDB skips a test when the live database is unavailable.
func requireDB(t *testing.T) {
	t.Helper()
	if !dbReady {
		t.Skip(SkipReason)
	}
}

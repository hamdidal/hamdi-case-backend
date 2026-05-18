// Package testutil provides shared helpers for Go test suites in this module.
// It is imported only by *_test.go files; it is never compiled into production
// binaries.
package testutil

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewTestDB opens a pure-Go SQLite in-memory database, auto-migrates every
// model, and wires it into database.DB. It returns a cleanup function that
// must be deferred by the caller to close the connection and reset the global.
//
// Usage:
//
//	func TestXxx(t *testing.T) {
//	    defer testutil.NewTestDB(t)()
//	    ...
//	}
func NewTestDB(t *testing.T) func() {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("testutil.NewTestDB: open failed: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Product{},
		&models.Material{},
		&models.ProductCare{},
		&models.CareInstruction{},
		&models.AuditLog{},
		&models.ProductVersion{},
	); err != nil {
		t.Fatalf("testutil.NewTestDB: AutoMigrate failed: %v", err)
	}

	database.DB = db

	return func() {
		rawDB, _ := db.DB()
		_ = rawDB.Close()
		database.DB = nil
	}
}

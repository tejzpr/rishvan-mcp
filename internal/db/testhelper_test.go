package db

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite DB for testing and sets the package-level instance.
func setupTestDB(t *testing.T) {
	t.Helper()
	var err error
	instance, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := instance.AutoMigrate(&Request{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
}

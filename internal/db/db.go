package db

import (
	"os"
	"path/filepath"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	instance *gorm.DB
	once     sync.Once
	initErr  error
)

func Init() (*gorm.DB, error) {
	once.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			initErr = err
			return
		}

		dir := filepath.Join(home, ".rishvan-mcp")
		if err := os.MkdirAll(dir, 0755); err != nil {
			initErr = err
			return
		}

		dbPath := filepath.Join(dir, "app.db")
		instance, initErr = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err := instance.AutoMigrate(&Request{}); err != nil {
			initErr = err
			return
		}
	})
	return instance, initErr
}

func Get() *gorm.DB {
	return instance
}

// InitWithDB allows injecting a pre-configured *gorm.DB (useful for testing).
func InitWithDB(d *gorm.DB) {
	instance = d
}

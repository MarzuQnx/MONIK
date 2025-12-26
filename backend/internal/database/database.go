package database

import (
	"fmt"
	"monik-enterprise/internal/config"
	"monik-enterprise/internal/models"
	appLogger "monik-enterprise/pkg/logger"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var db *gorm.DB

// InitDB initializes the database connection
func InitDB(dbPath string) *gorm.DB {
	// Ensure the data directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		appLogger.Error("Failed to create database directory: %v", err)
		panic(err)
	}

	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		appLogger.Error("Failed to connect to database: %v", err)
		panic(err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("Failed to get database instance: %v", err)
		panic(err)
	}

	cfg := config.Load()
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConn)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConn)

	// Enable WAL mode and performance optimizations
	if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		appLogger.Error("Failed to set journal_mode to WAL: %v", err)
	}
	if err := db.Exec("PRAGMA synchronous=NORMAL;").Error; err != nil {
		appLogger.Error("Failed to set synchronous to NORMAL: %v", err)
	}
	if err := db.Exec("PRAGMA cache_size=-2000;").Error; err != nil {
		appLogger.Error("Failed to set cache_size: %v", err)
	}
	if err := db.Exec("PRAGMA temp_store=MEMORY;").Error; err != nil {
		appLogger.Error("Failed to set temp_store to MEMORY: %v", err)
	}

	appLogger.Info("Database connected successfully: %s", dbPath)
	return db
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return db
}

// CloseDB closes the database connection
func CloseDB() {
	if db != nil {
		sqlDB, _ := db.DB()
		sqlDB.Close()
		appLogger.Info("Database connection closed")
	}
}

// RunMigrations runs all database migrations
func RunMigrations(db *gorm.DB) {
	appLogger.Info("Running database migrations...")

	// Auto-migrate all models
	err := db.AutoMigrate(
		&models.Interface{},
		&models.TrafficSnapshot{},
		&models.CounterResetLog{},
		&models.MonthlyQuota{},
		&models.SystemInfo{},
	)

	if err != nil {
		appLogger.Error("Failed to run migrations: %v", err)
		panic(fmt.Sprintf("Migration failed: %v", err))
	}

	appLogger.Info("Database migrations completed successfully")
}

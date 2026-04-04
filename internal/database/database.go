package database

import (
	"fmt"
	"log"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *config.Config) *Database {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Set up GORM logger
	var gormLogger logger.Interface
	if cfg.Database.Debug {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Successfully connected to database")

	return &Database{
		DB:     db,
		Config: cfg,
	}
}

// Close closes the database connection
func (d *Database) Close() {
	sqlDB, err := d.DB.DB()
	if err != nil {
		log.Printf("Failed to get database instance: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("Failed to close database connection: %v", err)
	} else {
		log.Println("Database connection closed")
	}
}

// Migrate runs GORM auto-migration to create tables
func (d *Database) Migrate() error {
	log.Println("Running GORM auto-migration...")

	// Auto-migrate all models except AuditLog (handled by schema.sql)
	err := d.DB.AutoMigrate(
		&domain.User{},
		&domain.Role{},
		&domain.UserRole{},
		&domain.Company{},
		&domain.Department{},
		&domain.JobPosition{},
		&domain.Employee{},
		&domain.EmployeeWorkInformation{},
		&domain.Grade{},
		&domain.EmployeeGrade{},
		&domain.EmployeeContract{},
		&domain.LeaveType{},
		&domain.LeaveBalance{},
		&domain.LeaveRequest{},
		&domain.LeaveDocument{},
		&domain.ExpenseType{},    // Expense Management
		&domain.ExpenseRequest{}, // Expense Management
		&domain.Attachment{},     // Document Management System
		// Note: AuditLog is excluded - it's created by schema.sql
	)

	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	log.Println("Auto-migration completed successfully")
	return nil
}

// GetDB returns the underlying GORM database instance
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}

// Transaction executes a function within a database transaction
func (d *Database) Transaction(fn func(*gorm.DB) error) error {
	return d.DB.Transaction(fn)
}

// Health checks the database connection health
func (d *Database) Health() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

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
	dsn := cfg.Database.URL

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

	database := &Database{
		DB:     db,
		Config: cfg,
	}
	// Restore/migration with explicit IDs can leave serial sequences behind MAX(id).
	// Sync independently of AutoMigrate so assign/create do not hit PK unique collisions.
	if err := SyncCriticalIDSequences(db); err != nil {
		log.Printf("WARNING: failed to sync critical id sequences: %v", err)
	}

	return database
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
	if !d.Config.Database.AutoMigrate {
		log.Println("Database auto-migration is disabled by configuration")
		return nil
	}

	log.Println("Running GORM auto-migration...")

	d.DB.Config.DisableForeignKeyConstraintWhenMigrating = true

	// Auto-migrate all models except AuditLog (handled by schema.sql)
	err := d.DB.AutoMigrate(
		&domain.User{},
		&domain.UserSetting{},
		&domain.KvkkLog{},
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
		&domain.ExpenseType{},      // Expense Management
		&domain.ExpenseRequest{},   // Expense Management
		&domain.Attachment{},       // Document Management System (for all modules)
		&domain.Job{},              // Job Scheduler
		&domain.JobHistory{},       // Job Scheduler
		&domain.Event{},            // Event Management
		&domain.EventParticipant{}, // Event Management
		&domain.FAQ{},              // FAQ Management
		&domain.RequestType{},
		&domain.OtherRequest{},
		&domain.MailConfiguration{},      // Mail Configuration Module
		&domain.MailRecipient{},          // Mail Configuration Module
		&domain.PortalContract{},         // Portal Contract Approval Tracking
		&domain.EmployeePortalContract{}, // Portal Contract Approval Tracking Pivot
		&domain.Training{},              // Kartezya Akademi
		&domain.TrainingAssignment{},    // Kartezya Akademi — Çalışan Ataması
		&domain.TrainingCertificate{},   // Kartezya Akademi — Tamamlama Sertifikası
		&domain.AcademySurvey{},         // Kartezya Akademi — Anket Modülü
		&domain.AcademySurveyOption{},
		&domain.AcademySurveyResponse{},
		&domain.InventoryItem{}, // Inventory Management
		// Note: AuditLog is excluded - it's created by schema.sql
	)

	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}
	backfilledGrades, err := BackfillMissingGradeBounds(d.DB)
	if err != nil {
		return fmt.Errorf("grade bounds backfill failed: %w", err)
	}
	if backfilledGrades > 0 {
		log.Printf("Backfilled bounds for %d grade records", backfilledGrades)
	}

	// Partial unique indexes / CHECK constraints are not managed by GORM AutoMigrate.
	// Employee grade: full status migration (column + backfill + CHECKs + indexes)
	// runs here via ApplyEmployeeGradeStatusMigration (advisory-locked transaction).
	// schema/migrate_employee_grade_status.sql mirrors the same builder SQL for ops.
	if err := EnsureJobHistoryReferenceDateUniqueIndex(d.DB); err != nil {
		return fmt.Errorf("job history index migration failed: %w", err)
	}
	if err := EnsureEmployeeGradeStatusConstraints(d.DB); err != nil {
		return fmt.Errorf("employee grade status migration failed: %w", err)
	}

	seedPortalContracts(d.DB)

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

func seedPortalContracts(db *gorm.DB) {
	var count int64
	db.Model(&domain.PortalContract{}).Count(&count)
	if count == 0 {
		contracts := []domain.PortalContract{
			{
				Title:   "Personel Aydınlatma Metni (KVKK)",
				Content: "KVKK kapsamında personel aydınlatma metni içeriği...",
				Version: "v1.0",
			},
			{
				Title:   "Personel Gizlilik Sözleşmesi",
				Content: "Personel gizlilik sözleşmesi içeriği...",
				Version: "v1.0",
			},
			{
				Title:   "Rüşvet ve Yolsuzlukla Mücadele Politikası",
				Content: "Rüşvet ve yolsuzlukla mücadele politikası içeriği...",
				Version: "v1.0",
			},
			{
				Title:   "Fotoğraf ve Görsel Paylaşım İzin Metni",
				Content: "Fotoğraf ve görsel paylaşım izin metni içeriği...",
				Version: "v1.0",
			},
		}

		for _, c := range contracts {
			c.CreatedBy = "System"
			db.Create(&c)
		}
		log.Println("Seeded default portal contracts successfully")
	}
}

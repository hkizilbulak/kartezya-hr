package database

import (
	"fmt"
	"testing"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newGradeBoundsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfg := &config.Config{}
	cfg.Database.TablePrefix = "hr"
	domain.SetConfig(cfg)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE hr_grades (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		min_year INTEGER,
		max_year INTEGER,
		deleted BOOLEAN NOT NULL DEFAULT false
	)`).Error; err != nil {
		t.Fatalf("create grades table: %v", err)
	}
	return db
}

func insertGradeBoundsTestRow(t *testing.T, db *gorm.DB, id int, name string, minYear, maxYear *int, deleted bool) {
	t.Helper()
	if err := db.Exec(
		`INSERT INTO hr_grades (id, name, min_year, max_year, deleted) VALUES (?, ?, ?, ?, ?)`,
		id, name, minYear, maxYear, deleted,
	).Error; err != nil {
		t.Fatalf("insert grade: %v", err)
	}
}

func readGradeBoundsTestRow(t *testing.T, db *gorm.DB, id int) (minYear, maxYear *int) {
	t.Helper()
	var row struct {
		MinYear *int
		MaxYear *int
	}
	if err := db.Raw(`SELECT min_year, max_year FROM hr_grades WHERE id = ?`, id).Scan(&row).Error; err != nil {
		t.Fatalf("read grade: %v", err)
	}
	return row.MinYear, row.MaxYear
}

func TestBackfillMissingGradeBoundsClosedAndIdempotent(t *testing.T) {
	db := newGradeBoundsTestDB(t)
	insertGradeBoundsTestRow(t, db, 1, "L3(4-6)", nil, nil, false)

	updated, err := BackfillMissingGradeBounds(db)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if updated != 1 {
		t.Fatalf("updated = %d, want 1", updated)
	}
	minYear, maxYear := readGradeBoundsTestRow(t, db, 1)
	if minYear == nil || *minYear != 4 || maxYear == nil || *maxYear != 6 {
		t.Fatalf("bounds = %v/%v, want 4/6", minYear, maxYear)
	}

	updated, err = BackfillMissingGradeBounds(db)
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if updated != 0 {
		t.Fatalf("second updated = %d, want 0", updated)
	}
}

func TestBackfillMissingGradeBoundsOpenEnded(t *testing.T) {
	db := newGradeBoundsTestDB(t)
	insertGradeBoundsTestRow(t, db, 1, "L6(10+)", nil, nil, false)

	updated, err := BackfillMissingGradeBounds(db)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if updated != 1 {
		t.Fatalf("updated = %d, want 1", updated)
	}
	minYear, maxYear := readGradeBoundsTestRow(t, db, 1)
	if minYear == nil || *minYear != 10 || maxYear != nil {
		t.Fatalf("bounds = %v/%v, want 10/NULL", minYear, maxYear)
	}

	updated, err = BackfillMissingGradeBounds(db)
	if err != nil || updated != 0 {
		t.Fatalf("second backfill updated=%d err=%v, want no-op", updated, err)
	}
}

func TestBackfillMissingGradeBoundsPreservesFilledAndDeleted(t *testing.T) {
	db := newGradeBoundsTestDB(t)
	filledMin, filledMax := 99, 100
	insertGradeBoundsTestRow(t, db, 1, "L3(4-6)", &filledMin, &filledMax, false)
	insertGradeBoundsTestRow(t, db, 2, "L4(6-8)", nil, nil, true)

	updated, err := BackfillMissingGradeBounds(db)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if updated != 0 {
		t.Fatalf("updated = %d, want 0", updated)
	}
	minYear, maxYear := readGradeBoundsTestRow(t, db, 1)
	if minYear == nil || *minYear != 99 || maxYear == nil || *maxYear != 100 {
		t.Fatalf("filled bounds changed: %v/%v", minYear, maxYear)
	}
	minYear, maxYear = readGradeBoundsTestRow(t, db, 2)
	if minYear != nil || maxYear != nil {
		t.Fatalf("deleted bounds changed: %v/%v", minYear, maxYear)
	}
}

func TestBackfillMissingGradeBoundsInvalidNameRollsBack(t *testing.T) {
	db := newGradeBoundsTestDB(t)
	insertGradeBoundsTestRow(t, db, 1, "L3(4-6)", nil, nil, false)
	insertGradeBoundsTestRow(t, db, 2, "invalid", nil, nil, false)

	if _, err := BackfillMissingGradeBounds(db); err == nil {
		t.Fatal("invalid name must fail")
	}
	minYear, maxYear := readGradeBoundsTestRow(t, db, 1)
	if minYear != nil || maxYear != nil {
		t.Fatalf("transaction did not roll back: %v/%v", minYear, maxYear)
	}
}

func TestDatabaseMigrateAutoMigrateFalseSkipsGradeBoundsBackfill(t *testing.T) {
	db := newGradeBoundsTestDB(t)
	insertGradeBoundsTestRow(t, db, 1, "L3(4-6)", nil, nil, false)
	cfg := &config.Config{}
	cfg.Database.TablePrefix = "hr"
	cfg.Database.AutoMigrate = false
	domain.SetConfig(cfg)

	database := &Database{DB: db, Config: cfg}
	if err := database.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	minYear, maxYear := readGradeBoundsTestRow(t, db, 1)
	if minYear != nil || maxYear != nil {
		t.Fatalf("DB_AUTO_MIGRATE=false changed bounds: %v/%v", minYear, maxYear)
	}
}

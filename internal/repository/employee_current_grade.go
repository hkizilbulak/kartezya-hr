package repository

import (
	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

// preloadActiveEmployeeGrade batch-loads the ACTIVE EmployeeGrade (+ Grade)
// for each employee in the result set (avoids N+1 on list/detail).
func preloadActiveEmployeeGrade(db *gorm.DB) *gorm.DB {
	return db.
		Preload("CurrentEmployeeGrade", func(tx *gorm.DB) *gorm.DB {
			return tx.Where("deleted = ? AND status = ?", false, domain.EmployeeGradeStatusActive)
		}).
		Preload("CurrentEmployeeGrade.Grade", func(tx *gorm.DB) *gorm.DB {
			return tx.Where("deleted = ?", false)
		})
}

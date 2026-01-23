package repository

import (
"fmt"
"kartezya-hr/internal/domain"
"kartezya-hr/internal/types"

"gorm.io/gorm"
)

type GradeRepository interface {
	Create(grade *domain.Grade, createdBy string) error
	GetByID(id int64) (*domain.Grade, error)
	GetByName(name string) (*domain.Grade, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Grade, int64, error)
	GetLookup() ([]domain.Grade, error)
	Update(grade *domain.Grade, modifiedBy string) error
	Delete(id int64, deletedBy string) error
	GetTotalCount() (int64, error)
}

type gradeRepository struct {
	db *gorm.DB
}

func NewGradeRepository(db *gorm.DB) GradeRepository {
	return &gradeRepository{db: db}
}

func (r *gradeRepository) Create(grade *domain.Grade, createdBy string) error {
	grade.CreatedBy = createdBy
	grade.ModifiedBy = createdBy
	return r.db.Create(grade).Error
}

func (r *gradeRepository) GetByID(id int64) (*domain.Grade, error) {
	var grade domain.Grade
	err := r.db.Where("id = ? AND deleted = ?", id, false).First(&grade).Error
	if err != nil {
		return nil, err
	}
	return &grade, nil
}

func (r *gradeRepository) GetByName(name string) (*domain.Grade, error) {
	var grade domain.Grade
	err := r.db.Where("name = ? AND deleted = ?", name, false).First(&grade).Error
	if err != nil {
		return nil, err
	}
	return &grade, nil
}

func (r *gradeRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Grade, int64, error) {
	var grades []*domain.Grade
	var total int64

	// Validate and sanitize sort field
	validSortFields := map[string]bool{
		"id":          true,
		"name":        true,
		"description": true,
		"created_at":  true,
		"updated_at":  true,
	}

	sortField := "id"
	if validSortFields[sortParams.Sort] {
		sortField = sortParams.Sort
	}

	direction := "ASC"
	if sortParams.Direction == "DESC" {
		direction = "DESC"
	}

	orderBy := fmt.Sprintf("%s %s", sortField, direction)

	// Count total records
	r.db.Model(&domain.Grade{}).Where("deleted = ?", false).Count(&total)

	// Get paginated records with sorting
	err := r.db.Where("deleted = ?", false).
		Order(orderBy).
		Limit(limit).Offset(offset).Find(&grades).Error

	return grades, total, err
}

func (r *gradeRepository) GetLookup() ([]domain.Grade, error) {
	var grades []domain.Grade
	err := r.db.Select("id, name").Where("deleted = ?", false).Order("name ASC").Find(&grades).Error
	return grades, err
}

func (r *gradeRepository) Update(grade *domain.Grade, modifiedBy string) error {
	grade.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(grade).Error
}

func (r *gradeRepository) Delete(id int64, deletedBy string) error {
	return r.db.Model(&domain.Grade{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
"deleted":     true,
"modified_by": deletedBy,
}).Error
}

// GetTotalCount returns the total number of grades
func (r *gradeRepository) GetTotalCount() (int64, error) {
	var count int64
	err := r.db.Model(&domain.Grade{}).Where("deleted = ?", false).Count(&count).Error
	return count, err
}

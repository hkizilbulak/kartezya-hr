package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type CompanyRepository interface {
	Create(company *domain.Company, createdBy string) error
	GetByID(id uint) (*domain.Company, error)
	GetByName(name string) (*domain.Company, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Company, int64, error)
	GetLookup() ([]domain.Company, error)
	Update(company *domain.Company, modifiedBy string) error
	Delete(id uint, deletedBy string) error
}

type companyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) CompanyRepository {
	return &companyRepository{db: db}
}

func (r *companyRepository) Create(company *domain.Company, createdBy string) error {
	company.CreatedBy = createdBy
	company.ModifiedBy = createdBy
	return r.db.Create(company).Error
}

func (r *companyRepository) GetByID(id uint) (*domain.Company, error) {
	var company domain.Company
	err := r.db.Preload("Departments").Where("id = ? AND deleted = ?", id, false).First(&company).Error
	if err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *companyRepository) GetByName(name string) (*domain.Company, error) {
	var company domain.Company
	err := r.db.Where("name = ? AND deleted = ?", name, false).First(&company).Error
	if err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *companyRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Company, int64, error) {
	var companies []*domain.Company
	var total int64

	// Validate and sanitize sort field
	validSortFields := map[string]bool{
		"id":         true,
		"name":       true,
		"address":    true,
		"phone":      true,
		"email":      true,
		"website":    true,
		"created_at": true,
		"updated_at": true,
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
	r.db.Model(&domain.Company{}).Where("deleted = ?", false).Count(&total)

	// Get paginated records with sorting and departments preload
	err := r.db.Preload("Departments").Where("deleted = ?", false).
		Order(orderBy).
		Limit(limit).Offset(offset).Find(&companies).Error

	return companies, total, err
}

func (r *companyRepository) GetLookup() ([]domain.Company, error) {
	var companies []domain.Company
	err := r.db.Select("id, name").Where("deleted = ?", false).Order("name ASC").Find(&companies).Error
	return companies, err
}

func (r *companyRepository) Update(company *domain.Company, modifiedBy string) error {
	company.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(company).Error
}

func (r *companyRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.Company{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

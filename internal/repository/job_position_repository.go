package repository

import (
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type JobPositionRepository interface {
	Create(jobPosition *domain.JobPosition, createdBy string) error
	GetByID(id uint) (*domain.JobPosition, error)
	GetByTitle(title string) (*domain.JobPosition, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.JobPosition, int64, error)
	GetLookup() ([]domain.JobPosition, error)
	Update(jobPosition *domain.JobPosition, modifiedBy string) error
	Delete(id uint, deletedBy string) error
}

type jobPositionRepository struct {
	db *gorm.DB
}

func NewJobPositionRepository(db *gorm.DB) JobPositionRepository {
	return &jobPositionRepository{db: db}
}

func (r *jobPositionRepository) Create(jobPosition *domain.JobPosition, createdBy string) error {
	jobPosition.CreatedBy = createdBy
	jobPosition.ModifiedBy = createdBy
	return r.db.Create(jobPosition).Error
}

func (r *jobPositionRepository) GetByID(id uint) (*domain.JobPosition, error) {
	var jobPosition domain.JobPosition
	err := r.db.Preload("EmployeeWorkInformation").
		Where("id = ? AND deleted = ?", id, false).First(&jobPosition).Error
	if err != nil {
		return nil, err
	}
	return &jobPosition, nil
}

func (r *jobPositionRepository) GetByTitle(title string) (*domain.JobPosition, error) {
	var jobPosition domain.JobPosition
	err := r.db.Where("title = ? AND deleted = ?", title, false).First(&jobPosition).Error
	if err != nil {
		return nil, err
	}
	return &jobPosition, nil
}

func (r *jobPositionRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.JobPosition, int64, error) {
	var jobPositions []*domain.JobPosition
	var total int64

	orderBy := buildJobPositionOrderClause(sortParams.Sort, sortParams.Direction)

	// Count total records
	r.db.Model(&domain.JobPosition{}).Where("deleted = ?", false).Count(&total)

	// Get paginated records with sorting
	err := r.db.Where("deleted = ?", false).
		Order(orderBy).
		Limit(limit).Offset(offset).Find(&jobPositions).Error

	return jobPositions, total, err
}

func (r *jobPositionRepository) GetLookup() ([]domain.JobPosition, error) {
	var jobPositions []domain.JobPosition
	err := r.db.Select("id, title").Where("deleted = ?", false).Order("title ASC").Find(&jobPositions).Error
	return jobPositions, err
}

func (r *jobPositionRepository) Update(jobPosition *domain.JobPosition, modifiedBy string) error {
	jobPosition.ModifiedBy = modifiedBy
	return r.db.Where("id = ? AND deleted = ?", jobPosition.ID, false).Save(jobPosition).Error
}

func (r *jobPositionRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.JobPosition{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

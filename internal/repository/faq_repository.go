package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type FAQRepository interface {
	Create(faq *domain.FAQ, createdBy string) error
	GetByID(id uint) (*domain.FAQ, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.FAQ, int64, error)
	Update(faq *domain.FAQ, modifiedBy string) error
	Delete(id uint, deletedBy string) error
}

type faqRepository struct {
	db *gorm.DB
}

func NewFAQRepository(db *gorm.DB) FAQRepository {
	return &faqRepository{db: db}
}

func (r *faqRepository) Create(faq *domain.FAQ, createdBy string) error {
	faq.CreatedBy = createdBy
	faq.ModifiedBy = createdBy
	return r.db.Create(faq).Error
}

func (r *faqRepository) GetByID(id uint) (*domain.FAQ, error) {
	var faq domain.FAQ
	err := r.db.Where("id = ? AND deleted = ?", id, false).First(&faq).Error
	if err != nil {
		return nil, err
	}
	return &faq, nil
}

func (r *faqRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.FAQ, int64, error) {
	var faqs []*domain.FAQ
	var total int64

	sortField := "created_at"
	direction := "DESC"

	if sortParams.Direction == "ASC" {
		direction = "ASC"
	}

	if sortParams.Sort != "" {
		switch sortParams.Sort {
		case "title":
			sortField = "title"
		case "status":
			sortField = "status"
		case "created_at":
			sortField = "created_at"
		case "updated_at":
			sortField = "updated_at"
		}
	}

	query := r.db.Model(&domain.FAQ{}).Where("deleted = ?", false)
	query.Count(&total)

	err := query.Order(fmt.Sprintf("%s %s", sortField, direction)).
		Limit(limit).Offset(offset).
		Find(&faqs).Error

	return faqs, total, err
}

func (r *faqRepository) Update(faq *domain.FAQ, modifiedBy string) error {
	faq.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(faq).Error
}

func (r *faqRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.FAQ{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}
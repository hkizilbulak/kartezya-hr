package repository

import (
	"fmt"
	"strings"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type ContractRepository interface {
	Create(contract *domain.Contract, createdBy string) error
	GetByID(id uint) (*domain.Contract, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Contract, int64, error)
	Update(contract *domain.Contract, modifiedBy string) error
	Delete(id uint, deletedBy string) error
}

type contractRepository struct {
	db *gorm.DB
}

func NewContractRepository(db *gorm.DB) ContractRepository {
	return &contractRepository{db: db}
}

func (r *contractRepository) Create(contract *domain.Contract, createdBy string) error {
	contract.CreatedBy = createdBy
	contract.ModifiedBy = createdBy
	return r.db.Create(contract).Error
}

func (r *contractRepository) GetByID(id uint) (*domain.Contract, error) {
	var contract domain.Contract
	if err := r.db.Where("id = ? AND deleted = ?", id, false).First(&contract).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("contract not found")
		}
		return nil, err
	}
	return &contract, nil
}

func (r *contractRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Contract, int64, error) {
	var contracts []*domain.Contract
	var total int64

	query := r.db.Model(&domain.Contract{}).Where("deleted = ?", false)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if sortParams.Sort != "" {
		direction := "ASC"
		if strings.ToUpper(sortParams.Direction) == "DESC" {
			direction = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", sortParams.Sort, direction))
	} else {
		query = query.Order("created_at DESC")
	}

	if err := query.Limit(limit).Offset(offset).Find(&contracts).Error; err != nil {
		return nil, 0, err
	}

	return contracts, total, nil
}

func (r *contractRepository) Update(contract *domain.Contract, modifiedBy string) error {
	contract.ModifiedBy = modifiedBy

	updates := map[string]interface{}{
		"customer_contact_name":  contract.CustomerContactName,
		"customer_contact_phone": contract.CustomerContactPhone,
		"customer_contact_email": contract.CustomerContactEmail,
		"project_name":           contract.ProjectName,
		"contract_no":            contract.ContractNo,
		"start_date":             contract.StartDate,
		"end_date":               contract.EndDate,
		"status":                 contract.Status,
		"modified_by":            modifiedBy,
	}

	return r.db.Where("deleted = ?", false).Model(contract).Updates(updates).Error
}

func (r *contractRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.Contract{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

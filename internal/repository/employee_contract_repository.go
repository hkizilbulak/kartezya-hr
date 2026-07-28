package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type EmployeeContractRepository interface {
	Create(contract *domain.EmployeeContract, createdBy string) error
	GetByID(id uint) (*domain.EmployeeContract, error)
	GetByContractAndEmployeeIncludingDeleted(contractID uint, employeeID uint) (*domain.EmployeeContract, error)
	GetByEmployeeID(employeeID uint, page int, limit int) ([]*domain.EmployeeContract, int64, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.EmployeeContract, int64, error)
	CheckExists(employeeID uint, contractID uint) (bool, error)
	ReviveByContractAndEmployee(contractID uint, employeeID uint, modifiedBy string) error
	Update(contract *domain.EmployeeContract, modifiedBy string) error
	Delete(id uint, deletedBy string) error
	DeleteByContractAndEmployee(contractID uint, employeeID uint, deletedBy string) error
	GetTotalCount() (int64, error)
}

type employeeContractRepository struct {
	db *gorm.DB
}

func NewEmployeeContractRepository(db *gorm.DB) EmployeeContractRepository {
	return &employeeContractRepository{db: db}
}

func (r *employeeContractRepository) Create(contract *domain.EmployeeContract, createdBy string) error {
	contract.CreatedBy = createdBy
	contract.ModifiedBy = createdBy
	return r.db.Create(contract).Error
}

func (r *employeeContractRepository) GetByID(id uint) (*domain.EmployeeContract, error) {
	var contract domain.EmployeeContract
	err := r.db.Preload("Employee").Where("deleted = ?", false).First(&contract, id).Error
	return &contract, err
}

func (r *employeeContractRepository) GetByContractAndEmployeeIncludingDeleted(contractID uint, employeeID uint) (*domain.EmployeeContract, error) {
	var contract domain.EmployeeContract
	err := r.db.
		Where("contract_id = ? AND employee_id = ?", contractID, employeeID).
		First(&contract).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &contract, nil
}

func (r *employeeContractRepository) GetByEmployeeID(employeeID uint, page int, limit int) ([]*domain.EmployeeContract, int64, error) {
	var contracts []*domain.EmployeeContract
	var total int64

	offset := (page - 1) * limit

	err := r.db.Model(&domain.EmployeeContract{}).
		Where("employee_id = ? AND deleted = ?", employeeID, false).
		Count(&total).
		Preload("Employee").
		Preload("Contract").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&contracts).Error

	if err != nil {
		return nil, 0, err
	}

	return contracts, total, nil
}

// CheckExists checks if an employee contract already exists
func (r *employeeContractRepository) CheckExists(employeeID uint, contractID uint) (bool, error) {
	var count int64
	err := r.db.Model(&domain.EmployeeContract{}).
		Where("employee_id = ? AND contract_id = ? AND deleted = ?", employeeID, contractID, false).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *employeeContractRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.EmployeeContract, int64, error) {
	var contracts []*domain.EmployeeContract
	var total int64

	validSortFields := map[string]bool{
		"id":         true,
		"start_date": true,
		"end_date":   true,
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

	r.db.Model(&domain.EmployeeContract{}).Where("deleted = ?", false).Count(&total)

	err := r.db.Preload("Employee").Preload("Contract").
		Where("deleted = ?", false).
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&contracts).Error

	return contracts, total, err
}

func (r *employeeContractRepository) Update(contract *domain.EmployeeContract, modifiedBy string) error {
	contract.ModifiedBy = modifiedBy

	updates := map[string]interface{}{
		"contract_id": contract.ContractID,
		"modified_by": modifiedBy,
	}

	return r.db.Where("deleted = ?", false).Model(contract).Updates(updates).Error
}

func (r *employeeContractRepository) ReviveByContractAndEmployee(contractID uint, employeeID uint, modifiedBy string) error {
	return r.db.Model(&domain.EmployeeContract{}).
		Where("contract_id = ? AND employee_id = ? AND deleted = ?", contractID, employeeID, true).
		Updates(map[string]interface{}{
			"deleted":     false,
			"modified_by": modifiedBy,
		}).Error
}

func (r *employeeContractRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.EmployeeContract{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

func (r *employeeContractRepository) DeleteByContractAndEmployee(contractID uint, employeeID uint, deletedBy string) error {
	return r.db.Model(&domain.EmployeeContract{}).
		Where("contract_id = ? AND employee_id = ? AND deleted = ?", contractID, employeeID, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

func (r *employeeContractRepository) GetTotalCount() (int64, error) {
	var count int64
	err := r.db.Model(&domain.EmployeeContract{}).Where("deleted = ?", false).Count(&count).Error
	return count, err
}

package repository

import (
	"kartezya-hr/internal/domain"
	"gorm.io/gorm"
)

type PortalContractRepository interface {
	GetAllActiveContracts() ([]*domain.PortalContract, error)
	GetEmployeeApprovals(employeeID uint) ([]*domain.EmployeePortalContract, error)
	SaveApproval(approval *domain.EmployeePortalContract) error
	GetUserSettingByUserID(userID uint) (*domain.UserSetting, error)
	GetKvkkLogsByUserID(userID uint) ([]*domain.KvkkLog, error)
}

type portalContractRepository struct {
	db *gorm.DB
}

func NewPortalContractRepository(db *gorm.DB) PortalContractRepository {
	return &portalContractRepository{db: db}
}

func (r *portalContractRepository) GetAllActiveContracts() ([]*domain.PortalContract, error) {
	var contracts []*domain.PortalContract
	err := r.db.Where("deleted = ?", false).Order("id ASC").Find(&contracts).Error
	return contracts, err
}

func (r *portalContractRepository) GetEmployeeApprovals(employeeID uint) ([]*domain.EmployeePortalContract, error) {
	var approvals []*domain.EmployeePortalContract
	err := r.db.Where("employee_id = ? AND deleted = ?", employeeID, false).Find(&approvals).Error
	return approvals, err
}

func (r *portalContractRepository) SaveApproval(approval *domain.EmployeePortalContract) error {
	return r.db.Save(approval).Error
}

func (r *portalContractRepository) GetUserSettingByUserID(userID uint) (*domain.UserSetting, error) {
	var setting domain.UserSetting
	err := r.db.Where("user_id = ? AND deleted = ?", userID, false).First(&setting).Error
	return &setting, err
}

func (r *portalContractRepository) GetKvkkLogsByUserID(userID uint) ([]*domain.KvkkLog, error) {
	var logs []*domain.KvkkLog
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&logs).Error
	return logs, err
}

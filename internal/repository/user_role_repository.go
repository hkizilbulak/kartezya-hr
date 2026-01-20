package repository

import (
	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

type UserRoleRepository interface {
	Create(userRole *domain.UserRole, createdBy string) error
	DeleteByUserID(userID uint, deletedBy string) error
	GetRolesByUserID(userID uint) ([]domain.Role, error)
	HasRole(userID uint, roleName string) (bool, error)
	Update(userRole *domain.UserRole, modifiedBy string) error
	Delete(userID, roleID uint, deletedBy string) error
}

type userRoleRepository struct {
	db *gorm.DB
}

func NewUserRoleRepository(db *gorm.DB) UserRoleRepository {
	return &userRoleRepository{db: db}
}

func (r *userRoleRepository) Create(userRole *domain.UserRole, createdBy string) error {
	userRole.CreatedBy = createdBy
	userRole.ModifiedBy = createdBy
	userRole.Deleted = false
	return r.db.Create(userRole).Error
}

func (r *userRoleRepository) DeleteByUserID(userID uint, deletedBy string) error {
	return r.db.Model(&domain.UserRole{}).
		Where("user_id = ? AND deleted = ?", userID, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

func (r *userRoleRepository) GetRolesByUserID(userID uint) ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.Table("hr_roles").
		Joins("JOIN hr_user_roles ON hr_roles.id = hr_user_roles.role_id").
		Where("hr_user_roles.user_id = ? AND hr_user_roles.deleted = ?", userID, false).
		Where("hr_roles.deleted = ?", false).
		Find(&roles).Error
	return roles, err
}

func (r *userRoleRepository) HasRole(userID uint, roleName string) (bool, error) {
	var count int64
	err := r.db.Table("hr_user_roles").
		Joins("JOIN hr_roles ON hr_user_roles.role_id = hr_roles.id").
		Where("hr_user_roles.user_id = ? AND hr_roles.name = ?", userID, roleName).
		Where("hr_user_roles.deleted = ? AND hr_roles.deleted = ?", false, false).
		Count(&count).Error
	return count > 0, err
}

func (r *userRoleRepository) Update(userRole *domain.UserRole, modifiedBy string) error {
	userRole.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(userRole).Error
}

func (r *userRoleRepository) Delete(userID, roleID uint, deletedBy string) error {
	return r.db.Model(&domain.UserRole{}).
		Where("user_id = ? AND role_id = ? AND deleted = ?", userID, roleID, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

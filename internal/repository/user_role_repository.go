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
	err := r.db.Table("roles").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND user_roles.deleted = ?", userID, false).
		Where("roles.deleted = ?", false).
		Find(&roles).Error
	return roles, err
}

func (r *userRoleRepository) HasRole(userID uint, roleName string) (bool, error) {
	var count int64
	err := r.db.Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, roleName).
		Where("user_roles.deleted = ? AND roles.deleted = ?", false, false).
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

package repository

import (
	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

type RoleRepository interface {
	Create(role *domain.Role, createdBy string) error
	GetByID(id uint) (*domain.Role, error)
	GetByName(name string) (*domain.Role, error)
	Update(role *domain.Role, modifiedBy string) error
	Delete(id uint, deletedBy string) error
	List() ([]domain.Role, error)
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(role *domain.Role, createdBy string) error {
	role.CreatedBy = createdBy
	role.ModifiedBy = createdBy
	return r.db.Create(role).Error
}

func (r *roleRepository) GetByID(id uint) (*domain.Role, error) {
	var role domain.Role
	err := r.db.Where("deleted = ?", false).First(&role, id).Error
	return &role, err
}

func (r *roleRepository) GetByName(name string) (*domain.Role, error) {
	var role domain.Role
	err := r.db.Where("name = ? AND deleted = ?", name, false).First(&role).Error
	return &role, err
}

func (r *roleRepository) Update(role *domain.Role, modifiedBy string) error {
	role.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(role).Error
}

func (r *roleRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.Role{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

func (r *roleRepository) List() ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.Where("deleted = ?", false).Find(&roles).Error
	return roles, err
}

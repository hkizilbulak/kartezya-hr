package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
	"time"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *domain.User, createdBy string) error
	GetByID(id uint) (*domain.User, error)
	GetByEmail(email string) (*domain.User, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.User, int64, error)
	Update(user *domain.User, modifiedBy string) error
	Delete(id uint, deletedBy string) error
	GetWithRoles(id uint) (*domain.User, error)
	GetEmployeeByUserID(userID uint) (*domain.Employee, error)
	UpdatePasswordResetToken(userID uint, token string, expiresAt *time.Time) error
	GetByPasswordResetToken(token string) (*domain.User, error)
	ClearPasswordResetToken(userID uint) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *domain.User, createdBy string) error {
	user.CreatedBy = createdBy
	user.ModifiedBy = createdBy
	return r.db.Create(user).Error
}

func (r *userRepository) GetByID(id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("deleted = ?", false).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ? AND deleted = ?", email, false).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.User, int64, error) {
	var users []*domain.User
	var total int64

	// Validate and sanitize sort field
	validSortFields := map[string]bool{
		"id":         true,
		"email":      true,
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
	r.db.Model(&domain.User{}).Where("deleted = ?", false).Count(&total)

	// Get paginated records with sorting
	err := r.db.Where("deleted = ?", false).
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&users).Error

	return users, total, err
}

func (r *userRepository) Update(user *domain.User, modifiedBy string) error {
	user.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(user).Error
}

func (r *userRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.User{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

func (r *userRepository) GetWithRoles(id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.Preload("UserRoles.Role").Where("deleted = ?", false).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetEmployeeByUserID(userID uint) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.Where("user_id = ? AND deleted = ?", userID, false).First(&employee).Error
	if err != nil {
		return nil, err
	}
	return &employee, nil
}

// UpdatePasswordResetToken updates the password reset token for a user
func (r *userRepository) UpdatePasswordResetToken(userID uint, token string, expiresAt *time.Time) error {
	return r.db.Model(&domain.User{}).
		Where("id = ? AND deleted = ?", userID, false).
		Updates(map[string]interface{}{
			"password_reset_token":   token,
			"password_reset_expires": expiresAt,
		}).Error
}

// GetByPasswordResetToken retrieves a user by password reset token
func (r *userRepository) GetByPasswordResetToken(token string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("password_reset_token = ? AND deleted = ? AND password_reset_expires > ?", token, false, time.Now()).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ClearPasswordResetToken clears the password reset token for a user
func (r *userRepository) ClearPasswordResetToken(userID uint) error {
	return r.db.Model(&domain.User{}).
		Where("id = ? AND deleted = ?", userID, false).
		Updates(map[string]interface{}{
			"password_reset_token":   nil,
			"password_reset_expires": nil,
		}).Error
}

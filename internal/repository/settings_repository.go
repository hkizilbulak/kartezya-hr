package repository

import (
	"kartezya-hr/internal/domain"
	"time"

	"gorm.io/gorm"
)

type SettingsRepository interface {
	GetByUserID(userID uint) (*domain.UserSetting, error)
	Create(setting *domain.UserSetting) error
	Update(setting *domain.UserSetting) error
	CreateKvkkLog(log *domain.KvkkLog) error
}

type settingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) GetByUserID(userID uint) (*domain.UserSetting, error) {
	var setting domain.UserSetting
	err := r.db.Where("user_id = ? AND deleted = ?", userID, false).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *settingsRepository) Create(setting *domain.UserSetting) error {
	now := time.Now()
	setting.CreatedAt = now
	setting.UpdatedAt = now
	return r.db.Create(setting).Error
}

func (r *settingsRepository) Update(setting *domain.UserSetting) error {
	setting.UpdatedAt = time.Now()
	return r.db.Model(setting).Updates(setting).Error
}

func (r *settingsRepository) CreateKvkkLog(log *domain.KvkkLog) error {
	log.CreatedAt = time.Now()
	return r.db.Create(log).Error
}

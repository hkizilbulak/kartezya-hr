package repository

import (
	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

type MailConfigRepository interface {
	GetAll() ([]domain.MailConfiguration, error)
	GetByID(id uint) (*domain.MailConfiguration, error)
	GetByKey(mailKey string) (*domain.MailConfiguration, error)
	Create(cfg *domain.MailConfiguration) error
	Update(cfg *domain.MailConfiguration) error
	Delete(id uint) error
	ReplaceRecipients(mailConfigID uint, recipients []domain.MailRecipient) error
}

type mailConfigRepository struct {
	db *gorm.DB
}

func NewMailConfigRepository(db *gorm.DB) MailConfigRepository {
	return &mailConfigRepository{db: db}
}

func (r *mailConfigRepository) GetAll() ([]domain.MailConfiguration, error) {
	var configs []domain.MailConfiguration
	err := r.db.Preload("Recipients").Order("mail_key ASC").Find(&configs).Error
	return configs, err
}

func (r *mailConfigRepository) GetByID(id uint) (*domain.MailConfiguration, error) {
	var cfg domain.MailConfiguration
	err := r.db.Preload("Recipients").First(&cfg, id).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *mailConfigRepository) GetByKey(mailKey string) (*domain.MailConfiguration, error) {
	var cfg domain.MailConfiguration
	err := r.db.Preload("Recipients").Where("mail_key = ?", mailKey).First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *mailConfigRepository) Create(cfg *domain.MailConfiguration) error {
	return r.db.Create(cfg).Error
}

func (r *mailConfigRepository) Update(cfg *domain.MailConfiguration) error {
	return r.db.Save(cfg).Error
}

func (r *mailConfigRepository) Delete(id uint) error {
	return r.db.Delete(&domain.MailConfiguration{}, id).Error
}

func (r *mailConfigRepository) ReplaceRecipients(mailConfigID uint, recipients []domain.MailRecipient) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing recipients
		if err := tx.Where("mail_config_id = ?", mailConfigID).Delete(&domain.MailRecipient{}).Error; err != nil {
			return err
		}
		// Insert new ones
		if len(recipients) == 0 {
			return nil
		}
		for i := range recipients {
			recipients[i].MailConfigID = mailConfigID
		}
		return tx.Create(&recipients).Error
	})
}

package repository

import (
	"kartezya-hr/internal/domain"
	"time"

	"gorm.io/gorm"
)

type AuditRepository interface {
	Create(audit *domain.AuditLog) error
	GetByEntityID(entityName string, entityID uint) ([]domain.AuditLog, error)
	List(limit, offset int) ([]domain.AuditLog, error)
}

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(audit *domain.AuditLog) error {
	audit.CreatedDate = time.Now()
	return r.db.Create(audit).Error
}

func (r *auditRepository) GetByEntityID(entityName string, entityID uint) ([]domain.AuditLog, error) {
	var audits []domain.AuditLog
	err := r.db.Where("entity_name = ? AND entity_id = ?", entityName, entityID).
		Order("created_date DESC").
		Find(&audits).Error
	return audits, err
}

func (r *auditRepository) List(limit, offset int) ([]domain.AuditLog, error) {
	var audits []domain.AuditLog
	err := r.db.Limit(limit).Offset(offset).
		Order("created_date DESC").
		Find(&audits).Error
	return audits, err
}

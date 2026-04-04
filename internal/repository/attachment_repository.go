package repository

import (
	"errors"
	"time"

	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

type AttachmentRepository interface {
	Create(attachment *domain.Attachment) error
	FindByID(id string) (*domain.Attachment, error)
	FindByOwnerID(ownerID uint) ([]domain.Attachment, error)
	FindByRelatedRecord(relatedType domain.AttachmentRelatedType, relatedID uint) ([]domain.Attachment, error)
	CountByRelatedRecord(relatedType domain.AttachmentRelatedType, relatedID uint) (int64, error)
	FindTemporaryOlderThan(hours int) ([]domain.Attachment, error)
	UpdateStatus(id string, status domain.AttachmentStatus, relatedID *uint) error
	Delete(id string) error
	PhysicalDelete(id string) error
	CheckHashExists(hash string, ownerID uint) (bool, error)
	LinkToRecord(ids []string, relatedType domain.AttachmentRelatedType, relatedID uint) error
	BeginTransaction() *gorm.DB
}

type attachmentRepository struct {
	db *gorm.DB
}

func NewAttachmentRepository(db *gorm.DB) AttachmentRepository {
	return &attachmentRepository{db: db}
}

// Create inserts a new attachment
func (r *attachmentRepository) Create(attachment *domain.Attachment) error {
	return r.db.Create(attachment).Error
}

// FindByID retrieves an attachment by its GUID
func (r *attachmentRepository) FindByID(id string) (*domain.Attachment, error) {
	var attachment domain.Attachment
	err := r.db.Where("id = ?", id).
		Preload("Owner").
		First(&attachment).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attachment, nil
}

// FindByOwnerID retrieves all attachments uploaded by a user
func (r *attachmentRepository) FindByOwnerID(ownerID uint) ([]domain.Attachment, error) {
	var attachments []domain.Attachment
	err := r.db.Where("owner_id = ? AND status != ?", ownerID, domain.AttachmentStatusArchived).
		Order("created_at DESC").
		Find(&attachments).Error
	return attachments, err
}

// FindByRelatedRecord retrieves all attachments linked to a specific record
func (r *attachmentRepository) FindByRelatedRecord(relatedType domain.AttachmentRelatedType, relatedID uint) ([]domain.Attachment, error) {
	var attachments []domain.Attachment

	// For Employee attachments, also consider records where the user is the owner,
	// even if they are not explicitly or properly linked yet.
	if relatedType == domain.AttachmentRelatedTypeEmployee {
		err := r.db.Where("related_type = ? AND (related_id = ? OR owner_id = ?) AND status != ?",
			relatedType, relatedID, relatedID, domain.AttachmentStatusArchived).
			Order("created_at DESC").
			Find(&attachments).Error
		return attachments, err
	}

	err := r.db.Where("related_type = ? AND related_id = ? AND status = ?",
		relatedType, relatedID, domain.AttachmentStatusLinked).
		Order("created_at DESC").
		Find(&attachments).Error
	return attachments, err
}

// CountByRelatedRecord counts attachments linked to a specific record
func (r *attachmentRepository) CountByRelatedRecord(relatedType domain.AttachmentRelatedType, relatedID uint) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Attachment{}).
		Where("related_type = ? AND related_id = ? AND status = ?",
			relatedType, relatedID, domain.AttachmentStatusLinked).
		Count(&count).Error
	return count, err
}

// FindTemporaryOlderThan finds temporary attachments older than specified hours (for cleanup job)
func (r *attachmentRepository) FindTemporaryOlderThan(hours int) ([]domain.Attachment, error) {
	var attachments []domain.Attachment
	cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	err := r.db.Where("status = ? AND created_at < ?",
		domain.AttachmentStatusTemporary, cutoffTime).
		Find(&attachments).Error
	return attachments, err
}

// UpdateStatus updates attachment status and optionally links it to a record
func (r *attachmentRepository) UpdateStatus(id string, status domain.AttachmentStatus, relatedID *uint) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if relatedID != nil {
		updates["related_id"] = *relatedID
	}

	return r.db.Model(&domain.Attachment{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Delete marks attachment as archived (soft delete)
func (r *attachmentRepository) Delete(id string) error {
	return r.UpdateStatus(id, domain.AttachmentStatusArchived, nil)
}

// PhysicalDelete permanently removes attachment from database
func (r *attachmentRepository) PhysicalDelete(id string) error {
	return r.db.Unscoped().Delete(&domain.Attachment{}, "id = ?", id).Error
}

// CheckHashExists checks if a file with the same hash already exists for this user
func (r *attachmentRepository) CheckHashExists(hash string, ownerID uint) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Attachment{}).
		Where("hash = ? AND owner_id = ? AND status != ?",
			hash, ownerID, domain.AttachmentStatusArchived).
		Count(&count).Error

	return count > 0, err
}

// LinkToRecord links multiple attachments to a specific record (transaction-safe)
func (r *attachmentRepository) LinkToRecord(ids []string, relatedType domain.AttachmentRelatedType, relatedID uint) error {
	if len(ids) == 0 {
		return nil
	}

	return r.db.Model(&domain.Attachment{}).
		Where("id IN ? AND status = ?", ids, domain.AttachmentStatusTemporary).
		Updates(map[string]interface{}{
			"related_type": relatedType,
			"related_id":   relatedID,
			"status":       domain.AttachmentStatusLinked,
			"updated_at":   time.Now(),
		}).Error
}

// BeginTransaction starts a new database transaction
func (r *attachmentRepository) BeginTransaction() *gorm.DB {
	return r.db.Begin()
}

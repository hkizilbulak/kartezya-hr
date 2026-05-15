package repository

import (
	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

type EventParticipantRepository interface {
	Create(participant *domain.EventParticipant, createdBy string) error
	GetByEventAndUser(eventId uint, userId uint) (*domain.EventParticipant, error)
	GetByEventID(eventId uint) ([]*domain.EventParticipant, error)
	Update(participant *domain.EventParticipant, modifiedBy string) error
}

type eventParticipantRepository struct {
	db *gorm.DB
}

func NewEventParticipantRepository(db *gorm.DB) EventParticipantRepository {
	return &eventParticipantRepository{db: db}
}

func (r *eventParticipantRepository) Create(participant *domain.EventParticipant, createdBy string) error {
	participant.CreatedBy = createdBy
	participant.ModifiedBy = createdBy
	return r.db.Create(participant).Error
}

func (r *eventParticipantRepository) GetByEventAndUser(eventId uint, userId uint) (*domain.EventParticipant, error) {
	var participant domain.EventParticipant
	err := r.db.Where("event_id = ? AND user_id = ? AND deleted = ?", eventId, userId, false).
		First(&participant).Error
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

func (r *eventParticipantRepository) GetByEventID(eventId uint) ([]*domain.EventParticipant, error) {
	var participants []*domain.EventParticipant
	err := r.db.Preload("User").Preload("User.Employee").Preload("User.Employee.EmployeeWorkInformation.Department").
		Where("event_id = ? AND deleted = ?", eventId, false).
		Find(&participants).Error
	return participants, err
}

func (r *eventParticipantRepository) Update(participant *domain.EventParticipant, modifiedBy string) error {
	participant.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(participant).Error
}

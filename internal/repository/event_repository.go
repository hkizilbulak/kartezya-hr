package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
	"time"

	"gorm.io/gorm"
)

type EventRepository interface {
	Create(event *domain.Event, createdBy string) error
	GetByID(id uint) (*domain.Event, error)
	GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Event, int64, error)
	Update(event *domain.Event, modifiedBy string) error
	Delete(id uint, deletedBy string) error

	// Dashboard: Get active events for a specific user
	GetActiveEventsForDashboard(userID uint) ([]*domain.Event, error)
	// Dashboard: Count published events with future end_date
	CountActiveEvents() (int64, error)
}

type eventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Create(event *domain.Event, createdBy string) error {
	event.CreatedBy = createdBy
	event.ModifiedBy = createdBy
	return r.db.Create(event).Error
}

func (r *eventRepository) GetByID(id uint) (*domain.Event, error) {
	var event domain.Event
	err := r.db.Where(fmt.Sprintf("%s.id = ? AND %s.deleted = ?", domain.GetTableName("events"), domain.GetTableName("events")), id, false).
		Preload("Participants", "deleted = ?", false).
		Preload("Participants.User").
		Preload("Participants.User.Employee").
		First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *eventRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Event, int64, error) {
	var events []*domain.Event
	var total int64

	query := r.db.Model(&domain.Event{}).Where("deleted = ?", false)

	// Count total records
	query.Count(&total)

	// Execute query — allowlisted ORDER BY before LIMIT/OFFSET
	err := query.Order(buildEventOrderClause(sortParams.Sort, sortParams.Direction)).
		Preload("Participants", "deleted = ?", false).
		Preload("Participants.User.Employee").
		Limit(limit).Offset(offset).
		Find(&events).Error

	return events, total, err
}

func (r *eventRepository) Update(event *domain.Event, modifiedBy string) error {
	event.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(event).Error
}

func (r *eventRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.Event{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

func (r *eventRepository) GetActiveEventsForDashboard(userID uint) ([]*domain.Event, error) {
	var events []*domain.Event
	now := time.Now()

	eventsTable := domain.GetTableName("events")
	participantsTable := domain.GetTableName("event_participants")

	query := r.db.Model(&domain.Event{}).Where(eventsTable+".deleted = ? AND "+eventsTable+".status = ? AND "+eventsTable+".end_date >= ?", false, domain.EventStatusPublished, now)

	query = query.Where(`
		`+eventsTable+`.audience_filter = ? 
		OR 
		EXISTS (SELECT 1 FROM `+participantsTable+` p WHERE p.event_id = `+eventsTable+`.id AND p.user_id = ? AND p.deleted = false)
	`, string(domain.EventAudienceAllCompany), userID)

	err := query.Order("start_date ASC").Find(&events).Error
	return events, err
}

func (r *eventRepository) CountActiveEvents() (int64, error) {
	var count int64
	now := time.Now()
	eventsTable := domain.GetTableName("events")
	err := r.db.Model(&domain.Event{}).
		Where(eventsTable+".deleted = ? AND "+eventsTable+".status = ? AND "+eventsTable+".end_date >= ?",
			false, domain.EventStatusPublished, now).
		Count(&count).Error
	return count, err
}

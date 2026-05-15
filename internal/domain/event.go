package domain

import (
	"time"
)

// EventStatus defines the state of an event
type EventStatus string

const (
	EventStatusDraft     EventStatus = "DRAFT"
	EventStatusPublished EventStatus = "PUBLISHED"
	EventStatusCancelled EventStatus = "CANCELLED"
)

// EventAudience defines the target audience filter for an event
type EventAudience string

const (
	EventAudienceAllCompany EventAudience = "ALL_COMPANY"
	EventAudienceDepartment EventAudience = "DEPARTMENT"
	EventAudienceLocation   EventAudience = "LOCATION"
)

// Event represents a company event created by the Admin
type Event struct {
	AuditableModel
	Name                 string        `json:"name" gorm:"size:255;not null"`
	Type                 string        `json:"type" gorm:"size:100;not null"`
	Description          string        `json:"description" gorm:"type:text"`
	StartDate            time.Time     `json:"start_date" gorm:"not null"`
	EndDate              time.Time     `json:"end_date" gorm:"not null"`
	Location             string        `json:"location" gorm:"size:500"`
	AudienceFilter       EventAudience `json:"audience_filter" gorm:"size:50;default:'ALL_COMPANY'"`
	Quota                int           `json:"quota" gorm:"default:0"` // 0 means unlimited
	AllowCompanion       bool          `json:"allow_companion" gorm:"default:false"`
	MaxCompanion         int           `json:"max_companion" gorm:"default:0"`
	LastChangeDate       *time.Time    `json:"last_change_date"`
	ResendTemplateId     string        `json:"resend_template_id" gorm:"size:100"`
	Status               EventStatus   `json:"status" gorm:"size:50;default:'DRAFT'"`
	
	// Relationships
	Participants []EventParticipant `json:"participants,omitempty"`
}

// TableName overrides the default table name
func (Event) TableName() string {
	return GetTableName("events")
}

// ParticipantStatus defines the user's participation state
type ParticipantStatus string

const (
	ParticipantStatusPending      ParticipantStatus = "PENDING"
	ParticipantStatusAttending    ParticipantStatus = "ATTENDING"
	ParticipantStatusNotAttending ParticipantStatus = "NOT_ATTENDING"
)

// EventParticipant represents a user's participation record for an event
type EventParticipant struct {
	AuditableModel
	EventID        uint              `json:"event_id" gorm:"not null;index"`
	UserID         uint              `json:"user_id" gorm:"not null;index"`
	Status         ParticipantStatus `json:"status" gorm:"size:50;default:'PENDING'"`
	CompanionCount int               `json:"companion_count" gorm:"default:0"`
	
	// Navigation Properties
	Event *Event `json:"event,omitempty" gorm:"foreignKey:EventID"`
	User  *User  `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName overrides the default table name
func (EventParticipant) TableName() string {
	return GetTableName("event_participants")
}

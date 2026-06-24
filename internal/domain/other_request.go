package domain

import "time"

const (
	RequestStatusActive    = "ACTIVE"
	RequestStatusCompleted = "COMPLETED"
	RequestStatusCancelled = "CANCELLED"
)

type RequestType struct {
	AuditableModel
	Name        string `json:"name" gorm:"not null;uniqueIndex"`
	Description string `json:"description" gorm:"type:text"`
	Active      bool   `json:"active" gorm:"default:true"`

	OtherRequests []OtherRequest `json:"other_requests,omitempty" gorm:"foreignKey:RequestTypeID"`
}

func (RequestType) TableName() string {
	return GetTableName("hr_request_types")
}

type OtherRequest struct {
	AuditableModel
	EmployeeID    uint   `json:"employee_id" gorm:"not null;index"`
	RequestTypeID uint   `json:"request_type_id" gorm:"not null;index"`
	Description   string `json:"description" gorm:"type:text;not null"`
	Status        string `json:"status" gorm:"size:20;not null;default:'ACTIVE';index"`

	// İK talebi tamamladığında doldurulacak alanlar
	CompletedAt *time.Time `json:"completed_at"`
	CompletedBy *uint      `json:"completed_by"`

	Employee    *Employee    `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
	RequestType *RequestType `json:"request_type,omitempty" gorm:"foreignKey:RequestTypeID"`
	Completer   *User        `json:"completer,omitempty" gorm:"foreignKey:CompletedBy"`

	Attachments []Attachment `json:"attachments,omitempty" gorm:"foreignKey:RelatedID;constraint:OnDelete:CASCADE;"`

	DocumentCount int `json:"document_count" gorm:"-"`
}

func (OtherRequest) TableName() string {
	return GetTableName("hr_other_requests")
}
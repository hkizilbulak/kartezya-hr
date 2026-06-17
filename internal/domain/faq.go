package domain

type FAQStatus string

const (
	FAQStatusActive   FAQStatus = "ACTIVE"
	FAQStatusInactive FAQStatus = "INACTIVE"
)

type FAQ struct {
	AuditableModel 
	Title       string    `json:"title" gorm:"type:varchar(255);not null"`
	Description string    `json:"description" gorm:"type:text;not null"`
	Status      FAQStatus `json:"status" gorm:"size:50;default:'ACTIVE'"`
}

func (FAQ) TableName() string {
	return "HR_FAQ"
}
package domain

import "time"

// TrainingStatus eğitimin yayın durumunu belirtir
type TrainingStatus string

const (
	TrainingStatusActive   TrainingStatus = "ACTIVE"
	TrainingStatusInactive TrainingStatus = "INACTIVE"
)

// AssignmentStatus çalışanın eğitimdeki ilerleme durumunu belirtir
type AssignmentStatus string

const (
	AssignmentStatusAssigned   AssignmentStatus = "ASSIGNED"
	AssignmentStatusInProgress AssignmentStatus = "IN_PROGRESS"
	AssignmentStatusCompleted  AssignmentStatus = "COMPLETED"
)

// Training admin tarafından tanımlanan eğitim kaydıdır.
type Training struct {
	AuditableModel
	Title       string         `json:"title" gorm:"type:varchar(255);not null"`
	Description string         `json:"description" gorm:"type:text"`
	Duration    int            `json:"duration" gorm:"default:0"` // dakika cinsinden tahmini süre
	Status      TrainingStatus `json:"status" gorm:"type:varchar(20);default:'ACTIVE'"`
	// İlişkili dosyalar Attachment tablosu üzerinden tutulur (AttachmentRelatedTypeAcademy).
}

func (Training) TableName() string {
	return GetTableName("trainings")
}

// TrainingAssignment çalışan–eğitim atamasını temsil eder.
type TrainingAssignment struct {
	AuditableModel
	TrainingID  uint             `json:"training_id" gorm:"not null;index"`
	EmployeeID  uint             `json:"employee_id" gorm:"not null;index"`
	Status      AssignmentStatus `json:"status" gorm:"size:20;not null;default:'ASSIGNED'"`
	StartedAt   *time.Time       `json:"started_at"`
	CompletedAt *time.Time       `json:"completed_at"`

	// Relationships
	Training Training `json:"training,omitempty" gorm:"foreignKey:TrainingID"`
	Employee Employee `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
}

func (TrainingAssignment) TableName() string {
	return GetTableName("training_assignments")
}

// TrainingCertificate tamamlanan eğitim için üretilen sertifika kaydıdır.
type TrainingCertificate struct {
	AuditableModel
	AssignmentID    uint      `json:"assignment_id" gorm:"not null;uniqueIndex"`
	EmployeeID      uint      `json:"employee_id" gorm:"not null;index"`
	TrainingID      uint      `json:"training_id" gorm:"not null;index"`
	CertificateCode string    `json:"certificate_code" gorm:"type:varchar(64);not null;uniqueIndex"`
	IssuedAt        time.Time `json:"issued_at" gorm:"not null"`

	// Relationships
	Assignment TrainingAssignment `json:"assignment,omitempty" gorm:"foreignKey:AssignmentID"`
	Employee   Employee           `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
	Training   Training           `json:"training,omitempty" gorm:"foreignKey:TrainingID"`
}

func (TrainingCertificate) TableName() string {
	return GetTableName("training_certificates")
}

// ─────────────────────────────────────────────────────────────────────────────
// Academy Survey Models
// ─────────────────────────────────────────────────────────────────────────────

// AcademySurvey admin tarafından oluşturulan akademideki ankettir.
type AcademySurvey struct {
	AuditableModel
	Title         string                `json:"title" gorm:"type:varchar(255);not null"`
	Description   string                `json:"description" gorm:"type:text"`
	IsMultiSelect bool                  `json:"is_multi_select" gorm:"default:false"`
	IsActive      bool                  `json:"is_active" gorm:"default:true"`
	Options       []AcademySurveyOption `json:"options,omitempty" gorm:"foreignKey:SurveyID;constraint:OnDelete:CASCADE"`
}

func (AcademySurvey) TableName() string {
	return GetTableName("academy_surveys")
}

// AcademySurveyOption anketin şıklarını/maddelerini belirtir.
type AcademySurveyOption struct {
	AuditableModel
	SurveyID uint   `json:"survey_id" gorm:"not null;index"`
	Text     string `json:"text" gorm:"type:varchar(255);not null"`
}

func (AcademySurveyOption) TableName() string {
	return GetTableName("academy_survey_options")
}

// AcademySurveyResponse çalışanların anketlere verdikleri oyları (seçimleri) tutar.
type AcademySurveyResponse struct {
	AuditableModel
	SurveyID   uint `json:"survey_id" gorm:"not null;index"`
	EmployeeID uint `json:"employee_id" gorm:"not null;index"`
	OptionID   uint `json:"option_id" gorm:"not null;index"`
}

func (AcademySurveyResponse) TableName() string {
	return GetTableName("academy_survey_responses")
}

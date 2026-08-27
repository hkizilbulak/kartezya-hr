package domain

import (
	"fmt"
	"time"

	"kartezya-hr/internal/config"

	"gorm.io/gorm"
)

// Global config variable for table naming
var globalConfig *config.Config

// SetConfig sets the global config for table naming
func SetConfig(cfg *config.Config) {
	globalConfig = cfg
}

// Ensure GetTableName is exported and accessible
func GetTableName(tableName string) string {
	if globalConfig != nil {
		return globalConfig.GetTableName(tableName)
	}
	return tableName
}

// AuditableModel with audit and soft-delete fields for all entities except AuditLog
type AuditableModel struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Deleted    bool      `json:"deleted" gorm:"not null;default:false"`
	CreatedBy  string    `json:"created_by" gorm:"size:50"`
	ModifiedBy string    `json:"modified_by" gorm:"size:50"`
}

// User represents the authentication entity
type User struct {
	AuditableModel
	Email                string     `json:"email" gorm:"uniqueIndex;not null"`
	Password             string     `json:"-" gorm:"not null"` // Hide password in JSON responses
	PasswordResetToken   string     `json:"-" gorm:"size:255"` // Token for password reset
	PasswordResetExpires *time.Time `json:"-"`                 // Expiration time for reset token

	// Relationships
	UserRoles   []UserRole   `json:"user_roles,omitempty"`
	Employee    *Employee    `json:"employee,omitempty" gorm:"foreignKey:UserID"`
	UserSetting *UserSetting `json:"user_setting,omitempty" gorm:"foreignKey:UserID"`
}

// Role represents system roles
type Role struct {
	AuditableModel
	Name        string `json:"name" gorm:"uniqueIndex;not null"`
	Description string `json:"description"`

	// Relationships
	UserRoles []UserRole `json:"user_roles,omitempty"`
}

// UserRole is the junction table for users and roles
type UserRole struct {
	AuditableModel
	UserID uint `json:"user_id" gorm:"not null"`
	RoleID uint `json:"role_id" gorm:"not null"`

	// Relationships
	User User `json:"user,omitempty"`
	Role Role `json:"role,omitempty"`
}

// Employee represents employee information
type Employee struct {
	AuditableModel
	UserID                   uint       `json:"user_id" gorm:"uniqueIndex;not null"`
	FirstName                string     `json:"first_name" gorm:"not null"`
	LastName                 string     `json:"last_name" gorm:"not null"`
	Email                    string     `json:"email" gorm:"size:255"`
	CompanyEmail             string     `json:"company_email" gorm:"size:255"`
	Phone                    string     `json:"phone"`
	Address                  string     `json:"address"`
	State                    string     `json:"state" gorm:"size:100"`
	City                     string     `json:"city" gorm:"size:30"`
	Gender                   string     `json:"gender" gorm:"size:20"`
	DateOfBirth              *time.Time `json:"date_of_birth"`
	HireDate                 *time.Time `json:"hire_date"`
	LeaveDate                *time.Time `json:"leave_date"`
	MaritalStatus            string     `json:"marital_status" gorm:"size:20"`
	EmergencyContact         string     `json:"emergency_contact" gorm:"size:15"`
	EmergencyContactName     string     `json:"emergency_contact_name" gorm:"size:20"`
	EmergencyContactRelation string     `json:"emergency_contact_relation" gorm:"size:20"`
	GradeID                  *int64     `json:"grade_id" gorm:"index"` // legacy DB column; API reads ACTIVE EmployeeGrade instead
	ProfessionStartDate      *time.Time `json:"profession_start_date"`
	Note                     string     `json:"note" gorm:"type:text"`
	FatherName               string     `json:"father_name" gorm:"size:255"`
	Nationality              string     `json:"nationality" gorm:"size:100"`
	IdentityNo               string     `json:"identity_no" gorm:"size:50"`
	Status                   string     `json:"status" gorm:"size:10;not null;default:'ACTIVE'"`

	// Relationships
	User  User   `json:"user,omitempty"`
	Grade *Grade `json:"-" gorm:"foreignKey:GradeID"` // legacy FK; unused by API (kept for AutoMigrate/scan)
	// CurrentEmployeeGrade is has-one onto hr_employee_grades (FK: employee_id), not a column on employees.
	CurrentEmployeeGrade    *EmployeeGrade            `json:"-" gorm:"foreignKey:EmployeeID"`
	EmployeeWorkInformation []EmployeeWorkInformation `json:"employee_work_information,omitempty"`
	LeaveBalances           []LeaveBalance            `json:"leave_balances,omitempty"`
	LeaveRequests           []LeaveRequest            `json:"leave_requests,omitempty"`
}

// Company represents company information
type Company struct {
	AuditableModel
	Name    string `json:"name" gorm:"not null"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Website string `json:"website"`

	// Relationships
	Departments []Department `json:"departments,omitempty"`
}

// Department represents organizational departments
type Department struct {
	AuditableModel
	CompanyID uint   `json:"company_id" gorm:"not null"`
	Name      string `json:"name" gorm:"not null"`
	Manager   string `json:"manager"`

	// Relationships
	Company                 Company                   `json:"company,omitempty"`
	EmployeeWorkInformation []EmployeeWorkInformation `json:"employee_work_information,omitempty"`
}

// JobPosition represents job positions
type JobPosition struct {
	AuditableModel
	Title string `json:"title" gorm:"not null"`

	// Relationships
	EmployeeWorkInformation []EmployeeWorkInformation `json:"employee_work_information,omitempty"`
}

// EmployeeWorkInformation represents employee work history and current position
type EmployeeWorkInformation struct {
	AuditableModel
	EmployeeID    uint       `json:"employee_id" gorm:"not null"`
	CompanyID     uint       `json:"company_id" gorm:"not null"`
	DepartmentID  uint       `json:"department_id" gorm:"not null"`
	JobPositionID uint       `json:"job_position_id" gorm:"not null"`
	StartDate     time.Time  `json:"start_date" gorm:"not null"`
	EndDate       *time.Time `json:"end_date"`
	PersonnelNo   string     `json:"personnel_no" gorm:"size:100"`
	WorkEmail     string     `json:"work_email" gorm:"size:255"`

	// Relationships
	Employee    Employee    `json:"employee,omitempty"`
	Company     Company     `json:"company,omitempty"`
	Department  Department  `json:"department,omitempty"`
	JobPosition JobPosition `json:"job_position,omitempty"`
}

// LeaveType represents different types of leave (Annual, Sick, etc.)
type LeaveType struct {
	AuditableModel
	Name               string `json:"name" gorm:"not null"`
	Description        string `json:"description"`
	IsPaid             bool   `json:"is_paid" gorm:"not null"`
	LimitAmount        *int   `json:"limit_amount" gorm:"default:null"`
	IsAccrual          bool   `json:"is_accrual" gorm:"not null"`
	IsRequiredDocument bool   `json:"is_required_document" gorm:"not null"`

	// Relationships
	LeaveBalances []LeaveBalance `json:"leave_balances,omitempty"`
	LeaveRequests []LeaveRequest `json:"leave_requests,omitempty"`
}

// LeaveBalance represents employee's leave balance for each leave type
type LeaveBalance struct {
	AuditableModel
	EmployeeID    uint    `json:"employee_id" gorm:"not null;uniqueIndex:idx_employee_leave_type"`
	LeaveTypeID   uint    `json:"leave_type_id" gorm:"not null;uniqueIndex:idx_employee_leave_type"`
	TotalDays     float64 `json:"total_days" gorm:"not null"`
	UsedDays      float64 `json:"used_days" gorm:"default:0"`
	RemainingDays float64 `json:"remaining_days" gorm:"not null"`

	// Relationships
	Employee  Employee  `json:"employee,omitempty"`
	LeaveType LeaveType `json:"leave_type,omitempty"`
}

// LeaveRequest represents employee leave requests
type LeaveRequest struct {
	AuditableModel
	EmployeeID          uint       `json:"employee_id" gorm:"not null"`
	LeaveTypeID         uint       `json:"leave_type_id" gorm:"not null"`
	StartDate           time.Time  `json:"start_date" gorm:"not null"`
	EndDate             time.Time  `json:"end_date" gorm:"not null"`
	IsStartDateFullDay  bool       `json:"is_start_date_full_day" gorm:"default:true"`  // true = tam gün, false = yarım gün
	IsFinishDateFullDay bool       `json:"is_finish_date_full_day" gorm:"default:true"` // true = tam gün, false = yarım gün
	RequestedDays       float64    `json:"requested_days" gorm:"not null"`
	Reason              string     `json:"reason"`
	Status              string     `json:"status" gorm:"default:'PENDING'"` // PENDING, APPROVED, REJECTED, CANCELLED
	IsPaid              bool       `json:"is_paid" gorm:"not null"`
	ApprovedBy          *uint      `json:"approved_by"`
	ApprovedAt          *time.Time `json:"approved_at"`
	RejectedAt          *time.Time `json:"rejected_at"`
	RejectionReason     string     `json:"rejection_reason"`
	CancelReason        string     `json:"cancel_reason"`
	CancelledAt         *time.Time `json:"cancelled_at"`
	Comments            string     `json:"comments" gorm:"type:text"`
	// Relationships
	Employee       Employee        `json:"employee,omitempty"`
	LeaveType      LeaveType       `json:"leave_type,omitempty"`
	Approver       *User           `json:"approver,omitempty" gorm:"foreignKey:ApprovedBy"`
	LeaveDocuments []LeaveDocument `json:"leave_documents,omitempty"`
}

// Leave represents a unified leave entity for backward compatibility (maps to leave_requests table)
type Leave struct {
	ID                  uint       `gorm:"primaryKey;column:id" json:"id"`
	EmployeeID          uint       `json:"employee_id" gorm:"not null;column:employee_id"`
	Employee            Employee   `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveTypeID         uint       `json:"leave_type_id" gorm:"not null;column:leave_type_id"`
	LeaveType           LeaveType  `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
	StartDate           time.Time  `json:"start_date" gorm:"not null;column:start_date"`
	EndDate             time.Time  `json:"end_date" gorm:"not null;column:end_date"`
	IsStartDateFullDay  bool       `json:"is_start_date_full_day" gorm:"default:true;column:is_start_date_full_day"`
	IsFinishDateFullDay bool       `json:"is_finish_date_full_day" gorm:"default:true;column:is_finish_date_full_day"`
	Days                float64    `json:"days" gorm:"not null;column:requested_days"`
	Status              string     `json:"status" gorm:"default:'PENDING';column:status"`
	Reason              string     `json:"reason" gorm:"type:text;column:reason"`
	ApproverID          *uint      `json:"approver_id" gorm:"column:approved_by"`
	Approver            *User      `gorm:"foreignKey:ApproverID;references:ID" json:"approver,omitempty"`
	ApprovedAt          *time.Time `json:"approved_at" gorm:"column:approved_at"`
	RejectedAt          *time.Time `json:"rejected_at"`
	Comments            string     `json:"comments" gorm:"type:text;column:rejection_reason"`
	CreatedAt           time.Time  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt           time.Time  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
	CreatedBy           string     `json:"created_by" gorm:"not null;column:created_by"`
	ModifiedBy          string     `json:"modified_by" gorm:"not null;column:modified_by"`
	Deleted             bool       `json:"-" gorm:"default:false;column:deleted"`
}

// TableName maps the Leave model to the hr_leave_requests table
func (Leave) TableName() string {
	return GetTableName("hr_leave_requests")
}

// LeaveDocument represents documents attached to leave requests
type LeaveDocument struct {
	AuditableModel
	LeaveRequestID uint   `json:"leave_request_id" gorm:"not null"`
	FileName       string `json:"file_name" gorm:"not null"`
	FilePath       string `json:"file_path" gorm:"not null"`
	FileSize       int64  `json:"file_size"`
	MimeType       string `json:"mime_type"`

	// Relationships
	LeaveRequest LeaveRequest `json:"leave_request,omitempty"`
}

// Holiday represents public holidays
type Holiday struct {
	AuditableModel
	HolidayDate time.Time `json:"holiday_date" gorm:"not null"`
	Name        string    `json:"name" gorm:"not null"`
	IsFullDay   bool      `json:"is_full_day" gorm:"not null;default:true"`
}

func (Holiday) TableName() string {
	return GetTableName("hr_holidays")
}

// Grade represents employee grades/levels
type Grade struct {
	AuditableModel
	Name        string `json:"name" gorm:"not null"`
	Description string `json:"description"`
	MinYear     *int   `json:"min_year"`
	MaxYear     *int   `json:"max_year"`

	// Relationships
	Employees      []Employee      `json:"employees,omitempty"`
	EmployeeGrades []EmployeeGrade `json:"employee_grades,omitempty"`
}

// EmployeeGradeStatus is the lifecycle status of an employee grade history row.
// Distinct from Employee.Status (employment ACTIVE/PASSIVE).
type EmployeeGradeStatus string

const (
	EmployeeGradeStatusActive   EmployeeGradeStatus = "ACTIVE"
	EmployeeGradeStatusInactive EmployeeGradeStatus = "INACTIVE"
)

// EmployeeGradeStatusFromEndDate derives status from end_date (ACTIVE ⇔ end_date IS NULL).
func EmployeeGradeStatusFromEndDate(endDate *time.Time) EmployeeGradeStatus {
	if endDate == nil {
		return EmployeeGradeStatusActive
	}
	return EmployeeGradeStatusInactive
}

// EmployeeGrade represents employee grade history.
// Invariants (also enforced in DB after migrate_employee_grade_status):
//   - ACTIVE  ⇒ end_date IS NULL and deleted = false
//   - INACTIVE ⇒ end_date IS NOT NULL
//   - at most one ACTIVE row per employee (deleted = false)
//
// employees.grade_id is intentionally left in place until a later phase.
type EmployeeGrade struct {
	AuditableModel
	EmployeeID uint                `json:"employee_id" gorm:"not null"`
	GradeID    uint                `json:"grade_id" gorm:"not null"`
	StartDate  time.Time           `json:"start_date" gorm:"not null"`
	EndDate    *time.Time          `json:"end_date"`
	Status     EmployeeGradeStatus `json:"status" gorm:"size:20;not null;default:'ACTIVE'"`

	// Relationships
	Employee Employee `json:"employee,omitempty"`
	Grade    Grade    `json:"grade,omitempty"`
}

// EmployeeContract represents employee contracts
type EmployeeContract struct {
	AuditableModel
	EmployeeID uint `json:"employee_id" gorm:"not null;index:idx_employee_contract"`
	ContractID uint `json:"contract_id" gorm:"index:idx_employee_contract"`

	// Relationships
	Employee Employee `json:"employee,omitempty"`
	Contract Contract `json:"contract,omitempty"`
}

// EmployeeContract table name
func (EmployeeContract) TableName() string {
	return GetTableName("hr_employee_contracts")
}

// Grade table name
func (Grade) TableName() string {
	return GetTableName("hr_grades")
}

// EmployeeGrade table name
func (EmployeeGrade) TableName() string {
	return GetTableName("hr_employee_grades")
}

// AuditLog represents system audit logs (NO SOFT DELETE - this table is append-only)
type AuditLog struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	EntityName  string    `json:"entity_name" gorm:"not null"`
	EntityID    uint      `json:"entity_id" gorm:"not null"`
	Action      string    `json:"action" gorm:"not null"` // CREATE, UPDATE, DELETE
	OldValue    string    `json:"old_value" gorm:"type:jsonb"`
	NewValue    string    `json:"new_value" gorm:"type:jsonb"`
	CreatedDate time.Time `json:"created_date" gorm:"not null"`
	CreatedBy   uint      `json:"created_by" gorm:"not null"`

	// Relationships
	Creator User `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// Constants for roles
const (
	RoleAdmin      = "ADMIN"
	RoleEmployee   = "EMPLOYEE"
	RoleHR         = "HR"
	RoleFinancial  = "FINANCIAL"
	RoleTeamLeader = "TEAM_LEADER"
)

// Constants for leave request status
const (
	LeaveStatusPending   = "PENDING"
	LeaveStatusApproved  = "APPROVED"
	LeaveStatusRejected  = "REJECTED"
	LeaveStatusCancelled = "CANCELLED"
)

// Constants for audit actions
const (
	AuditActionCreate = "CREATE"
	AuditActionUpdate = "UPDATE"
	AuditActionDelete = "DELETE"
)

// Contract status constants
const (
	ContractStatusPendingProposal  = "PENDING_PROPOSAL"  // teklif onay bekleniyor
	ContractStatusProposalSent     = "PROPOSAL_SENT"     // teklif iletildi
	ContractStatusProposalRevision = "PROPOSAL_REVISION" // teklif revize bekleniyor
	ContractStatusPendingRevision  = "PENDING_REVISION"  // revize bekleniyor
	ContractStatusPendingApproval  = "PENDING_APPROVAL"  // onay bekleniyor
	ContractStatusApproved         = "APPROVED"          // onaylandı
)

// Contract represents an agreement/project with a customer
type Contract struct {
	AuditableModel
	CustomerContactName  string     `json:"customer_contact_name" gorm:"size:255;not null"`
	CustomerContactPhone string     `json:"customer_contact_phone" gorm:"size:50"`
	CustomerContactEmail string     `json:"customer_contact_email" gorm:"size:255"`
	ProjectName          string     `json:"project_name" gorm:"size:255;not null"`
	ContractNo           string     `json:"contract_no" gorm:"size:100;uniqueIndex;not null"`
	StartDate            time.Time  `json:"start_date" gorm:"not null"`
	EndDate              *time.Time `json:"end_date"`
	Status               string     `json:"status" gorm:"size:50;not null;default:'PENDING_PROPOSAL'"`

	// Relationships
	EmployeeContracts []EmployeeContract `json:"employee_contracts,omitempty"`
}

// Contract table name
func (Contract) TableName() string {
	return GetTableName("hr_contracts")
}

// TableName methods to add hr_ prefix to all tables

// User table name
func (User) TableName() string {
	return GetTableName("hr_users")
}

// Role table name
func (Role) TableName() string {
	return GetTableName("hr_roles")
}

// UserRole table name
func (UserRole) TableName() string {
	return GetTableName("hr_user_roles")
}

// Employee table name
func (Employee) TableName() string {
	return GetTableName("hr_employees")
}

// Company table name
func (Company) TableName() string {
	return GetTableName("hr_companies")
}

// Department table name
func (Department) TableName() string {
	return GetTableName("hr_departments")
}

// JobPosition table name
func (JobPosition) TableName() string {
	return GetTableName("hr_job_positions")
}

// EmployeeWorkInformation table name
func (EmployeeWorkInformation) TableName() string {
	return GetTableName("hr_employee_work_information")
}

// LeaveType table name
func (LeaveType) TableName() string {
	return GetTableName("hr_leave_types")
}

// LeaveBalance table name
func (LeaveBalance) TableName() string {
	return GetTableName("hr_leave_balances")
}

// LeaveRequest table name
func (LeaveRequest) TableName() string {
	return GetTableName("hr_leave_requests")
}

// LeaveDocument table name
func (LeaveDocument) TableName() string {
	return GetTableName("hr_leave_documents")
}

// AuditLog table name
func (AuditLog) TableName() string {
	return GetTableName("hr_audit_logs")
}

// ==================== Document Management System (DYS) ====================

// Attachment Related Type Enum - Defines which module the attachment belongs to
type AttachmentRelatedType int

const (
	AttachmentRelatedTypeExpense      AttachmentRelatedType = 1 // Masraf
	AttachmentRelatedTypeLeave        AttachmentRelatedType = 2 // İzin
	AttachmentRelatedTypeUser         AttachmentRelatedType = 3 // Kullanıcı/Profil
	AttachmentRelatedTypeEmployee     AttachmentRelatedType = 4 // Çalışan Özlük Dosyası
	AttachmentRelatedTypeContract     AttachmentRelatedType = 5 // Sözleşme
	AttachmentRelatedTypeOtherRequest AttachmentRelatedType = 6 // Diğer Talepler
	AttachmentRelatedTypeAcademy      AttachmentRelatedType = 7 // Kartezya Akademi Eğitimleri
)

// AttachmentType Enum - Defines document category
type AttachmentType int

const (
	AttachmentTypeInvoice       AttachmentType = 1  // Fatura
	AttachmentTypeMedicalReport AttachmentType = 2  // Sağlık Raporu
	AttachmentTypeAvatar        AttachmentType = 3  // Profil Resmi
	AttachmentTypeReceipt       AttachmentType = 4  // Makbuz
	AttachmentTypeContract      AttachmentType = 5  // Sözleşme
	AttachmentTypeIdentity      AttachmentType = 6  // Kimlik
	AttachmentTypeDiploma       AttachmentType = 7  // Diploma
	AttachmentTypeCertificate   AttachmentType = 8  // Sertifika
	AttachmentTypeResume        AttachmentType = 9  // CV / Özgeçmiş
	AttachmentTypeDocument      AttachmentType = 99 // Döküman
	AttachmentTypeOther         AttachmentType = 99 // Diğer
)

// AttachmentStatus Enum - Defines attachment lifecycle status
type AttachmentStatus int

const (
	AttachmentStatusTemporary AttachmentStatus = 1 // Geçici
	AttachmentStatusLinked    AttachmentStatus = 2 // İlişkilendirilmiş
	AttachmentStatusArchived  AttachmentStatus = 3 // Arşivlenmiş
)

// Attachment represents a generic document/file in the system
// This is the Single Source of Truth for all file uploads across all modules
type Attachment struct {
	ID          string                `json:"id" gorm:"primaryKey;type:varchar(36)"`          // UUID as string
	OwnerID     uint                  `json:"owner_id" gorm:"not null;index"`                 // User who uploaded the file
	RelatedType AttachmentRelatedType `json:"related_type" gorm:"not null;index"`             // Which module (Expense, Leave, User, etc.)
	RelatedID   *uint                 `json:"related_id" gorm:"index"`                        // Related record ID (nullable until linked)
	Type        AttachmentType        `json:"type" gorm:"not null"`                           // Document category (Invoice, Medical, etc.)
	Status      AttachmentStatus      `json:"status" gorm:"not null;index;default:1"`         // Lifecycle status (Temporary, Linked, Archived)
	FileName    string                `json:"file_name" gorm:"type:varchar(255);not null"`    // Original filename
	Path        string                `json:"path" gorm:"type:varchar(500);not null"`         // Storage path (e.g., expense/2026/04/uuid_name.pdf)
	ContentType string                `json:"content_type" gorm:"type:varchar(100);not null"` // MIME type
	FileSize    int64                 `json:"file_size" gorm:"not null"`                      // File size in bytes
	Hash        string                `json:"hash" gorm:"type:varchar(64);index"`             // SHA256 hash for duplicate detection
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`

	// Relationships
	Owner User `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
}

// Attachment table name
func (Attachment) TableName() string {
	return GetTableName("hr_attachments")
}

// BeforeCreate hook to generate UUID
func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = GenerateUUID()
	}
	return nil
}

// GenerateUUID generates a new UUID string
func GenerateUUID() string {
	// Simple UUID generation without external dependency
	// Using timestamp + random for uniqueness
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

// ExpenseRequest represents an expense claim/reimbursement request
type ExpenseRequest struct {
	AuditableModel
	EmployeeID       uint       `json:"employee_id" gorm:"not null;index"`
	ExpenseTypeID    uint       `json:"expense_type_id" gorm:"not null;index"`
	Amount           float64    `json:"amount" gorm:"not null"`                      // Expense amount
	Currency         string     `json:"currency" gorm:"size:3;default:TRY"`          // Currency code (TRY, USD, EUR)
	ExpenseDate      time.Time  `json:"expense_date" gorm:"not null"`                // Date of expense
	Description      string     `json:"description" gorm:"type:text"`                // Expense description/notes
	Status           string     `json:"status" gorm:"size:20;default:PENDING;index"` // PENDING, APPROVED, REJECTED, PAID
	ApprovedBy       *uint      `json:"approved_by"`                                 // Admin who approved
	ApprovedAt       *time.Time `json:"approved_at"`                                 // Approval timestamp
	RejectedAt       *time.Time `json:"rejected_at"`                                 // Rejection timestamp
	RejectionReason  string     `json:"rejection_reason" gorm:"type:text"`           // Reason for rejection
	PaidAt           *time.Time `json:"paid_at"`                                     // Payment timestamp
	PaymentReference string     `json:"payment_reference" gorm:"size:255"`           // Payment reference/transaction ID

	// Relationships
	Employee    *Employee    `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
	ExpenseType *ExpenseType `json:"expense_type,omitempty" gorm:"foreignKey:ExpenseTypeID"`
	Approver    *User        `json:"approver,omitempty" gorm:"foreignKey:ApprovedBy"`

	// Computed fields (not stored in DB)
	DocumentCount int `json:"document_count" gorm:"-"` // Number of attached documents
}

func (ExpenseRequest) TableName() string {
	return GetTableName("hr_expense_requests")
}

// ExpenseType represents predefined expense categories
type ExpenseType struct {
	AuditableModel
	Name            string   `json:"name" gorm:"not null;uniqueIndex"`
	Description     string   `json:"description" gorm:"type:text"`
	RequiresReceipt bool     `json:"requires_receipt" gorm:"default:true"` // Whether receipt/invoice is mandatory
	MaxAmount       *float64 `json:"max_amount"`                           // Maximum allowed amount (null = no limit)
	Active          bool     `json:"active" gorm:"default:true"`           // Active/Inactive status
	RoleID          *uint    `json:"role_id"`                              // Role required to see/use this expense type (null = all can see)

	// Relationships
	Role *Role `json:"role,omitempty" gorm:"foreignKey:RoleID"`
}

func (ExpenseType) TableName() string {
	return GetTableName("hr_expense_types")
}

// ==================== Job Scheduler ====================

// Job represents a scheduled background task
type Job struct {
	AuditableModel
	JobKey         string `json:"job_key" gorm:"size:100;uniqueIndex;not null"`
	Name           string `json:"name" gorm:"size:255;not null"`
	CronExpression string `json:"cron_expression" gorm:"size:100;not null"`
	IsActive       bool   `json:"is_active" gorm:"default:true"`
	TimeoutSecond  int    `json:"timeout_second" gorm:"default:3600"`

	// Relationships
	Histories []JobHistory `json:"histories,omitempty" gorm:"foreignKey:JobID"`
}

func (Job) TableName() string {
	return GetTableName("hr_jobs")
}

// JobHistory logs the execution of a scheduled job
type JobHistory struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	JobID             uint       `json:"job_id" gorm:"not null;index"`
	StartTime         time.Time  `json:"start_time" gorm:"not null;index"`
	EndTime           *time.Time `json:"end_time"`
	ProcessedCount    int        `json:"processed_count" gorm:"default:0"`
	Status            string     `json:"status" gorm:"size:20;not null;index"` // SUCCESS, FAILED, RUNNING, TIMEOUT
	ErrorSummary      string     `json:"error_summary" gorm:"type:text"`
	ExecutionNode     string     `json:"execution_node" gorm:"size:255"`
	ReferenceDate     *time.Time `json:"reference_date" gorm:"type:date;index"`
	ExecutionType     string     `json:"execution_type" gorm:"size:20;default:scheduled"`
	TriggeredByUserID *uint      `json:"triggered_by_user_id"`

	// Relationships
	Job Job `json:"job,omitempty" gorm:"foreignKey:JobID"`
}

func (JobHistory) TableName() string {
	return GetTableName("hr_job_history")
}

// ──────────────────────────────────────────────────────────────────────────────
// Mail Configuration Module
// ──────────────────────────────────────────────────────────────────────────────

type MailProvider string
type MailRecipientType string
type MailValueType string

const (
	MailProviderResend MailProvider = "RESEND"
	MailProviderSMTP   MailProvider = "SMTP"

	RecipientTypeTo  MailRecipientType = "TO"
	RecipientTypeCC  MailRecipientType = "CC"
	RecipientTypeBCC MailRecipientType = "BCC"

	ValueTypeStatic  MailValueType = "STATIC"
	ValueTypeDynamic MailValueType = "DYNAMIC"
)

// MailConfiguration stores runtime email settings for each system event key
type MailConfiguration struct {
	ID                 uint            `json:"id" gorm:"primaryKey"`
	MailKey            string          `json:"mail_key" gorm:"uniqueIndex;size:100;not null"`
	Description        string          `json:"description" gorm:"size:255"`
	Provider           MailProvider    `json:"provider" gorm:"size:20;not null;default:'RESEND'"`
	ResendTemplateCode string          `json:"resend_template_code" gorm:"size:100"`
	IsActive           bool            `json:"is_active" gorm:"not null;default:true"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	Recipients         []MailRecipient `json:"recipients,omitempty" gorm:"foreignKey:MailConfigID;constraint:OnDelete:CASCADE"`
}

func (MailConfiguration) TableName() string {
	return GetTableName("hr_mail_configurations")
}

// MailRecipient stores normalized TO / CC / BCC entries for a configuration
type MailRecipient struct {
	ID             uint              `json:"id" gorm:"primaryKey"`
	MailConfigID   uint              `json:"mail_config_id" gorm:"not null;index"`
	RecipientType  MailRecipientType `json:"recipient_type" gorm:"size:10;not null"`
	ValueType      MailValueType     `json:"value_type" gorm:"size:20;not null"`
	RecipientValue string            `json:"recipient_value" gorm:"size:255;not null"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func (MailRecipient) TableName() string {
	return GetTableName("hr_mail_recipients")
}

// UserSetting represents user specific settings and consent statuses
type UserSetting struct {
	AuditableModel
	UserID uint `json:"user_id" gorm:"uniqueIndex;not null"`

	// Consent States for the 4 Documents
	PhotoConsent      string `json:"photo_consent" gorm:"not null;size:20;default:'PENDING'"`       // "PENDING", "APPROVED", "REJECTED"
	KvkkText          string `json:"kvkk_text" gorm:"not null;size:20;default:'PENDING'"`           // "PENDING", "READ"
	PrivacyPolicy     string `json:"privacy_policy" gorm:"not null;size:20;default:'PENDING'"`      // "PENDING", "READ"
	AntiBriberyPolicy string `json:"anti_bribery_policy" gorm:"not null;size:20;default:'PENDING'"` // "PENDING", "READ"

	// Timestamps
	PhotoConsentAt      *time.Time `json:"photo_consent_at"`
	KvkkTextAt          *time.Time `json:"kvkk_text_at"`
	PrivacyPolicyAt     *time.Time `json:"privacy_policy_at"`
	AntiBriberyPolicyAt *time.Time `json:"anti_bribery_policy_at"`
	KvkkLastPostponedAt *time.Time `json:"kvkk_last_postponed_at"`

	// Deprecated (Kept for backward compatibility)
	KvkkStatus     string     `json:"kvkk_status" gorm:"size:20;default:'PENDING'"`
	KvkkApproved   bool       `json:"kvkk_approved" gorm:"not null;default:false"`
	KvkkApprovedAt *time.Time `json:"kvkk_approved_at"`
	KvkkRejectedAt *time.Time `json:"kvkk_rejected_at"`

	PromotionEmailAllowed bool `json:"promotion_email_allowed" gorm:"not null;default:true"`
	PromotionSmsAllowed   bool `json:"promotion_sms_allowed" gorm:"not null;default:true"`
}

// TableName returns the table name for UserSetting
func (UserSetting) TableName() string {
	return GetTableName("hr_user_settings")
}

// KvkkLog represents append-only transaction logs for KVKK approvals/rejections
type KvkkLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"not null;index"`
	DocumentType string    `json:"document_type" gorm:"not null;size:50;default:'ALL'"` // "PHOTO_CONSENT", "KVKK_TEXT", "PRIVACY_POLICY", "ANTI_BRIBERY_POLICY", "ALL"
	Action       string    `json:"action" gorm:"not null;size:20"`                      // "APPROVED", "REJECTED", "READ", "REMIND_LATER"
	ClientIP     string    `json:"client_ip" gorm:"size:45"`
	UserAgent    string    `json:"user_agent" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null"`
}

// TableName returns the table name for KvkkLog
func (KvkkLog) TableName() string {
	return GetTableName("hr_kvkk_logs")
}

// PortalContract represents the mandatory agreements/contracts
type PortalContract struct {
	AuditableModel
	Title   string `json:"title" gorm:"size:255;not null"`
	Content string `json:"content" gorm:"type:text;not null"`
	Version string `json:"version" gorm:"size:50;not null"`
}

// TableName returns the table name for PortalContract
func (PortalContract) TableName() string {
	return GetTableName("hr_portal_contracts")
}

// EmployeePortalContract represents the junction/pivot table tracking employee signatures/approvals
type EmployeePortalContract struct {
	AuditableModel
	EmployeeID uint       `json:"employee_id" gorm:"not null;uniqueIndex:idx_emp_portal_contract"`
	ContractID uint       `json:"contract_id" gorm:"not null;uniqueIndex:idx_emp_portal_contract"`
	Status     string     `json:"status" gorm:"size:20;not null;default:'pending'"` // 'approved', 'pending', 'rejected'
	ApprovedAt *time.Time `json:"approved_at"`
	IPAddress  string     `json:"ip_address" gorm:"size:45"`

	// Relationships
	Employee Employee       `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
	Contract PortalContract `json:"contract,omitempty" gorm:"foreignKey:ContractID"`
}

// TableName returns the table name for EmployeePortalContract
func (EmployeePortalContract) TableName() string {
	return GetTableName("hr_employee_portal_contracts")
}

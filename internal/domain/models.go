package domain

import (
	"time"
)

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
	Email    string `json:"email" gorm:"uniqueIndex;not null"`
	Password string `json:"-" gorm:"not null"` // Hide password in JSON responses

	// Relationships
	UserRoles []UserRole `json:"user_roles,omitempty"`
	Employee  *Employee  `json:"employee,omitempty"`
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
	Gender                   string     `json:"gender" gorm:"type:gender_enum"`
	DateOfBirth              *time.Time `json:"date_of_birth"`
	HireDate                 *time.Time `json:"hire_date"`
	LeaveDate                *time.Time `json:"leave_date"`
	TotalExperience          float64    `json:"total_experience"`
	MaritalStatus            string     `json:"marital_status" gorm:"type:marital_status_enum"`
	EmergencyContact         string     `json:"emergency_contact" gorm:"size:15"`
	EmergencyContactName     string     `json:"emergency_contact_name" gorm:"size:20"`
	EmergencyContactRelation string     `json:"emergency_contact_relation" gorm:"type:emergency_contact_relation_enum"`

	// Relationships
	User                    User                      `json:"user,omitempty"`
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
	IsLimited          bool   `json:"is_limited" gorm:"not null"`
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
	EmployeeID      uint       `json:"employee_id" gorm:"not null"`
	LeaveTypeID     uint       `json:"leave_type_id" gorm:"not null"`
	StartDate       time.Time  `json:"start_date" gorm:"not null"`
	EndDate         time.Time  `json:"end_date" gorm:"not null"`
	RequestedDays   float64    `json:"requested_days" gorm:"not null"`
	Reason          string     `json:"reason"`
	Status          string     `json:"status" gorm:"default:'PENDING'"` // PENDING, APPROVED, REJECTED, CANCELLED
	IsPaid          bool       `json:"is_paid" gorm:"not null"`
	ApprovedBy      *uint      `json:"approved_by"`
	ApprovedAt      *time.Time `json:"approved_at"`
	RejectedAt      *time.Time `json:"rejected_at"`
	RejectionReason string     `json:"rejection_reason"`
	CancelReason    string     `json:"cancel_reason"`
	CancelledAt     *time.Time `json:"cancelled_at"`
	Comments        string     `json:"comments" gorm:"type:text"`
	// Relationships
	Employee       Employee        `json:"employee,omitempty"`
	LeaveType      LeaveType       `json:"leave_type,omitempty"`
	Approver       *User           `json:"approver,omitempty" gorm:"foreignKey:ApprovedBy"`
	LeaveDocuments []LeaveDocument `json:"leave_documents,omitempty"`
}

// Leave represents a unified leave entity for backward compatibility (maps to leave_requests table)
type Leave struct {
	ID          uint       `gorm:"primaryKey;column:id" json:"id"`
	EmployeeID  uint       `json:"employee_id" gorm:"not null;column:employee_id"`
	Employee    Employee   `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveTypeID uint       `json:"leave_type_id" gorm:"not null;column:leave_type_id"`
	LeaveType   LeaveType  `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
	StartDate   time.Time  `json:"start_date" gorm:"not null;column:start_date"`
	EndDate     time.Time  `json:"end_date" gorm:"not null;column:end_date"`
	Days        float64    `json:"days" gorm:"not null;column:requested_days"`
	Status      string     `json:"status" gorm:"default:'PENDING';column:status"`
	Reason      string     `json:"reason" gorm:"type:text;column:reason"`
	ApproverID  *uint      `json:"approver_id" gorm:"column:approved_by"`
	Approver    *User      `gorm:"foreignKey:ApproverID;references:ID" json:"approver,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at" gorm:"column:approved_at"`
	RejectedAt  *time.Time `json:"rejected_at"`
	Comments    string     `json:"comments" gorm:"type:text;column:rejection_reason"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
	CreatedBy   string     `json:"created_by" gorm:"not null;column:created_by"`
	ModifiedBy  string     `json:"modified_by" gorm:"not null;column:modified_by"`
	Deleted     bool       `json:"-" gorm:"default:false;column:deleted"`
}

// TableName maps the Leave model to the hr_leave_requests table
func (Leave) TableName() string {
	return "hr_leave_requests"
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
	return "hr_holidays"
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

// Constants for gender
const (
	GenderMale   = "Erkek"
	GenderFemale = "Kadın"
)

// Constants for emergency contact relation
const (
	EmergencyContactMother  = "Anne"
	EmergencyContactFather  = "Baba"
	EmergencyContactSpouse  = "Eş"
	EmergencyContactSibling = "Kardeş"
	EmergencyContactOther   = "Diğer"
)

// Constants for marital status
const (
	MaritalStatusMarried = "Evli"
	MaritalStatusSingle  = "Bekar"
)

// Constants for roles
const (
	RoleAdmin    = "ADMIN"
	RoleEmployee = "EMPLOYEE"
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

// TableName methods to add hr_ prefix to all tables

// User table name
func (User) TableName() string {
	return "hr_users"
}

// Role table name
func (Role) TableName() string {
	return "hr_roles"
}

// UserRole table name
func (UserRole) TableName() string {
	return "hr_user_roles"
}

// Employee table name
func (Employee) TableName() string {
	return "hr_employees"
}

// Company table name
func (Company) TableName() string {
	return "hr_companies"
}

// Department table name
func (Department) TableName() string {
	return "hr_departments"
}

// JobPosition table name
func (JobPosition) TableName() string {
	return "hr_job_positions"
}

// EmployeeWorkInformation table name
func (EmployeeWorkInformation) TableName() string {
	return "hr_employee_work_information"
}

// LeaveType table name
func (LeaveType) TableName() string {
	return "hr_leave_types"
}

// LeaveBalance table name
func (LeaveBalance) TableName() string {
	return "hr_leave_balances"
}

// LeaveRequest table name
func (LeaveRequest) TableName() string {
	return "hr_leave_requests"
}

// LeaveDocument table name
func (LeaveDocument) TableName() string {
	return "hr_leave_documents"
}

// AuditLog table name
func (AuditLog) TableName() string {
	return "hr_audit_logs"
}

package types

import "time"

// Lookup DTOs for basic listing responses
type CompanyLookup struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type LeaveTypeLookup struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// Grade response DTO
type GradeResponse struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Deleted     bool      `json:"deleted"`
	CreatedBy   string    `json:"created_by"`
	ModifiedBy  string    `json:"modified_by"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type GradeLookup struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// Company response DTO without departments relationship
type CompanyResponse struct {
	ID         uint      `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Deleted    bool      `json:"deleted"`
	CreatedBy  string    `json:"created_by"`
	ModifiedBy string    `json:"modified_by"`
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	Phone      string    `json:"phone"`
	Email      string    `json:"email"`
	Website    string    `json:"website"`
}

type DepartmentLookup struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Manager string `json:"manager"`
}

// Department response DTO with nested company object
type DepartmentResponse struct {
	ID         uint          `json:"id"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	Deleted    bool          `json:"deleted"`
	CreatedBy  string        `json:"created_by"`
	ModifiedBy string        `json:"modified_by"`
	Name       string        `json:"name"`
	Manager    string        `json:"manager"`
	Company    CompanyLookup `json:"company"`
}

type JobPositionLookup struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

// User nested object for EmployeeResponse
type UserInfo struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
}

// Employee response DTO with nested user object
type EmployeeResponse struct {
	ID                       uint                    `json:"id"`
	User                     UserInfo                `json:"user"`
	FirstName                string                  `json:"first_name"`
	LastName                 string                  `json:"last_name"`
	Email                    string                  `json:"email"`
	CompanyEmail             string                  `json:"company_email"`
	Phone                    string                  `json:"phone"`
	Address                  string                  `json:"address"`
	State                    string                  `json:"state"`
	City                     string                  `json:"city"`
	Gender                   string                  `json:"gender"`
	DateOfBirth              *string                 `json:"date_of_birth"`
	HireDate                 *string                 `json:"hire_date"`
	LeaveDate                *string                 `json:"leave_date,omitempty"`
	TotalExperience          float64                 `json:"total_experience"`
	MaritalStatus            string                  `json:"marital_status"`
	EmergencyContact         string                  `json:"emergency_contact"`
	EmergencyContactName     string                  `json:"emergency_contact_name"`
	EmergencyContactRelation string                  `json:"emergency_contact_relation"`
	GradeID                  *int64                  `json:"grade_id"`
	IsGradeUp                bool                    `json:"is_grade_up"`
	ContractNo               string                  `json:"contract_no"`
	ProfessionStartDate      *string                 `json:"profession_start_date"`
	Note                     string                  `json:"note"`
	MotherName               string                  `json:"mother_name"`
	FatherName               string                  `json:"father_name"`
	Nationality              string                  `json:"nationality"`
	IdentityNo               string                  `json:"identity_no"`
	Roles                    []string                `json:"roles"`
	WorkInformation          *EmployeeWorkInfoLookup `json:"work_information,omitempty"`
}

// EmployeeWorkInfoLookup for employee response
type EmployeeWorkInfoLookup struct {
	CompanyName    string `json:"company_name"`
	DepartmentName string `json:"department_name"`
	Manager        string `json:"manager"`
	JobTitle       string `json:"job_title"`
}

// Work Information response DTO with related entity names
type WorkInformationWithNames struct {
	ID              uint    `json:"id"`
	CompanyName     string  `json:"company_name"`
	DepartmentName  string  `json:"department_name"`
	Manager         string  `json:"manager"`
	JobPositionName string  `json:"job_position_name"`
	StartDate       string  `json:"start_date"`
	EndDate         *string `json:"end_date"`
}

// JobPosition response DTO without employee work information relationship
type JobPositionResponse struct {
	ID         uint   `json:"id"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Deleted    bool   `json:"deleted"`
	CreatedBy  string `json:"created_by"`
	ModifiedBy string `json:"modified_by"`
	Title      string `json:"title"`
}

// WorkInformation detail response structs
type WorkInformationEmployeeLookup struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type WorkInformationCompanyLookup struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type WorkInformationDepartmentLookup struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Manager string `json:"manager"`
}

type WorkInformationJobPositionLookup struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

// WorkInformation detail response DTO
type WorkInformationResponse struct {
	ID          uint                             `json:"id"`
	CreatedAt   time.Time                        `json:"created_at"`
	UpdatedAt   time.Time                        `json:"updated_at"`
	Deleted     bool                             `json:"deleted"`
	CreatedBy   string                           `json:"created_by"`
	ModifiedBy  string                           `json:"modified_by"`
	StartDate   time.Time                        `json:"start_date"`
	EndDate     *time.Time                       `json:"end_date"`
	PersonnelNo string                           `json:"personnel_no"`
	WorkEmail   string                           `json:"work_email"`
	Employee    WorkInformationEmployeeLookup    `json:"employee"`
	Company     WorkInformationCompanyLookup     `json:"company"`
	Department  WorkInformationDepartmentLookup  `json:"department"`
	JobPosition WorkInformationJobPositionLookup `json:"job_position"`
}

// LeaveType response DTO
type LeaveTypeResponse struct {
	ID                 uint      `json:"id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Deleted            bool      `json:"deleted"`
	CreatedBy          string    `json:"created_by"`
	ModifiedBy         string    `json:"modified_by"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	IsPaid             bool      `json:"is_paid"`
	IsLimited          bool      `json:"is_limited"`
	IsAccrual          bool      `json:"is_accrual"`
	IsRequiredDocument bool      `json:"is_required_document"`
}

type MyLeaveRequestResponse struct {
	ID              uint            `json:"id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	LeaveType       LeaveTypeLookup `json:"leave_type"`
	StartDate       time.Time       `json:"start_date"`
	EndDate         time.Time       `json:"end_date"`
	RequestedDays   float64         `json:"requested_days"`
	Reason          string          `json:"reason"`
	Status          string          `json:"status"`
	IsPaid          bool            `json:"is_paid"`
	ApprovedAt      *time.Time      `json:"approved_at"`
	RejectedAt      *time.Time      `json:"rejected_at"`
	RejectionReason string          `json:"rejection_reason"`
	CancelReason    string          `json:"cancel_reason"`
	CancelledAt     *time.Time      `json:"cancelled_at"`
	Comments        string          `json:"comments"`
}

type EmployeeLookup struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type AdminLeaveRequestResponse struct {
	ID              uint            `json:"id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Deleted         bool            `json:"deleted"`
	CreatedBy       string          `json:"created_by"`
	ModifiedBy      string          `json:"modified_by"`
	Employee        EmployeeLookup  `json:"employee"`
	LeaveType       LeaveTypeLookup `json:"leave_type"`
	StartDate       time.Time       `json:"start_date"`
	EndDate         time.Time       `json:"end_date"`
	RequestedDays   float64         `json:"requested_days"`
	RemainingDays   *float64        `json:"remaining_days"` // Leave balance remaining days (only for annual leave)
	Reason          string          `json:"reason"`
	Status          string          `json:"status"`
	IsPaid          bool            `json:"is_paid"`
	ApprovedBy      *uint           `json:"approved_by"`
	ApprovedAt      *time.Time      `json:"approved_at"`
	RejectedAt      *time.Time      `json:"rejected_at"`
	RejectionReason string          `json:"rejection_reason"`
	CancelReason    string          `json:"cancel_reason"`
	CancelledAt     *time.Time      `json:"cancelled_at"`
	Comments        string          `json:"comments"`
}

// LeaveBalance response DTO for My Leave Balances
type MyLeaveBalanceResponse struct {
	LeaveTypeName string  `json:"leave_type_name"`
	Year          int     `json:"year"`
	TotalDays     float64 `json:"total_days"`
	UsedDays      float64 `json:"used_days"`
	RemainingDays float64 `json:"remaining_days"`
}

// Enum conversion helpers for handling both English and Turkish enum values
// NormalizeGender converts gender values to English enum values
// Returns nil pointer if value is empty string
func NormalizeGender(value string) *string {
	if value == "" {
		return nil
	}
	var result string
	switch value {
	case "MALE", "Male", "male", "Erkek":
		result = "MALE"
	case "FEMALE", "Female", "female", "Kadın":
		result = "FEMALE"
	default:
		result = value // Return as-is if not recognized, let database validation catch it
	}
	return &result
}

// NormalizeMaritalStatus converts marital status values to English enum values
// Returns nil pointer if value is empty string
func NormalizeMaritalStatus(value string) *string {
	if value == "" {
		return nil
	}
	var result string
	switch value {
	case "MARRIED", "Married", "married", "Evli":
		result = "MARRIED"
	case "SINGLE", "Single", "single", "Bekar":
		result = "SINGLE"
	default:
		result = value
	}
	return &result
}

// NormalizeEmergencyContactRelation converts relation values to English enum values
// Returns nil pointer if value is empty string
func NormalizeEmergencyContactRelation(value string) *string {
	if value == "" {
		return nil
	}
	var result string
	switch value {
	case "MOTHER", "Mother", "mother", "Anne":
		result = "MOTHER"
	case "FATHER", "Father", "father", "Baba":
		result = "FATHER"
	case "SPOUSE", "Spouse", "spouse", "Eş":
		result = "SPOUSE"
	case "SIBLING", "Sibling", "sibling", "Kardeş":
		result = "SIBLING"
	case "OTHER", "Other", "other", "RELATIVE", "Relative", "relative", "Diğer":
		result = "OTHER"
	default:
		result = value
	}
	return &result
}

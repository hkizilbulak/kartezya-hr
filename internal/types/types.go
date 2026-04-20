package types

import "time"

// Lookup DTOs for basic listing responses
type CompanyLookup struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type LeaveTypeLookup struct {
	ID                 uint   `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	LimitAmount        *int   `json:"limit_amount"`
	IsRequiredDocument bool   `json:"is_required_document"`
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

type RoleLookup struct {
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
	TotalGap                 float64                 `json:"total_gap"`
	MaritalStatus            string                  `json:"marital_status"`
	EmergencyContact         string                  `json:"emergency_contact"`
	EmergencyContactName     string                  `json:"emergency_contact_name"`
	EmergencyContactRelation string                  `json:"emergency_contact_relation"`
	GradeID                  *int64                  `json:"grade_id"`
	ContractNo               string                  `json:"contract_no"`
	ProfessionStartDate      *string                 `json:"profession_start_date"`
	Note                     string                  `json:"note"`
	MotherName               string                  `json:"mother_name"`
	FatherName               string                  `json:"father_name"`
	Nationality              string                  `json:"nationality"`
	IdentityNo               string                  `json:"identity_no"`
	Roles                    []string                `json:"roles"`
	WorkInformation          *EmployeeWorkInfoLookup `json:"work_information,omitempty"`
	Status                   string                  `json:"status"`
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
	LimitAmount        *int      `json:"limit_amount"`
	IsAccrual          bool      `json:"is_accrual"`
	IsRequiredDocument bool      `json:"is_required_document"`
}

type MyLeaveRequestResponse struct {
	ID                  uint            `json:"id"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	LeaveType           LeaveTypeLookup `json:"leave_type"`
	StartDate           time.Time       `json:"start_date"`
	EndDate             time.Time       `json:"end_date"`
	IsStartDateFullDay  bool            `json:"is_start_date_full_day"`
	IsFinishDateFullDay bool            `json:"is_finish_date_full_day"`
	RequestedDays       float64         `json:"requested_days"`
	Reason              string          `json:"reason"`
	Status              string          `json:"status"`
	IsPaid              bool            `json:"is_paid"`
	ApprovedAt          *time.Time      `json:"approved_at"`
	RejectedAt          *time.Time      `json:"rejected_at"`
	RejectionReason     string          `json:"rejection_reason"`
	CancelReason        string          `json:"cancel_reason"`
	CancelledAt         *time.Time      `json:"cancelled_at"`
	Comments            string          `json:"comments"`
	DocumentCount       int             `json:"document_count"` // Number of attached documents
}

type EmployeeLookup struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type AdminLeaveRequestResponse struct {
	ID                  uint            `json:"id"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	Deleted             bool            `json:"deleted"`
	CreatedBy           string          `json:"created_by"`
	ModifiedBy          string          `json:"modified_by"`
	Employee            EmployeeLookup  `json:"employee"`
	LeaveType           LeaveTypeLookup `json:"leave_type"`
	StartDate           time.Time       `json:"start_date"`
	EndDate             time.Time       `json:"end_date"`
	IsStartDateFullDay  bool            `json:"is_start_date_full_day"`
	IsFinishDateFullDay bool            `json:"is_finish_date_full_day"`
	RequestedDays       float64         `json:"requested_days"`
	RemainingDays       *float64        `json:"remaining_days"` // Leave balance remaining days (only for annual leave)
	Reason              string          `json:"reason"`
	Status              string          `json:"status"`
	IsPaid              bool            `json:"is_paid"`
	ApprovedBy          *uint           `json:"approved_by"`
	ApprovedAt          *time.Time      `json:"approved_at"`
	RejectedAt          *time.Time      `json:"rejected_at"`
	RejectionReason     string          `json:"rejection_reason"`
	CancelReason        string          `json:"cancel_reason"`
	CancelledAt         *time.Time      `json:"cancelled_at"`
	Comments            string          `json:"comments"`
	DocumentCount       int             `json:"document_count"` // Number of attached documents
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

// EmployeeWorkInformationList for timeline view
type EmployeeWorkInformationList struct {
	ID             uint    `json:"id"`
	CompanyName    string  `json:"company_name"`
	DepartmentName string  `json:"department_name"`
	Manager        string  `json:"manager"`
	JobTitle       string  `json:"job_title"`
	StartDate      string  `json:"start_date"`
	EndDate        *string `json:"end_date"`
}

// EmployeeDetailResponse for detail endpoint - with work information list
type EmployeeDetailResponse struct {
	ID                       uint                          `json:"id"`
	User                     UserInfo                      `json:"user"`
	FirstName                string                        `json:"first_name"`
	LastName                 string                        `json:"last_name"`
	Email                    string                        `json:"email"`
	CompanyEmail             string                        `json:"company_email"`
	Phone                    string                        `json:"phone"`
	Address                  string                        `json:"address"`
	State                    string                        `json:"state"`
	City                     string                        `json:"city"`
	Gender                   string                        `json:"gender"`
	DateOfBirth              *string                       `json:"date_of_birth"`
	HireDate                 *string                       `json:"hire_date"`
	LeaveDate                *string                       `json:"leave_date,omitempty"`
	TotalGap                 float64                       `json:"total_gap"`
	MaritalStatus            string                        `json:"marital_status"`
	EmergencyContact         string                        `json:"emergency_contact"`
	EmergencyContactName     string                        `json:"emergency_contact_name"`
	EmergencyContactRelation string                        `json:"emergency_contact_relation"`
	GradeID                  *int64                        `json:"grade_id"`
	ContractNo               string                        `json:"contract_no"`
	ProfessionStartDate      *string                       `json:"profession_start_date"`
	Note                     string                        `json:"note"`
	MotherName               string                        `json:"mother_name"`
	FatherName               string                        `json:"father_name"`
	Nationality              string                        `json:"nationality"`
	IdentityNo               string                        `json:"identity_no"`
	Roles                    []string                      `json:"roles"`
	WorkInformation          []EmployeeWorkInformationList `json:"work_information,omitempty"`
	Status                   string                        `json:"status"`
}

// WorkDayReportFilter represents the filter criteria for work day report
type WorkDayReportFilter struct {
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	CompanyID     *uint     `json:"company_id"`
	DepartmentIDs []uint    `json:"department_ids"`
	IsActive      *bool     `json:"is_active"`
}

// ColumnConfig represents export column metadata from UI
type ColumnConfig struct {
	Key   string `json:"key" binding:"required"`
	Label string `json:"label" binding:"required"`
}

// WorkDayReportExportRequest represents request payload for dynamic work day Excel export
type WorkDayReportExportRequest struct {
	StartDate     string         `json:"start_date" binding:"required"`
	EndDate       string         `json:"end_date" binding:"required"`
	CompanyID     *uint          `json:"company_id"`
	DepartmentIDs []uint         `json:"department_ids"`
	ExportColumns []ColumnConfig `json:"export_columns" binding:"required"`
}

// WorkDayReportRow represents a single row in the work day report
type WorkDayReportRow struct {
	ID             uint    `json:"id"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	IdentityNo     string  `json:"identity_no"`
	CompanyName    string  `json:"company_name"`
	DepartmentName string  `json:"department_name"`
	Manager        string  `json:"manager"`
	TeamStartDate  *string `json:"team_start_date"`
	TeamEndDate    *string `json:"team_end_date"`
	HireDate       *string `json:"hire_date"`
	LeaveDate      *string `json:"leave_date"`
	WorkDays       float64 `json:"work_days"`
	UsedLeaveDays  float64 `json:"used_leave_days"`
	WorkedDays     float64 `json:"worked_days"`
	CurrentGrade   string  `json:"current_grade"`
}

// WorkDayReportResponse represents the complete work day report response
type WorkDayReportResponse struct {
	StartDate        time.Time          `json:"start_date"`
	EndDate          time.Time          `json:"end_date"`
	TotalWorkDays    float64            `json:"total_work_days"`
	TotalHolidayDays float64            `json:"total_holiday_days"`
	Rows             []WorkDayReportRow `json:"rows"`
}

// EforReportRow represents a single row in the efor report
type EforReportRow struct {
	ID             uint    `json:"id"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	IdentityNo     string  `json:"identity_no"`
	CompanyName    string  `json:"company_name"`
	DepartmentName string  `json:"department_name"`
	Manager        string  `json:"manager"`
	CurrentGrade   string  `json:"current_grade"`
	Grade          string  `json:"grade"`
	Rate           string  `json:"rate"`
	January        float64 `json:"january"`
	February       float64 `json:"february"`
	March          float64 `json:"march"`
	April          float64 `json:"april"`
	May            float64 `json:"may"`
	June           float64 `json:"june"`
	July           float64 `json:"july"`
	August         float64 `json:"august"`
	September      float64 `json:"september"`
	October        float64 `json:"october"`
	November       float64 `json:"november"`
	December       float64 `json:"december"`
	WorkedDays     float64 `json:"worked_days"`
}

// EforReportResponse represents the complete efor report response
type EforReportResponse struct {
	StartDate     time.Time       `json:"start_date"`
	EndDate       time.Time       `json:"end_date"`
	TotalWorkDays float64         `json:"total_work_days"`
	Rows          []EforReportRow `json:"rows"`
}

// EmployeeGradeLookup for lookup responses
type EmployeeGradeLookup struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// EmployeeGradeResponse for detail responses
type EmployeeGradeResponse struct {
	ID         uint           `json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Deleted    bool           `json:"deleted"`
	CreatedBy  string         `json:"created_by"`
	ModifiedBy string         `json:"modified_by"`
	Employee   EmployeeLookup `json:"employee"`
	Grade      GradeLookup    `json:"grade"`
	StartDate  time.Time      `json:"start_date"`
	EndDate    *time.Time     `json:"end_date"`
}

// EmployeeGradeWithNames for API responses with names
type EmployeeGradeWithNames struct {
	ID             uint    `json:"id"`
	EmployeeName   string  `json:"employee_name"`
	GradeName      string  `json:"grade_name"`
	StartDate      string  `json:"start_date"`
	EndDate        *string `json:"end_date"`
	IsCurrentGrade bool    `json:"is_current_grade"`
}

// EmployeeContractLookup for lookup responses
type EmployeeContractLookup struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// EmployeeContractResponse for detail responses
type EmployeeContractResponse struct {
	ID         uint           `json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Deleted    bool           `json:"deleted"`
	CreatedBy  string         `json:"created_by"`
	ModifiedBy string         `json:"modified_by"`
	Employee   EmployeeLookup `json:"employee"`
	ContractNo string         `json:"contract_no"`
	StartDate  time.Time      `json:"start_date"`
	EndDate    *time.Time     `json:"end_date"`
}

// EmployeeContractWithNames for API responses with names
type EmployeeContractWithNames struct {
	ID           uint    `json:"id"`
	EmployeeName string  `json:"employee_name"`
	ContractNo   string  `json:"contract_no"`
	StartDate    string  `json:"start_date"`
	EndDate      *string `json:"end_date"`
	IsActive     bool    `json:"is_active"`
}

// Pagination response wrapper
type PaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Page    int         `json:"page"`
	Limit   int         `json:"limit"`
	Total   int64       `json:"total"`
	Pages   int         `json:"pages"`
}

// GradeReportFilter represents the filter criteria for grade report
type GradeReportFilter struct {
	CompanyID    *uint `json:"company_id"`
	DepartmentID *uint `json:"department_id"`
	IsActive     *bool `json:"is_active"`
}

// GradeReportRow represents a single row in the grade report
type GradeReportRow struct {
	ID                  uint    `json:"id"`
	FirstName           string  `json:"first_name"`
	LastName            string  `json:"last_name"`
	HireDate            *string `json:"hire_date"`
	CompanyName         string  `json:"company_name"`
	DepartmentName      string  `json:"department_name"`
	Manager             string  `json:"manager"`
	TeamStartDate       *string `json:"team_start_date"`
	ProfessionStartDate *string `json:"profession_start_date"`
	TotalGap            float64 `json:"total_gap"`
	TotalExperienceText string  `json:"total_experience_text"`
	CurrentGrade        string  `json:"current_grade"`
	ExpectedGrade       string  `json:"expected_grade"`
}

// GradeReportResponse represents the complete grade report response
type GradeReportResponse struct {
	Rows []GradeReportRow `json:"rows"`
}

// GradeReportExportRequest represents the export request body from the frontend
type GradeReportExportRequest struct {
	CompanyID     *uint  `json:"companyId"`
	DepartmentID  *uint  `json:"departmentId"`
	DepartmentIDs []uint `json:"departmentIds"`
}

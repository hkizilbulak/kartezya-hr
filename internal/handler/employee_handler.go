package handler

import (
	"net/http"
	"strconv"
	"strings"

	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

type EmployeeHandler struct {
	employeeService service.EmployeeService
}

func NewEmployeeHandler(employeeService service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{
		employeeService: employeeService,
	}
}

type CreateEmployeeRequest struct {
	Email                    string   `json:"email" binding:"required,email"`
	CompanyEmail             string   `json:"company_email" binding:"required,email"`
	FirstName                string   `json:"first_name" binding:"required"`
	LastName                 string   `json:"last_name" binding:"required"`
	Phone                    string   `json:"phone"`
	Address                  string   `json:"address"`
	State                    string   `json:"state"`
	City                     string   `json:"city"`
	Gender                   string   `json:"gender"`
	DateOfBirth              string   `json:"date_of_birth"`
	HireDate                 string   `json:"hire_date"`
	LeaveDate                string   `json:"leave_date"`
	TotalGap                 float64  `json:"total_gap"`
	MaritalStatus            string   `json:"marital_status"`
	EmergencyContact         string   `json:"emergency_contact"`
	EmergencyContactName     string   `json:"emergency_contact_name"`
	EmergencyContactRelation string   `json:"emergency_contact_relation"`
	GradeID                  *int64   `json:"grade_id"`
	ContractNo               string   `json:"contract_no"`
	ProfessionStartDate      string   `json:"profession_start_date"`
	Note                     string   `json:"note"`
	MotherName               string   `json:"mother_name"`
	FatherName               string   `json:"father_name"`
	Nationality              string   `json:"nationality"`
	IdentityNo               string   `json:"identity_no"`
	Roles                    []string `json:"roles"`
}

type UpdateEmployeeRequest struct {
	Email                    string   `json:"email" binding:"omitempty,email"`
	CompanyEmail             string   `json:"company_email" binding:"omitempty,email"`
	FirstName                string   `json:"first_name" binding:"required"`
	LastName                 string   `json:"last_name" binding:"required"`
	Phone                    string   `json:"phone"`
	Address                  string   `json:"address"`
	State                    string   `json:"state"`
	City                     string   `json:"city"`
	Gender                   string   `json:"gender"`
	DateOfBirth              string   `json:"date_of_birth"`
	HireDate                 string   `json:"hire_date"`
	LeaveDate                string   `json:"leave_date"`
	TotalGap                 float64  `json:"total_gap"`
	MaritalStatus            string   `json:"marital_status"`
	EmergencyContact         string   `json:"emergency_contact"`
	EmergencyContactName     string   `json:"emergency_contact_name"`
	EmergencyContactRelation string   `json:"emergency_contact_relation"`
	GradeID                  *int64   `json:"grade_id"`
	ContractNo               string   `json:"contract_no"`
	ProfessionStartDate      string   `json:"profession_start_date"`
	Note                     string   `json:"note"`
	MotherName               string   `json:"mother_name"`
	FatherName               string   `json:"father_name"`
	Nationality              string   `json:"nationality"`
	IdentityNo               string   `json:"identity_no"`
	Roles                    []string `json:"roles"`
	Status                   string   `json:"status"`
}

type UpdateMyProfileRequest struct {
	Email                    string `json:"email"`
	Phone                    string `json:"phone"`
	Address                  string `json:"address"`
	State                    string `json:"state"`
	City                     string `json:"city"`
	Gender                   string `json:"gender"`
	DateOfBirth              string `json:"date_of_birth"`
	ProfessionStartDate      string `json:"profession_start_date"`
	MaritalStatus            string `json:"marital_status"`
	EmergencyContact         string `json:"emergency_contact"`
	EmergencyContactName     string `json:"emergency_contact_name"`
	EmergencyContactRelation string `json:"emergency_contact_relation"`
	MotherName               string `json:"mother_name"`
	FatherName               string `json:"father_name"`
	Nationality              string `json:"nationality"`
	IdentityNo               string `json:"identity_no"`
}

// CreateEmployee godoc
// @Summary Create a new employee
// @Description Create a new employee (Admin only)
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param employee body CreateEmployeeRequest true "Employee data"
// @Success 201 {object} map[string]interface{} "success: true, data: Employee"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /employees [post]
func (h *EmployeeHandler) CreateEmployee(c *gin.Context) {
	_, email, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !isAdmin(roles) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	var req CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	employee, err := h.employeeService.CreateEmployee(req.Email, req.CompanyEmail, req.FirstName, req.LastName, req.Phone, req.Address, req.State, req.City, req.Gender, req.DateOfBirth, req.HireDate, req.LeaveDate, req.TotalGap, req.MaritalStatus, req.EmergencyContact, req.EmergencyContactName, req.EmergencyContactRelation, req.GradeID, req.ContractNo, req.ProfessionStartDate, req.Note, req.MotherName, req.FatherName, req.Nationality, req.IdentityNo, email, req.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    employee,
		"message": "Employee created successfully",
	})
}

// GetEmployeeByID godoc
// @Summary Get employee by ID
// @Description Get employee details by ID (Admin only)
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} map[string]interface{} "success: true, data: Employee"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /employees/{id} [get]
func (h *EmployeeHandler) GetEmployeeByID(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !isAdmin(roles) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid employee ID",
		})
		return
	}

	// Include the status field in the response
	employee, err := h.employeeService.GetEmployeeByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Employee not found",
		})
		return
	}

	// Ensure the status field has a default value if empty
	status := employee.Status
	if status == "" {
		status = "UNKNOWN" // Default to UNKNOWN if status is empty
	}

	// Reverting to the original response structure with response.data
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employee, // Return the employee object directly under data
	})
}

// UpdateEmployee godoc
// @Summary Update employee
// @Description Update employee details by ID (Admin or own profile)
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param employee body UpdateEmployeeRequest true "Updated employee data"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employees/{id} [put]
func (h *EmployeeHandler) UpdateEmployee(c *gin.Context) {
	requestingUserID, email, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid employee ID",
		})
		return
	}

	var req UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// If company_email is not provided, use the current user's email from context
	if req.CompanyEmail == "" {
		req.CompanyEmail = email
	}

	if err := h.employeeService.UpdateEmployee(id, req.Email, req.CompanyEmail, req.FirstName, req.LastName, req.Phone, req.Address, req.State, req.City, req.Gender, req.DateOfBirth, req.HireDate, req.LeaveDate, req.TotalGap, req.MaritalStatus, req.EmergencyContact, req.EmergencyContactName, req.EmergencyContactRelation, req.GradeID, req.ContractNo, req.ProfessionStartDate, req.Note, req.MotherName, req.FatherName, req.Nationality, req.IdentityNo, req.Status, email, requestingUserID, isAdmin(roles), req.Roles); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized to update this employee profile" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get updated employee data to return
	employee, err := h.employeeService.GetEmployeeByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Employee updated but could not fetch updated data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employee,
		"message": "Employee updated successfully",
	})
}

// DeleteEmployee godoc
// @Summary Delete employee
// @Description Delete employee by ID (Admin only)
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employees/{id} [delete]
func (h *EmployeeHandler) DeleteEmployee(c *gin.Context) {
	_, email, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !isAdmin(roles) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid employee ID",
		})
		return
	}

	if err := h.employeeService.DeleteEmployee(id, email, isAdmin(roles)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Employee deleted successfully",
	})
}

// ListEmployees godoc
// @Summary List all employees
// @Description Get paginated list of all employees with filtering (Admin only)
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit number of results (default: 10)"
// @Param page query int false "Page number (default: 1)"
// @Param sort query string false "Sort field"
// @Param direction query string false "Sort direction (ASC/DESC)"
// @Param id query int false "Filter by employee ID"
// @Param first_name query string false "Filter by first name"
// @Param email query string false "Filter by email"
// @Param company_id query int false "Filter by company ID"
// @Param department_id query int false "Filter by department ID"
// @Param manager query string false "Filter by manager"
// @Param identity_no query string false "Filter by identity number"
// @Param gender query string false "Filter by gender"
// @Param marital_status query string false "Filter by marital status"
// @Param grade_id query int false "Filter by grade ID"
// @Success 200 {object} APIResponse{data=[]domain.Employee}
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employees [get]
func (h *EmployeeHandler) ListEmployees(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	if !isAdmin(roles) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	page := 1
	limit := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Parse filter parameters
	filters := make(map[string]interface{})

	if id := c.Query("id"); id != "" {
		if parsed, err := strconv.Atoi(id); err == nil && parsed > 0 {
			filters["id"] = parsed
		}
	}

	if firstName := c.Query("first_name"); firstName != "" {
		filters["first_name"] = firstName
	}

	if email := c.Query("email"); email != "" {
		filters["email"] = email
	}

	if companyID := c.Query("company_id"); companyID != "" {
		if parsed, err := strconv.Atoi(companyID); err == nil && parsed > 0 {
			filters["company_id"] = parsed
		}
	}

	if departmentIDs := c.Query("department_ids"); departmentIDs != "" {
		// Parse comma-separated department IDs
		departmentIDList := strings.Split(departmentIDs, ",")
		var validDepartmentIDs []int
		for _, deptIDStr := range departmentIDList {
			deptIDStr = strings.TrimSpace(deptIDStr)
			if parsed, err := strconv.Atoi(deptIDStr); err == nil && parsed > 0 {
				validDepartmentIDs = append(validDepartmentIDs, parsed)
			}
		}
		if len(validDepartmentIDs) > 0 {
			filters["department_ids"] = validDepartmentIDs
		}
	} else if departmentID := c.Query("department_id"); departmentID != "" {
		// Keep backward compatibility with single department_id
		if parsed, err := strconv.Atoi(departmentID); err == nil && parsed > 0 {
			filters["department_id"] = parsed
		}
	}

	if manager := c.Query("manager"); manager != "" {
		filters["manager"] = manager
	}

	if identityNo := c.Query("identity_no"); identityNo != "" {
		filters["identity_no"] = identityNo
	}

	if gender := c.Query("gender"); gender != "" {
		filters["gender"] = gender
	}

	if maritalStatus := c.Query("marital_status"); maritalStatus != "" {
		filters["marital_status"] = maritalStatus
	}

	if gradeID := c.Query("grade_id"); gradeID != "" {
		if parsed, err := strconv.Atoi(gradeID); err == nil && parsed > 0 {
			filters["grade_id"] = parsed
		}
	}

	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	// Parse sorting parameters
	sortField := c.Query("sort")
	sortDirection := c.Query("direction")

	// Set default sorting to first_name, last_name ASC if no sort field is provided
	if sortField == "" {
		sortField = "first_name"
	}

	// Default direction is ASC
	if sortDirection != "ASC" && sortDirection != "DESC" {
		sortDirection = "ASC"
	}

	offset := (page - 1) * limit

	employees, err := h.employeeService.ListEmployeesWithFilters(limit, offset, sortField, sortDirection, filters, isAdmin(roles))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get total count for pagination with filters
	totalCount, err := h.employeeService.GetTotalCountWithFilters(filters)
	if err != nil {
		totalCount = 0
	}

	totalPages := 1
	if totalCount > 0 {
		totalPages = int((totalCount + int64(limit) - 1) / int64(limit))
	}

	c.JSON(http.StatusOK, gin.H{
		"data": employees,
		"page": gin.H{
			"total":       totalCount,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
			"sort":        sortField,
			"direction":   sortDirection,
		},
		"success": true,
	})
}

// GetMyProfile godoc
// @Summary Get my employee profile
// @Description Get the profile of the authenticated employee
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "success: true, data: Employee"
// @Failure 401 {object} map[string]interface{}
// @Router /employees/me [get]
func (h *EmployeeHandler) GetMyProfile(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	employee, err := h.employeeService.GetEmployeeByUserID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Employee profile not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employee,
	})
}

// UpdateMyProfile godoc
// @Summary Update my employee profile
// @Description Update the profile of the authenticated employee (own profile only)
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param employee body UpdateMyProfileRequest true "Updated employee data"
// @Success 200 {object} map[string]interface{} "success: true, data: Employee"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /employees/me [put]
func (h *EmployeeHandler) UpdateMyProfile(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	var req UpdateMyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Update only the authenticated user's profile
	if err := h.employeeService.UpdateMyProfile(userID, req.Email, req.Phone, req.Address, req.State, req.City, req.Gender, req.DateOfBirth, req.ProfessionStartDate, req.MaritalStatus, req.EmergencyContact, req.EmergencyContactName, req.EmergencyContactRelation, req.MotherName, req.FatherName, req.Nationality, req.IdentityNo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get updated employee data to return
	employee, err := h.employeeService.GetEmployeeByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Profile updated but could not fetch updated data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employee,
		"message": "Profile updated successfully",
	})
}

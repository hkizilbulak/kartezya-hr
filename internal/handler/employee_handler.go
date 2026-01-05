package handler

import (
	"net/http"
	"strconv"

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
	CompanyEmail             string  `json:"company_email" binding:"required,email"`
	FirstName                string  `json:"first_name" binding:"required"`
	LastName                 string  `json:"last_name" binding:"required"`
	Phone                    string  `json:"phone"`
	Address                  string  `json:"address"`
	State                    string  `json:"state"`
	City                     string  `json:"city"`
	Gender                   string  `json:"gender"`
	DateOfBirth              string  `json:"date_of_birth"`
	HireDate                 string  `json:"hire_date"`
	LeaveDate                string  `json:"leave_date"`
	TotalExperience          float64 `json:"total_experience"`
	MaritalStatus            string  `json:"marital_status"`
	EmergencyContact         string  `json:"emergency_contact"`
	EmergencyContactName     string  `json:"emergency_contact_name"`
	EmergencyContactRelation string  `json:"emergency_contact_relation"`
}

type UpdateEmployeeRequest struct {
	CompanyEmail             string  `json:"company_email" binding:"required,email"`
	FirstName                string  `json:"first_name" binding:"required"`
	LastName                 string  `json:"last_name" binding:"required"`
	Phone                    string  `json:"phone"`
	Address                  string  `json:"address"`
	State                    string  `json:"state"`
	City                     string  `json:"city"`
	Gender                   string  `json:"gender"`
	DateOfBirth              string  `json:"date_of_birth"`
	HireDate                 string  `json:"hire_date"`
	LeaveDate                string  `json:"leave_date"`
	TotalExperience          float64 `json:"total_experience"`
	MaritalStatus            string  `json:"marital_status"`
	EmergencyContact         string  `json:"emergency_contact"`
	EmergencyContactName     string  `json:"emergency_contact_name"`
	EmergencyContactRelation string  `json:"emergency_contact_relation"`
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

	employee, err := h.employeeService.CreateEmployee(req.CompanyEmail, req.FirstName, req.LastName, req.Phone, req.Address, req.State, req.City, req.Gender, req.DateOfBirth, req.HireDate, req.LeaveDate, req.TotalExperience, req.MaritalStatus, req.EmergencyContact, req.EmergencyContactName, req.EmergencyContactRelation, email)
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

	employee, err := h.employeeService.GetEmployeeByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employee,
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

	if err := h.employeeService.UpdateEmployee(id, req.CompanyEmail, req.FirstName, req.LastName, req.Phone, req.Address, req.State, req.City, req.Gender, req.DateOfBirth, req.HireDate, req.LeaveDate, req.TotalExperience, req.MaritalStatus, req.EmergencyContact, req.EmergencyContactName, req.EmergencyContactRelation, email, requestingUserID, isAdmin(roles)); err != nil {
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
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
// @Description Get paginated list of all employees (Admin only)
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit number of results (default: 10)"
// @Param offset query int false "Offset for pagination (default: 0)"
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

	limit := 10
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	employees, err := h.employeeService.ListEmployees(limit, offset, isAdmin(roles))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employees,
		"meta": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(employees),
		},
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

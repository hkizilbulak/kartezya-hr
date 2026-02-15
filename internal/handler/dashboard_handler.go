package handler

import (
	"log"
	"net/http"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	employeeService   service.EmployeeService
	departmentService service.DepartmentService
	companyService    service.CompanyService
	leaveService      service.LeaveService
}

func NewDashboardHandler(
	employeeService service.EmployeeService,
	departmentService service.DepartmentService,
	companyService service.CompanyService,
	leaveService service.LeaveService,
) *DashboardHandler {
	return &DashboardHandler{
		employeeService:   employeeService,
		departmentService: departmentService,
		companyService:    companyService,
		leaveService:      leaveService,
	}
}

type DashboardDataResponse struct {
	TotalEmployees       int64 `json:"total_employees"`
	TotalDepartments     int64 `json:"total_departments"`
	TotalCompanies       int64 `json:"total_companies"`
	PendingLeaveRequests int64 `json:"pending_leave_requests"`
}

// Chart response types
type GenderChartData struct {
	Gender string `json:"gender"`
	Count  int64  `json:"count"`
}

type PositionChartData struct {
	PositionTitle string `json:"position_title"`
	Count         int64  `json:"count"`
}

type CompanyDepartmentChartData struct {
	CompanyName    string `json:"company_name"`
	DepartmentName string `json:"department_name"`
	Count          int64  `json:"count"`
}

// GetDashboardData godoc
// @Summary Get dashboard data
// @Description Get all dashboard statistics in a single request
// @Tags dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse{data=DashboardDataResponse}
// @Failure 401 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /dashboard/data [get]
func (h *DashboardHandler) GetDashboardData(c *gin.Context) {
	_, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	// Fetch total employees (only ACTIVE status)
	totalEmployees, err := h.employeeService.GetTotalCountWithFilters(map[string]interface{}{
		"status": "ACTIVE",
	})
	if err != nil {
		log.Printf("Error fetching total employees: %v", err)
		totalEmployees = 0
	}

	// Fetch total departments
	totalDepartments, err := h.departmentService.GetTotalCount()
	if err != nil {
		log.Printf("Error fetching total departments: %v", err)
		totalDepartments = 0
	}

	// Fetch total companies
	totalCompanies, err := h.companyService.GetTotalCount()
	if err != nil {
		log.Printf("Error fetching total companies: %v", err)
		totalCompanies = 0
	}

	// Fetch pending leave requests
	pendingLeaves, err := h.leaveService.GetLeavesByStatus(domain.LeaveStatusPending, "created_at", "DESC")
	if err != nil {
		log.Printf("Error fetching pending leave requests: %v", err)
		pendingLeaves = []*domain.LeaveRequest{}
	}
	pendingLeaveRequests := int64(len(pendingLeaves))

	dashboardData := DashboardDataResponse{
		TotalEmployees:       totalEmployees,
		TotalDepartments:     totalDepartments,
		TotalCompanies:       totalCompanies,
		PendingLeaveRequests: pendingLeaveRequests,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dashboardData,
	})
}

// GetEmployeesByGender godoc
// @Summary Get employees count by gender
// @Description Get employee statistics grouped by gender
// @Tags dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse{data=[]GenderChartData}
// @Failure 401 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /dashboard/employees-by-gender [get]
func (h *DashboardHandler) GetEmployeesByGender(c *gin.Context) {
	_, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	data, err := h.employeeService.GetEmployeeCountByGender()
	if err != nil {
		log.Printf("Error fetching employees by gender: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch employees by gender",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// GetEmployeesByPosition godoc
// @Summary Get employees count by position
// @Description Get employee statistics grouped by job position
// @Tags dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse{data=[]PositionChartData}
// @Failure 401 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /dashboard/employees-by-position [get]
func (h *DashboardHandler) GetEmployeesByPosition(c *gin.Context) {
	_, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	data, err := h.employeeService.GetEmployeeCountByPosition()
	if err != nil {
		log.Printf("Error fetching employees by position: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch employees by position",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// GetEmployeesByCompanyDepartment godoc
// @Summary Get employees count by company and department
// @Description Get employee statistics grouped by company and department
// @Tags dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse{data=[]CompanyDepartmentChartData}
// @Failure 401 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /dashboard/employees-by-company-department [get]
func (h *DashboardHandler) GetEmployeesByCompanyDepartment(c *gin.Context) {
	_, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	data, err := h.employeeService.GetEmployeeCountByCompanyDepartment()
	if err != nil {
		log.Printf("Error fetching employees by company and department: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch employees by company and department",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

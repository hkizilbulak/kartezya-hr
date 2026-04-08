package handler

import (
	"kartezya-hr/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type LookupHandler struct {
	lookupService service.LookupService
}

func NewLookupHandler(lookupService service.LookupService) *LookupHandler {
	return &LookupHandler{
		lookupService: lookupService,
	}
}

// GetCompaniesLookup godoc
// @Summary Get all companies for lookup
// @Description Get all companies as lookup data (public API)
// @Tags Lookup
// @Produce json
// @Success 200 {object} APIResponse{data=[]types.CompanyLookup}
// @Router /lookup/companies [get]
func (h *LookupHandler) GetCompaniesLookup(c *gin.Context) {
	companies, err := h.lookupService.GetCompaniesLookup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    companies,
	})
}

// GetDepartmentsLookup godoc
// @Summary Get all departments for lookup
// @Description Get all departments as lookup data (public API)
// @Tags Lookup
// @Produce json
// @Success 200 {object} APIResponse{data=[]types.DepartmentLookup}
// @Router /lookup/departments [get]
func (h *LookupHandler) GetDepartmentsLookup(c *gin.Context) {
	departments, err := h.lookupService.GetDepartmentsLookup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    departments,
	})
}

// GetDepartmentsByCompanyLookup godoc
// @Summary Get departments by company for lookup
// @Description Get departments filtered by company ID as lookup data (public API)
// @Tags Lookup
// @Produce json
// @Param company_id query int true "Company ID"
// @Success 200 {object} APIResponse{data=[]types.DepartmentLookup}
// @Router /lookup/departments-by-company [get]
func (h *LookupHandler) GetDepartmentsByCompanyLookup(c *gin.Context) {
	companyIDStr := c.Query("company_id")
	if companyIDStr == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "company_id is required",
		})
		return
	}

	companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "invalid company_id",
		})
		return
	}

	departments, err := h.lookupService.GetDepartmentsByCompanyLookup(uint(companyID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    departments,
	})
}

// GetJobPositionsLookup godoc
// @Summary Get all job positions for lookup
// @Description Get all job positions as lookup data (public API)
// @Tags Lookup
// @Produce json
// @Success 200 {object} APIResponse{data=[]types.JobPositionLookup}
// @Router /lookup/job-positions [get]
func (h *LookupHandler) GetJobPositionsLookup(c *gin.Context) {
	jobPositions, err := h.lookupService.GetJobPositionsLookup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    jobPositions,
	})
}

// GetLeaveTypesLookup godoc
// @Summary Get all leave types for lookup
// @Description Get all leave types as lookup data (public API)
// @Tags Lookup
// @Produce json
// @Success 200 {object} APIResponse{data=[]types.LeaveTypeLookup}
// @Router /lookup/leave-types [get]
func (h *LookupHandler) GetLeaveTypesLookup(c *gin.Context) {
	leaveTypes, err := h.lookupService.GetLeaveTypesLookup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    leaveTypes,
	})
}

// GetGradesLookup godoc
// @Summary Get all grades for lookup
// @Description Get all grades as lookup data (public API)
// @Tags Lookup
// @Produce json
// @Success 200 {object} APIResponse{data=[]types.GradeLookup}
// @Router /lookup/grades [get]
func (h *LookupHandler) GetGradesLookup(c *gin.Context) {
	grades, err := h.lookupService.GetGradesLookup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    grades,
	})
}

// GetRolesLookup godoc
// @Summary Get all roles for lookup
// @Description Get all roles as lookup data (admin API)
// @Tags Lookup
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse{data=[]types.RoleLookup}
// @Router /lookup/roles [get]
func (h *LookupHandler) GetRolesLookup(c *gin.Context) {
	roles, err := h.lookupService.GetRolesLookup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    roles,
	})
}

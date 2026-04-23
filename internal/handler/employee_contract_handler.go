package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type EmployeeContractHandler struct {
	employeeContractService service.EmployeeContractService
	employeeService         service.EmployeeService
}

func NewEmployeeContractHandler(employeeContractService service.EmployeeContractService, employeeService service.EmployeeService) *EmployeeContractHandler {
	return &EmployeeContractHandler{
		employeeContractService: employeeContractService,
		employeeService:         employeeService,
	}
}

type CreateEmployeeContractRequest struct {
	EmployeeID uint `json:"employee_id" binding:"required"`
	ContractID uint `json:"contract_id" binding:"required"`
}

type UpdateEmployeeContractRequest struct {
	EmployeeID uint `json:"employee_id" binding:"required"`
	ContractID uint `json:"contract_id" binding:"required"`
}

// CreateEmployeeContract godoc
// @Summary Create employee contract
// @Description Create employee contract record (Admin only)
// @Tags employee-contracts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateEmployeeContractRequest true "Employee contract data"
// @Success 201 {object} APIResponse{data=types.EmployeeContractResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employee-contracts [post]
func (h *EmployeeContractHandler) CreateEmployeeContract(c *gin.Context) {
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

	var req CreateEmployeeContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	employeeContract, err := h.employeeContractService.CreateContract(req.EmployeeID, req.ContractID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    employeeContract,
		"message": "Employee contract created successfully",
	})
}

// GetEmployeeContractByID godoc
// @Summary Get employee contract by ID
// @Description Get specific employee contract by ID
// @Tags employee-contracts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee Contract ID"
// @Success 200 {object} APIResponse{data=types.EmployeeContractResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /employee-contracts/{id} [get]
func (h *EmployeeContractHandler) GetEmployeeContractByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee contract ID"})
		return
	}

	employeeContract, err := h.employeeContractService.GetContractByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, employeeContract)
}

// GetMyEmployeeContracts godoc
// @Summary Get my employee contracts
// @Description Get employee contracts for the authenticated employee
// @Tags employee-contracts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse{data=types.EmployeeContractWithNames}
// @Failure 401 {object} APIResponse
// @Router /employee-contracts/me [get]
func (h *EmployeeContractHandler) GetMyEmployeeContracts(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	employeeContracts, err := h.employeeContractService.GetContractsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employeeContracts,
	})
}

// ListEmployeeContracts godoc
// @Summary List employee contracts
// @Description Get paginated list of all employee contracts (Admin only)
// @Tags employee-contracts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort query string false "Sort field (default: id)"
// @Param direction query string false "Sort direction (default: ASC)"
// @Param employee_id query int false "Filter by employee ID"
// @Success 200 {object} APIResponse{data=[]types.EmployeeContractResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employee-contracts [get]
func (h *EmployeeContractHandler) ListEmployeeContracts(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	if sortParams.Direction != "ASC" && sortParams.Direction != "DESC" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sort direction. Must be ASC or DESC"})
		return
	}

	var employeeID *uint
	if employeeIDStr := c.Query("employee_id"); employeeIDStr != "" {
		id, err := strconv.ParseUint(employeeIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee_id parameter"})
			return
		}
		empID := uint(id)
		employeeID = &empID
	}

	response, err := h.employeeContractService.GetAllContracts(page, limit, sortParams, employeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateEmployeeContract godoc
// @Summary Update employee contract
// @Description Update employee contract by ID (Admin only)
// @Tags employee-contracts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee Contract ID"
// @Param request body UpdateEmployeeContractRequest true "Updated employee contract data"
// @Success 200 {object} APIResponse{data=types.EmployeeContractResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employee-contracts/{id} [put]
func (h *EmployeeContractHandler) UpdateEmployeeContract(c *gin.Context) {
	requestingUserID, email, roles, ok := getUserContext(c)
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
			"error":   "Invalid employee contract ID",
		})
		return
	}

	var req UpdateEmployeeContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if err := h.employeeContractService.UpdateContract(id, req.EmployeeID, req.ContractID, email, requestingUserID, isAdmin(roles)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	employeeContract, err := h.employeeContractService.GetContractByID(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Employee contract updated successfully",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employeeContract,
		"message": "Employee contract updated successfully",
	})
}

// DeleteEmployeeContract godoc
// @Summary Delete employee contract
// @Description Delete employee contract by ID (Admin only)
// @Tags employee-contracts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee Contract ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /employee-contracts/{id} [delete]
func (h *EmployeeContractHandler) DeleteEmployeeContract(c *gin.Context) {
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
			"error":   "Invalid employee contract ID",
		})
		return
	}

	if err := h.employeeContractService.DeleteContract(id, email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Employee contract deleted successfully",
	})
}

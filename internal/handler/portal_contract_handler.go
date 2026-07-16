package handler

import (
	"net/http"
	"strconv"
	"kartezya-hr/internal/service"
	"github.com/gin-gonic/gin"
)

type PortalContractHandler struct {
	contractService service.PortalContractService
}

func NewPortalContractHandler(contractService service.PortalContractService) *PortalContractHandler {
	return &PortalContractHandler{
		contractService: contractService,
	}
}

// GetEmployeeContracts godoc
// @Summary Get employee portal contracts
// @Description Get all portal contracts with their approval status for a specific employee
// @Tags portal-contracts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} APIResponse{data=[]types.EmployeePortalContractResponse}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /employees/{id}/contracts [get]
func (h *PortalContractHandler) GetEmployeeContracts(c *gin.Context) {
	employeeIDStr := c.Param("id")
	employeeID, err := strconv.ParseUint(employeeIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid employee ID format",
		})
		return
	}

	response, err := h.contractService.GetEmployeeContractsStatus(uint(employeeID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

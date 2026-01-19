package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type CompanyHandler struct {
	companyService service.CompanyService
}

func NewCompanyHandler(companyService service.CompanyService) *CompanyHandler {
	return &CompanyHandler{
		companyService: companyService,
	}
}

// Company request/response DTOs
type CreateCompanyRequest struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Website string `json:"website"`
}

type UpdateCompanyRequest struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Website string `json:"website"`
}

// CreateCompany godoc
// @Summary Create company
// @Description Create a new company (Admin only)
// @Tags companies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param company body CreateCompanyRequest true "Company data"
// @Success 201 {object} APIResponse{data=types.CompanyResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /companies [post]
func (h *CompanyHandler) CreateCompany(c *gin.Context) {
	var req CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get current user ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	company := &domain.Company{
		Name:    req.Name,
		Address: req.Address,
		Phone:   req.Phone,
		Email:   req.Email,
		Website: req.Website,
	}

	err := h.companyService.CreateCompany(company, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    company,
		"message": "Company created successfully",
	})
}

// GetCompany godoc
// @Summary Get company by ID
// @Description Get a specific company by ID (Admin only)
// @Tags companies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Company ID"
// @Success 200 {object} APIResponse{data=types.CompanyResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /companies/{id} [get]
func (h *CompanyHandler) GetCompany(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid company ID",
		})
		return
	}

	company, err := h.companyService.GetCompanyByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    company,
	})
}

// GetCompanies godoc
// @Summary Get companies with pagination
// @Description Get paginated list of companies with sorting (Admin only)
// @Tags companies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param sort query string false "Sort field (default: id)"
// @Param direction query string false "Sort direction (default: ASC)"
// @Success 200 {object} APIResponse{data=[]types.CompanyResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /companies [get]
func (h *CompanyHandler) GetCompanies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "id"),
		Direction: c.DefaultQuery("direction", "ASC"),
	}

	result, err := h.companyService.GetAllCompanies(page, limit, sortParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    result.Data,
		"page":    result.Page,
		"success": true,
	})
}

// UpdateCompany godoc
// @Summary Update company
// @Description Update a company by ID (Admin only)
// @Tags companies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Company ID"
// @Param request body UpdateCompanyRequest true "Updated company data"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /companies/{id} [put]
func (h *CompanyHandler) UpdateCompany(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid company ID",
		})
		return
	}

	var req UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get current user ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	company := &domain.Company{
		Name:    req.Name,
		Address: req.Address,
		Phone:   req.Phone,
		Email:   req.Email,
		Website: req.Website,
	}

	err = h.companyService.UpdateCompany(id, company, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Company updated successfully",
	})
}

// DeleteCompany godoc
// @Summary Delete company
// @Description Delete a company by ID (Admin only)
// @Tags companies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Company ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Router /companies/{id} [delete]
func (h *CompanyHandler) DeleteCompany(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid company ID",
		})
		return
	}

	// Get current user ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	err = h.companyService.DeleteCompany(id, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Company deleted successfully",
	})
}

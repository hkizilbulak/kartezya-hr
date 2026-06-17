package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type FAQHandler struct {
	faqService service.FAQService
}

func NewFAQHandler(faqService service.FAQService) *FAQHandler {
	return &FAQHandler{faqService: faqService}
}

// DTOs (Request nesneleri)
type CreateFAQRequest struct {
	Title       string           `json:"title" binding:"required"`
	Description string           `json:"description" binding:"required"`
	Status      domain.FAQStatus `json:"status"`
}

type UpdateFAQRequest struct {
	Title       string           `json:"title" binding:"required"`
	Description string           `json:"description" binding:"required"`
	Status      domain.FAQStatus `json:"status"`
}

// Create handles FAQ creation
// @Summary Yeni bir SSS kaydı oluşturur
// @Description İK personeli tarafından yeni bir Sıkça Sorulan Soru ekler (Yalnızca Admin)
// @Tags faqs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateFAQRequest true "FAQ Data"
// @Success 201 {object} map[string]interface{}
// @Router /faqs [post]
func (h *FAQHandler) Create(c *gin.Context) {
	var req CreateFAQRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Proje standardına göre userID'yi al
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Modeldeki CreatedBy string =>  uint'i stringe çevir
	userIDStr := fmt.Sprintf("%v", userID)

	faq := &domain.FAQ{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	}

	if faq.Status == "" {
		faq.Status = domain.FAQStatusActive
	}

	if err := h.faqService.CreateFAQ(faq, userIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    faq,
		"message": "FAQ created successfully",
	})
}

// GetAll handles fetching FAQ list
// @Summary SSS kayıtlarını listeler
// @Description Sayfalama ve sıralama ile tüm SSS kayıtlarını getirir
// @Tags faqs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param sort query string false "Sıralama Alanı"
// @Param direction query string false "Sıralama Yönü (ASC/DESC)"
// @Success 200 {object} map[string]interface{}
// @Router /faqs [get]
func (h *FAQHandler) GetAll(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	sortParams := types.SortParams{
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	faqs, total, err := h.faqService.GetAllFAQs(limit, offset, sortParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    faqs,
		"total":   total,
	})
}

// GetByID handles fetching a single FAQ
// @Summary Tek bir SSS kaydını getirir
// @Description ID'sine göre SSS kaydının detaylarını döner
// @Tags faqs
// @Produce json
// @Security BearerAuth
// @Param id path int true "FAQ ID"
// @Success 200 {object} map[string]interface{}
// @Router /faqs/{id} [get]
func (h *FAQHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid ID format",
		})
		return
	}

	faq, err := h.faqService.GetFAQByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "FAQ not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    faq,
	})
}

// Update handles FAQ updating
// @Summary SSS kaydını günceller
// @Description Mevcut bir SSS kaydının başlık, açıklama ve durumunu günceller
// @Tags faqs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "FAQ ID"
// @Param request body UpdateFAQRequest true "FAQ Data"
// @Success 200 {object} map[string]interface{}
// @Router /faqs/{id} [put]
func (h *FAQHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid ID format",
		})
		return
	}

	var req UpdateFAQRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	userIDStr := fmt.Sprintf("%v", userID)

	existingFAQ, err := h.faqService.GetFAQByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "FAQ not found",
		})
		return
	}

	existingFAQ.Title = req.Title
	existingFAQ.Description = req.Description
	if req.Status != "" {
		existingFAQ.Status = req.Status
	}

	if err := h.faqService.UpdateFAQ(existingFAQ, userIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    existingFAQ,
		"message": "FAQ updated successfully",
	})
}

// Delete handles FAQ deletion
// @Summary SSS kaydını siler
// @Description ID'si verilen SSS kaydını soft-delete yöntemiyle siler
// @Tags faqs
// @Produce json
// @Security BearerAuth
// @Param id path int true "FAQ ID"
// @Success 200 {object} map[string]interface{}
// @Router /faqs/{id} [delete]
func (h *FAQHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid ID format",
		})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}
	userIDStr := fmt.Sprintf("%v", userID)

	if err := h.faqService.DeleteFAQ(uint(id), userIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "FAQ successfully deleted",
	})
}
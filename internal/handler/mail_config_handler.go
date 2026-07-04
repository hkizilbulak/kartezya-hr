package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

type MailConfigHandler struct {
	svc service.MailConfigService
}

func NewMailConfigHandler(svc service.MailConfigService) *MailConfigHandler {
	return &MailConfigHandler{svc: svc}
}

// GET /mail-configs
func (h *MailConfigHandler) GetAll(c *gin.Context) {
	configs, err := h.svc.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": configs})
}

// GET /mail-configs/:id
func (h *MailConfigHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	cfg, err := h.svc.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cfg})
}

// POST /mail-configs
func (h *MailConfigHandler) Create(c *gin.Context) {
	var req MailConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg := buildMailConfig(req)
	if err := h.svc.Create(cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": cfg})
}

// PUT /mail-configs/:id
func (h *MailConfigHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req MailConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg := buildMailConfig(req)
	if err := h.svc.Update(uint(id), cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "updated"})
}

// DELETE /mail-configs/:id
func (h *MailConfigHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "deleted"})
}

// ── DTOs ─────────────────────────────────────────────────────────────────────

type MailRecipientRequest struct {
	RecipientType  string `json:"recipient_type" binding:"required"`
	ValueType      string `json:"value_type" binding:"required"`
	RecipientValue string `json:"recipient_value" binding:"required"`
}

type MailConfigRequest struct {
	MailKey            string                 `json:"mail_key" binding:"required"`
	Description        string                 `json:"description"`
	Provider           string                 `json:"provider" binding:"required"`
	ResendTemplateCode string                 `json:"resend_template_code"`
	IsActive           bool                   `json:"is_active"`
	Recipients         []MailRecipientRequest `json:"recipients"`
}

func buildMailConfig(req MailConfigRequest) *domain.MailConfiguration {
	recipients := make([]domain.MailRecipient, 0, len(req.Recipients))
	for _, r := range req.Recipients {
		recipients = append(recipients, domain.MailRecipient{
			RecipientType:  domain.MailRecipientType(r.RecipientType),
			ValueType:      domain.MailValueType(r.ValueType),
			RecipientValue: r.RecipientValue,
		})
	}
	return &domain.MailConfiguration{
		MailKey:            req.MailKey,
		Description:        req.Description,
		Provider:           domain.MailProvider(req.Provider),
		ResendTemplateCode: req.ResendTemplateCode,
		IsActive:           req.IsActive,
		Recipients:         recipients,
	}
}

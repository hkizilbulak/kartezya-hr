package handler

import (
	"net/http"

	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

type EmailHandler struct {
	emailService service.EmailService
}

func NewEmailHandler(emailService service.EmailService) *EmailHandler {
	return &EmailHandler{emailService: emailService}
}

// SendEmailRequest is the request body for sending a custom email
type SendEmailRequest struct {
	To      []string `json:"to" binding:"required,min=1" example:"[\"user@example.com\"]"`
	Subject string   `json:"subject" binding:"required" example:"Bilgilendirme"`
	Body    string   `json:"body" binding:"required" example:"<p>Merhaba,</p><p>Bu bir test mailidir.</p>"`
}

// SendEmailResponse is the response for a successful email send
type SendEmailResponse struct {
	SentCount int      `json:"sent_count"`
	To        []string `json:"to"`
}

// SendTemplateEmailRequest is the request body for sending an email using a template
type SendTemplateEmailRequest struct {
	To         []string               `json:"to" binding:"required,min=1" example:"[\"user@example.com\"]"`
	Subject    string                 `json:"subject" example:"Bilgilendirme"`
	TemplateId string                 `json:"template_id" binding:"required" example:"b36f6d63-5477-4c7b-b5d1-67ec5d69ba2a"`
	Variables  map[string]interface{} `json:"variables" swaggertype:"object"`
}

// SendEmail godoc
// @Summary Send custom email
// @Description Send a custom HTML email to one or more recipients (Admin only)
// @Tags email
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SendEmailRequest true "Email request"
// @Success 200 {object} APIResponse{data=SendEmailResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /email/send [post]
func (h *EmailHandler) SendEmail(c *gin.Context) {
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

	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := h.emailService.SendCustomEmail(req.To, req.Subject, req.Body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": SendEmailResponse{
			SentCount: len(req.To),
			To:        req.To,
		},
		"message": "Email(s) sent successfully",
	})
}

// SendTemplateEmail godoc
// @Summary Send template email
// @Description Send an email using a predefined template (Admin only)
// @Tags email
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SendTemplateEmailRequest true "Template email request"
// @Success 200 {object} APIResponse{data=SendEmailResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /email/template/send [post]
func (h *EmailHandler) SendTemplateEmail(c *gin.Context) {
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

	var req SendTemplateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := h.emailService.SendTemplateEmail(req.To, req.Subject, req.TemplateId, req.Variables); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": SendEmailResponse{
			SentCount: len(req.To),
			To:        req.To,
		},
		"message": "Template email(s) sent successfully",
	})
}

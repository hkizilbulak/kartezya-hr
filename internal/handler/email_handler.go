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

// SendDynamicTemplateEmail godoc
// @Summary Send a dynamic template email via Resend
// @Description Sends an email using a Resend template with custom template_data variables
// @Tags email
// @Accept json
// @Produce json
// @Param body body SendDynamicEmailRequest true "Email payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /emails/send-template [post]
func (h *EmailHandler) SendDynamicTemplateEmail(c *gin.Context) {
	var req SendDynamicEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to (recipient email) is required"})
		return
	}
	if req.TemplateCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template_code is required"})
		return
	}
	if req.TemplateData == nil {
		req.TemplateData = make(map[string]interface{})
	}

	// kartezya_manager is mandatory in template_data
	if _, ok := req.TemplateData["kartezya_manager"]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kartezya_manager is required in template_data"})
		return
	}
	if km, ok := req.TemplateData["kartezya_manager"].(string); !ok || km == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kartezya_manager cannot be empty"})
		return
	}

	subject := req.Subject
	if subject == "" {
		subject = "Bilgilendirme"
	}

	err := h.emailService.SendTemplateEmailWithCC(
		[]string{req.To},
		[]string{"hr@kartezya.com"},
		subject,
		req.TemplateCode,
		req.TemplateData,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Mail başarıyla gönderildi",
		"to":      req.To,
	})
}

type SendDynamicEmailRequest struct {
	To           string                 `json:"to" binding:"required,email"`
	TemplateCode string                 `json:"template_code" binding:"required"`
	Subject      string                 `json:"subject"`
	TemplateData map[string]interface{} `json:"template_data" binding:"required"`
}

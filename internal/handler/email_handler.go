package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

type EmailHandler struct {
	emailService      service.EmailService
	mailConfigService service.MailConfigService
	cfg               *config.Config
}

func NewEmailHandler(emailService service.EmailService, mailConfigService service.MailConfigService, cfg *config.Config) *EmailHandler {
	return &EmailHandler{emailService: emailService, mailConfigService: mailConfigService, cfg: cfg}
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

	subject := req.Subject
	if subject == "" {
		subject = "Bilgilendirme"
	}

	// Resolve CC/BCC from mail config if mail_key provided
	var cc, bcc []string
	if req.MailKey != "" {
		ctx := map[string]string{"TRIGGER_USER": req.To}
		_, cc, bcc, _, _ = h.mailConfigService.ResolveRecipientsWithContext(req.MailKey, ctx)
	}

	err := h.emailService.SendTemplateEmailWithCC(
		[]string{req.To},
		cc,
		bcc,
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
	MailKey      string                 `json:"mail_key"`
	Subject      string                 `json:"subject"`
	TemplateData map[string]interface{} `json:"template_data" binding:"required"`
}

// ListResendTemplates proxies the Resend template list to the frontend
// GET /emails/templates
func (h *EmailHandler) ListResendTemplates(c *gin.Context) {
	if h.cfg.Email.ResendAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RESEND_API_KEY is not configured"})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.resend.com/templates?limit=100", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to build request: %v", err)})
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.Email.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("resend request failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read resend response"})
		return
	}

	if resp.StatusCode >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("resend API error (%d): %s", resp.StatusCode, string(body))})
		return
	}

	// Parse and return only the fields we need
	var resendResp struct {
		Data []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Alias  string `json:"alias"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resendResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse resend response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resendResp.Data,
	})
}

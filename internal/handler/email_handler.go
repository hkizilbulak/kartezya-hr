package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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

// GetTemplateVariables fetches a template from Resend and extracts variable names
// GET /emails/templates/:id/variables
func (h *EmailHandler) GetTemplateVariables(c *gin.Context) {
	if h.cfg.Email.ResendAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RESEND_API_KEY is not configured"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template id is required"})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Helper to fetch template body by template ID
	fetchTemplate := func(tid string) ([]byte, int, error) {
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.resend.com/templates/%s", tid), nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Authorization", "Bearer "+h.cfg.Email.ResendAPIKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, err
		}
		return b, resp.StatusCode, nil
	}

	body, status, err := fetchTemplate(id)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("resend request failed: %v", err)})
		return
	}

	// If not found, attempt to resolve by alias or name and retry
	if status == http.StatusNotFound {
		listReq, lerr := http.NewRequest("GET", "https://api.resend.com/templates?limit=100", nil)
		if lerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to build list request: %v", lerr)})
			return
		}
		listReq.Header.Set("Authorization", "Bearer "+h.cfg.Email.ResendAPIKey)
		listReq.Header.Set("Content-Type", "application/json")
		listResp, lerr := client.Do(listReq)
		if lerr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("resend list request failed: %v", lerr)})
			return
		}
		defer listResp.Body.Close()
		listBody, lerr := io.ReadAll(listResp.Body)
		if lerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read resend list response"})
			return
		}
		if listResp.StatusCode >= 300 {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("resend API error while listing templates (%d): %s", listResp.StatusCode, string(listBody))})
			return
		}

		var listRespStruct struct {
			Data []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Alias string `json:"alias"`
			} `json:"data"`
		}
		if err := json.Unmarshal(listBody, &listRespStruct); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse resend list response"})
			return
		}

		foundID := ""
		for _, t := range listRespStruct.Data {
			if t.ID == id || t.Alias == id || t.Name == id {
				foundID = t.ID
				break
			}
		}
		if foundID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("resend template not found for id/alias/name: %s", id)})
			return
		}

		// Retry fetching the template by the resolved ID
		body, status, err = fetchTemplate(foundID)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("resend request failed: %v", err)})
			return
		}
		if status >= 300 {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("resend API error (%d): %s", status, string(body))})
			return
		}
	} else if status >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("resend API error (%d): %s", status, string(body))})
		return
	}

	// Extract variable names of the form {{ variable_name }} from the template content
	// We'll run a regex on the full response body to be resilient to different response shapes.
	re := regexp.MustCompile(`{{\s*([a-zA-Z0-9_]+)\s*}}`)
	matches := re.FindAllStringSubmatch(string(body), -1)
	varsMap := map[string]bool{}
	for _, m := range matches {
		if len(m) > 1 {
			varsMap[m[1]] = true
		}
	}

	vars := make([]string, 0, len(varsMap))
	for k := range varsMap {
		vars = append(vars, k)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": vars})
}

package handler

import (
	"net/http"

	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settingsService service.SettingsService
}

func NewSettingsHandler(settingsService service.SettingsService) *SettingsHandler {
	return &SettingsHandler{settingsService: settingsService}
}

type SaveKvkkConsentRequest struct {
	Action            string `json:"action" binding:"required,oneof=SUBMIT REMIND_LATER"`
	PhotoConsent      string `json:"photo_consent" binding:"required_if=Action SUBMIT,omitempty,oneof=APPROVED REJECTED"`
	KvkkText          string `json:"kvkk_text" binding:"required_if=Action SUBMIT,omitempty,oneof=READ"`
	PrivacyPolicy     string `json:"privacy_policy" binding:"required_if=Action SUBMIT,omitempty,oneof=READ"`
	AntiBriberyPolicy string `json:"anti_bribery_policy" binding:"required_if=Action SUBMIT,omitempty,oneof=READ"`
}

// GetSettings retrieves the authenticated user's settings
// @Summary Get user settings
// @Description Get settings and preferences for the authenticated user, including KVKK approval status
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/settings [get]
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	settings, err := h.settingsService.GetSettings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve user settings: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// SaveKvkkConsent records user KVKK consent choice
// @Summary Save KVKK consent decision
// @Description Save the user's decision to accept or reject the photo/video sharing consent, and logs the operation
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SaveKvkkConsentRequest true "Consent Decision"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/kvkk [post]
func (h *SettingsHandler) SaveKvkkConsent(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	var req SaveKvkkConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	dto := service.SaveKvkkConsentDto{
		Action:            req.Action,
		PhotoConsent:      req.PhotoConsent,
		KvkkText:          req.KvkkText,
		PrivacyPolicy:     req.PrivacyPolicy,
		AntiBriberyPolicy: req.AntiBriberyPolicy,
	}

	settings, err := h.settingsService.SaveKvkkConsent(userID, dto, clientIP, userAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save consent decisions: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
		"message": "Consent decisions recorded successfully",
	})
}

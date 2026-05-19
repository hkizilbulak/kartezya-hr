package handler

import (
	"log"
	"net/http"
	"strconv"

	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type AuthHandler struct {
	authService  service.AuthService
	emailService service.EmailService
	userRepo     repository.UserRepository
	employeeRepo repository.EmployeeRepository
}

func NewAuthHandler(authService service.AuthService, emailService service.EmailService, userRepo repository.UserRepository, employeeRepo repository.EmployeeRepository) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		emailService: emailService,
		userRepo:     userRepo,
		employeeRepo: employeeRepo,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	NewPassword string `json:"new_password" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type ValidateResetTokenRequest struct {
	Token string `json:"token" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type SendPasswordResetEmailRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Login godoc
// @Summary User login
// @Description Authenticate user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param loginRequest body LoginRequest true "Login credentials"
// @Success 200 {object} APIResponse{data=service.LoginResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "Login successful",
	})
}

// Logout godoc
// @Summary User logout
// @Description Logout the authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APIResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a real implementation, you might want to blacklist the token
	// or store it in a revoked tokens list
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logout successful",
	})
}

// ValidateResetToken godoc
// @Summary Validate password reset token
// @Description Validate if a password reset token is valid and not expired
// @Tags auth
// @Accept json
// @Produce json
// @Param validateResetTokenRequest body ValidateResetTokenRequest true "Token validation request"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /auth/validate-reset-token [post]
func (h *AuthHandler) ValidateResetToken(c *gin.Context) {
	var req ValidateResetTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate token
	user, err := h.emailService.ValidatePasswordResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Invalid or expired password reset token",
		})
		return
	}

	// Verify email matches
	if user.Email != req.Email {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Email does not match the reset token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password reset token is valid",
		"data": gin.H{
			"user_id": user.ID,
			"email":   user.Email,
		},
	})
}

// ResetPassword godoc
// @Summary Reset user password with token
// @Description Reset user password using a valid password reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param resetPasswordRequest body ResetPasswordRequest true "Password reset request"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate token and email
	user, err := h.emailService.ValidatePasswordResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Invalid or expired password reset token",
		})
		return
	}

	if user.Email != req.Email {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Email does not match the reset token",
		})
		return
	}

	// Reset password
	if err := h.emailService.ResetPassword(req.Token, req.NewPassword, h.authService); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password reset successfully. Please login with your new password.",
	})
}

// ChangePassword godoc
// @Summary Change user password (authenticated)
// @Description Change password for the authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param changePasswordRequest body ChangePasswordRequest true "Password change request"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Invalid user ID",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(userIDUint)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	// Verify current password
	if err := h.authService.CheckPassword(user.Password, req.CurrentPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Current password is incorrect",
		})
		return
	}

	// Hash new password
	hashedPassword, err := h.authService.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to process password",
		})
		return
	}

	// Update password
	user.Password = hashedPassword
	user.PasswordResetToken = ""
	user.PasswordResetExpires = nil

	if err := h.userRepo.Update(user, ""); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to update password",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password changed successfully",
	})
}

// SendPasswordResetEmail godoc
// @Summary Send password reset email to a single user
// @Description Send password reset email to a user by user ID
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SendPasswordResetEmailRequest true "Single password reset email request"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /auth/send-password-reset-email [post]
func (h *AuthHandler) SendPasswordResetEmail(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req SendPasswordResetEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get user from database
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	// Get employee info for first/last name
	firstName := ""
	lastName := ""
	employee, empErr := h.employeeRepo.GetByUserID(user.ID)
	if empErr != nil {
		log.Printf("[AUTH] SendPasswordResetEmail - Employee not found for UserID: %d, error: %v", user.ID, empErr)
	} else {
		firstName = employee.FirstName
		lastName = employee.LastName
	}

	// Send password reset email
	if err := h.emailService.SendPasswordResetEmail(user.ID, user.Email, firstName, lastName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to send password reset email: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password reset email sent successfully",
		"data": gin.H{
			"user_id": user.ID,
			"email":   user.Email,
		},
	})
}

// Helper function to get user context from gin context
func getUserContext(c *gin.Context) (uint, string, []string, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, "", nil, false
	}

	email, exists := c.Get("email")
	if !exists {
		return 0, "", nil, false
	}

	roles, exists := c.Get("roles")
	if !exists {
		return 0, "", nil, false
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		return 0, "", nil, false
	}

	emailStr, ok := email.(string)
	if !ok {
		return 0, "", nil, false
	}

	rolesSlice, ok := roles.([]string)
	if !ok {
		return 0, "", nil, false
	}

	return userIDUint, emailStr, rolesSlice, true
}

// Helper function to check if user has admin role
func isAdmin(roles []string) bool {
	for _, role := range roles {
		if role == "ADMIN" {
			return true
		}
	}
	return false
}

// Helper function to parse uint from string parameter
func parseUintParam(c *gin.Context, paramName string) (uint, error) {
	param := c.Param(paramName)
	id, err := strconv.ParseUint(param, 10, 32)
	return uint(id), err
}

// YandexLogin godoc
// @Summary Initiate Yandex OAuth login
// @Description Redirects the user to Yandex OAuth authorization page
// @Tags auth
// @Accept json
// @Produce json
// @Success 302 {string} string "Redirect to Yandex OAuth"
// @Router /auth/yandex/login [get]
func (h *AuthHandler) YandexLogin(c *gin.Context) {
	oauthConfig := h.authService.GetYandexOAuthConfig()

	// Generate a random state for CSRF protection
	// You might want to store the state in a session or cache for verification
	state := "yandex_oauth_state"

	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// YandexCallback godoc
// @Summary Handle Yandex OAuth callback
// @Description Handles the OAuth callback from Yandex and authenticates the user
// @Tags auth
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Yandex"
// @Param state query string false "State parameter for CSRF protection"
// @Success 200 {object} APIResponse{data=service.LoginResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /auth/yandex/callback [get]
func (h *AuthHandler) YandexCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Authorization code is required",
		})
		return
	}

	// You might want to verify the state parameter here for CSRF protection
	// state := c.Query("state")

	response, err := h.authService.HandleYandexCallback(code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Failed to authenticate with Yandex: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "Yandex login successful",
	})
}

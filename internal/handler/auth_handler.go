package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService  service.AuthService
	emailService service.EmailService
	userRepo     repository.UserRepository
}

func NewAuthHandler(authService service.AuthService, emailService service.EmailService, userRepo repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		emailService: emailService,
		userRepo:     userRepo,
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
	UserID    uint   `json:"user_id" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

type SendPasswordResetEmailBatchRequest struct {
	Users []SendPasswordResetEmailRequest `json:"users" binding:"required"`
}

type SendPasswordResetEmailBatchResponse struct {
	TotalCount   int                                     `json:"total_count"`
	SuccessCount int                                     `json:"success_count"`
	FailureCount int                                     `json:"failure_count"`
	Results      []SendPasswordResetEmailBatchResultItem `json:"results"`
}

type SendPasswordResetEmailBatchResultItem struct {
	UserID  uint   `json:"user_id"`
	Email   string `json:"email"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
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

// SendPasswordResetEmailBatch godoc
// @Summary Send password reset emails to multiple users
// @Description Send password reset emails to a batch of users
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param batchRequest body SendPasswordResetEmailBatchRequest true "Batch password reset email request"
// @Success 200 {object} APIResponse{data=SendPasswordResetEmailBatchResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Router /auth/send-password-reset-email-batch [post]
func (h *AuthHandler) SendPasswordResetEmailBatch(c *gin.Context) {
	// Check if user is authenticated
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req SendPasswordResetEmailBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Process each user in the batch
	response := SendPasswordResetEmailBatchResponse{
		TotalCount: len(req.Users),
		Results:    make([]SendPasswordResetEmailBatchResultItem, 0),
	}

	for _, user := range req.Users {
		result := SendPasswordResetEmailBatchResultItem{
			UserID: user.UserID,
			Email:  user.Email,
		}

		// Send password reset email
		err := h.emailService.SendPasswordResetEmailWithUserId(
			user.UserID,
			user.Email,
			user.FirstName,
			user.LastName,
		)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			response.FailureCount++
		} else {
			result.Success = true
			response.SuccessCount++
		}

		response.Results = append(response.Results, result)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "Batch password reset emails processed",
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

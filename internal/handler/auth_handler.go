package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
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

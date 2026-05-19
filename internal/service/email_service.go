package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"encoding/base64"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
)

// EmailService interface for sending emails
type EmailService interface {
	SendWelcomeEmail(userId uint, email string, firstName string, lastName string) error
	SendPasswordResetEmail(userId uint, email string, firstName string, lastName string) error
	GeneratePasswordResetToken(userID uint) (string, error)
	ResetPassword(token string, newPassword string, authService AuthService) error
	ValidatePasswordResetToken(token string) (*domain.User, error)
	SendTemplateEmail(to []string, subject string, templateId string, variables map[string]interface{}) error
	SendReportEmail(to []string, subject string, variables map[string]interface{}, attachment io.Reader, filename string) error
}

type emailService struct {
	config   *config.Config
	userRepo repository.UserRepository
}

// NewEmailService creates a new EmailService instance
func NewEmailService(config *config.Config, userRepo repository.UserRepository) EmailService {
	return &emailService{
		config:   config,
		userRepo: userRepo,
	}
}

// GeneratePasswordResetToken generates a secure random token and stores it
func (s *emailService) GeneratePasswordResetToken(userID uint) (string, error) {
	// Generate a random token (32 bytes = 256 bits)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	// Convert to hex string
	token := hex.EncodeToString(tokenBytes)

	// Set expiration to 24 hours from now
	expiresAt := time.Now().Add(24 * time.Hour)

	// Store token in database
	if err := s.userRepo.UpdatePasswordResetToken(userID, token, &expiresAt); err != nil {
		return "", fmt.Errorf("failed to store reset token: %w", err)
	}

	return token, nil
}

// ValidatePasswordResetToken validates the token and returns the user
func (s *emailService) ValidatePasswordResetToken(token string) (*domain.User, error) {
	user, err := s.userRepo.GetByPasswordResetToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired password reset token")
	}

	// Check if token has expired (additional check, already done in repository)
	if user.PasswordResetExpires != nil && time.Now().After(*user.PasswordResetExpires) {
		return nil, fmt.Errorf("password reset token has expired")
	}

	return user, nil
}

// ResetPassword resets the user password after validating the token
func (s *emailService) ResetPassword(token string, newPassword string, authService AuthService) error {
	// Validate token and get user
	user, err := s.ValidatePasswordResetToken(token)
	if err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := authService.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and clear reset token
	user.Password = hashedPassword
	user.PasswordResetToken = ""
	user.PasswordResetExpires = nil

	if err := s.userRepo.Update(user, "system"); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Clear the reset token from database
	if err := s.userRepo.ClearPasswordResetToken(user.ID); err != nil {
		log.Printf("Warning: failed to clear reset token: %v", err)
	}

	return nil
}

func (s *emailService) SendWelcomeEmail(userId uint, email string, firstName string, lastName string) error {
	// Generate reset token
	resetToken, err := s.GeneratePasswordResetToken(userId)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Build reset URL
	resetURL := fmt.Sprintf("%s/reset-password?token=%s&email=%s", s.config.Email.FrontendURL, resetToken, email)

	// Send email using template
	variables := map[string]interface{}{
		"fullname": fmt.Sprintf("%s %s", firstName, lastName),
		"resetUrl": resetURL,
	}

	if err := s.SendTemplateEmail([]string{email}, "", "welcome-email", variables); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendPasswordResetEmailWithUserId sends a password reset email using a user ID
func (s *emailService) SendPasswordResetEmail(userId uint, email string, firstName string, lastName string) error {
	// Generate reset token
	resetToken, err := s.GeneratePasswordResetToken(userId)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Build reset URL
	resetURL := fmt.Sprintf("%s/reset-password?token=%s&email=%s", s.config.Email.FrontendURL, resetToken, email)

	// Send email using template
	variables := map[string]interface{}{
		"fullname": fmt.Sprintf("%s %s", firstName, lastName),
		"resetUrl": resetURL,
	}

	if err := s.SendTemplateEmail([]string{email}, "", "reset-password", variables); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendTemplateEmail sends an email using a predefined template.
// Currently only supported by the Resend provider.
func (s *emailService) SendTemplateEmail(to []string, subject string, templateId string, variables map[string]interface{}) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if templateId == "" {
		return fmt.Errorf("template_id is required")
	}

	if s.config.Email.Provider == "resend" {
		return s.sendViaResendTemplate(to, subject, templateId, variables, nil, "")
	}

	return fmt.Errorf("template emails are currently only supported with the resend provider")
}

// sendViaResendTemplate sends email using Resend HTTP API with a template
// Supports optional attachment - if attachment is not nil, it will be included
func (s *emailService) sendViaResendTemplate(to []string, subject string, templateId string, variables map[string]interface{}, attachment io.Reader, attachmentFilename string) error {
	if s.config.Email.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}

	if variables == nil {
		variables = make(map[string]interface{})
	}

	payload := map[string]interface{}{
		"from": fmt.Sprintf("%s <%s>", s.config.Email.FromName, s.config.Email.FromEmail),
		"to":   to,
		"template": map[string]interface{}{
			"id":        templateId,
			"variables": variables,
		},
	}

	if subject != "" {
		payload["subject"] = subject
	}

	// Add attachment if provided
	if attachment != nil && attachmentFilename != "" {
		attachmentBytes, err := ioutil.ReadAll(attachment)
		if err != nil {
			return fmt.Errorf("failed to read attachment: %w", err)
		}
		attachmentBase64 := base64.StdEncoding.EncodeToString(attachmentBytes)
		payload["attachments"] = []map[string]interface{}{
			{
				"filename": attachmentFilename,
				"content":  attachmentBase64,
			},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal resend payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.config.Email.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("resend HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("Template email sent via Resend to: %v (template: %s, attachment: %s)\n", to, templateId, attachmentFilename)
	return nil
}

// SendReportEmail sends a report email using "report-mail" template with optional attachment
func (s *emailService) SendReportEmail(to []string, subject string, variables map[string]interface{}, attachment io.Reader, filename string) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if subject == "" {
		return fmt.Errorf("subject is required")
	}

	if s.config.Email.Provider != "resend" {
		return fmt.Errorf("report email is only supported with the resend provider")
	}

	return s.sendViaResendTemplate(to, subject, "report-mail", variables, attachment, filename)
}

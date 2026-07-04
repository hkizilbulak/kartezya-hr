package service

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	SendWelcomeEmail(userId uint, email string, firstName string, lastName string, cc []string) error
	SendPasswordResetEmail(userId uint, email string, firstName string, lastName string, cc []string) error
	GeneratePasswordResetToken(userID uint) (string, error)
	ResetPassword(token string, newPassword string, authService AuthService) error
	ValidatePasswordResetToken(token string) (*domain.User, error)
	SendTemplateEmail(to []string, subject string, templateId string, variables map[string]interface{}) error
	SendTemplateEmailWithCC(to []string, cc []string, bcc []string, subject string, templateId string, variables map[string]interface{}) error
	SendReportEmail(to []string, cc []string, bcc []string, subject string, variables map[string]interface{}, attachment io.Reader, filename string) error
	// Diğer talep bildirimleri
	SendNewRequestEmail(req *domain.OtherRequest) error
	SendRequestCompletedEmail(req *domain.OtherRequest) error
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

// ==================== TALEP BİLDİRİMLERİ ====================

func (s *emailService) SendNewRequestEmail(req *domain.OtherRequest) error {
	if req.Employee == nil {
		return fmt.Errorf("employee info is missing for request email")
	}
	if req.Employee == nil {
		return fmt.Errorf("employee info is missing for request email")
	}

	variables := map[string]interface{}{
		"fullname":    fmt.Sprintf("%s %s", req.Employee.FirstName, req.Employee.LastName),
		"requestType": req.RequestType.Name,
	}

	// Config'deki İK listesini al, boşsa fallback olarak hr@kartezya.com kullan
	recipients := s.config.Email.HREmails
	if len(recipients) == 0 {
		recipients = []string{"hr@kartezya.com"}
	}

	return s.SendTemplateEmail(recipients, "Yeni Talep Oluşturuldu", "new-request-email", variables)
}

func (s *emailService) SendRequestCompletedEmail(req *domain.OtherRequest) error {
	if req.Employee == nil {
		return fmt.Errorf("employee info is missing for request email")
	}
	email := req.Employee.CompanyEmail
	if email == "" {
		email = req.Employee.Email
	}

	variables := map[string]interface{}{
		"fullname":    fmt.Sprintf("%s %s", req.Employee.FirstName, req.Employee.LastName),
		"requestType": req.RequestType.Name,
	}
	return s.SendTemplateEmail([]string{email}, "Talebiniz Tamamlandı", "request-completed-email", variables)
}

// ==================== DİĞER METODLAR ====================

// GeneratePasswordResetToken generates a secure random token and stores it
func (s *emailService) GeneratePasswordResetToken(userID uint) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(24 * time.Hour)

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

	if user.PasswordResetExpires != nil && time.Now().After(*user.PasswordResetExpires) {
		return nil, fmt.Errorf("password reset token has expired")
	}

	return user, nil
}

// ResetPassword resets the user password after validating the token
func (s *emailService) ResetPassword(token string, newPassword string, authService AuthService) error {
	user, err := s.ValidatePasswordResetToken(token)
	if err != nil {
		return err
	}

	hashedPassword, err := authService.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.Password = hashedPassword
	user.PasswordResetToken = ""
	user.PasswordResetExpires = nil

	if err := s.userRepo.Update(user, "system"); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.userRepo.ClearPasswordResetToken(user.ID); err != nil {
		log.Printf("Warning: failed to clear reset token: %v", err)
	}

	return nil
}

func (s *emailService) SendWelcomeEmail(userId uint, email string, firstName string, lastName string, cc []string) error {
	resetToken, err := s.GeneratePasswordResetToken(userId)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s&email=%s", s.config.Email.FrontendURL, resetToken, email)

	variables := map[string]interface{}{
		"fullname": fmt.Sprintf("%s %s", firstName, lastName),
		"resetUrl": resetURL,
	}

	if err := s.sendViaResendTemplate([]string{email}, cc, nil, "Hoş Geldiniz", "welcome-email", variables, nil, ""); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *emailService) SendPasswordResetEmail(userId uint, email string, firstName string, lastName string, cc []string) error {
	resetToken, err := s.GeneratePasswordResetToken(userId)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s&email=%s", s.config.Email.FrontendURL, resetToken, email)

	variables := map[string]interface{}{
		"fullname": fmt.Sprintf("%s %s", firstName, lastName),
		"resetUrl": resetURL,
	}

	if err := s.sendViaResendTemplate([]string{email}, cc, nil, "Şifre Sıfırlama", "reset-password", variables, nil, ""); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *emailService) SendTemplateEmail(to []string, subject string, templateId string, variables map[string]interface{}) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if templateId == "" {
		return fmt.Errorf("template_id is required")
	}

	if s.config.Email.Provider == "resend" {
		return s.sendViaResendTemplate(to, nil, nil, subject, templateId, variables, nil, "")
	}

	return fmt.Errorf("template emails are currently only supported with the resend provider")
}

func (s *emailService) SendTemplateEmailWithCC(to []string, cc []string, bcc []string, subject string, templateId string, variables map[string]interface{}) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if templateId == "" {
		return fmt.Errorf("template_id is required")
	}

	if s.config.Email.Provider == "resend" {
		return s.sendViaResendTemplate(to, cc, bcc, subject, templateId, variables, nil, "")
	}

	return fmt.Errorf("template emails are currently only supported with the resend provider")
}

func (s *emailService) sendViaResendTemplate(to []string, cc []string, bcc []string, subject string, templateId string, variables map[string]interface{}, attachment io.Reader, attachmentFilename string) error {
	if s.config.Email.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}

	if variables == nil {
		variables = make(map[string]interface{})
	}

	fromEmail := fmt.Sprintf("%s <%s>", s.config.Email.FromName, s.config.Email.FromEmail)

	var validTo []string
	for _, emailAddr := range to {
		if emailAddr != "" && emailAddr != "undefined" && emailAddr != "null" {
			validTo = append(validTo, emailAddr)
		}
	}

	if len(validTo) == 0 {
		validTo = []string{s.config.Email.FromEmail}
	}

	payload := map[string]interface{}{
		"from": fromEmail,
		"to":   validTo,
		"template": map[string]interface{}{
			"id":        templateId,
			"variables": variables,
		},
	}

	if len(cc) > 0 {
		payload["cc"] = cc
	}

	if len(bcc) > 0 {
		payload["bcc"] = bcc
	}

	if subject != "" {
		payload["subject"] = subject
	}

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

func (s *emailService) SendReportEmail(to []string, cc []string, bcc []string, subject string, variables map[string]interface{}, attachment io.Reader, filename string) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if subject == "" {
		return fmt.Errorf("subject is required")
	}

	if s.config.Email.Provider != "resend" {
		return fmt.Errorf("report email is only supported with the resend provider")
	}

	return s.sendViaResendTemplate(to, cc, bcc, subject, "report-mail", variables, attachment, filename)
}

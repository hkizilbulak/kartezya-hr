package service

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/mail"
	"net/smtp"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"
	templates "kartezya-hr/internal/email_templates"
	"kartezya-hr/internal/repository"
)

// EmailService interface for sending emails
type EmailService interface {
	SendPasswordResetEmail(user *domain.User, firstName string, lastName string) error
	SendPasswordResetEmailWithUserId(userId uint, email string, firstName string, lastName string) error
	GeneratePasswordResetToken(userID uint) (string, error)
	ResetPassword(token string, newPassword string, authService AuthService) error
	ValidatePasswordResetToken(token string) (*domain.User, error)
	SendCustomEmail(to []string, subject string, htmlBody string) error
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

// SendPasswordResetEmail sends a password reset email to the user
func (s *emailService) SendPasswordResetEmail(user *domain.User, firstName string, lastName string) error {
	// Generate reset token
	resetToken, err := s.GeneratePasswordResetToken(user.ID)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Build reset URL
	resetURL := fmt.Sprintf("%s/reset-password?token=%s&email=%s", s.config.Email.FrontendURL, resetToken, user.Email)

	// Build email content using template
	emailContent := templates.PasswordResetEmailTemplate(firstName, lastName, resetURL)

	// Send email
	if err := s.sendEmail(user.Email, emailContent.Subject, emailContent.Body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendPasswordResetEmailWithUserId sends a password reset email using a user ID
func (s *emailService) SendPasswordResetEmailWithUserId(userId uint, email string, firstName string, lastName string) error {
	// Generate reset token
	resetToken, err := s.GeneratePasswordResetToken(userId)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Build reset URL
	resetURL := fmt.Sprintf("%s/reset-password?token=%s&email=%s", s.config.Email.FrontendURL, resetToken, email)

	// Build email content using template
	emailContent := templates.PasswordResetEmailTemplate(firstName, lastName, resetURL)

	// Send email
	if err := s.sendEmail(email, emailContent.Subject, emailContent.Body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// sendEmail routes to Resend or SMTP depending on EMAIL_PROVIDER config.
func (s *emailService) sendEmail(to, subject, htmlBody string) error {
	if s.config.Email.Provider == "resend" {
		return s.sendViaResend([]string{to}, subject, htmlBody)
	}
	return s.sendSMTPEmail(to, subject, htmlBody)
}

// SendCustomEmail sends a custom HTML email to one or more recipients.
// Routes to Resend HTTP API or SMTP depending on EMAIL_PROVIDER config.
func (s *emailService) SendCustomEmail(to []string, subject string, htmlBody string) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if s.config.Email.Provider == "resend" {
		return s.sendViaResend(to, subject, htmlBody)
	}

	var errs []string
	for _, recipient := range to {
		if err := s.sendSMTPEmail(recipient, subject, htmlBody); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", recipient, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to send to some recipients: %v", errs)
	}
	return nil
}

// sendViaResend sends email using Resend HTTP API (https://resend.com)
// Works on Railway and other cloud platforms that block SMTP ports.
func (s *emailService) sendViaResend(to []string, subject, htmlBody string) error {
	if s.config.Email.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}

	payload := map[string]interface{}{
		"from":    fmt.Sprintf("%s <%s>", s.config.Email.FromName, s.config.Email.FromEmail),
		"to":      to,
		"subject": subject,
		"html":    htmlBody,
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

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("resend HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("Email sent via Resend to: %v\n", to)
	return nil
}

// sendSMTPEmail sends an email using SMTP.
// Port 465 → direct SSL/TLS (implicit TLS)
// Port 587/25 → STARTTLS (smtp.SendMail)
func (s *emailService) sendSMTPEmail(to, subject, htmlBody string) error {
	// Skip sending if SMTP not configured
	if s.config.Email.SMTPUser == "" || s.config.Email.SMTPPassword == "" {
		log.Printf("SMTP not configured. Would send email to: %s\nSubject: %s\n", to, subject)
		return nil
	}

	// Validate email address
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("invalid email address: %w", err)
	}

	// Encode subject with UTF-8
	encodedSubject := mime.QEncoding.Encode("utf-8", subject)

	// Build raw message
	message := fmt.Sprintf(
		"From: %s <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		s.config.Email.FromName,
		s.config.Email.FromEmail,
		to,
		encodedSubject,
		htmlBody,
	)

	addr := fmt.Sprintf("%s:%d", s.config.Email.SMTPHost, s.config.Email.SMTPPort)
	auth := smtp.PlainAuth("", s.config.Email.SMTPUser, s.config.Email.SMTPPassword, s.config.Email.SMTPHost)

	if s.config.Email.SMTPPort == 465 {
		// Port 465: implicit TLS — open TLS connection directly
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.config.Email.SMTPHost,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect via TLS: %w", err)
		}
		defer conn.Close()

		smtpClient, err := smtp.NewClient(conn, s.config.Email.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer smtpClient.Close()

		if err = smtpClient.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
		if err = smtpClient.Mail(s.config.Email.FromEmail); err != nil {
			return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
		}
		if err = smtpClient.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT TO failed: %w", err)
		}
		w, err := smtpClient.Data()
		if err != nil {
			return fmt.Errorf("SMTP DATA failed: %w", err)
		}
		if _, err = fmt.Fprint(w, message); err != nil {
			return fmt.Errorf("failed to write email body: %w", err)
		}
		if err = w.Close(); err != nil {
			return fmt.Errorf("failed to close email writer: %w", err)
		}
		smtpClient.Quit()
	} else {
		// Port 587/25: STARTTLS
		if err := smtp.SendMail(addr, auth, s.config.Email.FromEmail, []string{to}, []byte(message)); err != nil {
			return fmt.Errorf("failed to send SMTP email: %w", err)
		}
	}

	log.Printf("Email sent to: %s\n", to)
	return nil
}

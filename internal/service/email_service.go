package service

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"mime"
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
	if err := s.sendSMTPEmail(user.Email, emailContent.Subject, emailContent.Body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendPasswordResetEmail sends a password reset email to the user
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
	if err := s.sendSMTPEmail(email, emailContent.Subject, emailContent.Body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendCustomEmail sends a custom HTML email to one or more recipients
func (s *emailService) SendCustomEmail(to []string, subject string, htmlBody string) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
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

		client, err := smtp.NewClient(conn, s.config.Email.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
		if err = client.Mail(s.config.Email.FromEmail); err != nil {
			return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
		}
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT TO failed: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("SMTP DATA failed: %w", err)
		}
		if _, err = fmt.Fprint(w, message); err != nil {
			return fmt.Errorf("failed to write email body: %w", err)
		}
		if err = w.Close(); err != nil {
			return fmt.Errorf("failed to close email writer: %w", err)
		}
		client.Quit()
	} else {
		// Port 587/25: STARTTLS
		if err := smtp.SendMail(addr, auth, s.config.Email.FromEmail, []string{to}, []byte(message)); err != nil {
			return fmt.Errorf("failed to send SMTP email: %w", err)
		}
	}

	log.Printf("Email sent to: %s\n", to)
	return nil
}

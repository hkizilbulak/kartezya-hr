package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
)

// AuditService provides common audit logging functionality for all services
type AuditService interface {
	CreateAuditLog(entityName string, entityID uint, action string, oldValue any, newValue any, performedBy string) error
}

type auditService struct {
	auditRepo repository.AuditRepository
}

// NewAuditService creates a new instance of AuditService
func NewAuditService(auditRepo repository.AuditRepository) AuditService {
	return &auditService{
		auditRepo: auditRepo,
	}
}

// CreateAuditLog creates a new audit log entry
func (s *auditService) CreateAuditLog(entityName string, entityID uint, action string, oldValue any, newValue any, performedBy string) error {
	// Convert values to JSON strings
	oldValueJSON := s.valueToJSON(oldValue)
	newValueJSON := s.valueToJSON(newValue)

	// Create audit log entry
	auditLog := &domain.AuditLog{
		EntityName:  entityName,
		EntityID:    entityID,
		Action:      action,
		OldValue:    oldValueJSON,
		NewValue:    newValueJSON,
		CreatedBy:   s.getCurrentUserID(performedBy),
		CreatedDate: time.Now(),
	}

	// Save to repository
	if err := s.auditRepo.Create(auditLog); err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// valueToJSON safely converts a value to JSON string
func (s *auditService) valueToJSON(value any) string {
	if value == nil {
		return ""
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		// If JSON marshaling fails, convert to string representation
		return fmt.Sprintf("%v", value)
	}

	return string(jsonBytes)
}

// getCurrentUserID extracts user ID from performedBy string
// In a real application, this would typically come from JWT token or session context
func (s *auditService) getCurrentUserID(performedBy string) uint {
	if performedBy == "" {
		return 1 // Default system user
	}

	// Try to parse as uint first
	if id, err := strconv.ParseUint(performedBy, 10, 32); err == nil {
		return uint(id)
	}

	// If it's a username or email, you would typically look it up in the database
	// For now, return default system user
	return 1
}

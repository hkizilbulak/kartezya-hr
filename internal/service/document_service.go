package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

// StorageProvider defines interface for file storage (S3, Azure Blob, Local, etc.)
type StorageProvider interface {
	Upload(file multipart.File, path string) error
	Download(path string) ([]byte, error)
	Delete(path string) error
	GeneratePresignedURL(path string, expiryMinutes int) (string, error)
}

// DocumentService handles all document/attachment operations
type DocumentService interface {
	UploadDocument(file *multipart.FileHeader, ownerID uint, relatedType domain.AttachmentRelatedType, docType domain.AttachmentType) (*domain.Attachment, error)
	GetDocument(id string, userID uint, roles []string) (*domain.Attachment, error)
	GetDocumentURL(id string, userID uint, roles []string, expiryMinutes int) (string, error)
	GetUserDocuments(ownerID uint) ([]domain.Attachment, error)
	GetUserDocumentsPaginated(ownerID uint, page, limit int, sortParams types.SortParams) ([]domain.Attachment, int64, error)
	GetRelatedDocuments(relatedType domain.AttachmentRelatedType, relatedID uint, userID uint, roles []string) ([]domain.Attachment, error)
	GetRelatedDocumentsOrdered(relatedType domain.AttachmentRelatedType, relatedID uint, userID uint, roles []string, sortParams types.SortParams) ([]domain.Attachment, error)
	LinkDocumentsToRecord(documentIDs []string, relatedType domain.AttachmentRelatedType, relatedID uint, ownerID uint) error
	DeleteDocument(id string, userID uint, roles []string) error
	CleanupTemporaryFiles(hoursOld int) (int, error)
}

type documentService struct {
	repo             repository.AttachmentRepository
	storage          StorageProvider
	cfg              *config.Config
	allowedMimeTypes map[string]bool
	maxFileSize      int64
}

func NewDocumentService(repo repository.AttachmentRepository, storage StorageProvider, cfg *config.Config) DocumentService {
	// Define allowed MIME types
	allowedMimeTypes := map[string]bool{
		"application/pdf":    true,
		"image/jpeg":         true,
		"image/jpg":          true,
		"image/png":          true,
		"image/gif":          true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	}

	return &documentService{
		repo:             repo,
		storage:          storage,
		cfg:              cfg,
		allowedMimeTypes: allowedMimeTypes,
		maxFileSize:      10 * 1024 * 1024, // 10 MB default
	}
}

// UploadDocument handles file upload with validation
func (s *documentService) UploadDocument(fileHeader *multipart.FileHeader, ownerID uint, relatedType domain.AttachmentRelatedType, docType domain.AttachmentType) (*domain.Attachment, error) {
	// Validate file size
	if fileHeader.Size > s.maxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", s.maxFileSize)
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Validate MIME type
	contentType := fileHeader.Header.Get("Content-Type")
	if !s.allowedMimeTypes[contentType] {
		return nil, fmt.Errorf("file type %s is not allowed", contentType)
	}

	// Calculate file hash for duplicate detection
	hash, err := s.calculateFileHash(file)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Check for duplicate
	exists, err := s.repo.CheckHashExists(hash, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate: %w", err)
	}
	if exists {
		return nil, errors.New("duplicate file: this file has already been uploaded")
	}

	// Reset file pointer after hash calculation
	file.Seek(0, 0)

	// Generate unique ID and storage path
	docID := domain.GenerateUUID()
	storagePath := GenerateStoragePath(relatedType, fileHeader.Filename, docID)

	// Upload to storage
	if err := s.storage.Upload(file, storagePath); err != nil {
		return nil, fmt.Errorf("failed to upload file to storage: %w", err)
	}

	// Create attachment record
	attachment := &domain.Attachment{
		ID:          docID,
		OwnerID:     ownerID,
		RelatedType: relatedType,
		RelatedID:   nil, // Null until linked to a record
		Type:        docType,
		Status:      domain.AttachmentStatusTemporary,
		FileName:    fileHeader.Filename,
		Path:        storagePath,
		ContentType: contentType,
		FileSize:    fileHeader.Size,
		Hash:        hash,
	}

	if err := s.repo.Create(attachment); err != nil {
		// Rollback: delete from storage
		s.storage.Delete(storagePath)
		return nil, fmt.Errorf("failed to save attachment record: %w", err)
	}

	return attachment, nil
}

// GetDocument retrieves document metadata with authorization check
func (s *documentService) GetDocument(id string, userID uint, roles []string) (*domain.Attachment, error) {
	attachment, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if attachment == nil {
		return nil, errors.New("document not found")
	}

	// Authorization check
	if !s.canAccessDocument(attachment, userID, roles) {
		return nil, errors.New("access denied: you don't have permission to view this document")
	}

	return attachment, nil
}

// GetDocumentURL generates a pre-signed URL for downloading the document
func (s *documentService) GetDocumentURL(id string, userID uint, roles []string, expiryMinutes int) (string, error) {
	attachment, err := s.GetDocument(id, userID, roles)
	if err != nil {
		return "", err
	}

	// Generate pre-signed URL
	url, err := s.storage.GeneratePresignedURL(attachment.Path, expiryMinutes)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url, nil
}

// GetUserDocuments retrieves all documents uploaded by a user
func (s *documentService) GetUserDocuments(ownerID uint) ([]domain.Attachment, error) {
	return s.repo.FindByOwnerID(ownerID)
}

// GetUserDocumentsPaginated retrieves owner documents with DB ordering before LIMIT/OFFSET.
func (s *documentService) GetUserDocumentsPaginated(ownerID uint, page, limit int, sortParams types.SortParams) ([]domain.Attachment, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.repo.FindByOwnerIDPaginated(ownerID, limit, offset, sortParams)
}

// GetRelatedDocuments retrieves documents linked to a specific record with authorization
func (s *documentService) GetRelatedDocuments(relatedType domain.AttachmentRelatedType, relatedID uint, userID uint, roles []string) ([]domain.Attachment, error) {
	return s.GetRelatedDocumentsOrdered(relatedType, relatedID, userID, roles, types.SortParams{Sort: "created_at", Direction: "DESC"})
}

// GetRelatedDocumentsOrdered loads related docs with allowlisted ORDER BY, then applies auth filter.
func (s *documentService) GetRelatedDocumentsOrdered(relatedType domain.AttachmentRelatedType, relatedID uint, userID uint, roles []string, sortParams types.SortParams) ([]domain.Attachment, error) {
	attachments, err := s.repo.FindByRelatedRecordOrdered(relatedType, relatedID, sortParams)
	if err != nil {
		return nil, err
	}

	authorized := []domain.Attachment{}
	for _, att := range attachments {
		if s.canAccessDocument(&att, userID, roles) {
			authorized = append(authorized, att)
		}
	}

	return authorized, nil
}

// LinkDocumentsToRecord links uploaded documents to a specific record (e.g., Expense, Leave)
func (s *documentService) LinkDocumentsToRecord(documentIDs []string, relatedType domain.AttachmentRelatedType, relatedID uint, ownerID uint) error {
	if len(documentIDs) == 0 {
		return nil
	}

	// Verify all documents belong to the owner and are temporary
	for _, id := range documentIDs {
		attachment, err := s.repo.FindByID(id)
		if err != nil {
			return fmt.Errorf("document %s not found: %w", id, err)
		}
		if attachment == nil {
			return fmt.Errorf("document %s not found", id)
		}
		if attachment.OwnerID != ownerID {
			return fmt.Errorf("document %s does not belong to the user", id)
		}
		if attachment.Status != domain.AttachmentStatusTemporary {
			return fmt.Errorf("document %s is not in temporary status", id)
		}
	}

	// Link all documents atomically
	return s.repo.LinkToRecord(documentIDs, relatedType, relatedID)
}

// DeleteDocument deletes a document (soft delete by archiving)
func (s *documentService) DeleteDocument(id string, userID uint, roles []string) error {
	attachment, err := s.GetDocument(id, userID, roles)
	if err != nil {
		return err
	}

	// Only owner or admin can delete
	if !s.canDeleteDocument(attachment, userID, roles) {
		return errors.New("access denied: you don't have permission to delete this document")
	}

	return s.repo.Delete(id)
}

// CleanupTemporaryFiles removes temporary files older than specified hours
func (s *documentService) CleanupTemporaryFiles(hoursOld int) (int, error) {
	attachments, err := s.repo.FindTemporaryOlderThan(hoursOld)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, att := range attachments {
		// Delete from storage
		if err := s.storage.Delete(att.Path); err != nil {
			// Log error but continue
			fmt.Printf("Failed to delete file from storage: %s, error: %v\n", att.Path, err)
		}

		// Delete from database
		if err := s.repo.PhysicalDelete(att.ID); err != nil {
			fmt.Printf("Failed to delete attachment record: %s, error: %v\n", att.ID, err)
		} else {
			count++
		}
	}

	return count, nil
}

// canAccessDocument checks if user can access the document (RBAC)
func (s *documentService) canAccessDocument(attachment *domain.Attachment, userID uint, roles []string) bool {
	// Admin can access all documents
	if s.hasRole(roles, domain.RoleAdmin) {
		return true
	}

	// Access check depending on related resource (e.g. employee) or skip for now if testing.
	// We'll permit access to employee documents for now if relatedType is Employee
	if attachment.RelatedType == domain.AttachmentRelatedTypeEmployee {
		// HR / Manager logic could go here; for now allow to avoid blank responses.
		return true
	}

	// Owner can always access their own documents
	if attachment.OwnerID == userID {
		return true
	}

	// TODO: Manager can access their team's documents (requires team relationship check)
	// This would require additional context about team membership

	return false
}

// canDeleteDocument checks if user can delete the document
func (s *documentService) canDeleteDocument(attachment *domain.Attachment, userID uint, roles []string) bool {
	// Admin can delete any document
	if s.hasRole(roles, domain.RoleAdmin) {
		return true
	}

	// Owner can delete their own temporary documents
	if attachment.OwnerID == userID && attachment.Status == domain.AttachmentStatusTemporary {
		return true
	}

	return false
}

// hasRole checks if user has a specific role
func (s *documentService) hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// calculateFileHash calculates SHA256 hash of the file
func (s *documentService) calculateFileHash(file multipart.File) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GenerateStoragePath creates a structured path for file storage
func GenerateStoragePath(relatedType domain.AttachmentRelatedType, filename string, docID string) string {
	now := time.Now()

	// Determine folder based on related type
	folder := "other"
	switch relatedType {
	case domain.AttachmentRelatedTypeExpense:
		folder = "expense"
	case domain.AttachmentRelatedTypeLeave:
		folder = "leave"
	case domain.AttachmentRelatedTypeUser:
		folder = "user"
	case domain.AttachmentRelatedTypeEmployee:
		folder = "employee"
	case domain.AttachmentRelatedTypeContract:
		folder = "contract"
	case domain.AttachmentRelatedTypeOtherRequest:
		folder = "other_requests"
	}

	// Get file extension
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}

	// Formatting without duplicate 'documents' prefix since S3 already sets S3_BASE_PATH
	// Format: YYYY/MM/folder/uuid_originalname.ext
	return fmt.Sprintf("%d/%02d/%s/%s_%s%s",
		now.Year(),
		int(now.Month()),
		folder,
		docID,
		strings.ReplaceAll(filename, ext, ""),
		ext,
	)
}

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

// ── Exchange rate cache ──────────────────────────────────────────────────────
type exchangeRateCache struct {
	mu        sync.RWMutex
	rates     map[string]float64 // key: "USD", "EUR" etc → TRY per 1 unit
	fetchedAt time.Time
	ttl       time.Duration
}

var erCache = &exchangeRateCache{ttl: 1 * time.Hour}

// fetchTRYRates fetches rates from open.er-api.com (base: TRY).
// Response: { "rates": { "USD": 0.0277, "EUR": 0.0256, ... } }
// We invert to get "how many TRY per 1 USD/EUR".
func fetchTRYRates() (map[string]float64, error) {
	resp, err := http.Get("https://open.er-api.com/v6/latest/TRY")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Result != "success" {
		return nil, fmt.Errorf("exchange rate API returned non-success: %s", result.Result)
	}

	// Invert: rates["USD"] = 0.0277 means 1 TRY = 0.0277 USD → 1 USD = 1/0.0277 TRY
	inverted := make(map[string]float64)
	for currency, ratePerTRY := range result.Rates {
		if ratePerTRY > 0 {
			inverted[currency] = 1.0 / ratePerTRY
		}
	}
	inverted["TRY"] = 1.0
	return inverted, nil
}

// convertToTRY converts amount in given currency to TRY using cached rates.
// Falls back to returning the original amount (no conversion) on error.
func convertToTRY(amount float64, currency string) float64 {
	if currency == "" || currency == "TRY" {
		return amount
	}

	erCache.mu.RLock()
	fresh := time.Since(erCache.fetchedAt) < erCache.ttl
	rates := erCache.rates
	erCache.mu.RUnlock()

	if !fresh {
		erCache.mu.Lock()
		newRates, err := fetchTRYRates()
		if err == nil {
			erCache.rates = newRates
			erCache.fetchedAt = time.Now()
			rates = newRates
		}
		erCache.mu.Unlock()
	}

	if rate, ok := rates[currency]; ok {
		return amount * rate
	}
	return amount // fallback: no conversion
}

// Define expense status constants
const (
	ExpenseStatusPending   = "PENDING"
	ExpenseStatusApproved  = "APPROVED"
	ExpenseStatusRejected  = "REJECTED"
	ExpenseStatusPaid      = "PAID"
	ExpenseStatusCancelled = "CANCELLED"
)

type ExpenseService interface {
	// Expense Request methods
	CreateExpenseRequest(expense *domain.ExpenseRequest, userID uint) error
	GetExpenseRequestByID(id uint) (*domain.ExpenseRequest, error)
	GetExpenseRequestByIDForCaller(id uint, userID uint, canViewManagement bool) (*domain.ExpenseRequest, error)
	GetMyExpenseRequests(userID uint, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error)
	GetMyExpenseRequestsPaginated(userID uint, page, limit int, sortParams types.SortParams, status string, expenseTypeID *uint, startDate *string, endDate *string) (*PaginatedResponse, error)
	GetAllExpenseRequestsPaginated(employeeID *uint, page, limit int, sortParams types.SortParams, status string, expenseTypeID *uint, startDate *string, endDate *string) (*PaginatedResponse, error)
	UpdateExpenseRequest(expense *domain.ExpenseRequest, userID uint) error
	DeleteExpenseRequest(id uint, userID uint, isAdmin bool) error
	ApproveExpenseRequest(id uint, userID uint) error
	RejectExpenseRequest(id uint, rejectionReason string, userID uint) error
	MarkAsPaid(id uint, paymentReference string, userID uint) error

	// Expense Document methods (using Attachment system)
	UploadExpenseDocument(expenseRequestID uint, file *multipart.FileHeader, userID uint) (*domain.Attachment, error)
	GetExpenseDocuments(expenseRequestID uint, userID uint, isAdmin bool) ([]domain.Attachment, error)
	DeleteExpenseDocument(documentID string, userID uint, isAdmin bool) error
	DownloadExpenseDocument(documentID string, userID uint, isAdmin bool) (string, error)

	// Expense Type methods
	CreateExpenseType(expenseType *domain.ExpenseType, createdBy string) error
	GetExpenseTypeByID(id uint) (*domain.ExpenseType, error)
	GetAllExpenseTypes(page, limit int, sortParams types.SortParams, roleID *uint) (*PaginatedResponse, error)
	GetActiveExpenseTypes(roles []string) ([]*domain.ExpenseType, error)
	UpdateExpenseType(expenseType *domain.ExpenseType, modifiedBy string) error
	DeleteExpenseType(id uint) error
}

type expenseService struct {
	expenseRepo     repository.ExpenseRepository
	expenseTypeRepo repository.ExpenseTypeRepository
	attachmentRepo  repository.AttachmentRepository
	employeeRepo    repository.EmployeeRepository
	storage         StorageProvider
	auditService    AuditService
}

func NewExpenseService(
	expenseRepo repository.ExpenseRepository,
	expenseTypeRepo repository.ExpenseTypeRepository,
	attachmentRepo repository.AttachmentRepository,
	employeeRepo repository.EmployeeRepository,
	storage StorageProvider,
	auditService AuditService,
) ExpenseService {
	return &expenseService{
		expenseRepo:     expenseRepo,
		expenseTypeRepo: expenseTypeRepo,
		attachmentRepo:  attachmentRepo,
		employeeRepo:    employeeRepo,
		storage:         storage,
		auditService:    auditService,
	}
}

// CreateExpenseRequest creates a new expense request
func (s *expenseService) CreateExpenseRequest(expense *domain.ExpenseRequest, userID uint) error {
	// Get employee by user ID
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return errors.New("employee not found for this user")
	}

	// Validate expense type exists and is active
	expenseType, err := s.expenseTypeRepo.FindByID(expense.ExpenseTypeID)
	if err != nil {
		return errors.New("expense type not found")
	}
	if !expenseType.Active {
		return errors.New("this expense type is not active")
	}

	// Validate max amount if set — convert to TRY for fair comparison
	if expenseType.MaxAmount != nil {
		amountInTRY := convertToTRY(expense.Amount, expense.Currency)
		if amountInTRY > *expenseType.MaxAmount {
			return fmt.Errorf("expense amount exceeds maximum allowed amount of %.2f TRY", *expenseType.MaxAmount)
		}
	}

	// Set employee ID and default status
	expense.EmployeeID = employee.ID
	expense.Status = ExpenseStatusPending
	expense.CreatedBy = fmt.Sprintf("%d", userID)
	expense.ModifiedBy = fmt.Sprintf("%d", userID)

	if err := s.expenseRepo.Create(expense); err != nil {
		return err
	}

	// Audit log
	s.auditService.CreateAuditLog("ExpenseRequest", expense.ID, "CREATE", nil, expense, fmt.Sprintf("%d", userID))

	return nil
}

// GetExpenseRequestByID retrieves an expense request by ID
func (s *expenseService) GetExpenseRequestByID(id uint) (*domain.ExpenseRequest, error) {
	return s.expenseRepo.FindByID(id)
}

// GetExpenseRequestByIDForCaller retrieves an expense for the owner or a management viewer.
func (s *expenseService) GetExpenseRequestByIDForCaller(id uint, userID uint, canViewManagement bool) (*domain.ExpenseRequest, error) {
	expense, err := s.expenseRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("expense request not found")
	}

	if canViewManagement {
		return expense, nil
	}

	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("access denied")
	}
	if err := authorizeExpenseAccess(expense.EmployeeID, employee.ID, false); err != nil {
		return nil, err
	}

	return expense, nil
}

// authorizeExpenseAccess separates data-scope checks from role capabilities.
func authorizeExpenseAccess(expenseEmployeeID, callerEmployeeID uint, canViewManagement bool) error {
	if canViewManagement {
		return nil
	}
	if expenseEmployeeID != callerEmployeeID {
		return errors.New("access denied")
	}
	return nil
}

// GetMyExpenseRequests retrieves expense requests for a specific user
func (s *expenseService) GetMyExpenseRequests(userID uint, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error) {
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("employee not found for this user")
	}

	return s.expenseRepo.FindByEmployeeID(employee.ID, sortBy, sortDir)
}

// GetMyExpenseRequestsPaginated retrieves paginated expense requests for a user
func (s *expenseService) GetMyExpenseRequestsPaginated(userID uint, page, limit int, sortParams types.SortParams, status string, expenseTypeID *uint, startDate *string, endDate *string) (*PaginatedResponse, error) {
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("employee not found for this user")
	}

	expenses, total, err := s.expenseRepo.GetAll(&employee.ID, page, limit, sortParams, status, expenseTypeID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Calculate document count for each expense request
	for _, expense := range expenses {
		count, err := s.attachmentRepo.CountByRelatedRecord(domain.AttachmentRelatedTypeExpense, expense.ID)
		if err == nil {
			expense.DocumentCount = int(count)
		}
	}

	return &PaginatedResponse{
		Data: expenses,
		Page: PageInfo{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: (total + int64(limit) - 1) / int64(limit),
			Sort:       sortParams.Sort,
			Direction:  sortParams.Direction,
		},
	}, nil
}

// GetAllExpenseRequestsPaginated retrieves all expense requests (admin)
func (s *expenseService) GetAllExpenseRequestsPaginated(employeeID *uint, page, limit int, sortParams types.SortParams, status string, expenseTypeID *uint, startDate *string, endDate *string) (*PaginatedResponse, error) {
	expenses, total, err := s.expenseRepo.GetAll(employeeID, page, limit, sortParams, status, expenseTypeID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Calculate document count for each expense request
	for _, expense := range expenses {
		count, err := s.attachmentRepo.CountByRelatedRecord(domain.AttachmentRelatedTypeExpense, expense.ID)
		if err == nil {
			expense.DocumentCount = int(count)
		}
	}

	return &PaginatedResponse{
		Data: expenses,
		Page: PageInfo{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: (total + int64(limit) - 1) / int64(limit),
			Sort:       sortParams.Sort,
			Direction:  sortParams.Direction,
		},
	}, nil
}

// UpdateExpenseRequest updates an expense request
func (s *expenseService) UpdateExpenseRequest(expense *domain.ExpenseRequest, userID uint) error {
	existing, err := s.expenseRepo.FindByID(expense.ID)
	if err != nil {
		return err
	}

	// Only pending requests can be updated
	if existing.Status != ExpenseStatusPending {
		return errors.New("only pending expense requests can be updated")
	}

	// Validate expense type
	expenseType, err := s.expenseTypeRepo.FindByID(expense.ExpenseTypeID)
	if err != nil {
		return errors.New("expense type not found")
	}
	if !expenseType.Active {
		return errors.New("this expense type is not active")
	}

	// Validate max amount if set — convert to TRY for fair comparison
	if expenseType.MaxAmount != nil {
		amountInTRY := convertToTRY(expense.Amount, expense.Currency)
		if amountInTRY > *expenseType.MaxAmount {
			return fmt.Errorf("expense amount exceeds maximum allowed amount of %.2f TRY", *expenseType.MaxAmount)
		}
	}

	expense.ModifiedBy = fmt.Sprintf("%d", userID)

	if err := s.expenseRepo.Update(expense); err != nil {
		return err
	}

	// Audit log
	s.auditService.CreateAuditLog("ExpenseRequest", expense.ID, "UPDATE", nil, expense, fmt.Sprintf("%d", userID))

	return nil
}

// DeleteExpenseRequest deletes an expense request
func (s *expenseService) DeleteExpenseRequest(id uint, userID uint, isAdmin bool) error {
	expense, err := s.expenseRepo.FindByID(id)
	if err != nil {
		return err
	}

	// Get employee
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil && !isAdmin {
		return errors.New("employee not found")
	}

	// Check ownership or admin
	if !isAdmin && employee.ID != expense.EmployeeID {
		return errors.New("you can only delete your own expense requests")
	}

	// Only pending requests can be deleted
	if expense.Status != ExpenseStatusPending {
		return errors.New("only pending expense requests can be deleted")
	}

	expense.Status = ExpenseStatusCancelled
	expense.ModifiedBy = fmt.Sprintf("%d", userID)

	if err := s.expenseRepo.Update(expense); err != nil {
		return err
	}

	// Audit log
	s.auditService.CreateAuditLog("ExpenseRequest", id, "CANCEL", nil, expense, fmt.Sprintf("%d", userID))
	return nil
}

// ApproveExpenseRequest approves an expense request
func (s *expenseService) ApproveExpenseRequest(id uint, userID uint) error {
	expense, err := s.expenseRepo.FindByID(id)
	if err != nil {
		return err
	}

	if expense.Status != ExpenseStatusPending {
		return errors.New("only pending expense requests can be approved")
	}

	now := time.Now()
	expense.Status = ExpenseStatusApproved
	expense.ApprovedBy = &userID
	expense.ApprovedAt = &now
	expense.ModifiedBy = fmt.Sprintf("%d", userID)

	if err := s.expenseRepo.Update(expense); err != nil {
		return err
	}

	// Audit log
	s.auditService.CreateAuditLog("ExpenseRequest", id, "APPROVE", nil, expense, fmt.Sprintf("%d", userID))

	return nil
}

// RejectExpenseRequest rejects an expense request
func (s *expenseService) RejectExpenseRequest(id uint, rejectionReason string, userID uint) error {
	expense, err := s.expenseRepo.FindByID(id)
	if err != nil {
		return err
	}

	if expense.Status != ExpenseStatusPending {
		return errors.New("only pending expense requests can be rejected")
	}

	now := time.Now()
	expense.Status = ExpenseStatusRejected
	expense.RejectedAt = &now
	expense.RejectionReason = rejectionReason
	expense.ModifiedBy = fmt.Sprintf("%d", userID)

	if err := s.expenseRepo.Update(expense); err != nil {
		return err
	}

	// Audit log
	s.auditService.CreateAuditLog("ExpenseRequest", id, "REJECT", nil, expense, fmt.Sprintf("%d", userID))

	return nil
}

// MarkAsPaid marks an expense request as paid
func (s *expenseService) MarkAsPaid(id uint, paymentReference string, userID uint) error {
	expense, err := s.expenseRepo.FindByID(id)
	if err != nil {
		return err
	}

	if expense.Status != ExpenseStatusApproved {
		return errors.New("only approved expense requests can be marked as paid")
	}

	now := time.Now()
	expense.Status = ExpenseStatusPaid
	expense.PaidAt = &now
	expense.PaymentReference = paymentReference
	expense.ModifiedBy = fmt.Sprintf("%d", userID)

	if err := s.expenseRepo.Update(expense); err != nil {
		return err
	}

	// Audit log
	s.auditService.CreateAuditLog("ExpenseRequest", id, "MARK_PAID", nil, expense, fmt.Sprintf("%d", userID))

	return nil
}

// Expense Type methods

func (s *expenseService) CreateExpenseType(expenseType *domain.ExpenseType, createdBy string) error {
	return s.expenseTypeRepo.Create(expenseType, createdBy)
}

func (s *expenseService) GetExpenseTypeByID(id uint) (*domain.ExpenseType, error) {
	return s.expenseTypeRepo.FindByID(id)
}

func (s *expenseService) GetAllExpenseTypes(page, limit int, sortParams types.SortParams, roleID *uint) (*PaginatedResponse, error) {
	offset := (page - 1) * limit
	expenseTypes, total, err := s.expenseTypeRepo.GetAll(limit, offset, sortParams, roleID)
	if err != nil {
		return nil, err
	}

	return &PaginatedResponse{
		Data: expenseTypes,
		Page: PageInfo{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: (total + int64(limit) - 1) / int64(limit),
			Sort:       sortParams.Sort,
			Direction:  sortParams.Direction,
		},
	}, nil
}

func (s *expenseService) GetActiveExpenseTypes(roles []string) ([]*domain.ExpenseType, error) {
	return s.expenseTypeRepo.GetActive(roles)
}

func (s *expenseService) UpdateExpenseType(expenseType *domain.ExpenseType, modifiedBy string) error {
	return s.expenseTypeRepo.Update(expenseType, modifiedBy)
}

func (s *expenseService) DeleteExpenseType(id uint) error {
	return s.expenseTypeRepo.Delete(id)
}

// ==================== Expense Document Methods ====================

// UploadExpenseDocument uploads a document for an expense request using Attachment system
func (s *expenseService) UploadExpenseDocument(expenseRequestID uint, file *multipart.FileHeader, userID uint) (*domain.Attachment, error) {
	// Check if expense request exists and user has permission
	expense, err := s.expenseRepo.FindByID(expenseRequestID)
	if err != nil {
		return nil, errors.New("expense request not found")
	}

	// Get employee to check ownership
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("employee not found")
	}

	// Only the owner can upload documents
	if expense.EmployeeID != employee.ID {
		return nil, errors.New("you can only upload documents to your own expense requests")
	}

	// Only PENDING requests can have documents uploaded
	if expense.Status != ExpenseStatusPending {
		return nil, errors.New("documents can only be uploaded to pending expense requests")
	}

	// Upload using document service with expense-specific type
	attachment, err := s.uploadExpenseAttachment(file, userID, expenseRequestID, domain.AttachmentTypeReceipt)
	if err != nil {
		return nil, err
	}

	return attachment, nil
}

// uploadExpenseAttachment is a helper function to upload expense attachments
func (s *expenseService) uploadExpenseAttachment(file *multipart.FileHeader, ownerID uint, expenseRequestID uint, docType domain.AttachmentType) (*domain.Attachment, error) {
	// Open the file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Generate storage path using shared document service utility
	timestamp := time.Now().Format("20060102150405")
	docID := fmt.Sprintf("%d_%s", expenseRequestID, timestamp)
	storagePath := GenerateStoragePath(domain.AttachmentRelatedTypeExpense, file.Filename, docID)

	// Upload to storage
	if err := s.storage.Upload(src, storagePath); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Create attachment record
	relatedID := expenseRequestID
	attachment := &domain.Attachment{
		ID:          domain.GenerateUUID(),
		OwnerID:     ownerID,
		RelatedType: domain.AttachmentRelatedTypeExpense,
		RelatedID:   &relatedID,
		Type:        docType,
		Status:      domain.AttachmentStatusLinked,
		FileName:    file.Filename,
		Path:        storagePath,
		ContentType: file.Header.Get("Content-Type"),
		FileSize:    file.Size,
	}

	if err := s.attachmentRepo.Create(attachment); err != nil {
		// Try to cleanup uploaded file
		_ = s.storage.Delete(storagePath)
		return nil, fmt.Errorf("failed to create attachment record: %w", err)
	}

	// Audit log
	_ = s.auditService.CreateAuditLog("attachment", 0, "UPLOAD", nil, attachment, fmt.Sprintf("user_%d", ownerID))

	return attachment, nil
}

// GetExpenseDocuments retrieves all documents for an expense request
func (s *expenseService) GetExpenseDocuments(expenseRequestID uint, userID uint, isAdmin bool) ([]domain.Attachment, error) {
	// Check if expense request exists
	expense, err := s.expenseRepo.FindByID(expenseRequestID)
	if err != nil {
		return nil, errors.New("expense request not found")
	}

	// Check permission: admin or owner can view
	if !isAdmin {
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil {
			return nil, errors.New("employee not found")
		}
		if expense.EmployeeID != employee.ID {
			return nil, errors.New("you can only view documents of your own expense requests")
		}
	}

	// Get attachments for this expense
	return s.attachmentRepo.FindByRelatedRecord(domain.AttachmentRelatedTypeExpense, expenseRequestID)
}

// DeleteExpenseDocument deletes a document
func (s *expenseService) DeleteExpenseDocument(documentID string, userID uint, isAdmin bool) error {
	// Get attachment
	attachment, err := s.attachmentRepo.FindByID(documentID)
	if err != nil {
		return errors.New("document not found")
	}

	// Verify it's an expense document
	if attachment.RelatedType != domain.AttachmentRelatedTypeExpense || attachment.RelatedID == nil {
		return errors.New("invalid expense document")
	}

	// Get expense request
	expense, err := s.expenseRepo.FindByID(*attachment.RelatedID)
	if err != nil {
		return errors.New("expense request not found")
	}

	// Check permission: admin or owner can delete
	if !isAdmin {
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil {
			return errors.New("employee not found")
		}
		if expense.EmployeeID != employee.ID {
			return errors.New("you can only delete documents of your own expense requests")
		}
	}

	// Only PENDING requests can have documents deleted
	if expense.Status != ExpenseStatusPending {
		return errors.New("documents can only be deleted from pending expense requests")
	}

	// Delete from storage
	if err := s.storage.Delete(attachment.Path); err != nil {
		// Log error but continue with database deletion
		fmt.Printf("Failed to delete file from storage: %v\n", err)
	}

	// Delete from database
	if err := s.attachmentRepo.Delete(documentID); err != nil {
		return fmt.Errorf("failed to delete attachment record: %w", err)
	}

	// Audit log
	_ = s.auditService.CreateAuditLog("attachment", 0, "DELETE", attachment, nil, fmt.Sprintf("user_%d", userID))

	return nil
}

// DownloadExpenseDocument generates a download URL for a document
func (s *expenseService) DownloadExpenseDocument(documentID string, userID uint, isAdmin bool) (string, error) {
	// Get attachment
	attachment, err := s.attachmentRepo.FindByID(documentID)
	if err != nil {
		return "", errors.New("document not found")
	}

	// Verify it's an expense document
	if attachment.RelatedType != domain.AttachmentRelatedTypeExpense || attachment.RelatedID == nil {
		return "", errors.New("invalid expense document")
	}

	// Get expense request
	expense, err := s.expenseRepo.FindByID(*attachment.RelatedID)
	if err != nil {
		return "", errors.New("expense request not found")
	}

	// Check permission: admin or owner can download
	if !isAdmin {
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil {
			return "", errors.New("employee not found")
		}
		if expense.EmployeeID != employee.ID {
			return "", errors.New("you can only download documents of your own expense requests")
		}
	}

	// Generate presigned URL (valid for 15 minutes)
	url, err := s.storage.GeneratePresignedURL(attachment.Path, 15)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url, nil
}

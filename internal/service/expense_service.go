package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

// Define expense status constants
const (
	ExpenseStatusPending  = "PENDING"
	ExpenseStatusApproved = "APPROVED"
	ExpenseStatusRejected = "REJECTED"
	ExpenseStatusPaid     = "PAID"
)

type ExpenseService interface {
	// Expense Request methods
	CreateExpenseRequest(expense *domain.ExpenseRequest, userID uint) error
	GetExpenseRequestByID(id uint) (*domain.ExpenseRequest, error)
	GetMyExpenseRequests(userID uint, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error)
	GetMyExpenseRequestsPaginated(userID uint, page, limit int, sortParams types.SortParams, status string) (*PaginatedResponse, error)
	GetAllExpenseRequestsPaginated(employeeID *uint, page, limit int, sortParams types.SortParams, status string) (*PaginatedResponse, error)
	UpdateExpenseRequest(expense *domain.ExpenseRequest, userID uint) error
	DeleteExpenseRequest(id uint, userID uint, isAdmin bool) error
	ApproveExpenseRequest(id uint, userID uint) error
	RejectExpenseRequest(id uint, rejectionReason string, userID uint) error
	MarkAsPaid(id uint, paymentReference string, userID uint) error

	// Expense Type methods
	CreateExpenseType(expenseType *domain.ExpenseType, createdBy string) error
	GetExpenseTypeByID(id uint) (*domain.ExpenseType, error)
	GetAllExpenseTypes(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	GetActiveExpenseTypes() ([]*domain.ExpenseType, error)
	UpdateExpenseType(expenseType *domain.ExpenseType, modifiedBy string) error
	DeleteExpenseType(id uint) error
}

type expenseService struct {
	expenseRepo     repository.ExpenseRepository
	expenseTypeRepo repository.ExpenseTypeRepository
	employeeRepo    repository.EmployeeRepository
	auditService    AuditService
}

func NewExpenseService(
	expenseRepo repository.ExpenseRepository,
	expenseTypeRepo repository.ExpenseTypeRepository,
	employeeRepo repository.EmployeeRepository,
	auditService AuditService,
) ExpenseService {
	return &expenseService{
		expenseRepo:     expenseRepo,
		expenseTypeRepo: expenseTypeRepo,
		employeeRepo:    employeeRepo,
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

	// Validate max amount if set
	if expenseType.MaxAmount != nil && expense.Amount > *expenseType.MaxAmount {
		return fmt.Errorf("expense amount exceeds maximum allowed amount of %.2f %s", *expenseType.MaxAmount, expense.Currency)
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

// GetMyExpenseRequests retrieves expense requests for a specific user
func (s *expenseService) GetMyExpenseRequests(userID uint, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error) {
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("employee not found for this user")
	}

	return s.expenseRepo.FindByEmployeeID(employee.ID, sortBy, sortDir)
}

// GetMyExpenseRequestsPaginated retrieves paginated expense requests for a user
func (s *expenseService) GetMyExpenseRequestsPaginated(userID uint, page, limit int, sortParams types.SortParams, status string) (*PaginatedResponse, error) {
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("employee not found for this user")
	}

	expenses, total, err := s.expenseRepo.GetAll(&employee.ID, page, limit, sortParams, status)
	if err != nil {
		return nil, err
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
func (s *expenseService) GetAllExpenseRequestsPaginated(employeeID *uint, page, limit int, sortParams types.SortParams, status string) (*PaginatedResponse, error) {
	expenses, total, err := s.expenseRepo.GetAll(employeeID, page, limit, sortParams, status)
	if err != nil {
		return nil, err
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

	// Validate max amount if set
	if expenseType.MaxAmount != nil && expense.Amount > *expenseType.MaxAmount {
		return fmt.Errorf("expense amount exceeds maximum allowed amount of %.2f %s", *expenseType.MaxAmount, expense.Currency)
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

	if err := s.expenseRepo.Delete(id); err != nil {
		return err
	}

	// Audit log
	s.auditService.CreateAuditLog("ExpenseRequest", id, "DELETE", expense, nil, fmt.Sprintf("%d", userID))

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

func (s *expenseService) GetAllExpenseTypes(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
	offset := (page - 1) * limit
	expenseTypes, total, err := s.expenseTypeRepo.GetAll(limit, offset, sortParams)
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

func (s *expenseService) GetActiveExpenseTypes() ([]*domain.ExpenseType, error) {
	return s.expenseTypeRepo.GetActive()
}

func (s *expenseService) UpdateExpenseType(expenseType *domain.ExpenseType, modifiedBy string) error {
	return s.expenseTypeRepo.Update(expenseType, modifiedBy)
}

func (s *expenseService) DeleteExpenseType(id uint) error {
	return s.expenseTypeRepo.Delete(id)
}

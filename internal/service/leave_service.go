package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

// Define leave status constants
const (
	LeaveStatusPending   = "PENDING"
	LeaveStatusApproved  = "APPROVED"
	LeaveStatusRejected  = "REJECTED"
	LeaveStatusCancelled = "CANCELLED"
)

// Define leave type constants
const (
	LeaveTypeAnnual = "annual"
	LeaveTypeSick   = "sick"
)

type LeaveService interface {
	// Leave methods
	CreateLeave(leave *domain.LeaveRequest, createdBy string, isAdmin bool) error
	GetLeaveByID(id uint) (*domain.LeaveRequest, error)
	GetLeaveByIDFormatted(id uint) (*types.AdminLeaveRequestResponse, error)
	UpdateLeave(leave *domain.LeaveRequest, modifiedBy string) error
	DeleteLeave(id uint, deletedBy string) error
	GetLeavesByEmployeeID(employeeID uint, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error)
	GetLeavesByUserID(userID uint, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error)
	GetMyLeaveRequestsPaginated(userID uint, page, limit int, sortParams types.SortParams, status string) (*PaginatedResponse, error)
	GetAllLeaveRequestsPaginated(page, limit int, sortParams types.SortParams, status string) (*PaginatedResponse, error)
	GetLeavesByStatus(status string, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error)
	GetLeavesByDateRange(startDate, endDate string) ([]*domain.LeaveRequest, error)
	ApproveLeave(id uint, approverID uint, approvedBy string) error
	RejectLeave(id uint, rejectionReason string, rejectedBy string, rejectorID uint) error
	CancelLeave(id uint, cancelReason string, cancelledBy string, userID uint, isAdmin bool) error

	// Leave Balance methods
	ValidateLeaveBalance(employeeID, leaveTypeID uint, requestedDays float64, isAdmin bool) error
	DeductLeaveBalance(employeeID, leaveTypeID uint, requestedDays float64, modifiedBy string) error
	GetMyLeaveBalances(userID uint, page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)

	// Leave Type methods
	CreateLeaveType(leaveType *domain.LeaveType, createdBy string) error
	GetLeaveTypeByID(id uint) (*types.LeaveTypeResponse, error)
	GetAllLeaveTypes(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	GetLeaveTypesLookup() ([]types.LeaveTypeLookup, error)
	UpdateLeaveType(leaveType *domain.LeaveType, modifiedBy string) error
	DeleteLeaveType(id uint, deletedBy string) error
}

type leaveService struct {
	leaveRepo        repository.LeaveRepository
	leaveTypeRepo    repository.LeaveTypeRepository
	leaveBalanceRepo repository.LeaveBalanceRepository
	employeeRepo     repository.EmployeeRepository
	auditService     AuditService
}

func NewLeaveService(
	leaveRepo repository.LeaveRepository,
	leaveTypeRepo repository.LeaveTypeRepository,
	leaveBalanceRepo repository.LeaveBalanceRepository,
	employeeRepo repository.EmployeeRepository,
	auditService AuditService,
) LeaveService {
	return &leaveService{
		leaveRepo:        leaveRepo,
		leaveTypeRepo:    leaveTypeRepo,
		leaveBalanceRepo: leaveBalanceRepo,
		employeeRepo:     employeeRepo,
		auditService:     auditService,
	}
}

// CreateLeaveWithValidation creates a leave request with balance validation for employees
func (s *leaveService) CreateLeave(leave *domain.LeaveRequest, createdBy string, isAdmin bool) error {
	if leave == nil {
		return errors.New("leave cannot be nil")
	}

	// Validate required fields
	if leave.EmployeeID == 0 {
		return errors.New("employee_id is required")
	}
	if leave.LeaveTypeID == 0 {
		return errors.New("leave_type_id is required")
	}
	if leave.StartDate.IsZero() {
		return errors.New("start_date is required")
	}
	if leave.EndDate.IsZero() {
		return errors.New("end_date is required")
	}
	if leave.RequestedDays <= 0 {
		return errors.New("days must be greater than 0")
	}

	// Validate leave balance for non-admin users
	if err := s.ValidateLeaveBalance(leave.EmployeeID, leave.LeaveTypeID, leave.RequestedDays, isAdmin); err != nil {
		return err
	}

	// Set default values
	if leave.Status == "" {
		leave.Status = LeaveStatusPending
	}
	leave.CreatedAt = time.Now()
	leave.UpdatedAt = time.Now()

	// Create the leave
	if err := s.leaveRepo.Create(leave); err != nil {
		return fmt.Errorf("failed to create leave: %w", err)
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("Leave", leave.ID, domain.AuditActionCreate, nil, leave, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) GetLeaveByID(id uint) (*domain.LeaveRequest, error) {
	return s.leaveRepo.GetByID(id)
}

func (s *leaveService) GetLeaveByIDFormatted(id uint) (*types.AdminLeaveRequestResponse, error) {
	leave, err := s.leaveRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Convert to AdminLeaveRequestResponse format
	return &types.AdminLeaveRequestResponse{
		ID:         leave.ID,
		CreatedAt:  leave.CreatedAt,
		UpdatedAt:  leave.UpdatedAt,
		Deleted:    leave.Deleted,
		CreatedBy:  leave.CreatedBy,
		ModifiedBy: leave.ModifiedBy,
		Employee: types.EmployeeLookup{
			ID:        leave.Employee.ID,
			FirstName: leave.Employee.FirstName,
			LastName:  leave.Employee.LastName,
		},
		LeaveType: types.LeaveTypeLookup{
			ID:   leave.LeaveType.ID,
			Name: leave.LeaveType.Name,
		},
		StartDate:       leave.StartDate,
		EndDate:         leave.EndDate,
		RequestedDays:   leave.RequestedDays,
		Reason:          leave.Reason,
		Status:          leave.Status,
		IsPaid:          leave.IsPaid,
		ApprovedBy:      leave.ApprovedBy,
		ApprovedAt:      leave.ApprovedAt,
		RejectedAt:      leave.RejectedAt,
		RejectionReason: leave.RejectionReason,
		CancelReason:    leave.CancelReason,
		CancelledAt:     leave.CancelledAt,
		Comments:        leave.Comments,
	}, nil
}

func (s *leaveService) GetAllLeaves(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Set defaults for sorting
	if sortParams.Sort == "" {
		sortParams.Sort = "id"
	}
	if sortParams.Direction == "" {
		sortParams.Direction = "ASC"
	}

	offset := (page - 1) * limit
	leaves, total, err := s.leaveRepo.GetAll(limit, offset, sortParams)
	if err != nil {
		return nil, err
	}

	return &PaginatedResponse{
		Data: leaves,
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

func (s *leaveService) UpdateLeave(leave *domain.LeaveRequest, modifiedBy string) error {
	if leave == nil {
		return errors.New("leave cannot be nil")
	}

	if leave.ID == 0 {
		return errors.New("leave ID is required for update")
	}

	// Get existing leave for audit and status validation
	existingLeave, err := s.leaveRepo.GetByID(leave.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing leave: %w", err)
	}

	// Check if the leave request can be updated (only PENDING requests)
	if existingLeave.Status != LeaveStatusPending {
		return fmt.Errorf("only pending leave requests can be updated, current status: %s", existingLeave.Status)
	}

	leave.UpdatedAt = time.Now()

	// Update the leave
	if err := s.leaveRepo.Update(leave); err != nil {
		return fmt.Errorf("failed to update leave: %w", err)
	}

	// Get updated leave for audit
	updatedLeave, _ := s.leaveRepo.GetByID(leave.ID)

	// Audit the update
	if err := s.auditService.CreateAuditLog("Leave", leave.ID, domain.AuditActionUpdate, existingLeave, updatedLeave, modifiedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) DeleteLeave(id uint, deletedBy string) error {
	// Get existing leave for audit
	existingLeave, err := s.leaveRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete the leave
	if err := s.leaveRepo.Delete(id); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("Leave", id, domain.AuditActionDelete, existingLeave, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) GetLeavesByEmployeeID(employeeID uint, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error) {
	return s.leaveRepo.GetByEmployeeID(employeeID, sortBy, sortDir)
}

func (s *leaveService) GetLeavesByUserID(userID uint, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error) {
	// First, get the employee record using the user ID
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find employee for user ID %d: %w", userID, err)
	}

	// Then get the leave requests using the employee ID
	return s.leaveRepo.GetByEmployeeID(employee.ID, sortBy, sortDir)
}

func (s *leaveService) GetMyLeaveRequestsPaginated(userID uint, page, limit int, sortParams types.SortParams, status string) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Set defaults for sorting
	if sortParams.Sort == "" {
		sortParams.Sort = "created_at"
	}
	if sortParams.Direction == "" {
		sortParams.Direction = "DESC"
	}

	// First, get the employee record using the user ID
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find employee for user ID %d: %w", userID, err)
	}

	offset := (page - 1) * limit
	var leaves []*domain.LeaveRequest
	var total int64

	if status != "" {
		leaves, total, err = s.leaveRepo.GetByEmployeeIDWithLeaveTypeAndStatus(employee.ID, limit, offset, sortParams, status)
	} else {
		leaves, total, err = s.leaveRepo.GetByEmployeeIDWithLeaveType(employee.ID, limit, offset, sortParams)
	}

	if err != nil {
		return nil, err
	}

	// Convert to MyLeaveRequestResponse format
	var responseData []*types.MyLeaveRequestResponse
	for _, leave := range leaves {
		responseData = append(responseData, &types.MyLeaveRequestResponse{
			ID:        leave.ID,
			CreatedAt: leave.CreatedAt,
			UpdatedAt: leave.UpdatedAt,
			LeaveType: types.LeaveTypeLookup{
				ID:   leave.LeaveType.ID,
				Name: leave.LeaveType.Name,
			},
			StartDate:       leave.StartDate,
			EndDate:         leave.EndDate,
			RequestedDays:   leave.RequestedDays,
			Reason:          leave.Reason,
			Status:          leave.Status,
			IsPaid:          leave.IsPaid,
			ApprovedAt:      leave.ApprovedAt,
			RejectedAt:      leave.RejectedAt,
			RejectionReason: leave.RejectionReason,
			CancelReason:    leave.CancelReason,
			CancelledAt:     leave.CancelledAt,
			Comments:        leave.Comments,
		})
	}

	return &PaginatedResponse{
		Data: responseData,
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

func (s *leaveService) GetAllLeaveRequestsPaginated(page, limit int, sortParams types.SortParams, status string) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Set defaults for sorting
	if sortParams.Sort == "" {
		sortParams.Sort = "created_at"
	}
	if sortParams.Direction == "" {
		sortParams.Direction = "DESC"
	}

	offset := (page - 1) * limit
	leaves, total, err := s.leaveRepo.GetAllWithStatus(limit, offset, sortParams, status)
	if err != nil {
		return nil, err
	}

	// Convert to AdminLeaveRequestResponse format
	var responseData []*types.AdminLeaveRequestResponse
	for _, leave := range leaves {
		responseData = append(responseData, &types.AdminLeaveRequestResponse{
			ID:         leave.ID,
			CreatedAt:  leave.CreatedAt,
			UpdatedAt:  leave.UpdatedAt,
			Deleted:    leave.Deleted,
			CreatedBy:  leave.CreatedBy,
			ModifiedBy: leave.ModifiedBy,
			Employee: types.EmployeeLookup{
				ID:        leave.Employee.ID,
				FirstName: leave.Employee.FirstName,
				LastName:  leave.Employee.LastName,
			},
			LeaveType: types.LeaveTypeLookup{
				ID:   leave.LeaveType.ID,
				Name: leave.LeaveType.Name,
			},
			StartDate:       leave.StartDate,
			EndDate:         leave.EndDate,
			RequestedDays:   leave.RequestedDays,
			Reason:          leave.Reason,
			Status:          leave.Status,
			IsPaid:          leave.IsPaid,
			ApprovedBy:      leave.ApprovedBy,
			ApprovedAt:      leave.ApprovedAt,
			RejectedAt:      leave.RejectedAt,
			RejectionReason: leave.RejectionReason,
			CancelReason:    leave.CancelReason,
			CancelledAt:     leave.CancelledAt,
			Comments:        leave.Comments,
		})
	}

	return &PaginatedResponse{
		Data: responseData,
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

func (s *leaveService) GetLeavesByStatus(status string, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error) {
	return s.leaveRepo.GetByStatus(status, sortBy, sortDir)
}

func (s *leaveService) GetLeavesByDateRange(startDate, endDate string) ([]*domain.LeaveRequest, error) {
	return s.leaveRepo.GetByDateRange(startDate, endDate)
}

func (s *leaveService) ApproveLeave(id uint, approverID uint, approvedBy string) error {
	// Get existing leave for audit and balance deduction
	existingLeave, err := s.leaveRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Deduct leave balance when approving the request
	if err := s.DeductLeaveBalance(existingLeave.EmployeeID, existingLeave.LeaveTypeID, existingLeave.RequestedDays, approvedBy); err != nil {
		return fmt.Errorf("failed to deduct leave balance: %w", err)
	}

	leave := *existingLeave // Create a copy
	leave.Status = LeaveStatusApproved
	leave.ApprovedBy = &approverID
	now := time.Now()
	leave.ApprovedAt = &now
	leave.ModifiedBy = approvedBy

	// Update the leave
	if err := s.leaveRepo.Update(&leave); err != nil {
		return err
	}

	// Get updated leave for audit
	updatedLeave, _ := s.leaveRepo.GetByID(id)

	// Audit the approval
	if err := s.auditService.CreateAuditLog("Leave", id, domain.AuditActionUpdate, existingLeave, updatedLeave, approvedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) RejectLeave(id uint, rejectionReason string, rejectedBy string, rejectorID uint) error {
	// Get existing leave for audit
	existingLeave, err := s.leaveRepo.GetByID(id)
	if err != nil {
		return err
	}

	leave := *existingLeave // Create a copy
	leave.Status = LeaveStatusRejected
	leave.RejectionReason = rejectionReason
	leave.ApprovedBy = &rejectorID // Set the rejector's user ID in ApprovedBy field
	now := time.Now()
	leave.RejectedAt = &now // Set the rejection timestamp
	leave.ModifiedBy = rejectedBy

	// Update the leave
	if err := s.leaveRepo.Update(&leave); err != nil {
		return err
	}

	// Get updated leave for audit
	updatedLeave, _ := s.leaveRepo.GetByID(id)

	// Audit the rejection
	if err := s.auditService.CreateAuditLog("Leave", id, domain.AuditActionUpdate, existingLeave, updatedLeave, rejectedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) CancelLeave(id uint, cancelReason string, cancelledBy string, userID uint, isAdmin bool) error {
	// Get existing leave for validation and audit
	existingLeave, err := s.leaveRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check if the leave request can be cancelled (only PENDING requests)
	if existingLeave.Status != LeaveStatusPending {
		return fmt.Errorf("only pending leave requests can be cancelled, current status: %s", existingLeave.Status)
	}

	// Authorization check: employees can only cancel their own requests, admins can cancel any
	if !isAdmin {
		// Get employee ID from user ID
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil {
			return fmt.Errorf("failed to find employee for user: %w", err)
		}

		if existingLeave.EmployeeID != employee.ID {
			return errors.New("you can only cancel your own leave requests")
		}
	}

	leave := *existingLeave // Create a copy
	leave.Status = LeaveStatusCancelled
	leave.CancelReason = cancelReason
	now := time.Now()
	leave.CancelledAt = &now
	leave.ModifiedBy = cancelledBy

	// Update the leave
	if err := s.leaveRepo.Update(&leave); err != nil {
		return err
	}

	// Get updated leave for audit
	updatedLeave, _ := s.leaveRepo.GetByID(id)

	// Audit the cancellation
	if err := s.auditService.CreateAuditLog("Leave", id, domain.AuditActionUpdate, existingLeave, updatedLeave, cancelledBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

// Leave Balance methods implementation
func (s *leaveService) ValidateLeaveBalance(employeeID, leaveTypeID uint, requestedDays float64, isAdmin bool) error {
	// Skip balance validation for admins
	if isAdmin {
		return nil
	}

	// Get all leave balance records for the employee and leave type
	balances, err := s.leaveBalanceRepo.GetByEmployeeAndLeaveType(employeeID, leaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to get leave balances: %w", err)
	}

	if len(balances) == 0 {
		return fmt.Errorf("no leave balance found for employee %d and leave type %d", employeeID, leaveTypeID)
	}

	// Calculate total remaining days across all years
	var totalRemainingDays float64
	for _, balance := range balances {
		totalRemainingDays += balance.RemainingDays
	}

	// Check if sufficient balance exists
	if totalRemainingDays < requestedDays {
		return fmt.Errorf("insufficient leave balance: requested %.1f days, available %.1f days", requestedDays, totalRemainingDays)
	}

	return nil
}

func (s *leaveService) DeductLeaveBalance(employeeID, leaveTypeID uint, requestedDays float64, modifiedBy string) error {
	// Get all leave balance records ordered by year ASC (oldest first)
	balances, err := s.leaveBalanceRepo.GetByEmployeeAndLeaveType(employeeID, leaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to get leave balances: %w", err)
	}

	if len(balances) == 0 {
		return fmt.Errorf("no leave balance found for employee %d and leave type %d", employeeID, leaveTypeID)
	}

	remainingDaysToDeduct := requestedDays
	var balancesToUpdate []*domain.LeaveBalance

	// First pass: Deduct from oldest years with remaining_days > 0
	for _, balance := range balances {
		if remainingDaysToDeduct <= 0 {
			break
		}

		if balance.RemainingDays > 0 {
			// Create a copy for modification
			updatedBalance := *balance

			// Calculate deduction amount
			deductAmount := remainingDaysToDeduct
			if deductAmount > balance.RemainingDays {
				deductAmount = balance.RemainingDays
			}

			// Update the balance
			updatedBalance.UsedDays += deductAmount
			updatedBalance.RemainingDays -= deductAmount
			remainingDaysToDeduct -= deductAmount

			balancesToUpdate = append(balancesToUpdate, &updatedBalance)
		}
	}

	// Second pass: If there are still days to deduct, apply to most recent year (allow negative)
	if remainingDaysToDeduct > 0 {
		// Find the most recent year balance (last in the ordered list)
		mostRecentBalance := balances[len(balances)-1]

		// Check if we already have this balance in our update list
		var mostRecentUpdated *domain.LeaveBalance
		for _, balance := range balancesToUpdate {
			if balance.ID == mostRecentBalance.ID {
				mostRecentUpdated = balance
				break
			}
		}

		// If not in update list, create a copy
		if mostRecentUpdated == nil {
			mostRecentCopy := *mostRecentBalance
			mostRecentUpdated = &mostRecentCopy
			balancesToUpdate = append(balancesToUpdate, mostRecentUpdated)
		}

		// Deduct remaining days (allow negative remaining_days)
		mostRecentUpdated.UsedDays += remainingDaysToDeduct
		mostRecentUpdated.RemainingDays -= remainingDaysToDeduct
	}

	// Update all modified balances in a single transaction
	if len(balancesToUpdate) > 0 {
		if err := s.leaveBalanceRepo.UpdateMultiple(balancesToUpdate, modifiedBy); err != nil {
			return fmt.Errorf("failed to update leave balances: %w", err)
		}

		// Audit the balance changes
		for _, balance := range balancesToUpdate {
			if err := s.auditService.CreateAuditLog("LeaveBalance", balance.ID, domain.AuditActionUpdate, nil, balance, modifiedBy); err != nil {
				// Log error but don't fail the operation
			}
		}
	}

	return nil
}

func (s *leaveService) GetMyLeaveBalances(userID uint, page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Set defaults for sorting
	if sortParams.Sort == "" {
		sortParams.Sort = "leave_type_id"
	}
	if sortParams.Direction == "" {
		sortParams.Direction = "ASC"
	}

	// First, get the employee record using the user ID
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find employee for user ID %d: %w", userID, err)
	}

	offset := (page - 1) * limit
	leaveBalances, total, err := s.leaveBalanceRepo.GetByEmployeeIDPaginated(employee.ID, limit, offset, sortParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get leave balances: %w", err)
	}

	// Convert to MyLeaveBalanceResponse format
	var responseData []*types.MyLeaveBalanceResponse
	for _, balance := range leaveBalances {
		responseData = append(responseData, &types.MyLeaveBalanceResponse{
			LeaveTypeName: balance.LeaveType.Name,
			Year:          balance.Year,
			TotalDays:     balance.TotalDays,
			UsedDays:      balance.UsedDays,
			RemainingDays: balance.RemainingDays,
		})
	}

	return &PaginatedResponse{
		Data: responseData,
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

// Leave Type methods implementation
func (s *leaveService) CreateLeaveType(leaveType *domain.LeaveType, createdBy string) error {
	if leaveType == nil {
		return errors.New("leave type cannot be nil")
	}

	// Validate required fields
	if leaveType.Name == "" {
		return errors.New("leave type name is required")
	}

	// Check if a leave type with the same name already exists
	existingLeaveType, err := s.leaveTypeRepo.GetByName(leaveType.Name)
	if err == nil && existingLeaveType != nil {
		return fmt.Errorf("leave type with name '%s' already exists", leaveType.Name)
	}

	// Set audit fields
	leaveType.Deleted = false

	// Create the leave type
	if err := s.leaveTypeRepo.Create(leaveType, createdBy); err != nil {
		return err
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("LeaveType", leaveType.ID, domain.AuditActionCreate, nil, leaveType, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) GetLeaveTypeByID(id uint) (*types.LeaveTypeResponse, error) {
	leaveType, err := s.leaveTypeRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return &types.LeaveTypeResponse{
		ID:                 leaveType.ID,
		CreatedAt:          leaveType.CreatedAt,
		UpdatedAt:          leaveType.UpdatedAt,
		Deleted:            leaveType.Deleted,
		CreatedBy:          leaveType.CreatedBy,
		ModifiedBy:         leaveType.ModifiedBy,
		Name:               leaveType.Name,
		Description:        leaveType.Description,
		IsPaid:             leaveType.IsPaid,
		IsLimited:          leaveType.IsLimited,
		IsAccrual:          leaveType.IsAccrual,
		IsRequiredDocument: leaveType.IsRequiredDocument,
	}, nil
}

func (s *leaveService) GetAllLeaveTypes(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Set defaults for sorting
	if sortParams.Sort == "" {
		sortParams.Sort = "id"
	}
	if sortParams.Direction == "" {
		sortParams.Direction = "ASC"
	}

	offset := (page - 1) * limit
	leaveTypes, total, err := s.leaveTypeRepo.GetAll(limit, offset, sortParams)
	if err != nil {
		return nil, err
	}

	// Convert domain objects to response DTOs
	var leaveTypeResponses []*types.LeaveTypeResponse
	for _, leaveType := range leaveTypes {
		leaveTypeResponses = append(leaveTypeResponses, &types.LeaveTypeResponse{
			ID:                 leaveType.ID,
			CreatedAt:          leaveType.CreatedAt,
			UpdatedAt:          leaveType.UpdatedAt,
			Deleted:            leaveType.Deleted,
			CreatedBy:          leaveType.CreatedBy,
			ModifiedBy:         leaveType.ModifiedBy,
			Name:               leaveType.Name,
			Description:        leaveType.Description,
			IsPaid:             leaveType.IsPaid,
			IsLimited:          leaveType.IsLimited,
			IsAccrual:          leaveType.IsAccrual,
			IsRequiredDocument: leaveType.IsRequiredDocument,
		})
	}

	return &PaginatedResponse{
		Data: leaveTypeResponses,
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

func (s *leaveService) GetLeaveTypesLookup() ([]types.LeaveTypeLookup, error) {
	leaveTypes, err := s.leaveTypeRepo.GetLookup()
	if err != nil {
		return nil, err
	}

	// Convert to lookup DTOs
	lookupData := make([]types.LeaveTypeLookup, len(leaveTypes))
	for i, leaveType := range leaveTypes {
		lookupData[i] = types.LeaveTypeLookup{
			ID:   leaveType.ID,
			Name: leaveType.Name,
		}
	}

	return lookupData, nil
}

func (s *leaveService) UpdateLeaveType(leaveType *domain.LeaveType, modifiedBy string) error {
	if leaveType == nil {
		return errors.New("leave type cannot be nil")
	}

	// Validate required fields
	if leaveType.Name == "" {
		return errors.New("leave type name is required")
	}

	// Get existing leave type for audit
	existingLeaveType, err := s.leaveTypeRepo.GetByID(leaveType.ID)
	if err != nil {
		return err
	}

	// Check if the name is being changed and if it conflicts with another leave type
	if existingLeaveType.Name != leaveType.Name {
		conflictingLeaveType, err := s.leaveTypeRepo.GetByName(leaveType.Name)
		if err == nil && conflictingLeaveType != nil && conflictingLeaveType.ID != leaveType.ID {
			return fmt.Errorf("leave type with name '%s' already exists", leaveType.Name)
		}
	}

	// Update the leave type
	if err := s.leaveTypeRepo.Update(leaveType, modifiedBy); err != nil {
		return err
	}

	// Get updated leave type for audit
	updatedLeaveType, _ := s.leaveTypeRepo.GetByID(leaveType.ID)

	// Audit the update
	if err := s.auditService.CreateAuditLog("LeaveType", leaveType.ID, domain.AuditActionUpdate, existingLeaveType, updatedLeaveType, modifiedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) DeleteLeaveType(id uint, deletedBy string) error {
	// Get existing leave type for audit
	existingLeaveType, err := s.leaveTypeRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete the leave type
	if err := s.leaveTypeRepo.Delete(id); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("LeaveType", id, domain.AuditActionDelete, existingLeaveType, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

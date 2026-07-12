package service

import (
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
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

var (
	ErrLeaveTypeLimitExceeded = errors.New("leave type limit exceeded")
)

type LeaveService interface {
	// Leave methods
	CreateLeave(leave *domain.LeaveRequest, userID uint, isAdmin bool) error
	GetLeaveByID(id uint) (*domain.LeaveRequest, error)
	GetLeaveByIDFormatted(id uint) (*types.AdminLeaveRequestResponse, error)
	UpdateLeave(leave *domain.LeaveRequest, userID uint) error
	DeleteLeave(id uint, userID uint) error
	GetLeavesByEmployeeID(employeeID uint, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error)
	GetLeavesByUserID(userID uint, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error)
	GetMyLeaveRequestsPaginated(userID uint, page, limit int, sortParams types.SortParams, status string, leaveTypeID *uint, startDate *string, endDate *string) (*PaginatedResponse, error)
	GetAllLeaveRequestsPaginated(employeeID *uint, page, limit int, sortParams types.SortParams, status string, leaveTypeID *uint, startDate *string, endDate *string, listGroup string) (*PaginatedResponse, error)
	GetLeavesByStatus(status string, sortBy string, sortDir types.SortDirection) ([]*domain.LeaveRequest, error)
	GetLeavesByDateRange(startDate, endDate string) ([]*domain.LeaveRequest, error)
	ApproveLeave(id uint, userID uint) error
	RejectLeave(id uint, rejectionReason string, userID uint) error
	CancelLeave(id uint, cancelReason string, userID uint, isAdmin bool) error

	// Leave Balance methods
	ValidateLeaveBalance(employeeID, leaveTypeID uint, requestedDays float64, isAdmin bool) error
	DeductLeaveBalance(employeeID, leaveTypeID uint, requestedDays float64, userID uint) error
	GetMyLeaveBalances(userID uint, page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	GetEmployeeLeaveBalances(employeeID uint, page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	GetLeaveBalanceForType(employeeID, leaveTypeID uint) (*domain.LeaveBalance, error)
	GetAnnualLeaveBalance(employeeID uint) (*domain.LeaveBalance, error)

	// Leave Type methods
	CreateLeaveType(leaveType *domain.LeaveType, userID uint) error
	GetLeaveTypeByID(id uint) (*types.LeaveTypeResponse, error)
	GetAllLeaveTypes(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	UpdateLeaveType(leaveType *domain.LeaveType, userID uint) error
	DeleteLeaveType(id uint, userID uint) error

	// Working days calculation
	CalculateWorkingDays(startDate, endDate time.Time, isStartDateFullDay, isFinishDateFullDay bool) (float64, error)
	CalculateEndDate(startDate time.Time, requestedDays float64, isStartDateFullDay, isFinishDateFullDay bool) (time.Time, error)

	// Leave Document methods (using Attachment system)
	UploadLeaveDocument(leaveRequestID uint, file *multipart.FileHeader, userID uint) (*domain.Attachment, error)
	GetLeaveDocuments(leaveRequestID uint, userID uint, isAdmin bool) ([]domain.Attachment, error)
	DeleteLeaveDocument(documentID string, userID uint, isAdmin bool) error
	DownloadLeaveDocument(documentID string, userID uint, isAdmin bool) (string, error)
}

type leaveService struct {
	leaveRepo        repository.LeaveRepository
	leaveTypeRepo    repository.LeaveTypeRepository
	leaveBalanceRepo repository.LeaveBalanceRepository
	employeeRepo     repository.EmployeeRepository
	holidayRepo      repository.HolidayRepository
	attachmentRepo   repository.AttachmentRepository
	storage          StorageProvider
	auditService     AuditService
}

func NewLeaveService(
	leaveRepo repository.LeaveRepository,
	leaveTypeRepo repository.LeaveTypeRepository,
	leaveBalanceRepo repository.LeaveBalanceRepository,
	employeeRepo repository.EmployeeRepository,
	holidayRepo repository.HolidayRepository,
	attachmentRepo repository.AttachmentRepository,
	storage StorageProvider,
	auditService AuditService,
) LeaveService {
	return &leaveService{
		leaveRepo:        leaveRepo,
		leaveTypeRepo:    leaveTypeRepo,
		leaveBalanceRepo: leaveBalanceRepo,
		employeeRepo:     employeeRepo,
		holidayRepo:      holidayRepo,
		attachmentRepo:   attachmentRepo,
		storage:          storage,
		auditService:     auditService,
	}
}

// CreateLeaveWithValidation creates a leave request with balance validation for employees
func (s *leaveService) CreateLeave(leave *domain.LeaveRequest, userID uint, isAdmin bool) error {
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

	// Get leave type to check if it's birthday leave
	leaveType, err := s.leaveTypeRepo.GetByID(leave.LeaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to get leave type: %w", err)
	}

	// Check for date overlap with existing PENDING leave requests
	pendingLeaves, err := s.leaveRepo.GetPendingLeavesByEmployeeIDAndDateRange(leave.EmployeeID, leave.LeaveTypeID, leave.StartDate, leave.EndDate)
	if err != nil {
		return fmt.Errorf("failed to check pending leaves: %w", err)
	}
	if len(pendingLeaves) > 0 {
		// Get the leave type name for the error message
		typeName := leaveType.Name
		return fmt.Errorf("Seçtiğiniz tarih aralığında bekleyen %s talebiniz olduğundan yeni talebiniz için izin girişi yapamazsınız", typeName)
	}

	// Check for birthday leave restrictions (Doğum Günü İzni)
	if leaveType.Name == "Doğum Günü İzni" || leaveType.Name == "Birthday Leave" {
		// Check if employee has completed one year of employment
		employee, err := s.employeeRepo.GetByID(leave.EmployeeID)
		if err != nil {
			return fmt.Errorf("failed to get employee: %w", err)
		}

		if employee.HireDate == nil {
			return errors.New("İşe başlama tarihi bulunamadığı için doğum günü izni kullanılamaz")
		}

		// Calculate if employee has completed one year
		oneYearAfterHire := employee.HireDate.AddDate(1, 0, 0)
		now := time.Now()
		if now.Before(oneYearAfterHire) {
			return errors.New("Doğum günü izni kullanabilmek için en az 1 yıl çalışma sürenizi doldurmuş olmanız gerekmektedir")
		}

		// Check if requested days exceeds 1 day limit
		if leave.RequestedDays > 1 {
			return errors.New("Doğum günü izni en fazla 1 gün girilebilir")
		}
		// Check if there's already an approved birthday leave in this year
		year := leave.StartDate.Year()
		existingBirthdayLeaves, err := s.leaveRepo.GetApprovedBirthdayLeaveInYear(leave.EmployeeID, leave.LeaveTypeID, year)
		if err == nil && len(existingBirthdayLeaves) > 0 {
			return errors.New("Bir takvim yılı içerisinde en fazla 1 kez doğum günü izni girebilirsiniz")
		}
	}

	// Limitli izinler yılda bir kez kullanılabilir.
	if leaveType.LimitAmount != nil && *leaveType.LimitAmount > 0 {
		year := leave.StartDate.Year()
		existingLeaves, err := s.leaveRepo.GetPendingOrApprovedLeavesByEmployeeAndTypeInYear(leave.EmployeeID, leave.LeaveTypeID, year)
		if err != nil {
			return fmt.Errorf("failed to check existing limited leaves: %w", err)
		}
		if len(existingLeaves) > 0 {
			return errors.New("Limitli izinler yılda bir kez kullanılabilir. Bu izin türü için daha önce giriş yapılmış.")
		}
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
	leave.CreatedBy = strconv.FormatUint(uint64(userID), 10)
	leave.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

	// Create the leave
	if err := s.leaveRepo.Create(leave); err != nil {
		return fmt.Errorf("failed to create leave: %w", err)
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("Leave", leave.ID, domain.AuditActionCreate, nil, leave, strconv.FormatUint(uint64(userID), 10)); err != nil {
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
			ID:                 leave.LeaveType.ID,
			Name:               leave.LeaveType.Name,
			LimitAmount:        leave.LeaveType.LimitAmount,
			IsRequiredDocument: leave.LeaveType.IsRequiredDocument,
		},
		StartDate:           leave.StartDate,
		EndDate:             leave.EndDate,
		IsStartDateFullDay:  leave.IsStartDateFullDay,
		IsFinishDateFullDay: leave.IsFinishDateFullDay,
		RequestedDays:       leave.RequestedDays,
		Reason:              leave.Reason,
		Status:              leave.Status,
		IsPaid:              leave.IsPaid,
		ApprovedBy:          leave.ApprovedBy,
		ApprovedAt:          leave.ApprovedAt,
		RejectedAt:          leave.RejectedAt,
		RejectionReason:     leave.RejectionReason,
		CancelReason:        leave.CancelReason,
		CancelledAt:         leave.CancelledAt,
		Comments:            leave.Comments,
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

func (s *leaveService) UpdateLeave(leave *domain.LeaveRequest, userID uint) error {
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

	// Validate that start date is not in the past
	today := time.Now().Truncate(24 * time.Hour)
	startDateNormalized := leave.StartDate.Truncate(24 * time.Hour)
	if startDateNormalized.Before(today) {
		return errors.New("start date cannot be in the past")
	}

	// Get the leave type to get the IsPaid value
	leaveType, err := s.leaveTypeRepo.GetByID(leave.LeaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to get leave type: %w", err)
	}

	// Check for date overlap with existing PENDING or APPROVED leave requests for the same leave type
	// Exclude the current leave request being updated
	pendingLeaves, err := s.leaveRepo.GetPendingLeavesByEmployeeIDAndDateRange(leave.EmployeeID, leave.LeaveTypeID, leave.StartDate, leave.EndDate)
	if err != nil {
		return fmt.Errorf("failed to check pending leaves: %w", err)
	}
	// Filter out the current leave request from the results
	for _, pendingLeave := range pendingLeaves {
		if pendingLeave.ID != leave.ID {
			return fmt.Errorf("Seçtiğiniz tarih aralığında bekleyen %s talebiniz olduğundan yeni talebiniz için izin girişi yapamazsınız", leaveType.Name)
		}
	}

	// Limitli izinler yılda bir kez kullanılabilir.
	if leaveType.LimitAmount != nil && *leaveType.LimitAmount > 0 {
		year := leave.StartDate.Year()
		existingLeaves, err := s.leaveRepo.GetPendingOrApprovedLeavesByEmployeeAndTypeInYear(leave.EmployeeID, leave.LeaveTypeID, year)
		if err != nil {
			return fmt.Errorf("failed to check existing limited leaves: %w", err)
		}

		hasOtherExistingLeave := false
		for _, existingLeave := range existingLeaves {
			if existingLeave.ID != leave.ID {
				hasOtherExistingLeave = true
				break
			}
		}

		if hasOtherExistingLeave {
			return errors.New("Limitli izinler yılda bir kez kullanılabilir. Bu izin türü için daha önce giriş yapılmış.")
		}
	}

	// Clone the existing leave and update only the provided fields
	updatedLeave := *existingLeave
	updatedLeave.LeaveTypeID = leave.LeaveTypeID
	updatedLeave.LeaveType = *leaveType // Set the LeaveType object
	updatedLeave.StartDate = leave.StartDate
	updatedLeave.EndDate = leave.EndDate
	updatedLeave.IsStartDateFullDay = leave.IsStartDateFullDay
	updatedLeave.IsFinishDateFullDay = leave.IsFinishDateFullDay
	updatedLeave.RequestedDays = leave.RequestedDays
	updatedLeave.Reason = leave.Reason
	updatedLeave.IsPaid = leaveType.IsPaid
	updatedLeave.UpdatedAt = time.Now()
	updatedLeave.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

	// Log the request before update
	fmt.Printf("UpdateLeave - Request to update: ID=%d, Reason=%s, LeaveTypeID=%d, StartDate=%v, EndDate=%v, RequestedDays=%v\n",
		updatedLeave.ID, updatedLeave.Reason, updatedLeave.LeaveTypeID, updatedLeave.StartDate, updatedLeave.EndDate, updatedLeave.RequestedDays)

	// Update the leave
	if err := s.leaveRepo.Update(&updatedLeave); err != nil {
		return fmt.Errorf("failed to update leave: %w", err)
	}

	// Get updated leave for audit
	auditedLeave, _ := s.leaveRepo.GetByID(leave.ID)

	// Audit the update
	if err := s.auditService.CreateAuditLog("Leave", leave.ID, domain.AuditActionUpdate, existingLeave, auditedLeave, strconv.FormatUint(uint64(userID), 10)); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) DeleteLeave(id uint, userID uint) error {
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
	if err := s.auditService.CreateAuditLog("Leave", id, domain.AuditActionDelete, existingLeave, nil, strconv.FormatUint(uint64(userID), 10)); err != nil {
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

func (s *leaveService) GetMyLeaveRequestsPaginated(userID uint, page, limit int, sortParams types.SortParams, status string, leaveTypeID *uint, startDate *string, endDate *string) (*PaginatedResponse, error) {
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

	leaves, total, err := s.leaveRepo.GetAllWithStatus(&employee.ID, limit, offset, sortParams, status, leaveTypeID, startDate, endDate, "")

	if err != nil {
		return nil, err
	}

	// Convert to MyLeaveRequestResponse format
	var responseData []*types.MyLeaveRequestResponse
	for _, leave := range leaves {
		// Get document count
		docCount, _ := s.attachmentRepo.CountByRelatedRecord(domain.AttachmentRelatedTypeLeave, leave.ID)

		responseData = append(responseData, &types.MyLeaveRequestResponse{
			ID:        leave.ID,
			CreatedAt: leave.CreatedAt,
			UpdatedAt: leave.UpdatedAt,
			LeaveType: types.LeaveTypeLookup{
				ID:                 leave.LeaveType.ID,
				Name:               leave.LeaveType.Name,
				LimitAmount:        leave.LeaveType.LimitAmount,
				IsRequiredDocument: leave.LeaveType.IsRequiredDocument,
			},
			StartDate:           leave.StartDate,
			EndDate:             leave.EndDate,
			IsStartDateFullDay:  leave.IsStartDateFullDay,
			IsFinishDateFullDay: leave.IsFinishDateFullDay,
			RequestedDays:       leave.RequestedDays,
			Reason:              leave.Reason,
			Status:              leave.Status,
			IsPaid:              leave.IsPaid,
			ApprovedAt:          leave.ApprovedAt,
			RejectedAt:          leave.RejectedAt,
			RejectionReason:     leave.RejectionReason,
			CancelReason:        leave.CancelReason,
			CancelledAt:         leave.CancelledAt,
			Comments:            leave.Comments,
			DocumentCount:       int(docCount),
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

func (s *leaveService) GetAllLeaveRequestsPaginated(employeeID *uint, page, limit int, sortParams types.SortParams, status string, leaveTypeID *uint, startDate *string, endDate *string, listGroup string) (*PaginatedResponse, error) {
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
	leaves, total, err := s.leaveRepo.GetAllWithStatus(employeeID, limit, offset, sortParams, status, leaveTypeID, startDate, endDate, listGroup)
	if err != nil {
		return nil, err
	}

	// Convert to AdminLeaveRequestResponse format
	var responseData []*types.AdminLeaveRequestResponse
	for _, leave := range leaves {
		// Get leave balance remaining days for annual leave only
		var remainingDays *float64
		if leave.LeaveType.Name == "Yıllık İzin" || leave.LeaveType.Name == "Annual Leave" {
			balances, err := s.leaveBalanceRepo.GetByEmployeeAndLeaveType(leave.EmployeeID, leave.LeaveTypeID)
			if err == nil && balances != nil && len(balances) > 0 {
				remainingDays = &balances[0].RemainingDays
			}
		}

		// Get document count
		docCount, _ := s.attachmentRepo.CountByRelatedRecord(domain.AttachmentRelatedTypeLeave, leave.ID)

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
				ID:                 leave.LeaveType.ID,
				Name:               leave.LeaveType.Name,
				LimitAmount:        leave.LeaveType.LimitAmount,
				IsRequiredDocument: leave.LeaveType.IsRequiredDocument,
			},
			StartDate:           leave.StartDate,
			EndDate:             leave.EndDate,
			IsStartDateFullDay:  leave.IsStartDateFullDay,
			IsFinishDateFullDay: leave.IsFinishDateFullDay,
			RequestedDays:       leave.RequestedDays,
			RemainingDays:       remainingDays,
			Reason:              leave.Reason,
			Status:              leave.Status,
			IsPaid:              leave.IsPaid,
			ApprovedBy:          leave.ApprovedBy,
			ApprovedAt:          leave.ApprovedAt,
			RejectedAt:          leave.RejectedAt,
			RejectionReason:     leave.RejectionReason,
			CancelReason:        leave.CancelReason,
			CancelledAt:         leave.CancelledAt,
			Comments:            leave.Comments,
			DocumentCount:       int(docCount),
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

func (s *leaveService) ApproveLeave(id uint, userID uint) error {
	// Get existing leave for audit and balance deduction
	existingLeave, err := s.leaveRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Get leave type to check if it's annual leave
	leaveType, err := s.leaveTypeRepo.GetByID(existingLeave.LeaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to get leave type: %w", err)
	}

	// For annual leave type, validate balance before approval
	if leaveType.Name == "Yıllık İzin" || leaveType.Name == "Annual Leave" {
		// Deduct leave balance for annual leave
		if err := s.DeductLeaveBalance(existingLeave.EmployeeID, existingLeave.LeaveTypeID, existingLeave.RequestedDays, userID); err != nil {
			return fmt.Errorf("failed to deduct leave balance: %w", err)
		}
	}

	leave := *existingLeave // Create a copy
	leave.Status = LeaveStatusApproved
	leave.ApprovedBy = &userID
	now := time.Now()
	leave.ApprovedAt = &now
	leave.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

	// Update the leave
	if err := s.leaveRepo.Update(&leave); err != nil {
		return err
	}

	// Get updated leave for audit
	updatedLeave, _ := s.leaveRepo.GetByID(id)

	// Audit the approval
	if err := s.auditService.CreateAuditLog("Leave", id, domain.AuditActionUpdate, existingLeave, updatedLeave, strconv.FormatUint(uint64(userID), 10)); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) RejectLeave(id uint, rejectionReason string, userID uint) error {
	// Get existing leave for audit
	existingLeave, err := s.leaveRepo.GetByID(id)
	if err != nil {
		return err
	}

	leave := *existingLeave // Create a copy
	leave.Status = LeaveStatusRejected
	leave.RejectionReason = rejectionReason
	leave.ApprovedBy = &userID // Set the rejector's user ID in ApprovedBy field
	now := time.Now()
	leave.RejectedAt = &now // Set the rejection timestamp
	leave.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

	// Update the leave
	if err := s.leaveRepo.Update(&leave); err != nil {
		return err
	}

	// Get updated leave for audit
	updatedLeave, _ := s.leaveRepo.GetByID(id)

	// Audit the rejection
	if err := s.auditService.CreateAuditLog("Leave", id, domain.AuditActionUpdate, existingLeave, updatedLeave, strconv.FormatUint(uint64(userID), 10)); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) CancelLeave(id uint, cancelReason string, userID uint, isAdmin bool) error {
	// Get existing leave for validation and audit
	existingLeave, err := s.leaveRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check if the leave request can be cancelled (only PENDING and APPROVED requests)
	if existingLeave.Status != LeaveStatusPending && existingLeave.Status != LeaveStatusApproved {
		return fmt.Errorf("only pending or approved leave requests can be cancelled, current status: %s", existingLeave.Status)
	}

	// Authorization check: employees can only cancel their own PENDING requests, admins can cancel any PENDING or APPROVED
	if !isAdmin {
		// Get employee ID from user ID
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil {
			return fmt.Errorf("failed to find employee for user: %w", err)
		}

		if existingLeave.EmployeeID != employee.ID {
			return errors.New("you can only cancel your own leave requests")
		}

		// Employees can only cancel PENDING requests, or APPROVED requests that haven't started yet
		if existingLeave.Status != LeaveStatusPending {
			if existingLeave.Status == LeaveStatusApproved && existingLeave.StartDate.After(time.Now()) {
				// Allowed: approved but future leave — reverse balance if needed
				leaveType, err := s.leaveTypeRepo.GetByID(existingLeave.LeaveTypeID)
				if err != nil {
					return fmt.Errorf("failed to get leave type: %w", err)
				}
				if leaveType.Name == "Yıllık İzin" || leaveType.Name == "Annual Leave" {
					if err := s.ReverseLeaveBalance(existingLeave.EmployeeID, existingLeave.LeaveTypeID, existingLeave.RequestedDays, userID); err != nil {
						return fmt.Errorf("failed to reverse leave balance: %w", err)
					}
				}
			} else {
				return fmt.Errorf("you can only cancel pending leave requests or approved future leave requests, current status: %s", existingLeave.Status)
			}
		}
	}

	// If the leave is APPROVED and being cancelled by admin, reverse the deduction
	if existingLeave.Status == LeaveStatusApproved && isAdmin {
		// Get leave type to check if it's annual leave
		leaveType, err := s.leaveTypeRepo.GetByID(existingLeave.LeaveTypeID)
		if err != nil {
			return fmt.Errorf("failed to get leave type: %w", err)
		}

		// Reverse leave balance deduction only for annual leave type
		if leaveType.Name == "Yıllık İzin" || leaveType.Name == "Annual Leave" {
			if err := s.ReverseLeaveBalance(existingLeave.EmployeeID, existingLeave.LeaveTypeID, existingLeave.RequestedDays, userID); err != nil {
				return fmt.Errorf("failed to reverse leave balance: %w", err)
			}
		}
	}

	leave := *existingLeave // Create a copy
	leave.Status = LeaveStatusCancelled
	leave.CancelReason = cancelReason
	now := time.Now()
	leave.CancelledAt = &now
	leave.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

	// Update the leave
	if err := s.leaveRepo.Update(&leave); err != nil {
		return err
	}

	// Get updated leave for audit
	updatedLeave, _ := s.leaveRepo.GetByID(id)

	// Audit the cancellation
	if err := s.auditService.CreateAuditLog("Leave", id, domain.AuditActionUpdate, existingLeave, updatedLeave, strconv.FormatUint(uint64(userID), 10)); err != nil {
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

	// Get leave type to check if it has a limit
	leaveType, err := s.leaveTypeRepo.GetByID(leaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to get leave type: %w", err)
	}

	// If leave type has a limit amount, check against approved leaves in current year
	if leaveType.LimitAmount != nil && *leaveType.LimitAmount > 0 {
		currentYear := time.Now().Year()

		// Get all approved leaves for this employee and leave type in current year
		approvedLeaves, err := s.leaveRepo.GetApprovedLeavesByEmployeeAndTypeInYear(employeeID, leaveTypeID, currentYear)
		if err != nil {
			return fmt.Errorf("failed to get approved leaves: %w", err)
		}

		// Calculate total used days from approved leaves
		var totalUsedDays float64
		for _, leave := range approvedLeaves {
			totalUsedDays += leave.RequestedDays
		}

		// Check if requested days + used days exceeds limit
		if totalUsedDays+requestedDays > float64(*leaveType.LimitAmount) {
			remainingLimit := float64(*leaveType.LimitAmount) - totalUsedDays
			return fmt.Errorf("%w: %s izin türü için yıllık limitiniz %d gündür. Bu yıl içinde %.1f gün kullanmışsınız. En fazla %.1f gün izin girişi yapabilirsiniz",
				ErrLeaveTypeLimitExceeded,
				leaveType.Name,
				*leaveType.LimitAmount,
				totalUsedDays,
				remainingLimit)
		}
	}

	// Get all leave balance records for the employee and leave type
	balances, err := s.leaveBalanceRepo.GetByEmployeeAndLeaveType(employeeID, leaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to get leave balances: %w", err)
	}

	/*if len(balances) == 0 {
		return fmt.Errorf("no leave balance found for employee %d and leave type %d", employeeID, leaveTypeID)
	}*/

	// Calculate total remaining days across all years
	var totalRemainingDays float64
	for _, balance := range balances {
		totalRemainingDays += balance.RemainingDays
	}

	// Check if sufficient balance exists
	/*if totalRemainingDays < requestedDays {
		return fmt.Errorf("insufficient leave balance: requested %.1f days, available %.1f days", requestedDays, totalRemainingDays)
	}*/

	return nil
}

func (s *leaveService) DeductLeaveBalance(employeeID, leaveTypeID uint, requestedDays float64, userID uint) error {
	// Get leave balance record for employee and leave type
	// Since there's only one record per employee+leaveType, we use FirstOrCreate
	balance := &domain.LeaveBalance{
		EmployeeID:  employeeID,
		LeaveTypeID: leaveTypeID,
	}

	// Try to find existing balance, if not found create with default values
	result, err := s.leaveBalanceRepo.GetByEmployeeAndLeaveType(employeeID, leaveTypeID)
	if err != nil || result == nil || len(result) == 0 {
		// Create new balance with 0 values
		balance.TotalDays = 0
		balance.UsedDays = 0
		balance.RemainingDays = 0
		balance.CreatedBy = strconv.FormatUint(uint64(userID), 10)
		balance.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

		// Create the new balance
		if err := s.leaveBalanceRepo.Create(balance, strconv.FormatUint(uint64(userID), 10)); err != nil {
			return fmt.Errorf("failed to create leave balance: %w", err)
		}
	} else {
		// Use the existing balance
		balance = result[0]
	}

	// Deduct the requested days from remaining_days (allow negative)
	balance.UsedDays += requestedDays
	balance.RemainingDays -= requestedDays
	balance.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

	// Update the balance
	if err := s.leaveBalanceRepo.Update(balance, strconv.FormatUint(uint64(userID), 10)); err != nil {
		return fmt.Errorf("failed to update leave balance: %w", err)
	}

	// Audit the balance change
	if err := s.auditService.CreateAuditLog("LeaveBalance", balance.ID, domain.AuditActionUpdate, nil, balance, strconv.FormatUint(uint64(userID), 10)); err != nil {
		// Log error but don't fail the operation
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

func (s *leaveService) GetEmployeeLeaveBalances(employeeID uint, page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
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

	offset := (page - 1) * limit
	leaveBalances, total, err := s.leaveBalanceRepo.GetByEmployeeIDPaginated(employeeID, limit, offset, sortParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get leave balances: %w", err)
	}

	// Convert to MyLeaveBalanceResponse format
	var responseData []*types.MyLeaveBalanceResponse
	for _, balance := range leaveBalances {
		responseData = append(responseData, &types.MyLeaveBalanceResponse{
			LeaveTypeName: balance.LeaveType.Name,
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

// ReverseLeaveBalance reverses the leave balance deduction when an approved leave is cancelled
func (s *leaveService) ReverseLeaveBalance(employeeID, leaveTypeID uint, requestedDays float64, userID uint) error {
	// Get leave balance record for employee and leave type
	result, err := s.leaveBalanceRepo.GetByEmployeeAndLeaveType(employeeID, leaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to get leave balance: %w", err)
	}

	if len(result) == 0 {
		// If no balance record exists, nothing to reverse
		return nil
	}

	balance := result[0]

	// Reverse the deduction (add back the days)
	balance.UsedDays -= requestedDays
	balance.RemainingDays += requestedDays
	balance.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

	// Update the balance
	if err := s.leaveBalanceRepo.Update(balance, strconv.FormatUint(uint64(userID), 10)); err != nil {
		return fmt.Errorf("failed to update leave balance: %w", err)
	}

	// Audit the balance change
	if err := s.auditService.CreateAuditLog("LeaveBalance", balance.ID, domain.AuditActionUpdate, nil, balance, strconv.FormatUint(uint64(userID), 10)); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

// Leave Type methods implementation
func (s *leaveService) CreateLeaveType(leaveType *domain.LeaveType, userID uint) error {
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
	leaveType.CreatedBy = strconv.FormatUint(uint64(userID), 10)
	leaveType.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

	// Create the leave type
	if err := s.leaveTypeRepo.Create(leaveType, strconv.FormatUint(uint64(userID), 10)); err != nil {
		return err
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("LeaveType", leaveType.ID, domain.AuditActionCreate, nil, leaveType, strconv.FormatUint(uint64(userID), 10)); err != nil {
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
		LimitAmount:        leaveType.LimitAmount,
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
			LimitAmount:        leaveType.LimitAmount,
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

func (s *leaveService) UpdateLeaveType(leaveType *domain.LeaveType, userID uint) error {
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

	// Clone the existing leave type and update only the provided fields
	updatedLeaveType := *existingLeaveType
	updatedLeaveType.Name = leaveType.Name
	updatedLeaveType.Description = leaveType.Description
	updatedLeaveType.IsPaid = leaveType.IsPaid
	updatedLeaveType.LimitAmount = leaveType.LimitAmount
	updatedLeaveType.IsAccrual = leaveType.IsAccrual
	updatedLeaveType.IsRequiredDocument = leaveType.IsRequiredDocument
	updatedLeaveType.ModifiedBy = strconv.FormatUint(uint64(userID), 10)

	// Update the leave type
	if err := s.leaveTypeRepo.Update(&updatedLeaveType, strconv.FormatUint(uint64(userID), 10)); err != nil {
		return err
	}

	// Get updated leave type for audit
	auditedLeaveType, _ := s.leaveTypeRepo.GetByID(leaveType.ID)

	// Audit the update
	if err := s.auditService.CreateAuditLog("LeaveType", leaveType.ID, domain.AuditActionUpdate, existingLeaveType, auditedLeaveType, strconv.FormatUint(uint64(userID), 10)); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *leaveService) DeleteLeaveType(id uint, userID uint) error {
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
	if err := s.auditService.CreateAuditLog("LeaveType", id, domain.AuditActionDelete, existingLeaveType, nil, strconv.FormatUint(uint64(userID), 10)); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

// CalculateWorkingDays calculates the actual number of working days excluding weekends and holidays
// It takes into account weekends, holidays (full-day and half-day), and half-day leaves
// isStartDateFullDay and isFinishDateFullDay parameters indicate if the start/end dates are full or half days
func (s *leaveService) CalculateWorkingDays(startDate, endDate time.Time, isStartDateFullDay, isFinishDateFullDay bool) (float64, error) {
	if startDate.After(endDate) {
		return 0, errors.New("start date must be before or equal to end date")
	}

	// Normalize dates to midnight to ensure consistent date comparison
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	// Get holidays between the date range
	holidays, err := s.holidayRepo.GetHolidaysBetweenDates(startDate, endDate)
	if err != nil {
		return 0, fmt.Errorf("failed to get holidays: %w", err)
	}

	// Create a map of holiday dates with their full-day/half-day status
	// map[date_string]holidayDayValue where holidayDayValue is 1.0 for full-day, 0.5 for half-day
	holidayMap := make(map[string]float64)
	for _, holiday := range holidays {
		// Normalize holiday date to midnight
		holidayDate := time.Date(holiday.HolidayDate.Year(), holiday.HolidayDate.Month(), holiday.HolidayDate.Day(), 0, 0, 0, 0, holiday.HolidayDate.Location())
		dateKey := holidayDate.Format("2006-01-02")

		// If it's a full-day holiday, it takes the entire day (1.0)
		// If it's a half-day holiday, it only takes half the day (0.5)
		if holiday.IsFullDay {
			holidayMap[dateKey] = 1.0
		} else {
			holidayMap[dateKey] = 0.5
		}
	}

	// Count working days (excluding weekends and considering holidays)
	workingDays := 0.0
	currentDate := startDate

	// Loop through each day from start to end date (inclusive)
	for {
		// Check if it's a weekend (Saturday = 6, Sunday = 0)
		dayOfWeek := currentDate.Weekday()
		isWeekend := dayOfWeek == time.Saturday || dayOfWeek == time.Sunday

		// Check if it's a holiday and get its value (1.0 for full-day, 0.5 for half-day, 0 if not a holiday)
		dateKey := currentDate.Format("2006-01-02")
		holidayDayValue, isHoliday := holidayMap[dateKey]

		// Skip weekends entirely
		if !isWeekend {
			dayValue := 1.0 // Default to full working day

			// Apply half-day rules for start and end dates
			if currentDate.Equal(startDate) && !isStartDateFullDay {
				dayValue = 0.5
			}

			// Check end date separately (not else-if to handle same-day leaves)
			if currentDate.Equal(endDate) && !isFinishDateFullDay {
				// If start and end are the same day and both are half-day
				if currentDate.Equal(startDate) && !isStartDateFullDay {
					// Same day, both half-day = 0.5 total (not 1.0)
					dayValue = 0.5
				} else {
					// Different days or only end date is half-day
					dayValue = 0.5
				}
			}

			// If this day is a holiday, subtract the holiday impact
			if isHoliday {
				// If it's a full-day holiday (1.0), no working day counts
				// If it's a half-day holiday (0.5), only half working day counts
				dayValue = dayValue - holidayDayValue

				// Ensure dayValue doesn't go negative
				if dayValue < 0 {
					dayValue = 0
				}
			}

			// Add the calculated day value
			workingDays += dayValue
		}

		// If we've reached the end date, break the loop
		if currentDate.Equal(endDate) {
			break
		}

		// Move to next day
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return workingDays, nil
}

// CalculateEndDate calculates the end date given a start date and requested working days
func (s *leaveService) CalculateEndDate(startDate time.Time, requestedDays float64, isStartDateFullDay, isFinishDateFullDay bool) (time.Time, error) {
	if requestedDays <= 0 {
		return time.Time{}, errors.New("requested days must be greater than 0")
	}

	currentDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())

	for {
		days, err := s.CalculateWorkingDays(startDate, currentDate, isStartDateFullDay, isFinishDateFullDay)
		if err != nil {
			return time.Time{}, err
		}

		if days == requestedDays {
			return currentDate, nil
		}

		if days > requestedDays {
			return time.Time{}, errors.New("impossible to reach exactly the requested days with the given full/half day settings")
		}

		currentDate = currentDate.AddDate(0, 0, 1)

		// Hard limit to prevent infinite loops (e.g. searching for 1000 days or due to zero progress)
		if currentDate.Sub(startDate).Hours() > 24*365*10 {
			return time.Time{}, errors.New("requested days too large or calculation loop exceeded limit")
		}
	}
}

// ==================== Leave Document Methods ====================

// UploadLeaveDocument uploads a document for a leave request using Attachment system
func (s *leaveService) UploadLeaveDocument(leaveRequestID uint, file *multipart.FileHeader, userID uint) (*domain.Attachment, error) {
	// Check if leave request exists
	leave, err := s.leaveRepo.GetByID(leaveRequestID)
	if err != nil {
		return nil, errors.New("leave request not found")
	}

	// Get employee to check ownership
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("employee not found")
	}

	// Only the owner can upload documents (admin can also upload via their own user context)
	if leave.EmployeeID != employee.ID {
		return nil, errors.New("you can only upload documents to your own leave requests")
	}

	// Documents can be uploaded to any leave (not just PENDING like expenses)
	// This allows uploading medical reports after sick leave approval, etc.

	// Upload using attachment helper
	attachment, err := s.uploadLeaveAttachment(file, userID, leaveRequestID, domain.AttachmentTypeMedicalReport)
	if err != nil {
		return nil, err
	}

	return attachment, nil
}

// uploadLeaveAttachment is a helper function to upload leave attachments
func (s *leaveService) uploadLeaveAttachment(file *multipart.FileHeader, ownerID uint, leaveRequestID uint, docType domain.AttachmentType) (*domain.Attachment, error) {
	// Open the file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Generate storage path using shared document service utility
	timestamp := time.Now().Format("20060102150405")
	docID := fmt.Sprintf("%d_%s", leaveRequestID, timestamp)
	storagePath := GenerateStoragePath(domain.AttachmentRelatedTypeLeave, file.Filename, docID)

	// Upload to storage
	if err := s.storage.Upload(src, storagePath); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Create attachment record
	relatedID := leaveRequestID
	attachment := &domain.Attachment{
		ID:          domain.GenerateUUID(),
		OwnerID:     ownerID,
		RelatedType: domain.AttachmentRelatedTypeLeave,
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

// GetLeaveDocuments retrieves all documents for a leave request
func (s *leaveService) GetLeaveDocuments(leaveRequestID uint, userID uint, isAdmin bool) ([]domain.Attachment, error) {
	// Check if leave request exists
	leave, err := s.leaveRepo.GetByID(leaveRequestID)
	if err != nil {
		return nil, errors.New("leave request not found")
	}

	// Check permission: admin or owner can view
	if !isAdmin {
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil {
			return nil, errors.New("employee not found")
		}
		if leave.EmployeeID != employee.ID {
			return nil, errors.New("you can only view documents of your own leave requests")
		}
	}

	// Get all attachments for this leave request
	attachments, err := s.attachmentRepo.FindByRelatedRecord(domain.AttachmentRelatedTypeLeave, leaveRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}

	return attachments, nil
}

// DeleteLeaveDocument deletes a document
func (s *leaveService) DeleteLeaveDocument(documentID string, userID uint, isAdmin bool) error {
	// Get attachment
	attachment, err := s.attachmentRepo.FindByID(documentID)
	if err != nil {
		return errors.New("document not found")
	}

	// Verify it's a leave document
	if attachment.RelatedType != domain.AttachmentRelatedTypeLeave || attachment.RelatedID == nil {
		return errors.New("invalid leave document")
	}

	// Get leave request
	leave, err := s.leaveRepo.GetByID(*attachment.RelatedID)
	if err != nil {
		return errors.New("leave request not found")
	}

	// Check permission: admin or owner can delete
	if !isAdmin {
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil {
			return errors.New("employee not found")
		}
		if leave.EmployeeID != employee.ID {
			return errors.New("you can only delete documents of your own leave requests")
		}
	}

	// Documents can be deleted from any leave status (not restricted to PENDING)

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

// DownloadLeaveDocument generates a download URL for a document
func (s *leaveService) DownloadLeaveDocument(documentID string, userID uint, isAdmin bool) (string, error) {
	// Get attachment
	attachment, err := s.attachmentRepo.FindByID(documentID)
	if err != nil {
		return "", errors.New("document not found")
	}

	// Verify it's a leave document
	if attachment.RelatedType != domain.AttachmentRelatedTypeLeave || attachment.RelatedID == nil {
		return "", errors.New("invalid leave document")
	}

	// Get leave request
	leave, err := s.leaveRepo.GetByID(*attachment.RelatedID)
	if err != nil {
		return "", errors.New("leave request not found")
	}

	// Check permission: admin or owner can download
	if !isAdmin {
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil {
			return "", errors.New("employee not found")
		}
		if leave.EmployeeID != employee.ID {
			return "", errors.New("you can only download documents of your own leave requests")
		}
	}

	// Generate presigned URL (valid for 15 minutes)
	url, err := s.storage.GeneratePresignedURL(attachment.Path, 15)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url, nil
}

// GetLeaveBalanceForType returns the leave balance for a specific employee and leave type.
func (s *leaveService) GetLeaveBalanceForType(employeeID, leaveTypeID uint) (*domain.LeaveBalance, error) {
	balances, err := s.leaveBalanceRepo.GetByEmployeeAndLeaveType(employeeID, leaveTypeID)
	if err != nil {
		return nil, err
	}
	if len(balances) == 0 {
		return nil, nil
	}
	return balances[0], nil
}

// GetAnnualLeaveBalance returns the first (annual) leave balance for the employee.
// Since the system only has one leave type, this returns whichever balance exists.
func (s *leaveService) GetAnnualLeaveBalance(employeeID uint) (*domain.LeaveBalance, error) {
	balances, _, err := s.leaveBalanceRepo.GetByEmployeeIDPaginated(employeeID, 1, 0, types.SortParams{Sort: "leave_type_id", Direction: "ASC"})
	if err != nil {
		return nil, err
	}
	if len(balances) == 0 {
		return nil, nil
	}
	return balances[0], nil
}

package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"

	"gorm.io/gorm"
)

type EmployeeGradeService interface {
	CreateEmployeeGrade(employeeID, gradeID uint, startDate, endDate, createdBy string) (*domain.EmployeeGrade, error)
	GetEmployeeGradeByID(id uint) (*types.EmployeeGradeResponse, error)
	GetEmployeeGradeByUserID(userID uint) ([]*types.EmployeeGradeWithNames, error)
	UpdateEmployeeGrade(id uint, employeeID, gradeID uint, startDate, endDate, modifiedBy string, requestingUserID uint, isAdmin bool) error
	DeleteEmployeeGrade(id uint, deletedBy string) error
	GetAllEmployeeGrades(page, limit int, sortParams types.SortParams, employeeID *uint) (*PaginatedResponse, error)
}

type employeeGradeService struct {
	employeeGradeRepo repository.EmployeeGradeRepository
	employeeRepo      repository.EmployeeRepository
	gradeRepo         repository.GradeRepository
	auditService      AuditService
}

func NewEmployeeGradeService(employeeGradeRepo repository.EmployeeGradeRepository, employeeRepo repository.EmployeeRepository, gradeRepo repository.GradeRepository, auditService AuditService) EmployeeGradeService {
	return &employeeGradeService{
		employeeGradeRepo: employeeGradeRepo,
		employeeRepo:      employeeRepo,
		gradeRepo:         gradeRepo,
		auditService:      auditService,
	}
}

// CreateEmployeeGrade assigns a new ACTIVE grade inside a transaction.
// Request end_date is ignored: assignment always creates ACTIVE with end_date NULL.
// Any previous ACTIVE row for the employee is closed (INACTIVE, end_date = start-1 day).
func (s *employeeGradeService) CreateEmployeeGrade(employeeID, gradeID uint, startDate, endDate, createdBy string) (*domain.EmployeeGrade, error) {
	_ = endDate // client end_date is ignored for assign lifecycle (backward-compatible request field)

	startDateParsed, err := parseDate(startDate)
	if err != nil || startDateParsed == nil {
		return nil, domain.ErrEmployeeGradeInvalidStartDate
	}
	startDay := dateOnlyUTC(*startDateParsed)

	if _, err := s.employeeRepo.GetByID(employeeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEmployeeGradeEmployeeNotFound
		}
		return nil, fmt.Errorf("failed to get employee: %w", err)
	}

	if _, err := s.gradeRepo.GetByID(int64(gradeID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEmployeeGradeGradeNotFound
		}
		return nil, fmt.Errorf("failed to get grade: %w", err)
	}

	var created *domain.EmployeeGrade
	var closedBefore *domain.EmployeeGrade
	var closedAfter *domain.EmployeeGrade

	err = s.employeeGradeRepo.Transaction(func(txRepo repository.EmployeeGradeRepository) error {
		exists, err := txRepo.ExistsByEmployeeGradeStartDate(employeeID, gradeID, startDay)
		if err != nil {
			return fmt.Errorf("failed to check duplicate employee grade: %w", err)
		}
		if exists {
			return domain.ErrEmployeeGradeDuplicateAssignment
		}

		active, err := txRepo.GetActiveByEmployeeIDForUpdate(employeeID)
		if err != nil {
			return fmt.Errorf("failed to lock active employee grade: %w", err)
		}

		if active != nil {
			endDay, err := domain.ActiveGradeCloseEndDate(active.StartDate, startDay)
			if err != nil {
				return err
			}

			before := *active
			closedBefore = &before

			if err := txRepo.CloseActiveAsInactive(active.ID, endDay, createdBy); err != nil {
				return fmt.Errorf("failed to close active employee grade: %w", err)
			}

			after := *active
			after.Status = domain.EmployeeGradeStatusInactive
			after.EndDate = &endDay
			after.ModifiedBy = createdBy
			closedAfter = &after
		}

		employeeGrade := &domain.EmployeeGrade{
			EmployeeID: employeeID,
			GradeID:    gradeID,
			StartDate:  startDay,
			EndDate:    nil,
			Status:     domain.EmployeeGradeStatusActive,
		}

		if err := txRepo.Create(employeeGrade, createdBy); err != nil {
			if repository.IsEmployeeGradeActiveUniqueViolation(err) {
				return domain.ErrEmployeeGradeActiveConflict
			}
			return fmt.Errorf("failed to create employee grade: %w", err)
		}

		created = employeeGrade
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Audit after commit (audit service is not transaction-aware).
	if closedBefore != nil && closedAfter != nil {
		_ = s.auditService.CreateAuditLog("EmployeeGrade", closedBefore.ID, domain.AuditActionUpdate, closedBefore, closedAfter, createdBy)
	}
	if created != nil {
		_ = s.auditService.CreateAuditLog("EmployeeGrade", created.ID, domain.AuditActionCreate, nil, created, createdBy)
	}

	return created, nil
}

func (s *employeeGradeService) GetEmployeeGradeByID(id uint) (*types.EmployeeGradeResponse, error) {
	if id == 0 {
		return nil, domain.ErrEmployeeGradeNotFound
	}

	employeeGrade, err := s.employeeGradeRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return toEmployeeGradeResponse(employeeGrade), nil
}

func (s *employeeGradeService) GetEmployeeGradeByUserID(userID uint) ([]*types.EmployeeGradeWithNames, error) {
	employeeGrades, err := s.employeeGradeRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	var result []*types.EmployeeGradeWithNames

	for _, employeeGrade := range employeeGrades {
		grade, err := s.gradeRepo.GetByID(int64(employeeGrade.GradeID))
		if err != nil {
			return nil, fmt.Errorf("failed to get grade: %v", err)
		}

		var startDateStr string
		var endDateStr *string

		if !employeeGrade.StartDate.IsZero() {
			startDateStr = employeeGrade.StartDate.Format(time.RFC3339)
		}

		if employeeGrade.EndDate != nil {
			dateStr := employeeGrade.EndDate.Format(time.RFC3339)
			endDateStr = &dateStr
		}

		employeeGradeDTO := &types.EmployeeGradeWithNames{
			ID:        employeeGrade.ID,
			GradeName: grade.Name,
			StartDate: startDateStr,
			EndDate:   endDateStr,
			Status:    string(employeeGrade.Status),
		}

		result = append(result, employeeGradeDTO)
	}

	return result, nil
}

// UpdateEmployeeGrade corrects an existing history row without changing its lifecycle state.
// employee_id is immutable; ACTIVE updates are serialized with grade assignment.
func (s *employeeGradeService) UpdateEmployeeGrade(id uint, employeeID, gradeID uint, startDate, endDate, modifiedBy string, requestingUserID uint, isAdmin bool) error {
	existingEmployeeGrade, err := s.employeeGradeRepo.GetByID(id)
	if err != nil {
		return err
	}

	if !isAdmin {
		employee, err := s.employeeRepo.GetByID(existingEmployeeGrade.EmployeeID)
		if err != nil {
			return fmt.Errorf("failed to get employee for authorization: %v", err)
		}
		if employee.UserID != requestingUserID {
			return errors.New("unauthorized to update this employee grade")
		}
	}

	if employeeID != existingEmployeeGrade.EmployeeID {
		return domain.ErrEmployeeGradeEmployeeImmutable
	}

	startDateParsed, err := parseDate(startDate)
	if err != nil || startDateParsed == nil {
		return domain.ErrEmployeeGradeInvalidStartDate
	}
	startDay := dateOnlyUTC(*startDateParsed)

	if _, err := s.gradeRepo.GetByID(int64(gradeID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrEmployeeGradeGradeNotFound
		}
		return fmt.Errorf("failed to get grade: %w", err)
	}

	employeeGrade := &domain.EmployeeGrade{
		EmployeeID: existingEmployeeGrade.EmployeeID,
		GradeID:    gradeID,
		StartDate:  startDay,
		Status:     existingEmployeeGrade.Status,
	}
	employeeGrade.ID = id

	if existingEmployeeGrade.Status == domain.EmployeeGradeStatusActive {
		employeeGrade.EndDate = nil
		if err := s.employeeGradeRepo.Transaction(func(txRepo repository.EmployeeGradeRepository) error {
			active, err := txRepo.GetActiveByEmployeeIDForUpdate(existingEmployeeGrade.EmployeeID)
			if err != nil {
				return fmt.Errorf("failed to lock active employee grade: %w", err)
			}
			if active == nil || active.ID != id {
				return domain.ErrEmployeeGradeActiveConflict
			}
			return txRepo.Update(employeeGrade, modifiedBy)
		}); err != nil {
			return err
		}
	} else {
		if endDate == "" {
			return domain.ErrEmployeeGradeInactiveRequiresEndDate
		}
		endDateParsed, err := parseDate(endDate)
		if err != nil || endDateParsed == nil {
			return fmt.Errorf("%w: invalid end date format", domain.ErrEmployeeGradeInactiveRequiresEndDate)
		}
		endDay := dateOnlyUTC(*endDateParsed)
		if endDay.Before(startDay) {
			return domain.ErrEmployeeGradeEndBeforeStart
		}
		employeeGrade.EndDate = &endDay
		if err := s.employeeGradeRepo.Update(employeeGrade, modifiedBy); err != nil {
			return err
		}
	}

	updatedEmployeeGrade, _ := s.employeeGradeRepo.GetByID(id)
	_ = s.auditService.CreateAuditLog("EmployeeGrade", id, domain.AuditActionUpdate, existingEmployeeGrade, updatedEmployeeGrade, modifiedBy)

	return nil
}

func (s *employeeGradeService) DeleteEmployeeGrade(id uint, deletedBy string) error {
	existingEmployeeGrade, err := s.employeeGradeRepo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.employeeGradeRepo.Transaction(func(txRepo repository.EmployeeGradeRepository) error {
		if existingEmployeeGrade.Status == domain.EmployeeGradeStatusActive {
			active, err := txRepo.GetActiveByEmployeeIDForUpdate(existingEmployeeGrade.EmployeeID)
			if err != nil {
				return fmt.Errorf("failed to lock active employee grade: %w", err)
			}
			if active == nil || active.ID != id {
				return domain.ErrEmployeeGradeActiveConflict
			}
		}
		return txRepo.Delete(id, deletedBy)
	}); err != nil {
		return err
	}

	_ = s.auditService.CreateAuditLog("EmployeeGrade", id, domain.AuditActionDelete, existingEmployeeGrade, nil, deletedBy)

	return nil
}

func (s *employeeGradeService) GetAllEmployeeGrades(page, limit int, sortParams types.SortParams, employeeID *uint) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	if sortParams.Sort == "" {
		sortParams.Sort = "id"
	}
	if sortParams.Direction == "" {
		sortParams.Direction = "ASC"
	}

	offset := (page - 1) * limit
	employeeGrades, total, err := s.employeeGradeRepo.GetAll(limit, offset, sortParams, employeeID)
	if err != nil {
		return nil, err
	}

	employeeGradeResponses := make([]types.EmployeeGradeResponse, len(employeeGrades))
	for i, employeeGrade := range employeeGrades {
		employeeGradeResponses[i] = *toEmployeeGradeResponse(&employeeGrade)
	}

	return &PaginatedResponse{
		Data: employeeGradeResponses,
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

func toEmployeeGradeResponse(employeeGrade *domain.EmployeeGrade) *types.EmployeeGradeResponse {
	return &types.EmployeeGradeResponse{
		ID:         employeeGrade.ID,
		CreatedAt:  employeeGrade.CreatedAt,
		UpdatedAt:  employeeGrade.UpdatedAt,
		Deleted:    employeeGrade.Deleted,
		CreatedBy:  employeeGrade.CreatedBy,
		ModifiedBy: employeeGrade.ModifiedBy,
		StartDate:  employeeGrade.StartDate,
		EndDate:    employeeGrade.EndDate,
		Status:     string(employeeGrade.Status),
		Employee: types.EmployeeLookup{
			ID:        employeeGrade.Employee.ID,
			FirstName: employeeGrade.Employee.FirstName,
			LastName:  employeeGrade.Employee.LastName,
		},
		Grade: types.GradeLookup{
			ID:   employeeGrade.Grade.ID,
			Name: employeeGrade.Grade.Name,
		},
	}
}

func dateOnlyUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// IsEmployeeGradeClientError reports validation / domain errors that should map to HTTP 4xx.
func IsEmployeeGradeClientError(err error) bool {
	switch {
	case errors.Is(err, domain.ErrEmployeeGradeNotFound),
		errors.Is(err, domain.ErrEmployeeGradeEmployeeNotFound),
		errors.Is(err, domain.ErrEmployeeGradeGradeNotFound),
		errors.Is(err, domain.ErrEmployeeGradeInvalidStartDate),
		errors.Is(err, domain.ErrEmployeeGradeInvalidCloseDate),
		errors.Is(err, domain.ErrEmployeeGradeDuplicateAssignment),
		errors.Is(err, domain.ErrEmployeeGradeActiveConflict),
		errors.Is(err, domain.ErrEmployeeGradeActiveCannotDelete),
		errors.Is(err, domain.ErrEmployeeGradeActiveUpdateForbidden),
		errors.Is(err, domain.ErrEmployeeGradeEmployeeImmutable),
		errors.Is(err, domain.ErrEmployeeGradeInactiveRequiresEndDate),
		errors.Is(err, domain.ErrEmployeeGradeEndBeforeStart):
		return true
	default:
		return false
	}
}

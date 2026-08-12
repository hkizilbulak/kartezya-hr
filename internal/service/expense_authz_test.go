package service

import (
	"errors"
	"testing"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type stubExpenseRepo struct {
	expense *domain.ExpenseRequest
	err     error
}

func (s *stubExpenseRepo) Create(expense *domain.ExpenseRequest) error { return nil }
func (s *stubExpenseRepo) FindByID(id uint) (*domain.ExpenseRequest, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.expense, nil
}
func (s *stubExpenseRepo) FindByEmployeeID(employeeID uint, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error) {
	return nil, nil
}
func (s *stubExpenseRepo) FindByStatus(status string, sortBy string, sortDir types.SortDirection) ([]*domain.ExpenseRequest, error) {
	return nil, nil
}
func (s *stubExpenseRepo) GetAll(employeeID *uint, page, limit int, sortParams types.SortParams, status string, expenseTypeID *uint, startDate, endDate *string) ([]*domain.ExpenseRequest, int64, error) {
	return nil, 0, nil
}
func (s *stubExpenseRepo) Update(expense *domain.ExpenseRequest) error { return nil }
func (s *stubExpenseRepo) Delete(id uint) error                        { return nil }

type stubEmployeeRepoForExpense struct {
	employee *domain.Employee
	err      error
}

func (s *stubEmployeeRepoForExpense) Create(employee *domain.Employee, createdBy string) error {
	return nil
}
func (s *stubEmployeeRepoForExpense) GetByID(id uint) (*domain.Employee, error) { return nil, nil }
func (s *stubEmployeeRepoForExpense) GetByIDs(ids []uint) ([]*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetByUserID(userID uint) (*domain.Employee, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.employee, nil
}
func (s *stubEmployeeRepoForExpense) GetByEmail(email string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetByIdentityNo(identityNo string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetByPhone(phone string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetByCompanyEmail(companyEmail string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Employee, int64, error) {
	return nil, 0, nil
}
func (s *stubEmployeeRepoForExpense) GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.Employee, int64, error) {
	return nil, 0, nil
}
func (s *stubEmployeeRepoForExpense) Update(employee *domain.Employee, modifiedBy string) error {
	return nil
}
func (s *stubEmployeeRepoForExpense) Delete(id uint, deletedBy string) error { return nil }
func (s *stubEmployeeRepoForExpense) GetTotalCount() (int64, error)          { return 0, nil }
func (s *stubEmployeeRepoForExpense) GetTotalCountWithFilters(filters map[string]interface{}) (int64, error) {
	return 0, nil
}
func (s *stubEmployeeRepoForExpense) GetEmployeeCountByGender() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetEmployeeCountByPosition() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetEmployeeCountByCompanyDepartment() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetInternCountByCompanyDepartment() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetEmployeeCountByGrade() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetWorkDayReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.WorkDayReportRow, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetGradeReportData(companyID *uint, departmentIDs []uint, isActive *bool) ([]types.GradeReportRow, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetContractReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.ContractReportRow, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForExpense) GetLookupList() ([]*domain.Employee, error) { return nil, nil }
func (s *stubEmployeeRepoForExpense) InTransaction(fn func(empRepo repository.EmployeeRepository, gradeRepo repository.EmployeeGradeRepository) error) error {
	return fn(s, nil)
}

func TestGetExpenseRequestByIDForCaller_OwnerAllowed(t *testing.T) {
	svc := &expenseService{
		expenseRepo: &stubExpenseRepo{
			expense: &domain.ExpenseRequest{AuditableModel: domain.AuditableModel{ID: 10}, EmployeeID: 5},
		},
		employeeRepo: &stubEmployeeRepoForExpense{
			employee: &domain.Employee{AuditableModel: domain.AuditableModel{ID: 5}, UserID: 100},
		},
	}

	got, err := svc.GetExpenseRequestByIDForCaller(10, 100, false)
	if err != nil {
		t.Fatalf("owner should view own expense: %v", err)
	}
	if got.ID != 10 {
		t.Fatalf("unexpected expense id %d", got.ID)
	}
}

func TestGetExpenseRequestByIDForCaller_UnrelatedEmployeeDenied(t *testing.T) {
	svc := &expenseService{
		expenseRepo: &stubExpenseRepo{
			expense: &domain.ExpenseRequest{AuditableModel: domain.AuditableModel{ID: 10}, EmployeeID: 5},
		},
		employeeRepo: &stubEmployeeRepoForExpense{
			employee: &domain.Employee{AuditableModel: domain.AuditableModel{ID: 99}, UserID: 200},
		},
	}

	_, err := svc.GetExpenseRequestByIDForCaller(10, 200, false)
	if err == nil || err.Error() != "access denied" {
		t.Fatalf("expected access denied, got %v", err)
	}
}

func TestGetExpenseRequestByIDForCaller_ManagementAllowed(t *testing.T) {
	svc := &expenseService{
		expenseRepo: &stubExpenseRepo{
			expense: &domain.ExpenseRequest{AuditableModel: domain.AuditableModel{ID: 10}, EmployeeID: 5},
		},
		employeeRepo: &stubEmployeeRepoForExpense{
			err: errors.New("not needed"),
		},
	}

	got, err := svc.GetExpenseRequestByIDForCaller(10, 200, true)
	if err != nil {
		t.Fatalf("management viewer should access expense: %v", err)
	}
	if got.ID != 10 {
		t.Fatalf("unexpected expense id %d", got.ID)
	}
}

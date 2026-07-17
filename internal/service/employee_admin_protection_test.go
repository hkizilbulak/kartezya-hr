package service

import (
	"errors"
	"testing"
	"time"

	"kartezya-hr/internal/authz"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
)

type stubEmployeeRepoForProtection struct {
	employee *domain.Employee
	updated  bool
	deleted  bool
}

func (s *stubEmployeeRepoForProtection) Create(employee *domain.Employee, createdBy string) error {
	return nil
}
func (s *stubEmployeeRepoForProtection) GetByID(id uint) (*domain.Employee, error) {
	if s.employee == nil {
		return nil, errors.New("not found")
	}
	return s.employee, nil
}
func (s *stubEmployeeRepoForProtection) GetByIDs(ids []uint) ([]*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetByUserID(userID uint) (*domain.Employee, error) {
	return s.employee, nil
}
func (s *stubEmployeeRepoForProtection) GetByEmail(email string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetByIdentityNo(identityNo string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetByPhone(phone string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetByCompanyEmail(companyEmail string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Employee, int64, error) {
	return nil, 0, nil
}
func (s *stubEmployeeRepoForProtection) GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.Employee, int64, error) {
	return nil, 0, nil
}
func (s *stubEmployeeRepoForProtection) Update(employee *domain.Employee, modifiedBy string) error {
	s.updated = true
	return nil
}
func (s *stubEmployeeRepoForProtection) Delete(id uint, deletedBy string) error {
	s.deleted = true
	return nil
}
func (s *stubEmployeeRepoForProtection) GetTotalCount() (int64, error) { return 0, nil }
func (s *stubEmployeeRepoForProtection) GetTotalCountWithFilters(filters map[string]interface{}) (int64, error) {
	return 0, nil
}
func (s *stubEmployeeRepoForProtection) GetEmployeeCountByGender() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetEmployeeCountByPosition() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetEmployeeCountByCompanyDepartment() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetInternCountByCompanyDepartment() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetEmployeeCountByGrade() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetWorkDayReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.WorkDayReportRow, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetGradeReportData(companyID *uint, departmentIDs []uint, isActive *bool) ([]types.GradeReportRow, error) {
	return nil, nil
}
func (s *stubEmployeeRepoForProtection) GetContractReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.ContractReportRow, error) {
	return nil, nil
}

type stubUserRepoForProtection struct {
	user *domain.User
}

func (s *stubUserRepoForProtection) Create(user *domain.User, createdBy string) error { return nil }
func (s *stubUserRepoForProtection) GetByID(id uint) (*domain.User, error)         { return s.user, nil }
func (s *stubUserRepoForProtection) GetByEmail(email string) (*domain.User, error) {
	return nil, errors.New("not found")
}
func (s *stubUserRepoForProtection) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.User, int64, error) {
	return nil, 0, nil
}
func (s *stubUserRepoForProtection) Update(user *domain.User, modifiedBy string) error { return nil }
func (s *stubUserRepoForProtection) Delete(id uint, deletedBy string) error            { return nil }
func (s *stubUserRepoForProtection) GetWithRoles(id uint) (*domain.User, error)        { return s.user, nil }
func (s *stubUserRepoForProtection) GetEmployeeByUserID(userID uint) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubUserRepoForProtection) UpdatePasswordResetToken(userID uint, token string, expiresAt *time.Time) error {
	return nil
}
func (s *stubUserRepoForProtection) GetByPasswordResetToken(token string) (*domain.User, error) {
	return nil, nil
}
func (s *stubUserRepoForProtection) ClearPasswordResetToken(userID uint) error { return nil }

type stubUserRoleRepoForProtection struct {
	hasAdmin      bool
	assignedRoles []string
}

func (s *stubUserRoleRepoForProtection) Create(userRole *domain.UserRole, createdBy string) error {
	return nil
}
func (s *stubUserRoleRepoForProtection) DeleteByUserID(userID uint, deletedBy string) error {
	return nil
}
func (s *stubUserRoleRepoForProtection) GetRolesByUserID(userID uint) ([]domain.Role, error) {
	return nil, nil
}
func (s *stubUserRoleRepoForProtection) HasRole(userID uint, roleName string) (bool, error) {
	if roleName == domain.RoleAdmin {
		return s.hasAdmin, nil
	}
	return false, nil
}
func (s *stubUserRoleRepoForProtection) Update(userRole *domain.UserRole, modifiedBy string) error {
	return nil
}
func (s *stubUserRoleRepoForProtection) Delete(userID, roleID uint, deletedBy string) error {
	return nil
}

type stubRoleRepoForProtection struct{}

func (s *stubRoleRepoForProtection) Create(role *domain.Role, createdBy string) error { return nil }
func (s *stubRoleRepoForProtection) GetByID(id uint) (*domain.Role, error)           { return nil, nil }
func (s *stubRoleRepoForProtection) GetByName(name string) (*domain.Role, error) {
	return &domain.Role{AuditableModel: domain.AuditableModel{ID: 1}, Name: name}, nil
}
func (s *stubRoleRepoForProtection) List() ([]domain.Role, error) { return nil, nil }
func (s *stubRoleRepoForProtection) Update(role *domain.Role, modifiedBy string) error {
	return nil
}
func (s *stubRoleRepoForProtection) Delete(id uint, deletedBy string) error { return nil }

type stubAuditForProtection struct{}

func (s *stubAuditForProtection) CreateAuditLog(entityType string, entityID uint, action string, oldValue, newValue interface{}, performedBy string) error {
	return nil
}

func newProtectionEmployeeService(emp *domain.Employee, hasAdmin bool) (*employeeService, *stubEmployeeRepoForProtection, *stubUserRoleRepoForProtection) {
	empRepo := &stubEmployeeRepoForProtection{employee: emp}
	roleRepo := &stubUserRoleRepoForProtection{hasAdmin: hasAdmin}
	svc := &employeeService{
		employeeRepo: empRepo,
		userRepo:     &stubUserRepoForProtection{user: &domain.User{AuditableModel: domain.AuditableModel{ID: emp.UserID}, Email: "u@example.com"}},
		userRoleRepo: roleRepo,
		roleRepo:     &stubRoleRepoForProtection{},
		auditService: &stubAuditForProtection{},
	}
	return svc, empRepo, roleRepo
}

func TestUpdateEmployeeAdminCanAssignAdmin(t *testing.T) {
	emp := &domain.Employee{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 10, CompanyEmail: "t@example.com"}
	svc, empRepo, _ := newProtectionEmployeeService(emp, false)

	err := svc.UpdateEmployee(1, "p@example.com", "t@example.com", "A", "B", "", "", "", "", "", "", "", "", 0, "", "", "", "", nil, "", "", "", "", "", "", "", "ACTIVE", "admin@x", 99, []string{domain.RoleAdmin}, []string{domain.RoleAdmin})
	if err != nil {
		t.Fatalf("ADMIN should assign ADMIN: %v", err)
	}
	if !empRepo.updated {
		t.Fatal("expected employee update")
	}
}

func TestUpdateEmployeeHRCannotAssignAdmin(t *testing.T) {
	emp := &domain.Employee{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 10, CompanyEmail: "t@example.com"}
	svc, empRepo, _ := newProtectionEmployeeService(emp, false)

	err := svc.UpdateEmployee(1, "p@example.com", "t@example.com", "A", "B", "", "", "", "", "", "", "", "", 0, "", "", "", "", nil, "", "", "", "", "", "", "", "ACTIVE", "hr@x", 99, []string{domain.RoleHR}, []string{domain.RoleAdmin})
	if !errors.Is(err, authz.ErrForbiddenAssignAdmin) {
		t.Fatalf("expected ErrForbiddenAssignAdmin, got %v", err)
	}
	if empRepo.updated {
		t.Fatal("must not update when assigning ADMIN is forbidden")
	}
}

func TestUpdateEmployeeHRCanAssignAllowedRoles(t *testing.T) {
	emp := &domain.Employee{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 10, CompanyEmail: "t@example.com"}
	svc, _, _ := newProtectionEmployeeService(emp, false)

	err := svc.UpdateEmployee(1, "p@example.com", "t@example.com", "A", "B", "", "", "", "", "", "", "", "", 0, "", "", "", "", nil, "", "", "", "", "", "", "", "ACTIVE", "hr@x", 99, []string{domain.RoleHR}, []string{domain.RoleEmployee, domain.RoleHR, domain.RoleFinancial})
	if err != nil {
		t.Fatalf("HR should assign allowed roles: %v", err)
	}
}

func TestUpdateEmployeeHRCannotUpdateAdminTarget(t *testing.T) {
	emp := &domain.Employee{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 10, CompanyEmail: "t@example.com"}
	svc, empRepo, _ := newProtectionEmployeeService(emp, true)

	err := svc.UpdateEmployee(1, "p@example.com", "t@example.com", "A", "B", "", "", "", "", "", "", "", "", 0, "", "", "", "", nil, "", "", "", "", "", "", "", "ACTIVE", "hr@x", 99, []string{domain.RoleHR}, []string{domain.RoleEmployee})
	if !errors.Is(err, authz.ErrForbiddenAdminTarget) {
		t.Fatalf("expected ErrForbiddenAdminTarget, got %v", err)
	}
	if empRepo.updated {
		t.Fatal("must not update ADMIN target")
	}
}

func TestUpdateEmployeeAdminCanUpdateAdminTarget(t *testing.T) {
	emp := &domain.Employee{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 10, CompanyEmail: "t@example.com"}
	svc, _, _ := newProtectionEmployeeService(emp, true)

	err := svc.UpdateEmployee(1, "p@example.com", "t@example.com", "A", "B", "", "", "", "", "", "", "", "", 0, "", "", "", "", nil, "", "", "", "", "", "", "", "ACTIVE", "admin@x", 99, []string{domain.RoleAdmin}, []string{domain.RoleAdmin})
	if err != nil {
		t.Fatalf("ADMIN should manage ADMIN target: %v", err)
	}
}

func TestDeleteEmployeeHRCannotDeleteAdminTarget(t *testing.T) {
	emp := &domain.Employee{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 10}
	svc, empRepo, _ := newProtectionEmployeeService(emp, true)

	err := svc.DeleteEmployee(1, "hr@x", []string{domain.RoleHR})
	if !errors.Is(err, authz.ErrForbiddenAdminTarget) {
		t.Fatalf("expected ErrForbiddenAdminTarget, got %v", err)
	}
	if empRepo.deleted {
		t.Fatal("must not delete ADMIN target")
	}
}

func TestDeleteEmployeeAdminCanDeleteAdminTarget(t *testing.T) {
	emp := &domain.Employee{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 10}
	svc, empRepo, _ := newProtectionEmployeeService(emp, true)

	err := svc.DeleteEmployee(1, "admin@x", []string{domain.RoleAdmin})
	if err != nil {
		t.Fatalf("ADMIN should delete ADMIN target: %v", err)
	}
	if !empRepo.deleted {
		t.Fatal("expected delete")
	}
}

func TestDeleteEmployeeFinanceCannotMutate(t *testing.T) {
	emp := &domain.Employee{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 10}
	svc, empRepo, _ := newProtectionEmployeeService(emp, false)

	err := svc.DeleteEmployee(1, "finance@x", []string{domain.RoleFinancial})
	if err == nil {
		t.Fatal("FINANCIAL must not delete employees")
	}
	if empRepo.deleted {
		t.Fatal("must not delete")
	}
}

func TestPasswordResetDeniedForHRAgainstAdminTarget(t *testing.T) {
	if err := authz.DenyHRMutatingAdminTarget(authz.ClassifyActor([]string{domain.RoleHR}), true); !errors.Is(err, authz.ErrForbiddenAdminTarget) {
		t.Fatalf("HR password reset against ADMIN must be forbidden, got %v", err)
	}
	if err := authz.DenyHRMutatingAdminTarget(authz.ClassifyActor([]string{domain.RoleAdmin}), true); err != nil {
		t.Fatalf("ADMIN password reset against ADMIN must be allowed: %v", err)
	}
}

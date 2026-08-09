package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type stubEGAudit struct {
	logs []string
}

func (s *stubEGAudit) CreateAuditLog(entityName string, entityID uint, action string, oldValue any, newValue any, performedBy string) error {
	s.logs = append(s.logs, action)
	return nil
}

type stubEGEmployeeRepo struct {
	employee        *domain.Employee
	gradeReportRows []types.GradeReportRow
	err             error
}

func (s *stubEGEmployeeRepo) Create(employee *domain.Employee, createdBy string) error { return nil }
func (s *stubEGEmployeeRepo) GetByID(id uint) (*domain.Employee, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.employee == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if s.employee.ID != 0 && s.employee.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return s.employee, nil
}
func (s *stubEGEmployeeRepo) GetByIDs(ids []uint) ([]*domain.Employee, error) { return nil, nil }
func (s *stubEGEmployeeRepo) GetByUserID(userID uint) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEGEmployeeRepo) GetByEmail(email string) (*domain.Employee, error) { return nil, nil }
func (s *stubEGEmployeeRepo) GetByIdentityNo(identityNo string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEGEmployeeRepo) GetByPhone(phone string) (*domain.Employee, error) { return nil, nil }
func (s *stubEGEmployeeRepo) GetByCompanyEmail(companyEmail string) (*domain.Employee, error) {
	return nil, nil
}
func (s *stubEGEmployeeRepo) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Employee, int64, error) {
	return nil, 0, nil
}
func (s *stubEGEmployeeRepo) GetAllWithFilters(limit, offset int, sortParams types.SortParams, filters map[string]interface{}) ([]*domain.Employee, int64, error) {
	return nil, 0, nil
}
func (s *stubEGEmployeeRepo) Update(employee *domain.Employee, modifiedBy string) error { return nil }
func (s *stubEGEmployeeRepo) Delete(id uint, deletedBy string) error                    { return nil }
func (s *stubEGEmployeeRepo) GetTotalCount() (int64, error)                             { return 0, nil }
func (s *stubEGEmployeeRepo) GetTotalCountWithFilters(filters map[string]interface{}) (int64, error) {
	return 0, nil
}
func (s *stubEGEmployeeRepo) GetEmployeeCountByGender() ([]interface{}, error) { return nil, nil }
func (s *stubEGEmployeeRepo) GetEmployeeCountByPosition() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEGEmployeeRepo) GetEmployeeCountByCompanyDepartment() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEGEmployeeRepo) GetInternCountByCompanyDepartment() ([]interface{}, error) {
	return nil, nil
}
func (s *stubEGEmployeeRepo) GetEmployeeCountByGrade() ([]interface{}, error) { return nil, nil }
func (s *stubEGEmployeeRepo) GetWorkDayReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.WorkDayReportRow, error) {
	return nil, nil
}
func (s *stubEGEmployeeRepo) GetGradeReportData(companyID *uint, departmentIDs []uint, isActive *bool) ([]types.GradeReportRow, error) {
	return s.gradeReportRows, nil
}
func (s *stubEGEmployeeRepo) GetContractReportData(startDate, endDate string, companyID *uint, departmentIDs []uint, isActive *bool) ([]types.ContractReportRow, error) {
	return nil, nil
}
func (s *stubEGEmployeeRepo) InTransaction(fn func(empRepo repository.EmployeeRepository, gradeRepo repository.EmployeeGradeRepository) error) error {
	return fn(s, newStubEmployeeGradeRepo())
}

type stubEGGradeRepo struct {
	grade  *domain.Grade
	grades []*domain.Grade
	err    error
}

func (s *stubEGGradeRepo) Create(grade *domain.Grade, createdBy string) error { return nil }
func (s *stubEGGradeRepo) GetByID(id int64) (*domain.Grade, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.grade == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return s.grade, nil
}
func (s *stubEGGradeRepo) GetByName(name string) (*domain.Grade, error) { return nil, nil }
func (s *stubEGGradeRepo) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Grade, int64, error) {
	return s.grades, int64(len(s.grades)), s.err
}
func (s *stubEGGradeRepo) GetLookup() ([]domain.Grade, error)                  { return nil, nil }
func (s *stubEGGradeRepo) Update(grade *domain.Grade, modifiedBy string) error { return nil }
func (s *stubEGGradeRepo) Delete(id int64, deletedBy string) error             { return nil }
func (s *stubEGGradeRepo) GetTotalCount() (int64, error) {
	return int64(len(s.grades)), s.err
}

type stubEmployeeGradeRepo struct {
	mu         sync.Mutex
	records    map[uint]*domain.EmployeeGrade
	nextID     uint
	failClose  error
	failCreate error
}

func newStubEmployeeGradeRepo(records ...*domain.EmployeeGrade) *stubEmployeeGradeRepo {
	s := &stubEmployeeGradeRepo{
		records: make(map[uint]*domain.EmployeeGrade),
		nextID:  1,
	}
	for _, r := range records {
		cp := *r
		s.records[r.ID] = &cp
		if r.ID >= s.nextID {
			s.nextID = r.ID + 1
		}
	}
	return s
}

func (s *stubEmployeeGradeRepo) cloneRecords() map[uint]*domain.EmployeeGrade {
	out := make(map[uint]*domain.EmployeeGrade, len(s.records))
	for id, r := range s.records {
		cp := *r
		if r.EndDate != nil {
			end := *r.EndDate
			cp.EndDate = &end
		}
		out[id] = &cp
	}
	return out
}

func (s *stubEmployeeGradeRepo) Transaction(fn func(txRepo repository.EmployeeGradeRepository) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx := &stubEmployeeGradeRepo{
		records:    s.cloneRecords(),
		nextID:     s.nextID,
		failClose:  s.failClose,
		failCreate: s.failCreate,
	}
	if err := fn(tx); err != nil {
		return err
	}
	s.records = tx.records
	s.nextID = tx.nextID
	return nil
}

func (s *stubEmployeeGradeRepo) Create(employeeGrade *domain.EmployeeGrade, createdBy string) error {
	if s.failCreate != nil {
		return s.failCreate
	}
	if employeeGrade.ID == 0 {
		employeeGrade.ID = s.nextID
		s.nextID++
	}
	employeeGrade.CreatedBy = createdBy
	employeeGrade.ModifiedBy = createdBy
	cp := *employeeGrade
	s.records[employeeGrade.ID] = &cp
	return nil
}

func (s *stubEmployeeGradeRepo) GetByID(id uint) (*domain.EmployeeGrade, error) {
	r, ok := s.records[id]
	if !ok || r.Deleted {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *r
	return &cp, nil
}

func (s *stubEmployeeGradeRepo) GetByUserID(userID uint) ([]domain.EmployeeGrade, error) {
	return nil, nil
}

func (s *stubEmployeeGradeRepo) GetAll(limit, offset int, sortParams types.SortParams, employeeID *uint) ([]domain.EmployeeGrade, int64, error) {
	return nil, 0, nil
}

func (s *stubEmployeeGradeRepo) Update(employeeGrade *domain.EmployeeGrade, modifiedBy string) error {
	r, ok := s.records[employeeGrade.ID]
	if !ok || r.Deleted {
		return gorm.ErrRecordNotFound
	}
	r.GradeID = employeeGrade.GradeID
	r.StartDate = employeeGrade.StartDate
	r.EndDate = employeeGrade.EndDate
	r.Status = employeeGrade.Status
	r.ModifiedBy = modifiedBy
	return nil
}

func (s *stubEmployeeGradeRepo) Delete(id uint, deletedBy string) error {
	r, ok := s.records[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	r.Deleted = true
	r.Status = domain.EmployeeGradeStatusInactive
	if r.EndDate == nil {
		end := r.StartDate
		r.EndDate = &end
	}
	r.ModifiedBy = deletedBy
	return nil
}

func (s *stubEmployeeGradeRepo) GetActiveByEmployeeIDForUpdate(employeeID uint) (*domain.EmployeeGrade, error) {
	for _, r := range s.records {
		if r.EmployeeID == employeeID && !r.Deleted && r.Status == domain.EmployeeGradeStatusActive {
			cp := *r
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *stubEmployeeGradeRepo) CloseActiveAsInactive(id uint, endDate time.Time, modifiedBy string) error {
	if s.failClose != nil {
		return s.failClose
	}
	r, ok := s.records[id]
	if !ok || r.Deleted || r.Status != domain.EmployeeGradeStatusActive {
		return errors.New("failed to close active employee grade: no matching ACTIVE row")
	}
	r.Status = domain.EmployeeGradeStatusInactive
	end := endDate
	r.EndDate = &end
	r.ModifiedBy = modifiedBy
	return nil
}

func (s *stubEmployeeGradeRepo) ExistsByEmployeeGradeStartDate(employeeID, gradeID uint, startDate time.Time) (bool, error) {
	startDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	for _, r := range s.records {
		if r.Deleted {
			continue
		}
		rd := time.Date(r.StartDate.Year(), r.StartDate.Month(), r.StartDate.Day(), 0, 0, 0, 0, time.UTC)
		if r.EmployeeID == employeeID && r.GradeID == gradeID && rd.Equal(startDay) {
			return true, nil
		}
	}
	return false, nil
}

var (
	_ repository.EmployeeGradeRepository = (*stubEmployeeGradeRepo)(nil)
	_ repository.EmployeeRepository      = (*stubEGEmployeeRepo)(nil)
	_ repository.GradeRepository         = (*stubEGGradeRepo)(nil)
)

func newEmployeeGradeServiceForTest(egRepo *stubEmployeeGradeRepo) (*employeeGradeService, *stubEGAudit) {
	audit := &stubEGAudit{}
	svc := &employeeGradeService{
		employeeGradeRepo: egRepo,
		employeeRepo: &stubEGEmployeeRepo{
			employee: &domain.Employee{AuditableModel: domain.AuditableModel{ID: 1}, UserID: 10},
		},
		gradeRepo: &stubEGGradeRepo{
			grade: &domain.Grade{AuditableModel: domain.AuditableModel{ID: 2}, Name: "G2"},
		},
		auditService: audit,
	}
	return svc, audit
}

func TestCreateEmployeeGrade_NoActiveCreatesActive(t *testing.T) {
	repo := newStubEmployeeGradeRepo()
	svc, audit := newEmployeeGradeServiceForTest(repo)

	created, err := svc.CreateEmployeeGrade(1, 2, "2026-07-30", "2099-01-01", "hr@test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != domain.EmployeeGradeStatusActive || created.EndDate != nil {
		t.Fatalf("expected ACTIVE with null end_date, got %#v", created)
	}
	if len(audit.logs) == 0 || audit.logs[len(audit.logs)-1] != domain.AuditActionCreate {
		t.Fatalf("expected create audit, got %#v", audit.logs)
	}
}

func TestCreateEmployeeGrade_ClosesPreviousActive(t *testing.T) {
	activeStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      activeStart,
		Status:         domain.EmployeeGradeStatusActive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	created, err := svc.CreateEmployeeGrade(1, 2, "2026-07-30", "", "hr@test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != domain.EmployeeGradeStatusActive || created.EndDate != nil {
		t.Fatalf("new row: %#v", created)
	}

	old := repo.records[5]
	if old.Status != domain.EmployeeGradeStatusInactive {
		t.Fatalf("old status = %s", old.Status)
	}
	wantEnd := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	if old.EndDate == nil || !old.EndDate.Equal(wantEnd) {
		t.Fatalf("old end_date = %v, want %v", old.EndDate, wantEnd)
	}
}

func TestCreateEmployeeGrade_SameGradeDifferentStartDate(t *testing.T) {
	activeStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        2,
		StartDate:      activeStart,
		Status:         domain.EmployeeGradeStatusActive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	created, err := svc.CreateEmployeeGrade(1, 2, "2026-03-01", "", "hr@test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.GradeID != 2 || created.Status != domain.EmployeeGradeStatusActive {
		t.Fatalf("unexpected new row %#v", created)
	}
	if repo.records[5].Status != domain.EmployeeGradeStatusInactive {
		t.Fatal("expected previous same-grade row closed")
	}
}

func TestCreateEmployeeGrade_DuplicateSameStartRejected(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        2,
		StartDate:      start,
		Status:         domain.EmployeeGradeStatusActive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	_, err := svc.CreateEmployeeGrade(1, 2, "2026-01-01", "", "hr@test")
	if !errors.Is(err, domain.ErrEmployeeGradeDuplicateAssignment) {
		t.Fatalf("got %v", err)
	}
	if len(repo.records) != 1 {
		t.Fatalf("expected no new rows, got %d", len(repo.records))
	}
}

func TestCreateEmployeeGrade_SameDayStartRejected(t *testing.T) {
	activeStart := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      activeStart,
		Status:         domain.EmployeeGradeStatusActive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	_, err := svc.CreateEmployeeGrade(1, 2, "2026-07-30", "", "hr@test")
	if !errors.Is(err, domain.ErrEmployeeGradeInvalidCloseDate) {
		t.Fatalf("got %v", err)
	}
	if repo.records[5].Status != domain.EmployeeGradeStatusActive {
		t.Fatal("active row must remain ACTIVE after rollback")
	}
}

func TestCreateEmployeeGrade_EarlierStartRejected(t *testing.T) {
	activeStart := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      activeStart,
		Status:         domain.EmployeeGradeStatusActive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	_, err := svc.CreateEmployeeGrade(1, 2, "2026-07-01", "", "hr@test")
	if !errors.Is(err, domain.ErrEmployeeGradeInvalidCloseDate) {
		t.Fatalf("got %v", err)
	}
	if repo.records[5].Status != domain.EmployeeGradeStatusActive {
		t.Fatal("expected rollback keep ACTIVE")
	}
}

func TestCreateEmployeeGrade_CloseFailureRollsBack(t *testing.T) {
	activeStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      activeStart,
		Status:         domain.EmployeeGradeStatusActive,
	})
	repo.failClose = errors.New("close failed")
	svc, _ := newEmployeeGradeServiceForTest(repo)

	_, err := svc.CreateEmployeeGrade(1, 2, "2026-07-30", "", "hr@test")
	if err == nil {
		t.Fatal("expected error")
	}
	if len(repo.records) != 1 || repo.records[5].Status != domain.EmployeeGradeStatusActive {
		t.Fatalf("rollback failed: %#v", repo.records)
	}
}

func TestCreateEmployeeGrade_CreateFailureRollsBackClose(t *testing.T) {
	activeStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      activeStart,
		Status:         domain.EmployeeGradeStatusActive,
	})
	repo.failCreate = errors.New("create failed")
	svc, _ := newEmployeeGradeServiceForTest(repo)

	_, err := svc.CreateEmployeeGrade(1, 2, "2026-07-30", "", "hr@test")
	if err == nil {
		t.Fatal("expected error")
	}
	old := repo.records[5]
	if old.Status != domain.EmployeeGradeStatusActive || old.EndDate != nil {
		t.Fatalf("active must be restored by rollback, got %#v", old)
	}
	if len(repo.records) != 1 {
		t.Fatalf("no new rows expected, got %d", len(repo.records))
	}
}

func TestCreateEmployeeGrade_UniqueViolationMapped(t *testing.T) {
	repo := newStubEmployeeGradeRepo()
	repo.failCreate = &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "ux_hr_employee_grades_employee_id_status_active",
		Message:        "duplicate key",
	}
	svc, _ := newEmployeeGradeServiceForTest(repo)

	_, err := svc.CreateEmployeeGrade(1, 2, "2026-07-30", "", "hr@test")
	if !errors.Is(err, domain.ErrEmployeeGradeActiveConflict) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateEmployeeGrade_PrimaryKeyUniqueNotMappedAsActiveConflict(t *testing.T) {
	repo := newStubEmployeeGradeRepo()
	repo.failCreate = &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "hr_employee_grades_pkey",
		Message:        "duplicate key value violates unique constraint",
	}
	svc, _ := newEmployeeGradeServiceForTest(repo)

	_, err := svc.CreateEmployeeGrade(1, 2, "2026-07-30", "", "hr@test")
	if errors.Is(err, domain.ErrEmployeeGradeActiveConflict) {
		t.Fatal("PK unique must not map to active conflict")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateEmployeeGrade_IgnoresClientEndDate(t *testing.T) {
	repo := newStubEmployeeGradeRepo()
	svc, _ := newEmployeeGradeServiceForTest(repo)

	created, err := svc.CreateEmployeeGrade(1, 2, "2026-07-30", "2026-08-01", "hr@test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.EndDate != nil || created.Status != domain.EmployeeGradeStatusActive {
		t.Fatalf("client end_date must be ignored, got %#v", created)
	}
}

func TestDeleteEmployeeGrade_ActiveSoftDeletesAndClearsCurrent(t *testing.T) {
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeGradeStatusActive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	if err := svc.DeleteEmployeeGrade(5, "hr@test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !repo.records[5].Deleted || repo.records[5].Status != domain.EmployeeGradeStatusInactive {
		t.Fatalf("unexpected delete state %#v", repo.records[5])
	}
	active, err := repo.GetActiveByEmployeeIDForUpdate(1)
	if err != nil {
		t.Fatalf("get current: %v", err)
	}
	if active != nil {
		t.Fatalf("deleted ACTIVE must not remain current: %#v", active)
	}
}

func TestDeleteEmployeeGrade_InactiveSoftDeletes(t *testing.T) {
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        &end,
		Status:         domain.EmployeeGradeStatusInactive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	if err := svc.DeleteEmployeeGrade(5, "hr@test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !repo.records[5].Deleted || repo.records[5].Status != domain.EmployeeGradeStatusInactive {
		t.Fatalf("unexpected delete state %#v", repo.records[5])
	}
}

func TestUpdateEmployeeGrade_ActiveUpdatesInPlace(t *testing.T) {
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeGradeStatusActive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	if err := svc.UpdateEmployeeGrade(5, 1, 2, "2026-01-05", "", "hr@test", 10, true); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := repo.records[5]
	if got.GradeID != 2 || got.Status != domain.EmployeeGradeStatusActive || got.EndDate != nil {
		t.Fatalf("unexpected active update %#v", got)
	}
	activeCount := 0
	for _, record := range repo.records {
		if !record.Deleted && record.Status == domain.EmployeeGradeStatusActive {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Fatalf("active count = %d, want 1", activeCount)
	}
}

func TestUpdateEmployeeGrade_EmployeeIDImmutable(t *testing.T) {
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        &end,
		Status:         domain.EmployeeGradeStatusInactive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	err := svc.UpdateEmployeeGrade(5, 99, 1, "2026-01-01", "2026-06-01", "hr@test", 10, true)
	if !errors.Is(err, domain.ErrEmployeeGradeEmployeeImmutable) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateEmployeeGrade_InactiveAllowsDateCorrection(t *testing.T) {
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        &end,
		Status:         domain.EmployeeGradeStatusInactive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)

	if err := svc.UpdateEmployeeGrade(5, 1, 2, "2026-01-05", "2026-06-15", "hr@test", 10, true); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := repo.records[5]
	if got.GradeID != 2 || got.Status != domain.EmployeeGradeStatusInactive {
		t.Fatalf("unexpected %#v", got)
	}
}

func TestIsEmployeeGradeClientError(t *testing.T) {
	if !IsEmployeeGradeClientError(domain.ErrEmployeeGradeInvalidCloseDate) {
		t.Fatal("expected client error")
	}
	if IsEmployeeGradeClientError(errors.New("db down")) {
		t.Fatal("unexpected client error")
	}
}

func TestCreateEmployeeGrade_EmployeeNotFound(t *testing.T) {
	repo := newStubEmployeeGradeRepo()
	svc, _ := newEmployeeGradeServiceForTest(repo)
	svc.employeeRepo = &stubEGEmployeeRepo{err: gorm.ErrRecordNotFound}

	_, err := svc.CreateEmployeeGrade(1, 2, "2026-07-30", "", "hr@test")
	if !errors.Is(err, domain.ErrEmployeeGradeEmployeeNotFound) {
		t.Fatalf("got %v", err)
	}
}

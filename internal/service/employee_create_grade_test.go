package service

import (
	"errors"
	"testing"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
)

type createEmpRepo struct {
	stubEGEmployeeRepo
	created       []*domain.Employee
	failCreate    error
	inTxGradeRepo *stubEmployeeGradeRepo
}

func (r *createEmpRepo) Create(employee *domain.Employee, createdBy string) error {
	if r.failCreate != nil {
		return r.failCreate
	}
	if employee.ID == 0 {
		employee.ID = uint(len(r.created) + 1)
	}
	employee.CreatedBy = createdBy
	cp := *employee
	r.created = append(r.created, &cp)
	return nil
}

func (r *createEmpRepo) GetLookupList() ([]*domain.Employee, error) { return nil, nil }
func (r *createEmpRepo) InTransaction(fn func(empRepo repository.EmployeeRepository, gradeRepo repository.EmployeeGradeRepository) error) error {
	gradeRepo := r.inTxGradeRepo
	if gradeRepo == nil {
		gradeRepo = newStubEmployeeGradeRepo()
		r.inTxGradeRepo = gradeRepo
	}
	beforeEmp := len(r.created)
	beforeGrades := gradeRepo.cloneRecords()
	beforeNext := gradeRepo.nextID
	err := fn(r, gradeRepo)
	if err != nil {
		r.created = r.created[:beforeEmp]
		gradeRepo.records = beforeGrades
		gradeRepo.nextID = beforeNext
		return err
	}
	return nil
}

func TestCreateEmployee_WithGradeCreatesOneActiveHistory(t *testing.T) {
	empRepo := &createEmpRepo{}
	gradeHist := newStubEmployeeGradeRepo()
	empRepo.inTxGradeRepo = gradeHist

	hire := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	employee := &domain.Employee{
		UserID:    10,
		FirstName: "A",
		LastName:  "B",
		HireDate:  &hire,
	}

	err := empRepo.InTransaction(func(txEmp repository.EmployeeRepository, txGrade repository.EmployeeGradeRepository) error {
		if err := txEmp.Create(employee, "hr@test"); err != nil {
			return err
		}
		eg := &domain.EmployeeGrade{
			EmployeeID: employee.ID,
			GradeID:    2,
			StartDate:  dateOnlyUTC(hire),
			EndDate:    nil,
			Status:     domain.EmployeeGradeStatusActive,
		}
		return txGrade.Create(eg, "hr@test")
	})
	if err != nil {
		t.Fatalf("tx: %v", err)
	}
	if len(empRepo.created) != 1 {
		t.Fatalf("employees=%d", len(empRepo.created))
	}
	active := 0
	for _, r := range gradeHist.records {
		if !r.Deleted && r.Status == domain.EmployeeGradeStatusActive && r.EndDate == nil {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("active history=%d want 1", active)
	}
}

func TestCreateEmployee_WithoutGradeCreatesZeroActiveHistory(t *testing.T) {
	empRepo := &createEmpRepo{}
	gradeHist := newStubEmployeeGradeRepo()
	empRepo.inTxGradeRepo = gradeHist

	employee := &domain.Employee{UserID: 10, FirstName: "A", LastName: "B"}
	err := empRepo.InTransaction(func(txEmp repository.EmployeeRepository, _ repository.EmployeeGradeRepository) error {
		return txEmp.Create(employee, "hr@test")
	})
	if err != nil {
		t.Fatalf("tx: %v", err)
	}
	if len(gradeHist.records) != 0 {
		t.Fatalf("expected 0 history, got %d", len(gradeHist.records))
	}
}

func TestCreateEmployee_GradeCreateFailureRollsBackEmployee(t *testing.T) {
	empRepo := &createEmpRepo{}
	gradeHist := newStubEmployeeGradeRepo()
	gradeHist.failCreate = errors.New("grade insert failed")
	empRepo.inTxGradeRepo = gradeHist

	employee := &domain.Employee{UserID: 10, FirstName: "A", LastName: "B"}
	err := empRepo.InTransaction(func(txEmp repository.EmployeeRepository, txGrade repository.EmployeeGradeRepository) error {
		if err := txEmp.Create(employee, "hr@test"); err != nil {
			return err
		}
		return txGrade.Create(&domain.EmployeeGrade{
			EmployeeID: employee.ID,
			GradeID:    2,
			StartDate:  time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Status:     domain.EmployeeGradeStatusActive,
		}, "hr@test")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(empRepo.created) != 0 {
		t.Fatalf("employee must roll back, got %d", len(empRepo.created))
	}
	if len(gradeHist.records) != 0 {
		t.Fatalf("grade must roll back, got %d", len(gradeHist.records))
	}
}

func TestCreateEmployeeGrade_FirstAssignSucceedsWhenNoActive(t *testing.T) {
	repo := newStubEmployeeGradeRepo()
	svc, _ := newEmployeeGradeServiceForTest(repo)
	created, err := svc.CreateEmployeeGrade(1, 2, "2026-08-06", "", "hr@test")
	if err != nil {
		t.Fatalf("first assign: %v", err)
	}
	if created.Status != domain.EmployeeGradeStatusActive || created.EndDate != nil {
		t.Fatalf("got %#v", created)
	}
}

func TestCreateEmployeeGrade_SecondAssignClosesPreviousActive(t *testing.T) {
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 5},
		EmployeeID:     1,
		GradeID:        1,
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeGradeStatusActive,
	})
	svc, _ := newEmployeeGradeServiceForTest(repo)
	created, err := svc.CreateEmployeeGrade(1, 2, "2026-08-06", "", "hr@test")
	if err != nil {
		t.Fatalf("second assign should close previous: %v", err)
	}
	if created.Status != domain.EmployeeGradeStatusActive {
		t.Fatalf("got %#v", created)
	}
	if repo.records[5].Status != domain.EmployeeGradeStatusInactive {
		t.Fatal("previous must be INACTIVE")
	}
}

func TestHistoryListAndCurrentGrade_SameActiveRow(t *testing.T) {
	repo := newStubEmployeeGradeRepo(&domain.EmployeeGrade{
		AuditableModel: domain.AuditableModel{ID: 9},
		EmployeeID:     1,
		GradeID:        2,
		StartDate:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Status:         domain.EmployeeGradeStatusActive,
		Grade:          domain.Grade{AuditableModel: domain.AuditableModel{ID: 2}, Name: "G2"},
	})
	active, err := repo.GetActiveByEmployeeIDForUpdate(1)
	if err != nil || active == nil || active.ID != 9 {
		t.Fatalf("active=%#v err=%v", active, err)
	}
	var list []domain.EmployeeGrade
	for _, r := range repo.records {
		if !r.Deleted && r.EmployeeID == 1 {
			list = append(list, *r)
		}
	}
	if len(list) != 1 || list[0].ID != active.ID {
		t.Fatalf("history/current mismatch list=%#v active=%#v", list, active)
	}
	emp := &domain.Employee{CurrentEmployeeGrade: active}
	gid, cur := mapCurrentEmployeeGrade(emp)
	if gid == nil || *gid != 2 || cur == nil || cur.GradeID != 2 {
		t.Fatalf("detail current mismatch: %v %#v", gid, cur)
	}
}

package service

import (
	"errors"
	"testing"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type stubContractRepo struct {
	contract    *domain.Contract
	updateCalls int
	createCalls int
	updateErr   error
	createErr   error
	getByIDErr  error
	lastUpdated *domain.Contract
	lastCreated *domain.Contract
}

func (s *stubContractRepo) Create(contract *domain.Contract, createdBy string) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.createCalls++
	if contract.ID == 0 {
		contract.ID = 99
	}
	s.lastCreated = contract
	s.contract = contract
	return nil
}

func (s *stubContractRepo) GetByID(id uint) (*domain.Contract, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	if s.contract == nil {
		return nil, errors.New("contract not found")
	}
	return s.contract, nil
}

func (s *stubContractRepo) GetAll(limit, offset int, sortParams types.SortParams, filters types.ContractFilters) ([]*domain.Contract, int64, error) {
	return nil, 0, nil
}

func (s *stubContractRepo) Update(contract *domain.Contract, modifiedBy string) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updateCalls++
	s.lastUpdated = contract
	s.contract = &domain.Contract{
		AuditableModel:       domain.AuditableModel{ID: contract.ID, CreatedAt: contract.CreatedAt},
		CustomerContactName:  contract.CustomerContactName,
		CustomerContactPhone: contract.CustomerContactPhone,
		CustomerContactEmail: contract.CustomerContactEmail,
		ProjectName:          contract.ProjectName,
		ContractNo:           contract.ContractNo,
		StartDate:            contract.StartDate,
		EndDate:              contract.EndDate,
		Status:               contract.Status,
		EmployeeContracts:    s.contract.EmployeeContracts,
	}
	return nil
}

func (s *stubContractRepo) Delete(id uint, deletedBy string) error { return nil }

type employeeContractKey struct {
	contractID uint
	employeeID uint
}

type stubEmployeeContractRepo struct {
	records     map[employeeContractKey]*domain.EmployeeContract
	createErr   error
	reviveErr   error
	deleteErr   error
	createCalls []employeeContractKey
	reviveCalls []employeeContractKey
	deleteCalls []employeeContractKey
}

func newStubEmployeeContractRepo(records ...*domain.EmployeeContract) *stubEmployeeContractRepo {
	store := make(map[employeeContractKey]*domain.EmployeeContract, len(records))
	for _, record := range records {
		key := employeeContractKey{contractID: record.ContractID, employeeID: record.EmployeeID}
		copyRecord := *record
		store[key] = &copyRecord
	}
	return &stubEmployeeContractRepo{records: store}
}

func (s *stubEmployeeContractRepo) Create(contract *domain.EmployeeContract, createdBy string) error {
	if s.createErr != nil {
		return s.createErr
	}
	key := employeeContractKey{contractID: contract.ContractID, employeeID: contract.EmployeeID}
	s.createCalls = append(s.createCalls, key)
	copyRecord := *contract
	copyRecord.Deleted = false
	copyRecord.CreatedBy = createdBy
	copyRecord.ModifiedBy = createdBy
	s.records[key] = &copyRecord
	return nil
}

func (s *stubEmployeeContractRepo) GetByID(id uint) (*domain.EmployeeContract, error) {
	for _, record := range s.records {
		if record.ID == id {
			copyRecord := *record
			return &copyRecord, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *stubEmployeeContractRepo) GetByContractAndEmployeeIncludingDeleted(contractID uint, employeeID uint) (*domain.EmployeeContract, error) {
	key := employeeContractKey{contractID: contractID, employeeID: employeeID}
	record, ok := s.records[key]
	if !ok {
		return nil, nil
	}
	copyRecord := *record
	return &copyRecord, nil
}

func (s *stubEmployeeContractRepo) GetByEmployeeID(employeeID uint, page int, limit int) ([]*domain.EmployeeContract, int64, error) {
	return nil, 0, nil
}

func (s *stubEmployeeContractRepo) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.EmployeeContract, int64, error) {
	return nil, 0, nil
}

func (s *stubEmployeeContractRepo) CheckExists(employeeID uint, contractID uint) (bool, error) {
	record, ok := s.records[employeeContractKey{contractID: contractID, employeeID: employeeID}]
	return ok && !record.Deleted, nil
}

func (s *stubEmployeeContractRepo) ReviveByContractAndEmployee(contractID uint, employeeID uint, modifiedBy string) error {
	if s.reviveErr != nil {
		return s.reviveErr
	}
	key := employeeContractKey{contractID: contractID, employeeID: employeeID}
	record, ok := s.records[key]
	if !ok {
		return errors.New("record not found")
	}
	record.Deleted = false
	record.ModifiedBy = modifiedBy
	s.reviveCalls = append(s.reviveCalls, key)
	return nil
}

func (s *stubEmployeeContractRepo) Update(contract *domain.EmployeeContract, modifiedBy string) error {
	return nil
}

func (s *stubEmployeeContractRepo) Delete(id uint, deletedBy string) error { return nil }

func (s *stubEmployeeContractRepo) DeleteByContractAndEmployee(contractID uint, employeeID uint, deletedBy string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	key := employeeContractKey{contractID: contractID, employeeID: employeeID}
	record, ok := s.records[key]
	if !ok {
		record = &domain.EmployeeContract{ContractID: contractID, EmployeeID: employeeID}
		s.records[key] = record
	}
	record.Deleted = true
	record.ModifiedBy = deletedBy
	s.deleteCalls = append(s.deleteCalls, key)
	return nil
}

func (s *stubEmployeeContractRepo) GetTotalCount() (int64, error) { return int64(len(s.records)), nil }

type stubAuditForContracts struct{}

func (s *stubAuditForContracts) CreateAuditLog(entityName string, entityID uint, action string, oldValue, newValue interface{}, performedBy string) error {
	return nil
}

var _ repository.ContractRepository = (*stubContractRepo)(nil)
var _ repository.EmployeeContractRepository = (*stubEmployeeContractRepo)(nil)

func newContractServiceForTest(contract *domain.Contract, employeeRecords ...*domain.EmployeeContract) (*contractService, *stubContractRepo, *stubEmployeeContractRepo) {
	contractRepo := &stubContractRepo{contract: contract}
	employeeRepo := newStubEmployeeContractRepo(employeeRecords...)
	svc := &contractService{
		contractRepo:         contractRepo,
		employeeContractRepo: employeeRepo,
		auditService:         &stubAuditForContracts{},
	}
	return svc, contractRepo, employeeRepo
}

func contractWithEmployees(contractID uint, employeeIDs ...uint) *domain.Contract {
	employeeContracts := make([]domain.EmployeeContract, 0, len(employeeIDs))
	for _, employeeID := range employeeIDs {
		employeeContracts = append(employeeContracts, domain.EmployeeContract{
			AuditableModel: domain.AuditableModel{ID: employeeID},
			ContractID:     contractID,
			EmployeeID:     employeeID,
		})
	}

	return &domain.Contract{
		AuditableModel:      domain.AuditableModel{ID: contractID, CreatedAt: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)},
		CustomerContactName: "Jane Customer",
		ProjectName:         "Migration",
		ContractNo:          "CN-001",
		StartDate:           time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Status:              domain.ContractStatusApproved,
		EmployeeContracts:   employeeContracts,
	}
}

func TestSyncTargetEmployeesCreatesNewAssignment(t *testing.T) {
	svc, _, employeeRepo := newContractServiceForTest(
		contractWithEmployees(10, 1, 2),
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 1},
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 2},
	)

	if err := svc.syncTargetEmployees(10, []uint{1, 2, 3}, "7"); err != nil {
		t.Fatalf("syncTargetEmployees returned error: %v", err)
	}

	if len(employeeRepo.createCalls) != 1 || employeeRepo.createCalls[0] != (employeeContractKey{contractID: 10, employeeID: 3}) {
		t.Fatalf("expected one create call for employee 3, got %#v", employeeRepo.createCalls)
	}
	if len(employeeRepo.deleteCalls) != 0 {
		t.Fatalf("expected no delete calls, got %#v", employeeRepo.deleteCalls)
	}
}

func TestSyncTargetEmployeesRevivesSoftDeletedAssignment(t *testing.T) {
	svc, _, employeeRepo := newContractServiceForTest(
		contractWithEmployees(10, 1, 2),
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 1},
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 2},
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 3, AuditableModel: domain.AuditableModel{Deleted: true}},
	)

	if err := svc.syncTargetEmployees(10, []uint{1, 2, 3}, "7"); err != nil {
		t.Fatalf("syncTargetEmployees returned error: %v", err)
	}

	if len(employeeRepo.createCalls) != 0 {
		t.Fatalf("expected no create calls for soft-deleted employee, got %#v", employeeRepo.createCalls)
	}
	if len(employeeRepo.reviveCalls) != 1 || employeeRepo.reviveCalls[0] != (employeeContractKey{contractID: 10, employeeID: 3}) {
		t.Fatalf("expected revive call for employee 3, got %#v", employeeRepo.reviveCalls)
	}
	if employeeRepo.records[employeeContractKey{contractID: 10, employeeID: 3}].Deleted {
		t.Fatal("expected revived employee contract to be active")
	}
}

func TestSyncTargetEmployeesSoftDeletesRemovedAssignment(t *testing.T) {
	svc, _, employeeRepo := newContractServiceForTest(
		contractWithEmployees(10, 1, 2),
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 1},
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 2},
	)

	if err := svc.syncTargetEmployees(10, []uint{1}, "7"); err != nil {
		t.Fatalf("syncTargetEmployees returned error: %v", err)
	}

	if len(employeeRepo.deleteCalls) != 1 || employeeRepo.deleteCalls[0] != (employeeContractKey{contractID: 10, employeeID: 2}) {
		t.Fatalf("expected delete call for employee 2, got %#v", employeeRepo.deleteCalls)
	}
	if employeeRepo.records[employeeContractKey{contractID: 10, employeeID: 1}].Deleted {
		t.Fatal("expected employee 1 to remain active")
	}
}

func TestSyncTargetEmployeesNoOpWhenTargetMatchesActiveSet(t *testing.T) {
	svc, _, employeeRepo := newContractServiceForTest(
		contractWithEmployees(10, 1, 2),
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 1},
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 2},
	)

	if err := svc.syncTargetEmployees(10, []uint{1, 2}, "7"); err != nil {
		t.Fatalf("syncTargetEmployees returned error: %v", err)
	}

	if len(employeeRepo.createCalls) != 0 || len(employeeRepo.reviveCalls) != 0 || len(employeeRepo.deleteCalls) != 0 {
		t.Fatalf("expected no repository mutations, got create=%#v revive=%#v delete=%#v", employeeRepo.createCalls, employeeRepo.reviveCalls, employeeRepo.deleteCalls)
	}
}

func TestSyncTargetEmployeesPropagatesCreateAndReviveErrors(t *testing.T) {
	t.Run("create error", func(t *testing.T) {
		svc, _, employeeRepo := newContractServiceForTest(
			contractWithEmployees(10, 1, 2),
			&domain.EmployeeContract{ContractID: 10, EmployeeID: 1},
			&domain.EmployeeContract{ContractID: 10, EmployeeID: 2},
		)
		employeeRepo.createErr = errors.New("create failed")

		err := svc.syncTargetEmployees(10, []uint{1, 2, 3}, "7")
		if err == nil || err.Error() != "create failed" {
			t.Fatalf("expected create error to be returned, got %v", err)
		}
	})

	t.Run("revive error", func(t *testing.T) {
		svc, _, employeeRepo := newContractServiceForTest(
			contractWithEmployees(10, 1, 2),
			&domain.EmployeeContract{ContractID: 10, EmployeeID: 1},
			&domain.EmployeeContract{ContractID: 10, EmployeeID: 2},
			&domain.EmployeeContract{ContractID: 10, EmployeeID: 3, AuditableModel: domain.AuditableModel{Deleted: true}},
		)
		employeeRepo.reviveErr = errors.New("revive failed")

		err := svc.syncTargetEmployees(10, []uint{1, 2, 3}, "7")
		if err == nil || err.Error() != "revive failed" {
			t.Fatalf("expected revive error to be returned, got %v", err)
		}
	})
}

func TestSyncTargetEmployeesDeduplicatesTargetIDs(t *testing.T) {
	svc, _, employeeRepo := newContractServiceForTest(
		contractWithEmployees(10, 1, 2),
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 1},
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 2},
	)

	if err := svc.syncTargetEmployees(10, []uint{1, 1, 2, 2, 3, 3}, "7"); err != nil {
		t.Fatalf("syncTargetEmployees returned error: %v", err)
	}

	if len(employeeRepo.createCalls) != 1 {
		t.Fatalf("expected one create call for duplicate target IDs, got %#v", employeeRepo.createCalls)
	}
}

func TestUpdateContractPropagatesSyncErrors(t *testing.T) {
	contract := contractWithEmployees(10, 1, 2)
	svc, contractRepo, employeeRepo := newContractServiceForTest(
		contract,
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 1},
		&domain.EmployeeContract{ContractID: 10, EmployeeID: 2},
	)
	employeeRepo.createErr = errors.New("sync failed")

	err := svc.UpdateContract(10, types.ContractRequest{
		CustomerContactName: "Jane Customer",
		ProjectName:         "Migration",
		ContractNo:          "CN-001",
		StartDate:           "2026-07-01",
		Status:              domain.ContractStatusApproved,
		TargetEmployeeIDs:   []uint{1, 2, 3},
	}, "7")
	if err == nil || err.Error() != "sync failed" {
		t.Fatalf("expected sync error to be returned, got %v", err)
	}
	if contractRepo.updateCalls != 1 {
		t.Fatalf("expected contract update to run once, got %d", contractRepo.updateCalls)
	}
}

func TestCreateContractPropagatesSyncErrors(t *testing.T) {
	svc, contractRepo, employeeRepo := newContractServiceForTest(nil)
	employeeRepo.createErr = errors.New("sync failed")

	_, err := svc.CreateContract(types.ContractRequest{
		CustomerContactName: "Jane Customer",
		ProjectName:         "Migration",
		ContractNo:          "CN-001",
		StartDate:           "2026-07-01",
		Status:              domain.ContractStatusApproved,
		TargetEmployeeIDs:   []uint{3},
	}, "7")
	if err == nil || err.Error() != "sync failed" {
		t.Fatalf("expected sync error to be returned, got %v", err)
	}
	if contractRepo.createCalls != 1 {
		t.Fatalf("expected contract create to run once, got %d", contractRepo.createCalls)
	}
}

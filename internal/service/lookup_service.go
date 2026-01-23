package service

import (
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

// LookupService provides centralized lookup data for all domains
// This service exposes public APIs for dropdowns and selects
type LookupService interface {
	GetCompaniesLookup() ([]types.CompanyLookup, error)
	GetDepartmentsLookup() ([]types.DepartmentLookup, error)
	GetDepartmentsByCompanyLookup(companyID uint) ([]types.DepartmentLookup, error)
	GetJobPositionsLookup() ([]types.JobPositionLookup, error)
	GetLeaveTypesLookup() ([]types.LeaveTypeLookup, error)
	GetGradesLookup() ([]types.GradeLookup, error)
}

type lookupService struct {
	companyRepo     repository.CompanyRepository
	departmentRepo  repository.DepartmentRepository
	jobPositionRepo repository.JobPositionRepository
	leaveTypeRepo   repository.LeaveTypeRepository
	gradeRepo       repository.GradeRepository
}

func NewLookupService(
	companyRepo repository.CompanyRepository,
	departmentRepo repository.DepartmentRepository,
	jobPositionRepo repository.JobPositionRepository,
	leaveTypeRepo repository.LeaveTypeRepository,
	gradeRepo repository.GradeRepository,
) LookupService {
	return &lookupService{
		companyRepo:     companyRepo,
		departmentRepo:  departmentRepo,
		jobPositionRepo: jobPositionRepo,
		leaveTypeRepo:   leaveTypeRepo,
		gradeRepo:       gradeRepo,
	}
}

// GetCompaniesLookup returns all companies as lookup data
func (s *lookupService) GetCompaniesLookup() ([]types.CompanyLookup, error) {
	companies, err := s.companyRepo.GetLookup()
	if err != nil {
		return nil, err
	}

	lookupData := make([]types.CompanyLookup, len(companies))
	for i, company := range companies {
		lookupData[i] = types.CompanyLookup{
			ID:   company.ID,
			Name: company.Name,
		}
	}

	return lookupData, nil
}

// GetDepartmentsLookup returns all departments as lookup data
func (s *lookupService) GetDepartmentsLookup() ([]types.DepartmentLookup, error) {
	departments, _, err := s.departmentRepo.GetAll(1000, 0, types.SortParams{Sort: "name", Direction: "ASC"})
	if err != nil {
		return nil, err
	}

	lookupData := make([]types.DepartmentLookup, len(departments))
	for i, department := range departments {
		lookupData[i] = types.DepartmentLookup{
			ID:      department.ID,
			Name:    department.Name,
			Manager: department.Manager,
		}
	}

	return lookupData, nil
}

// GetDepartmentsByCompanyLookup returns departments filtered by company as lookup data
func (s *lookupService) GetDepartmentsByCompanyLookup(companyID uint) ([]types.DepartmentLookup, error) {
	departments, err := s.departmentRepo.GetByCompanyID(companyID)
	if err != nil {
		return nil, err
	}

	lookupData := make([]types.DepartmentLookup, len(departments))
	for i, department := range departments {
		lookupData[i] = types.DepartmentLookup{
			ID:      department.ID,
			Name:    department.Name,
			Manager: department.Manager,
		}
	}

	return lookupData, nil
}

// GetJobPositionsLookup returns all job positions as lookup data
func (s *lookupService) GetJobPositionsLookup() ([]types.JobPositionLookup, error) {
	jobPositions, err := s.jobPositionRepo.GetLookup()
	if err != nil {
		return nil, err
	}

	lookupData := make([]types.JobPositionLookup, len(jobPositions))
	for i, jobPosition := range jobPositions {
		lookupData[i] = types.JobPositionLookup{
			ID:    jobPosition.ID,
			Title: jobPosition.Title,
		}
	}

	return lookupData, nil
}

// GetLeaveTypesLookup returns all leave types as lookup data
func (s *lookupService) GetLeaveTypesLookup() ([]types.LeaveTypeLookup, error) {
	leaveTypes, err := s.leaveTypeRepo.GetLookup()
	if err != nil {
		return nil, err
	}

	lookupData := make([]types.LeaveTypeLookup, len(leaveTypes))
	for i, leaveType := range leaveTypes {
		lookupData[i] = types.LeaveTypeLookup{
			ID:   leaveType.ID,
			Name: leaveType.Name,
		}
	}

	return lookupData, nil
}

// GetGradesLookup returns all grades as lookup data
func (s *lookupService) GetGradesLookup() ([]types.GradeLookup, error) {
	grades, err := s.gradeRepo.GetLookup()
	if err != nil {
		return nil, err
	}

	lookupData := make([]types.GradeLookup, len(grades))
	for i, grade := range grades {
		lookupData[i] = types.GradeLookup{
			ID:   grade.ID,
			Name: grade.Name,
		}
	}

	return lookupData, nil
}

package service

import (
"errors"
"fmt"
"kartezya-hr/internal/domain"
"kartezya-hr/internal/repository"
"kartezya-hr/internal/types"
)

type GradeService interface {
	CreateGrade(grade *domain.Grade, userID uint) error
	GetGradeByID(id int64) (*types.GradeResponse, error)
	GetAllGrades(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	UpdateGrade(id int64, grade *domain.Grade, userID uint) error
	DeleteGrade(id int64, userID uint) error
	GetTotalCount() (int64, error)
}

type gradeService struct {
	gradeRepo   repository.GradeRepository
	auditService AuditService
}

func NewGradeService(gradeRepo repository.GradeRepository, auditService AuditService) GradeService {
	return &gradeService{
		gradeRepo:    gradeRepo,
		auditService: auditService,
	}
}

func (s *gradeService) CreateGrade(grade *domain.Grade, userID uint) error {
	// Validation
	if grade.Name == "" {
		return errors.New("grade name is required")
	}

	// Check if a grade with the same name already exists
	existingGrade, err := s.gradeRepo.GetByName(grade.Name)
	if err == nil && existingGrade != nil {
		return errors.New("grade with this name already exists")
	}

	// Create audit identifier
	createdBy := fmt.Sprintf("%d", userID)

	// Create the grade
	if err := s.gradeRepo.Create(grade, createdBy); err != nil {
		return err
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("Grade", grade.ID, domain.AuditActionCreate, nil, grade, createdBy); err != nil {
		// Log error but don't fail the operation
}

return nil
}

func (s *gradeService) GetGradeByID(id int64) (*types.GradeResponse, error) {
if id == 0 {
return nil, errors.New("invalid grade ID")
}

grade, err := s.gradeRepo.GetByID(id)
if err != nil {
return nil, err
}

return &types.GradeResponse{
ID:          grade.ID,
CreatedAt:   grade.CreatedAt,
UpdatedAt:   grade.UpdatedAt,
Deleted:     grade.Deleted,
CreatedBy:   grade.CreatedBy,
ModifiedBy:  grade.ModifiedBy,
Name:        grade.Name,
Description: grade.Description,
}, nil
}

func (s *gradeService) GetAllGrades(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
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
grades, total, err := s.gradeRepo.GetAll(limit, offset, sortParams)
if err != nil {
return nil, err
}

// Transform domain models to response DTOs
gradeResponses := make([]types.GradeResponse, len(grades))
for i, grade := range grades {
gradeResponses[i] = types.GradeResponse{
ID:          grade.ID,
CreatedAt:   grade.CreatedAt,
UpdatedAt:   grade.UpdatedAt,
Deleted:     grade.Deleted,
CreatedBy:   grade.CreatedBy,
ModifiedBy:  grade.ModifiedBy,
Name:        grade.Name,
Description: grade.Description,
}
}

return &PaginatedResponse{
Data: gradeResponses,
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

func (s *gradeService) UpdateGrade(id int64, grade *domain.Grade, userID uint) error {
if id == 0 {
return errors.New("invalid grade ID")
}

// Check if grade exists and get old value for audit
existingGrade, err := s.gradeRepo.GetByID(id)
if err != nil {
return errors.New("grade not found")
}
if existingGrade == nil {
return errors.New("grade not found")
}

// Validation
if grade.Name == "" {
return errors.New("grade name is required")
}

// Check if the new name is different from the current name
if existingGrade.Name != grade.Name {
// Check if another grade with the same name already exists
existingWithName, err := s.gradeRepo.GetByName(grade.Name)
if err == nil && existingWithName != nil && existingWithName.ID != existingGrade.ID {
return errors.New("grade with this name already exists")
}
}

// Create audit identifier
modifiedBy := fmt.Sprintf("%d", userID)

// Clone the existing grade and update only the provided fields
updatedGrade := *existingGrade
updatedGrade.Name = grade.Name
updatedGrade.Description = grade.Description

// Update the grade
if err := s.gradeRepo.Update(&updatedGrade, modifiedBy); err != nil {
return err
}

// Get updated grade for audit
auditedGrade, _ := s.gradeRepo.GetByID(id)

// Audit the update
if err := s.auditService.CreateAuditLog("Grade", uint(id), domain.AuditActionUpdate, existingGrade, auditedGrade, modifiedBy); err != nil {
// Log error but don't fail the operation
	}

	return nil
}

func (s *gradeService) DeleteGrade(id int64, userID uint) error {
	if id == 0 {
		return errors.New("invalid grade ID")
	}

	// Check if grade exists and get old value for audit
	existingGrade, err := s.gradeRepo.GetByID(id)
	if err != nil {
		return errors.New("grade not found")
	}
	if existingGrade == nil {
		return errors.New("grade not found")
	}

	// Create audit identifier
	deletedBy := fmt.Sprintf("%d", userID)

	// Delete the grade
	if err := s.gradeRepo.Delete(id, deletedBy); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("Grade", uint(id), domain.AuditActionDelete, existingGrade, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
}

return nil
}

// GetTotalCount returns the total number of grades
func (s *gradeService) GetTotalCount() (int64, error) {
count, err := s.gradeRepo.GetTotalCount()
if err != nil {
return 0, err
}
return count, nil
}

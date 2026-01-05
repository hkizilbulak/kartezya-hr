package service

import (
	"errors"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type JobPositionService interface {
	CreateJobPosition(jobPosition *domain.JobPosition, createdBy string) error
	GetJobPositionByID(id uint) (*types.JobPositionResponse, error)
	GetAllJobPositions(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error)
	GetJobPositionsLookup() ([]types.JobPositionLookup, error)
	UpdateJobPosition(id uint, title string, modifiedBy string) error
	DeleteJobPosition(id uint, deletedBy string) error
}

type jobPositionService struct {
	jobPositionRepo repository.JobPositionRepository
	auditService    AuditService
}

func NewJobPositionService(jobPositionRepo repository.JobPositionRepository, auditService AuditService) JobPositionService {
	return &jobPositionService{
		jobPositionRepo: jobPositionRepo,
		auditService:    auditService,
	}
}

func (s *jobPositionService) CreateJobPosition(jobPosition *domain.JobPosition, createdBy string) error {
	if jobPosition.Title == "" {
		return errors.New("job position title is required")
	}

	// Check if a job position with the same title already exists
	existingJobPosition, err := s.jobPositionRepo.GetByTitle(jobPosition.Title)
	if err == nil && existingJobPosition != nil {
		return errors.New("job position with this title already exists")
	}

	// Create the job position
	if err := s.jobPositionRepo.Create(jobPosition, createdBy); err != nil {
		return err
	}

	// Audit the creation
	if err := s.auditService.CreateAuditLog("JobPosition", jobPosition.ID, domain.AuditActionCreate, nil, jobPosition, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *jobPositionService) GetJobPositionByID(id uint) (*types.JobPositionResponse, error) {
	if id == 0 {
		return nil, errors.New("invalid job position ID")
	}

	jobPosition, err := s.jobPositionRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Convert domain model to response DTO
	response := &types.JobPositionResponse{
		ID:         jobPosition.ID,
		CreatedAt:  jobPosition.CreatedAt.Format("2006-01-02T15:04:05.000000Z07:00"),
		UpdatedAt:  jobPosition.UpdatedAt.Format("2006-01-02T15:04:05.000000Z07:00"),
		Deleted:    jobPosition.Deleted,
		CreatedBy:  jobPosition.CreatedBy,
		ModifiedBy: jobPosition.ModifiedBy,
		Title:      jobPosition.Title,
	}

	return response, nil
}

func (s *jobPositionService) GetAllJobPositions(page, limit int, sortParams types.SortParams) (*PaginatedResponse, error) {
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
	jobPositions, total, err := s.jobPositionRepo.GetAll(limit, offset, sortParams)
	if err != nil {
		return nil, err
	}

	// Convert domain models to response DTOs
	jobPositionResponses := make([]types.JobPositionResponse, len(jobPositions))
	for i, jobPosition := range jobPositions {
		jobPositionResponses[i] = types.JobPositionResponse{
			ID:         jobPosition.ID,
			CreatedAt:  jobPosition.CreatedAt.Format("2006-01-02T15:04:05.000000Z07:00"),
			UpdatedAt:  jobPosition.UpdatedAt.Format("2006-01-02T15:04:05.000000Z07:00"),
			Deleted:    jobPosition.Deleted,
			CreatedBy:  jobPosition.CreatedBy,
			ModifiedBy: jobPosition.ModifiedBy,
			Title:      jobPosition.Title,
		}
	}

	return &PaginatedResponse{
		Data: jobPositionResponses,
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

func (s *jobPositionService) GetJobPositionsLookup() ([]types.JobPositionLookup, error) {
	jobPositions, err := s.jobPositionRepo.GetLookup()
	if err != nil {
		return nil, err
	}

	// Convert to lookup DTOs
	lookupData := make([]types.JobPositionLookup, len(jobPositions))
	for i, jobPosition := range jobPositions {
		lookupData[i] = types.JobPositionLookup{
			ID:    jobPosition.ID,
			Title: jobPosition.Title,
		}
	}

	return lookupData, nil
}

func (s *jobPositionService) UpdateJobPosition(id uint, title string, modifiedBy string) error {
	// Get existing job position for audit trail and validation
	existingJobPosition, err := s.jobPositionRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check if the new title is different from the current title
	if existingJobPosition.Title != title {
		// Check if another job position with the same title already exists
		existingWithTitle, err := s.jobPositionRepo.GetByTitle(title)
		if err == nil && existingWithTitle != nil && existingWithTitle.ID != id {
			return errors.New("job position with this title already exists")
		}
	}

	// Update the job position
	existingJobPosition.Title = title

	if err := s.jobPositionRepo.Update(existingJobPosition, modifiedBy); err != nil {
		return err
	}

	// Get updated job position for audit
	updatedJobPosition, _ := s.jobPositionRepo.GetByID(id)

	// Audit the update
	if err := s.auditService.CreateAuditLog("JobPosition", id, domain.AuditActionUpdate, existingJobPosition, updatedJobPosition, modifiedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

func (s *jobPositionService) DeleteJobPosition(id uint, deletedBy string) error {
	if id == 0 {
		return errors.New("invalid job position ID")
	}

	// Get existing job position for audit trail
	existingJobPosition, err := s.jobPositionRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete the job position
	if err := s.jobPositionRepo.Delete(id, deletedBy); err != nil {
		return err
	}

	// Audit the deletion
	if err := s.auditService.CreateAuditLog("JobPosition", id, domain.AuditActionDelete, existingJobPosition, nil, deletedBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

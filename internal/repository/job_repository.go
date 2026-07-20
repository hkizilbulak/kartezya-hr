package repository

import (
	"errors"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
	"time"

	"gorm.io/gorm"
)

type JobRepository interface {
	// Job methods
	Create(job *domain.Job) error
	Update(job *domain.Job) error
	GetByID(id uint) (*domain.Job, error)
	GetByKey(key string) (*domain.Job, error)
	GetAll(sortParams types.SortParams) ([]domain.Job, error)
	GetActiveJobs() ([]domain.Job, error)

	// JobHistory methods
	CreateHistory(history *domain.JobHistory) error
	UpdateHistory(history *domain.JobHistory) error
	GetHistoryByJobID(jobID uint, limit int) ([]domain.JobHistory, error)
	HasHistoryForReferenceDate(jobID uint, referenceDate time.Time, statuses []string) (bool, error)
}

type jobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) JobRepository {
	return &jobRepository{db: db}
}

func (r *jobRepository) Create(job *domain.Job) error {
	return r.db.Create(job).Error
}

func (r *jobRepository) Update(job *domain.Job) error {
	return r.db.Save(job).Error
}

func (r *jobRepository) GetByID(id uint) (*domain.Job, error) {
	var job domain.Job
	if err := r.db.First(&job, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

func (r *jobRepository) GetByKey(key string) (*domain.Job, error) {
	var job domain.Job
	if err := r.db.Where("job_key = ?", key).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

func (r *jobRepository) GetAll(sortParams types.SortParams) ([]domain.Job, error) {
	var jobs []domain.Job
	query := r.db.Model(&domain.Job{}).
		Order(buildJobOrderClause(sortParams.Sort, sortParams.Direction))

	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *jobRepository) GetActiveJobs() ([]domain.Job, error) {
	var jobs []domain.Job
	if err := r.db.Where("is_active = ?", true).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *jobRepository) CreateHistory(history *domain.JobHistory) error {
	return r.db.Create(history).Error
}

func (r *jobRepository) UpdateHistory(history *domain.JobHistory) error {
	return r.db.Save(history).Error
}

func (r *jobRepository) GetHistoryByJobID(jobID uint, limit int) ([]domain.JobHistory, error) {
	var history []domain.JobHistory
	query := r.db.Where("job_id = ?", jobID).Order("start_time desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&history).Error; err != nil {
		return nil, err
	}
	return history, nil
}

func (r *jobRepository) HasHistoryForReferenceDate(jobID uint, referenceDate time.Time, statuses []string) (bool, error) {
	if len(statuses) == 0 {
		return false, nil
	}

	dateOnly := time.Date(referenceDate.Year(), referenceDate.Month(), referenceDate.Day(), 0, 0, 0, 0, time.UTC)

	var count int64
	err := r.db.Model(&domain.JobHistory{}).
		Where("job_id = ? AND reference_date = ? AND status IN ?", jobID, dateOnly, statuses).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

package repository

import (
	"time"

	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// Training Repository
// ─────────────────────────────────────────────────────────────────────────────

type TrainingRepository interface {
	Create(training *domain.Training, createdBy string) error
	GetByID(id uint) (*domain.Training, error)
	ListActive(limit, offset int) ([]*domain.Training, int64, error)
	ListAll(limit, offset int) ([]*domain.Training, int64, error)
	Update(training *domain.Training, modifiedBy string) error
	Delete(id uint, deletedBy string) error
}

type trainingRepository struct {
	db *gorm.DB
}

func NewTrainingRepository(db *gorm.DB) TrainingRepository {
	return &trainingRepository{db: db}
}

func (r *trainingRepository) Create(t *domain.Training, createdBy string) error {
	t.CreatedBy = createdBy
	t.ModifiedBy = createdBy
	return r.db.Create(t).Error
}

func (r *trainingRepository) GetByID(id uint) (*domain.Training, error) {
	var t domain.Training
	err := r.db.Where("id = ? AND deleted = ?", id, false).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *trainingRepository) ListActive(limit, offset int) ([]*domain.Training, int64, error) {
	var list []*domain.Training
	var total int64
	q := r.db.Model(&domain.Training{}).Where("deleted = ? AND status = ?", false, domain.TrainingStatusActive)
	q.Count(&total)
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

func (r *trainingRepository) ListAll(limit, offset int) ([]*domain.Training, int64, error) {
	var list []*domain.Training
	var total int64
	q := r.db.Model(&domain.Training{}).Where("deleted = ?", false)
	q.Count(&total)
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

func (r *trainingRepository) Update(t *domain.Training, modifiedBy string) error {
	t.ModifiedBy = modifiedBy
	return r.db.Where("deleted = ?", false).Save(t).Error
}

func (r *trainingRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.Training{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

// ─────────────────────────────────────────────────────────────────────────────
// Assignment Repository
// ─────────────────────────────────────────────────────────────────────────────

type AssignmentRepository interface {
	Create(a *domain.TrainingAssignment, createdBy string) error
	GetByID(id uint) (*domain.TrainingAssignment, error)
	GetByEmployeeAndTraining(employeeID, trainingID uint) (*domain.TrainingAssignment, error)
	ListByEmployee(employeeID uint) ([]*domain.TrainingAssignment, error)
	ListByTraining(trainingID uint, limit, offset int) ([]*domain.TrainingAssignment, int64, error)
	UpdateStatus(id uint, status domain.AssignmentStatus, startedAt, completedAt *time.Time, modifiedBy string) error
	Delete(id uint, deletedBy string) error
}

type assignmentRepository struct {
	db *gorm.DB
}

func NewAssignmentRepository(db *gorm.DB) AssignmentRepository {
	return &assignmentRepository{db: db}
}

func (r *assignmentRepository) Create(a *domain.TrainingAssignment, createdBy string) error {
	a.CreatedBy = createdBy
	a.ModifiedBy = createdBy
	return r.db.Create(a).Error
}

func (r *assignmentRepository) GetByID(id uint) (*domain.TrainingAssignment, error) {
	var a domain.TrainingAssignment
	err := r.db.Preload("Training").Preload("Employee").
		Where("id = ? AND deleted = ?", id, false).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *assignmentRepository) GetByEmployeeAndTraining(employeeID, trainingID uint) (*domain.TrainingAssignment, error) {
	var a domain.TrainingAssignment
	err := r.db.Where("employee_id = ? AND training_id = ? AND deleted = ?", employeeID, trainingID, false).
		First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *assignmentRepository) ListByEmployee(employeeID uint) ([]*domain.TrainingAssignment, error) {
	var list []*domain.TrainingAssignment
	
	trainingsTable := domain.GetTableName("trainings")
	assignmentsTable := domain.GetTableName("training_assignments")
	
	err := r.db.Preload("Training").
		Joins("JOIN " + trainingsTable + " ON " + trainingsTable + ".id = " + assignmentsTable + ".training_id").
		Where(assignmentsTable + ".employee_id = ? AND " + assignmentsTable + ".deleted = ?", employeeID, false).
		Where(trainingsTable + ".deleted = ?", false).
		Order(assignmentsTable + ".created_at DESC").Find(&list).Error
		
	return list, err
}

func (r *assignmentRepository) ListByTraining(trainingID uint, limit, offset int) ([]*domain.TrainingAssignment, int64, error) {
	var list []*domain.TrainingAssignment
	var total int64
	q := r.db.Model(&domain.TrainingAssignment{}).Preload("Employee").
		Where("training_id = ? AND deleted = ?", trainingID, false)
	q.Count(&total)
	err := q.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

func (r *assignmentRepository) UpdateStatus(id uint, status domain.AssignmentStatus, startedAt, completedAt *time.Time, modifiedBy string) error {
	updates := map[string]interface{}{
		"status":      status,
		"modified_by": modifiedBy,
	}
	if startedAt != nil {
		updates["started_at"] = startedAt
	}
	if completedAt != nil {
		updates["completed_at"] = completedAt
	}
	return r.db.Model(&domain.TrainingAssignment{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(updates).Error
}

func (r *assignmentRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.TrainingAssignment{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"modified_by": deletedBy,
		}).Error
}

// ─────────────────────────────────────────────────────────────────────────────
// Certificate Repository
// ─────────────────────────────────────────────────────────────────────────────

type CertificateRepository interface {
	Create(cert *domain.TrainingCertificate, createdBy string) error
	GetByCode(code string) (*domain.TrainingCertificate, error)
	GetByAssignmentID(assignmentID uint) (*domain.TrainingCertificate, error)
	ListByEmployee(employeeID uint) ([]*domain.TrainingCertificate, error)
}

type certificateRepository struct {
	db *gorm.DB
}

func NewCertificateRepository(db *gorm.DB) CertificateRepository {
	return &certificateRepository{db: db}
}

func (r *certificateRepository) Create(cert *domain.TrainingCertificate, createdBy string) error {
	cert.CreatedBy = createdBy
	cert.ModifiedBy = createdBy
	return r.db.Create(cert).Error
}

func (r *certificateRepository) GetByCode(code string) (*domain.TrainingCertificate, error) {
	var cert domain.TrainingCertificate
	err := r.db.Preload("Employee").Preload("Training").
		Where("certificate_code = ? AND deleted = ?", code, false).First(&cert).Error
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func (r *certificateRepository) GetByAssignmentID(assignmentID uint) (*domain.TrainingCertificate, error) {
	var cert domain.TrainingCertificate
	err := r.db.Preload("Employee").Preload("Training").
		Where("assignment_id = ? AND deleted = ?", assignmentID, false).First(&cert).Error
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func (r *certificateRepository) ListByEmployee(employeeID uint) ([]*domain.TrainingCertificate, error) {
	var list []*domain.TrainingCertificate
	err := r.db.Preload("Training").
		Where("employee_id = ? AND deleted = ?", employeeID, false).
		Order("issued_at DESC").Find(&list).Error
	return list, err
}

// ─────────────────────────────────────────────────────────────────────────────
// Academy Survey Repository
// ─────────────────────────────────────────────────────────────────────────────

type AcademySurveyRepository interface {
	Create(survey *domain.AcademySurvey, createdBy string) error
	GetByID(id uint) (*domain.AcademySurvey, error)
	ListAll(limit, offset int) ([]*domain.AcademySurvey, int64, error)
	ListActive(limit, offset int) ([]*domain.AcademySurvey, int64, error)
	Delete(id uint, deletedBy string) error
	SubmitResponse(responses []domain.AcademySurveyResponse) error
	HasUserResponded(surveyID, employeeID uint) (bool, error)
	GetSurveyResults(surveyID uint) (map[uint]int, error) // map[OptionID]voteCount
}

type academySurveyRepository struct {
	db *gorm.DB
}

func NewAcademySurveyRepository(db *gorm.DB) AcademySurveyRepository {
	return &academySurveyRepository{db: db}
}

func (r *academySurveyRepository) Create(survey *domain.AcademySurvey, createdBy string) error {
	survey.CreatedBy = createdBy
	survey.ModifiedBy = createdBy
	for i := range survey.Options {
		survey.Options[i].CreatedBy = createdBy
		survey.Options[i].ModifiedBy = createdBy
	}
	return r.db.Create(survey).Error
}

func (r *academySurveyRepository) GetByID(id uint) (*domain.AcademySurvey, error) {
	var s domain.AcademySurvey
	err := r.db.Preload("Options").Where("id = ? AND deleted = ?", id, false).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *academySurveyRepository) ListAll(limit, offset int) ([]*domain.AcademySurvey, int64, error) {
	var list []*domain.AcademySurvey
	var total int64
	q := r.db.Model(&domain.AcademySurvey{}).Preload("Options").Where("deleted = ?", false)
	q.Count(&total)
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

func (r *academySurveyRepository) ListActive(limit, offset int) ([]*domain.AcademySurvey, int64, error) {
	var list []*domain.AcademySurvey
	var total int64
	q := r.db.Model(&domain.AcademySurvey{}).Preload("Options").Where("deleted = ? AND is_active = ?", false, true)
	q.Count(&total)
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

func (r *academySurveyRepository) Delete(id uint, deletedBy string) error {
	return r.db.Model(&domain.AcademySurvey{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"deleted":     true,
			"is_active":   false,
			"modified_by": deletedBy,
		}).Error
}

func (r *academySurveyRepository) SubmitResponse(responses []domain.AcademySurveyResponse) error {
	if len(responses) == 0 {
		return nil
	}
	return r.db.Create(&responses).Error
}

func (r *academySurveyRepository) HasUserResponded(surveyID, employeeID uint) (bool, error) {
	var count int64
	err := r.db.Model(&domain.AcademySurveyResponse{}).
		Where("survey_id = ? AND employee_id = ? AND deleted = ?", surveyID, employeeID, false).
		Count(&count).Error
	return count > 0, err
}

func (r *academySurveyRepository) GetSurveyResults(surveyID uint) (map[uint]int, error) {
	type result struct {
		OptionID uint
		Count    int
	}
	var results []result
	err := r.db.Model(&domain.AcademySurveyResponse{}).
		Select("option_id, count(*) as count").
		Where("survey_id = ? AND deleted = ?", surveyID, false).
		Group("option_id").
		Find(&results).Error

	resMap := make(map[uint]int)
	if err != nil {
		return resMap, err
	}
	for _, r := range results {
		resMap[r.OptionID] = r.Count
	}
	return resMap, nil
}

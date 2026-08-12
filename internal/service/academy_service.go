package service

import (
	"errors"
	"fmt"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// Training Service
// ─────────────────────────────────────────────────────────────────────────────

type AcademyService interface {
	// Training CRUD (admin)
	CreateTraining(training *domain.Training, userEmail string) error
	GetTraining(id uint) (*domain.Training, error)
	ListTrainings(isAdmin bool, limit, offset int) ([]*domain.Training, int64, error)
	UpdateTraining(training *domain.Training, userEmail string) error
	DeleteTraining(id uint, userEmail string) error

	// Assignment
	AssignToEmployees(trainingID uint, employeeIDs []uint, userEmail string) error
	RemoveAssignment(assignmentID uint, userEmail string) error
	ListAssignmentsByTraining(trainingID uint, limit, offset int) ([]*domain.TrainingAssignment, int64, error)
	GetMyAssignments(employeeID uint) ([]*domain.TrainingAssignment, error)

	// Progress (çalışan)
	StartTraining(assignmentID, employeeID uint, userEmail string) error
	CompleteTraining(assignmentID, employeeID uint, userEmail string) (*domain.TrainingCertificate, error)

	// Certificates
	GetMyCertificates(employeeID uint) ([]*domain.TrainingCertificate, error)
	GetCertificateByCode(code string) (*domain.TrainingCertificate, error)

	// Surveys
	CreateSurvey(survey *domain.AcademySurvey, userEmail string) error
	GetSurvey(id uint) (*domain.AcademySurvey, error)
	ListSurveys(isAdmin bool, limit, offset int) ([]*domain.AcademySurvey, int64, error)
	DeleteSurvey(id uint, userEmail string) error
	SubmitSurveyVote(surveyID, employeeID uint, optionIDs []uint, userEmail string) error
	GetSurveyResults(surveyID uint) (map[uint]int, error)
}

type academyService struct {
	trainingRepo    repository.TrainingRepository
	assignmentRepo  repository.AssignmentRepository
	certificateRepo repository.CertificateRepository
	employeeRepo    repository.EmployeeRepository
	surveyRepo      repository.AcademySurveyRepository
	auditService    AuditService
}

func NewAcademyService(
	trainingRepo repository.TrainingRepository,
	assignmentRepo repository.AssignmentRepository,
	certificateRepo repository.CertificateRepository,
	employeeRepo repository.EmployeeRepository,
	surveyRepo repository.AcademySurveyRepository,
	auditService AuditService,
) AcademyService {
	return &academyService{
		trainingRepo:    trainingRepo,
		assignmentRepo:  assignmentRepo,
		certificateRepo: certificateRepo,
		employeeRepo:    employeeRepo,
		surveyRepo:      surveyRepo,
		auditService:    auditService,
	}
}

// ── Training CRUD ────────────────────────────────────────────────────────────

func (s *academyService) CreateTraining(training *domain.Training, userEmail string) error {
	if training.Status == "" {
		training.Status = domain.TrainingStatusActive
	}
	err := s.trainingRepo.Create(training, userEmail)
	if err != nil {
		return err
	}

	return nil
}

func (s *academyService) GetTraining(id uint) (*domain.Training, error) {
	return s.trainingRepo.GetByID(id)
}

func (s *academyService) ListTrainings(isAdmin bool, limit, offset int) ([]*domain.Training, int64, error) {
	if isAdmin {
		return s.trainingRepo.ListAll(limit, offset)
	}
	return s.trainingRepo.ListActive(limit, offset)
}

func (s *academyService) UpdateTraining(training *domain.Training, userEmail string) error {
	existing, err := s.trainingRepo.GetByID(training.ID)
	if err != nil {
		return fmt.Errorf("eğitim bulunamadı: %w", err)
	}
	// Korunan alanları koru
	training.CreatedBy = existing.CreatedBy
	training.CreatedAt = existing.CreatedAt
	return s.trainingRepo.Update(training, userEmail)
}

func (s *academyService) DeleteTraining(id uint, userEmail string) error {
	_, err := s.trainingRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("eğitim bulunamadı: %w", err)
	}
	return s.trainingRepo.Delete(id, userEmail)
}

// ── Assignment ────────────────────────────────────────────────────────────────

func (s *academyService) AssignToEmployees(trainingID uint, employeeIDs []uint, userEmail string) error {
	for _, employeeID := range employeeIDs {
		// Daha önce atanmış mı?
		existing, err := s.assignmentRepo.GetByEmployeeAndTraining(employeeID, trainingID)
		if err == nil && existing != nil {
			continue // Atanmışsa atla
		}

		a := &domain.TrainingAssignment{
			TrainingID: trainingID,
			EmployeeID: employeeID,
			Status:     domain.AssignmentStatusAssigned,
		}
		if err := s.assignmentRepo.Create(a, userEmail); err != nil {
			return err
		}
	}
	return nil
}

func (s *academyService) RemoveAssignment(assignmentID uint, userEmail string) error {
	_, err := s.assignmentRepo.GetByID(assignmentID)
	if err != nil {
		return fmt.Errorf("atama bulunamadı: %w", err)
	}
	return s.assignmentRepo.Delete(assignmentID, userEmail)
}

func (s *academyService) ListAssignmentsByTraining(trainingID uint, limit, offset int) ([]*domain.TrainingAssignment, int64, error) {
	return s.assignmentRepo.ListByTraining(trainingID, limit, offset)
}

func (s *academyService) GetMyAssignments(employeeID uint) ([]*domain.TrainingAssignment, error) {
	return s.assignmentRepo.ListByEmployee(employeeID)
}

// ── Progress ──────────────────────────────────────────────────────────────────

func (s *academyService) StartTraining(assignmentID, employeeID uint, userEmail string) error {
	a, err := s.assignmentRepo.GetByID(assignmentID)
	if err != nil {
		return fmt.Errorf("atama bulunamadı: %w", err)
	}
	if a.EmployeeID != employeeID {
		return errors.New("bu eğitim size ait değil")
	}
	if a.Status == domain.AssignmentStatusCompleted {
		return errors.New("eğitim zaten tamamlanmış")
	}
	if a.Status == domain.AssignmentStatusInProgress {
		return nil // idempotent
	}
	now := time.Now().UTC()
	return s.assignmentRepo.UpdateStatus(assignmentID, domain.AssignmentStatusInProgress, &now, nil, userEmail)
}

func (s *academyService) CompleteTraining(assignmentID, employeeID uint, userEmail string) (*domain.TrainingCertificate, error) {
	a, err := s.assignmentRepo.GetByID(assignmentID)
	if err != nil {
		return nil, fmt.Errorf("atama bulunamadı: %w", err)
	}
	if a.EmployeeID != employeeID {
		return nil, errors.New("bu eğitim size ait değil")
	}
	if a.Status == domain.AssignmentStatusCompleted {
		// Zaten tamamlanmış — sertifikayı döndür
		cert, certErr := s.certificateRepo.GetByAssignmentID(assignmentID)
		if certErr == nil {
			return cert, nil
		}
	}

	now := time.Now().UTC()
	var startedAt *time.Time
	if a.Status == domain.AssignmentStatusAssigned {
		startedAt = &now // Eğitime hiç start basmadıysa şimdi başlatılmış say
	}
	if err := s.assignmentRepo.UpdateStatus(assignmentID, domain.AssignmentStatusCompleted, startedAt, &now, userEmail); err != nil {
		return nil, err
	}

	// Sertifika oluştur
	cert := &domain.TrainingCertificate{
		AssignmentID:    assignmentID,
		EmployeeID:      employeeID,
		TrainingID:      a.TrainingID,
		CertificateCode: generateCertCode(),
		IssuedAt:        now,
	}
	if err := s.certificateRepo.Create(cert, userEmail); err != nil {
		return nil, fmt.Errorf("sertifika oluşturulamadı: %w", err)
	}

	// İlişkileri doldur
	cert.Training = a.Training
	cert.Employee = a.Employee
	return cert, nil
}

// ── Certificates ──────────────────────────────────────────────────────────────

func (s *academyService) GetMyCertificates(employeeID uint) ([]*domain.TrainingCertificate, error) {
	return s.certificateRepo.ListByEmployee(employeeID)
}

func (s *academyService) GetCertificateByCode(code string) (*domain.TrainingCertificate, error) {
	if code == "" {
		return nil, errors.New("geçersiz sertifika kodu")
	}
	return s.certificateRepo.GetByCode(code)
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func generateCertCode() string {
	id := uuid.New()
	return fmt.Sprintf("KA-%s", id.String()[:8])
}

// ─────────────────────────────────────────────────────────────────────────────
// Academy Survey
// ─────────────────────────────────────────────────────────────────────────────

func (s *academyService) CreateSurvey(survey *domain.AcademySurvey, userEmail string) error {
	return s.surveyRepo.Create(survey, userEmail)
}

func (s *academyService) GetSurvey(id uint) (*domain.AcademySurvey, error) {
	return s.surveyRepo.GetByID(id)
}

func (s *academyService) ListSurveys(isAdmin bool, limit, offset int) ([]*domain.AcademySurvey, int64, error) {
	if isAdmin {
		return s.surveyRepo.ListAll(limit, offset)
	}
	return s.surveyRepo.ListActive(limit, offset)
}

func (s *academyService) DeleteSurvey(id uint, userEmail string) error {
	return s.surveyRepo.Delete(id, userEmail)
}

func (s *academyService) SubmitSurveyVote(surveyID, employeeID uint, optionIDs []uint, userEmail string) error {
	survey, err := s.surveyRepo.GetByID(surveyID)
	if err != nil {
		return err
	}
	if !survey.IsActive {
		return errors.New("anket aktif değil")
	}

	hasResponded, err := s.surveyRepo.HasUserResponded(surveyID, employeeID)
	if err != nil {
		return err
	}
	if hasResponded {
		return errors.New("bu ankete daha önce katıldınız")
	}

	if !survey.IsMultiSelect && len(optionIDs) > 1 {
		return errors.New("bu ankette birden fazla seçim yapılamaz")
	}
	if len(optionIDs) == 0 {
		return errors.New("en az bir seçim yapmalısınız")
	}

	// Validate options exist in survey
	validOptions := make(map[uint]bool)
	for _, opt := range survey.Options {
		validOptions[opt.ID] = true
	}
	var responses []domain.AcademySurveyResponse
	for _, optID := range optionIDs {
		if !validOptions[optID] {
			return fmt.Errorf("geçersiz seçenek: %d", optID)
		}
		responses = append(responses, domain.AcademySurveyResponse{
			SurveyID:   surveyID,
			EmployeeID: employeeID,
			OptionID:   optID,
		})
	}
	
	for i := range responses {
		responses[i].CreatedBy = userEmail
		responses[i].ModifiedBy = userEmail
	}

	return s.surveyRepo.SubmitResponse(responses)
}

func (s *academyService) GetSurveyResults(surveyID uint) (map[uint]int, error) {
	return s.surveyRepo.GetSurveyResults(surveyID)
}

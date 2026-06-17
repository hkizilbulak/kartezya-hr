package service

import (
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type FAQService interface {
	CreateFAQ(faq *domain.FAQ, userEmail string) error
	GetFAQByID(id uint) (*domain.FAQ, error)
	GetAllFAQs(limit, offset int, sortParams types.SortParams) ([]*domain.FAQ, int64, error)
	UpdateFAQ(faq *domain.FAQ, userEmail string) error
	DeleteFAQ(id uint, userEmail string) error
}

type faqService struct {
	repo         repository.FAQRepository
	auditService AuditService // Sistemin loglama yapısı varsa diye ekledim
}

func NewFAQService(repo repository.FAQRepository, auditService AuditService) FAQService {
	return &faqService{
		repo:         repo,
		auditService: auditService,
	}
}

func (s *faqService) CreateFAQ(faq *domain.FAQ, userEmail string) error {
	return s.repo.Create(faq, userEmail)
}

func (s *faqService) GetFAQByID(id uint) (*domain.FAQ, error) {
	return s.repo.GetByID(id)
}

func (s *faqService) GetAllFAQs(limit, offset int, sortParams types.SortParams) ([]*domain.FAQ, int64, error) {
	return s.repo.GetAll(limit, offset, sortParams)
}

func (s *faqService) UpdateFAQ(faq *domain.FAQ, userEmail string) error {
	return s.repo.Update(faq, userEmail)
}

func (s *faqService) DeleteFAQ(id uint, userEmail string) error {
	return s.repo.Delete(id, userEmail)
}
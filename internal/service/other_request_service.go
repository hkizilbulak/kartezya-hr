package service

import (
	"errors"
	"mime/multipart"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type OtherRequestService interface {
	// Talep Türü
	CreateRequestType(reqType *domain.RequestType, userEmail string) error
	GetRequestTypeByID(id uint) (*domain.RequestType, error)
	GetAllRequestTypes(limit, offset int, sortParams types.SortParams) ([]*domain.RequestType, int64, error)
	UpdateRequestType(reqType *domain.RequestType, userEmail string) error
	DeleteRequestType(id uint, userEmail string) error

	// Talep
	CreateRequest(req *domain.OtherRequest, userID uint, userEmail string) error
	GetRequestByID(id uint) (*domain.OtherRequest, error)
	GetAllRequests(limit, offset int, sortParams types.SortParams) ([]*domain.OtherRequest, int64, error)
	UpdateRequest(req *domain.OtherRequest, userEmail string) error
	CancelRequest(id uint, userEmail string) error
	CompleteRequest(id uint, completerID uint, userEmail string) error

	// Yüklü Dosya / Doküman Yönetimi
	UploadRequestDocument(requestID uint, file *multipart.FileHeader) (*domain.Attachment, error)
	GetRequestDocuments(requestID uint) ([]*domain.Attachment, error)
	DeleteRequestDocument(documentID string) error
}

type otherRequestService struct {
	repo         repository.OtherRequestRepository
	auditService AuditService
	emailService EmailService
}

func NewOtherRequestService(repo repository.OtherRequestRepository, auditService AuditService, emailService EmailService) OtherRequestService {
	return &otherRequestService{
		repo:         repo,
		auditService: auditService,
		emailService: emailService,
	}
}

// ==================== TALEP TÜRÜ İŞLEMLERİ ====================

func (s *otherRequestService) CreateRequestType(reqType *domain.RequestType, userEmail string) error {
	return s.repo.CreateRequestType(reqType, userEmail)
}

func (s *otherRequestService) GetRequestTypeByID(id uint) (*domain.RequestType, error) {
	return s.repo.GetRequestTypeByID(id)
}

func (s *otherRequestService) GetAllRequestTypes(limit, offset int, sortParams types.SortParams) ([]*domain.RequestType, int64, error) {
	return s.repo.GetAllRequestTypes(limit, offset, sortParams)
}

func (s *otherRequestService) UpdateRequestType(reqType *domain.RequestType, userEmail string) error {
	return s.repo.UpdateRequestType(reqType, userEmail)
}

func (s *otherRequestService) DeleteRequestType(id uint, userEmail string) error {
	return s.repo.DeleteRequestType(id, userEmail)
}

// ==================== TALEP İŞLEMLERİ ====================

func (s *otherRequestService) CreateRequest(req *domain.OtherRequest, userID uint, userEmail string) error {
	req.Status = domain.RequestStatusActive

	err := s.repo.CreateRequest(req, userID, userEmail)
	if err != nil {
		return err
	}

	subject := "Yeni Talep Alındı - " + userEmail
	variables := map[string]interface{}{
		"email":       userEmail,
		"status":      req.Status,
		"description": req.Description,
	}
	_ = s.emailService.SendTemplateEmail([]string{"hr@kartezya.com"}, subject, "notification-template", variables)

	return nil
}

func (s *otherRequestService) GetRequestByID(id uint) (*domain.OtherRequest, error) {
	return s.repo.GetRequestByID(id)
}

func (s *otherRequestService) GetAllRequests(limit, offset int, sortParams types.SortParams) ([]*domain.OtherRequest, int64, error) {
	return s.repo.GetAllRequests(limit, offset, sortParams)
}

func (s *otherRequestService) UpdateRequest(req *domain.OtherRequest, userEmail string) error {
	existingReq, err := s.repo.GetRequestByID(req.ID)
	if err != nil {
		return err
	}

	if existingReq.Status == domain.RequestStatusCompleted {
		return errors.New("Tamamlanmış talepler güncellenemez")
	}

	// Talep Tipi veya Açıklamada değişiklik yapıldığında statü otomatik ACTIVE olur
	if existingReq.RequestTypeID != req.RequestTypeID || existingReq.Description != req.Description {
		req.Status = domain.RequestStatusActive
	}

	return s.repo.UpdateRequest(req, userEmail)
}

func (s *otherRequestService) CancelRequest(id uint, userEmail string) error {
	existingReq, err := s.repo.GetRequestByID(id)
	if err != nil {
		return err
	}

	if existingReq.Status == domain.RequestStatusCompleted {
		return errors.New("Tamamlanmış talepler silinemez")
	}

	existingReq.Status = domain.RequestStatusCancelled
	return s.repo.UpdateRequest(existingReq, userEmail)
}

func (s *otherRequestService) CompleteRequest(id uint, completerID uint, userEmail string) error {
	existingReq, err := s.repo.GetRequestByID(id)
	if err != nil {
		return err
	}

	now := time.Now()
	existingReq.Status = domain.RequestStatusCompleted
	existingReq.CompletedBy = &completerID
	existingReq.CompletedAt = &now

	err = s.repo.UpdateRequest(existingReq, userEmail)
	if err != nil {
		return err
	}

	subject := "Talebiniz Tamamlandı"
	variables := map[string]interface{}{
		"type":        existingReq.RequestType.Name,
		"completedAt": now.Format("02.01.2006 15:04"),
		"description": existingReq.Description,
	}

	if existingReq.Employee != nil && existingReq.Employee.CompanyEmail != "" {
		_ = s.emailService.SendTemplateEmail([]string{existingReq.Employee.CompanyEmail}, subject, "notification-template", variables)
	}

	return nil
}

// ==================== DÖKÜMAN / DOSYA YÖNETİMİ ====================

func (s *otherRequestService) UploadRequestDocument(requestID uint, file *multipart.FileHeader) (*domain.Attachment, error) {
	return nil, nil
}

func (s *otherRequestService) GetRequestDocuments(requestID uint) ([]*domain.Attachment, error) {
	return nil, nil
}

func (s *otherRequestService) DeleteRequestDocument(documentID string) error {
	return nil
}
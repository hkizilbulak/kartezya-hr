package service

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"

	"github.com/xuri/excelize/v2"
)

type EventService interface {
	CreateEvent(event *domain.Event, createdBy string) error
	UpdateEvent(event *domain.Event, modifiedBy string) error
	GetEvent(id uint) (*domain.Event, error)
	GetAllEvents(limit, offset int, sortParams types.SortParams) ([]*domain.Event, int64, error)
	DeleteEvent(id uint, deletedBy string) error

	PublishEvent(id uint, modifiedBy string) error
	
	GetActiveEventsForDashboard(userID uint, audience string) ([]*domain.Event, error)
	ParticipateInEvent(eventId uint, userId uint, status domain.ParticipantStatus, companionCount int) error

	ExportEventParticipants(eventId uint) ([]byte, error)
}

type eventService struct {
	eventRepo            repository.EventRepository
	eventParticipantRepo repository.EventParticipantRepository
	userRepo             repository.UserRepository
	employeeRepo         repository.EmployeeRepository
	emailService         EmailService
}

func NewEventService(
	eventRepo repository.EventRepository,
	eventParticipantRepo repository.EventParticipantRepository,
	userRepo repository.UserRepository,
	employeeRepo repository.EmployeeRepository,
	emailService EmailService,
) EventService {
	return &eventService{
		eventRepo:            eventRepo,
		eventParticipantRepo: eventParticipantRepo,
		userRepo:             userRepo,
		employeeRepo:         employeeRepo,
		emailService:         emailService,
	}
}

func (s *eventService) CreateEvent(event *domain.Event, createdBy string) error {
	return s.eventRepo.Create(event, createdBy)
}

func (s *eventService) UpdateEvent(event *domain.Event, modifiedBy string) error {
	return s.eventRepo.Update(event, modifiedBy)
}

func (s *eventService) GetEvent(id uint) (*domain.Event, error) {
	return s.eventRepo.GetByID(id)
}

func (s *eventService) GetAllEvents(limit, offset int, sortParams types.SortParams) ([]*domain.Event, int64, error) {
	return s.eventRepo.GetAll(limit, offset, sortParams)
}

func (s *eventService) DeleteEvent(id uint, deletedBy string) error {
	return s.eventRepo.Delete(id, deletedBy)
}

func (s *eventService) PublishEvent(id uint, modifiedBy string) error {
	event, err := s.eventRepo.GetByID(id)
	if err != nil {
		return err
	}

	event.Status = domain.EventStatusPublished
	if err := s.eventRepo.Update(event, modifiedBy); err != nil {
		return err
	}

	// Send Email using Resend template if configured
	if event.ResendTemplateId != "" {
		// Get target audience emails. For simplicity, we get all active employees' emails.
		// If AudienceFilter is specific, you would filter here.
		filters := map[string]interface{}{
			"status": "ACTIVE",
		}
		
		employees, _, err := s.employeeRepo.GetAllWithFilters(10000, 0, types.SortParams{Sort: "id", Direction: "ASC"}, filters)
		if err == nil && len(employees) > 0 {
			var emails []string
			for _, emp := range employees {
				if emp.CompanyEmail != "" {
					emails = append(emails, emp.CompanyEmail)
				} else if emp.Email != "" {
					emails = append(emails, emp.Email)
				}
			}

			if len(emails) > 0 {
				variables := map[string]interface{}{
					"event_name":  event.Name,
					"date":        event.StartDate.Format("02.01.2006 15:04"),
					"location":    event.Location,
					// "portal_link": "https://portal.domain.com/events", // Configurable
				}
				
				// Send to all emails. Note: In production, batching is better.
				_ = s.emailService.SendTemplateEmail(emails, "Yeni Etkinlik: "+event.Name, event.ResendTemplateId, variables)
			}
		}
	} else {
		log.Println("WARN: Event published but no ResendTemplateId provided. Emails not sent for event:", event.Name)
	}

	return nil
}

func (s *eventService) GetActiveEventsForDashboard(userID uint, audience string) ([]*domain.Event, error) {
	// Active published events
	events, err := s.eventRepo.GetActiveEventsForDashboard(audience)
	if err != nil {
		return nil, err
	}

	// Preload participant status for the current user
	for i, ev := range events {
		participant, err := s.eventParticipantRepo.GetByEventAndUser(ev.ID, userID)
		if err == nil && participant != nil {
			// Ensure Participants slice is initialized
			events[i].Participants = []domain.EventParticipant{*participant}
		}
	}

	// Sort logic: "New" (no participant record) on top, "Answered" (has record) below.
	// This is done by splitting and concatenating.
	var newEvents []*domain.Event
	var answeredEvents []*domain.Event

	for _, ev := range events {
		if len(ev.Participants) > 0 {
			answeredEvents = append(answeredEvents, ev)
		} else {
			newEvents = append(newEvents, ev)
		}
	}

	return append(newEvents, answeredEvents...), nil
}

func (s *eventService) ParticipateInEvent(eventId uint, userId uint, status domain.ParticipantStatus, companionCount int) error {
	event, err := s.eventRepo.GetByID(eventId)
	if err != nil {
		return err
	}

	// Check deadline
	if event.LastChangeDate != nil && time.Now().After(*event.LastChangeDate) {
		return fmt.Errorf("participation deadline has passed")
	}

	if event.AllowCompanion && companionCount > event.MaxCompanion {
		return fmt.Errorf("max companion limit exceeded")
	}

	if !event.AllowCompanion {
		companionCount = 0
	}

	participant, err := s.eventParticipantRepo.GetByEventAndUser(eventId, userId)
	if err == nil && participant != nil {
		participant.Status = status
		participant.CompanionCount = companionCount
		return s.eventParticipantRepo.Update(participant, fmt.Sprintf("%d", userId))
	}

	// Create new
	newParticipant := &domain.EventParticipant{
		EventID:        eventId,
		UserID:         userId,
		Status:         status,
		CompanionCount: companionCount,
	}

	return s.eventParticipantRepo.Create(newParticipant, fmt.Sprintf("%d", userId))
}

func (s *eventService) ExportEventParticipants(eventId uint) ([]byte, error) {
	_, err := s.eventRepo.GetByID(eventId)
	if err != nil {
		return nil, err
	}

	participants, err := s.eventParticipantRepo.GetByEventID(eventId)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheetName := "Katılımcılar"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// Headers
	headers := []string{"Ad Soyad", "Departman", "Katılım Durumu", "Refakatçi Sayısı", "Toplam Kişi", "Kayıt Tarihi"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	// Data rows
	for rowIdx, p := range participants {
		rowNum := rowIdx + 2
		
		fullName := ""
		department := ""
		
		if p.User != nil && p.User.Employee != nil {
			fullName = p.User.Employee.FirstName + " " + p.User.Employee.LastName
			if len(p.User.Employee.EmployeeWorkInformation) > 0 {
				for _, winfo := range p.User.Employee.EmployeeWorkInformation {
					if winfo.EndDate == nil || winfo.EndDate.After(time.Now()) {
						department = winfo.Department.Name
						break
					}
				}
				if department == "" {
					department = p.User.Employee.EmployeeWorkInformation[0].Department.Name
				}
			}
		}

		statusStr := "Evet"
		if p.Status == domain.ParticipantStatusNotAttending {
			statusStr = "Hayır"
		} else if p.Status == domain.ParticipantStatusPending {
			statusStr = "Beklemede"
		}

		totalPerson := 1 + p.CompanionCount
		if p.Status != domain.ParticipantStatusAttending {
			totalPerson = 0
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), fullName)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), department)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), statusStr)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), p.CompanionCount)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), totalPerson)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), p.CreatedAt.Format("02.01.2006 15:04:05"))
	}

	// Auto-fit columns
	// ... (simplified for space) ...

	// Delete Default Sheet1 if it's not the active one
	if sheetName != "Sheet1" {
		f.DeleteSheet("Sheet1")
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

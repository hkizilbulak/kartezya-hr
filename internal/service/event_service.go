package service

import (
	"bytes"
	"fmt"
	"log"
	"time"
	_ "time/tzdata" // embed timezone database for LoadLocation

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"

	"github.com/xuri/excelize/v2"
)

type EventService interface {
	CreateEvent(event *domain.Event, targetEmployeeIDs []uint, createdBy string) error
	UpdateEvent(event *domain.Event, targetEmployeeIDs []uint, modifiedBy string) error
	GetEvent(id uint) (*domain.Event, error)
	GetAllEvents(limit, offset int, sortParams types.SortParams) ([]*domain.Event, int64, error)
	DeleteEvent(id uint, deletedBy string) error

	PublishEvent(id uint, modifiedBy string) error

	GetActiveEventsForDashboard(userID uint) ([]*domain.Event, error)
	GetActiveEventsCount() (int64, error)
	ParticipateInEvent(eventId uint, userId uint, status domain.ParticipantStatus, companionCount int) error

	ExportEventParticipants(eventId uint) ([]byte, error)
}

type eventService struct {
	eventRepo            repository.EventRepository
	eventParticipantRepo repository.EventParticipantRepository
	userRepo             repository.UserRepository
	employeeRepo         repository.EmployeeRepository
	emailService         EmailService
	config               *config.Config
}

func NewEventService(
	eventRepo repository.EventRepository,
	eventParticipantRepo repository.EventParticipantRepository,
	userRepo repository.UserRepository,
	employeeRepo repository.EmployeeRepository,
	emailService EmailService,
	cfg *config.Config,
) EventService {
	return &eventService{
		eventRepo:            eventRepo,
		eventParticipantRepo: eventParticipantRepo,
		userRepo:             userRepo,
		employeeRepo:         employeeRepo,
		emailService:         emailService,
		config:               cfg,
	}
}

func (s *eventService) CreateEvent(event *domain.Event, targetEmployeeIDs []uint, createdBy string) error {
	err := s.eventRepo.Create(event, createdBy)
	if err != nil {
		return err
	}
	return s.syncTargetEmployees(event.ID, targetEmployeeIDs, createdBy)
}

func (s *eventService) UpdateEvent(event *domain.Event, targetEmployeeIDs []uint, modifiedBy string) error {
	err := s.eventRepo.Update(event, modifiedBy)
	if err != nil {
		return err
	}
	return s.syncTargetEmployees(event.ID, targetEmployeeIDs, modifiedBy)
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

	log.Printf("[EVENT] Event published: ID=%d, Name=%s", event.ID, event.Name)

	// Send Email using Resend template if configured
	if event.ResendTemplateId != "" {
		istanbul, _ := time.LoadLocation("Europe/Istanbul")
		eventDate := event.StartDate.In(istanbul).Format("02/01/2006 15:04") + " - " + event.EndDate.In(istanbul).Format("02/01/2006 15:04")
		importantNote := "Katılım durumunu en kısa sürede bildirmeni rica ederiz."
		if event.AllowCompanion && event.MaxCompanion > 0 {
			importantNote = fmt.Sprintf("Bu etkinliğe +%d kişiyle katılabilirsin. Katılım durumunu en kısa sürede bildirmeni rica ederiz.", event.MaxCompanion)
		}
		eventUrl := "https://personel.kartezya.com"

		// Get target audience emails
		if event.AudienceFilter == domain.EventAudienceAllCompany {
			// Use configured mail group addresses to avoid Resend BCC limit (max 50)
			groupEmails := s.config.Email.EventAllCompanyGroup
			if len(groupEmails) == 0 {
				log.Printf("[EVENT] WARNING: EVENT_EMAIL_ALL_COMPANY is not configured, skipping ALL_COMPANY email for event %d", event.ID)
			} else {
				variables := map[string]interface{}{
					"eventTitle":       event.Name,
					"eventDate":        eventDate,
					"eventLocation":    event.Location,
					"eventDescription": event.Description,
					"eventUrl":         eventUrl,
					"importantNote":    importantNote,
				}
				err := s.emailService.SendTemplateEmail(groupEmails, "", event.ResendTemplateId, variables)
				if err != nil {
					log.Printf("[EVENT] ERROR: Failed to send event email to group %v: %v", groupEmails, err)
				} else {
					log.Printf("[EVENT] Successfully sent event email to all-company groups: %v", groupEmails)
				}
			}
		} else {
			// Targeted audience, fetch participants
			participants, err := s.eventParticipantRepo.GetByEventID(event.ID)
			if err == nil && len(participants) > 0 {
				var emails []string
				for _, p := range participants {
					if p.User != nil && p.User.Employee != nil {
						emp := p.User.Employee
						if emp.CompanyEmail != "" {
							emails = append(emails, emp.CompanyEmail)
						} else if emp.Email != "" {
							emails = append(emails, emp.Email)
						}
					}
				}

				if len(emails) > 0 {
					variables := map[string]interface{}{
						"eventTitle":       event.Name,
						"eventDate":        eventDate,
						"eventLocation":    event.Location,
						"eventDescription": event.Description,
						"eventUrl":         eventUrl,
						"importantNote":    importantNote,
					}
					err := s.emailService.SendTemplateEmail(emails, "", event.ResendTemplateId, variables)
					if err != nil {
						log.Printf("[EVENT] ERROR: Failed to send event email: %v", err)
					} else {
						log.Printf("[EVENT] Successfully sent event email to %d recipients: %v", len(emails), emails)
					}
				}
			}
		}
	} else {
		log.Println("WARN: Event published but no ResendTemplateId provided. Emails not sent for event:", event.Name)
	}

	return nil
}

func (s *eventService) GetActiveEventsForDashboard(userID uint) ([]*domain.Event, error) {
	// Active published events
	events, err := s.eventRepo.GetActiveEventsForDashboard(userID)
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

func (s *eventService) GetActiveEventsCount() (int64, error) {
	return s.eventRepo.CountActiveEvents()
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

func (s *eventService) syncTargetEmployees(eventID uint, targetEmployeeIDs []uint, modifiedBy string) error {
	if targetEmployeeIDs == nil {
		targetEmployeeIDs = []uint{}
	}

	// Fetch existing participants
	existingParticipants, err := s.eventParticipantRepo.GetByEventID(eventID)
	if err != nil {
		return err
	}

	existingUserIDs := make(map[uint]*domain.EventParticipant)
	for i, p := range existingParticipants {
		existingUserIDs[p.UserID] = existingParticipants[i]
	}

	// Fetch target employees to get their UserIDs
	var targetUserIDs []uint
	if len(targetEmployeeIDs) > 0 {
		employees, err := s.employeeRepo.GetByIDs(targetEmployeeIDs)
		if err != nil {
			return err
		}
		for _, emp := range employees {
			if emp.UserID != 0 {
				targetUserIDs = append(targetUserIDs, emp.UserID)
			}
		}
	}

	targetUserIDsMap := make(map[uint]bool)
	for _, uid := range targetUserIDs {
		targetUserIDsMap[uid] = true

		if _, exists := existingUserIDs[uid]; !exists {
			// Add new PENDING participant
			newParticipant := &domain.EventParticipant{
				EventID: eventID,
				UserID:  uid,
				Status:  domain.ParticipantStatusPending,
			}
			err := s.eventParticipantRepo.Create(newParticipant, modifiedBy)
			if err != nil {
				log.Printf("Failed to create participant for event %d, user %d: %v", eventID, uid, err)
			}
		}
	}

	// Remove PENDING participants that are not in targetUserIDs
	for uid, p := range existingUserIDs {
		if !targetUserIDsMap[uid] && p.Status == domain.ParticipantStatusPending {
			err := s.eventParticipantRepo.Delete(p.ID, modifiedBy)
			if err != nil {
				log.Printf("Failed to delete participant for event %d, user %d: %v", eventID, uid, err)
			}
		}
	}

	return nil
}

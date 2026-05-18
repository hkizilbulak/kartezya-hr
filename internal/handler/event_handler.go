package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	eventService service.EventService
}

func NewEventHandler(eventService service.EventService) *EventHandler {
	return &EventHandler{
		eventService: eventService,
	}
}

// Request DTOs
type CreateEventRequest struct {
	Name             string               `json:"name" binding:"required"`
	Type             string               `json:"type" binding:"required"`
	Description      string               `json:"description"`
	StartDate        time.Time            `json:"start_date" binding:"required"`
	EndDate          time.Time            `json:"end_date" binding:"required"`
	Location         string               `json:"location"`
	AudienceFilter   domain.EventAudience `json:"audience_filter"`
	Quota            int                  `json:"quota"`
	AllowCompanion   bool                 `json:"allow_companion"`
	MaxCompanion     int                  `json:"max_companion"`
	LastChangeDate   *time.Time           `json:"last_change_date"`
	ResendTemplateId string               `json:"resend_template_id"`
	TargetEmployeeIDs []uint               `json:"target_employee_ids"`
}

type ParticipateRequest struct {
	Status         domain.ParticipantStatus `json:"status" binding:"required"`
	CompanionCount int                      `json:"companion_count"`
}

// CreateEvent handles event creation (Admin)
func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")

	if len(req.TargetEmployeeIDs) > 0 {
		req.AudienceFilter = "TARGETED" // Custom value to indicate it is targeted
	} else if req.AudienceFilter == "" {
		req.AudienceFilter = domain.EventAudienceAllCompany
	}

	event := &domain.Event{
		Name:             req.Name,
		Type:             req.Type,
		Description:      req.Description,
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
		Location:         req.Location,
		AudienceFilter:   req.AudienceFilter,
		Quota:            req.Quota,
		AllowCompanion:   req.AllowCompanion,
		MaxCompanion:     req.MaxCompanion,
		LastChangeDate:   req.LastChangeDate,
		ResendTemplateId: req.ResendTemplateId,
		Status:           domain.EventStatusDraft,
	}

	if err := h.eventService.CreateEvent(event, req.TargetEmployeeIDs, fmt.Sprintf("%v", userID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": event})
}

// UpdateEvent handles event updates (Admin)
func (h *EventHandler) UpdateEvent(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
		return
	}

	var req CreateEventRequest // Re-use struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")

	event, err := h.eventService.GetEvent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Event not found"})
		return
	}

	event.Name = req.Name
	event.Type = req.Type
	event.Description = req.Description
	event.StartDate = req.StartDate
	event.EndDate = req.EndDate
	event.Location = req.Location
	if len(req.TargetEmployeeIDs) > 0 {
		event.AudienceFilter = "TARGETED"
	} else if req.AudienceFilter == "" {
		event.AudienceFilter = domain.EventAudienceAllCompany
	} else {
		event.AudienceFilter = req.AudienceFilter
	}
	event.Quota = req.Quota
	event.AllowCompanion = req.AllowCompanion
	event.MaxCompanion = req.MaxCompanion
	event.LastChangeDate = req.LastChangeDate
	event.ResendTemplateId = req.ResendTemplateId

	if err := h.eventService.UpdateEvent(event, req.TargetEmployeeIDs, fmt.Sprintf("%v", userID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": event})
}

// PublishEvent handles publishing an event (Admin)
func (h *EventHandler) PublishEvent(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
		return
	}

	userID, _ := c.Get("userID")

	if err := h.eventService.PublishEvent(id, fmt.Sprintf("%v", userID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Event published successfully"})
}

// DeleteEvent handles deleting an event (Admin)
func (h *EventHandler) DeleteEvent(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
		return
	}

	userID, _ := c.Get("userID")

	if err := h.eventService.DeleteEvent(id, fmt.Sprintf("%v", userID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Event deleted successfully"})
}

// GetEvents lists events for Admin
func (h *EventHandler) GetEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "start_date"),
		Direction: c.DefaultQuery("direction", "DESC"),
	}

	offset := (page - 1) * limit

	events, total, err := h.eventService.GetAllEvents(limit, offset, sortParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    events,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GetDashboardEvents lists active events for user portal
func (h *EventHandler) GetDashboardEvents(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	events, err := h.eventService.GetActiveEventsForDashboard(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": events})
}

// ParticipateInEvent handles user participation
func (h *EventHandler) ParticipateInEvent(c *gin.Context) {
	eventId, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid Event ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	var req ParticipateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.eventService.ParticipateInEvent(eventId, userID.(uint), req.Status, req.CompanionCount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Participation updated"})
}

// ExportParticipants exports the participant list to Excel
func (h *EventHandler) ExportParticipants(c *gin.Context) {
	eventId, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid Event ID"})
		return
	}

	fileBytes, err := h.eventService.ExportEventParticipants(eventId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename=participants.xlsx")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}

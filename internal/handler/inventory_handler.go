package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"
	"kartezya-hr/internal/types"

	"github.com/gin-gonic/gin"
)

type InventoryHandler struct {
	inventoryService service.InventoryService
	employeeService  service.EmployeeService
	documentService  service.DocumentService
}

type InventoryItemRequest struct {
	DeviceType     string                     `json:"device_type"`
	Brand          string                     `json:"brand"`
	Model          string                     `json:"model"`
	SerialNumber   string                     `json:"serial_number"`
	Status         domain.InventoryItemStatus `json:"status"`
	AssignmentDate string                     `json:"assignment_date"`
	Notes          string                     `json:"notes"`
	Specifications interface{}                `json:"specifications"`
	DocumentID     *string                    `json:"document_id"`
}

func parseAssignmentDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, dateStr)
		if t.IsZero() {
			return nil
		}
	}
	return &t
}

func parseSpecifications(specs interface{}) string {
	if specs == nil {
		return ""
	}
	switch v := specs.(type) {
	case string:
		return v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func NewInventoryHandler(
	inventoryService service.InventoryService,
	employeeService service.EmployeeService,
	documentService service.DocumentService,
) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
		employeeService:  employeeService,
		documentService:  documentService,
	}
}

// GetMyItems retrieves inventory items assigned to the current user
func (h *InventoryHandler) GetMyItems(c *gin.Context) {
	userID, _, _, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
		return
	}

	employee, err := h.employeeService.GetEmployeeByUserID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Employee profile not found"})
		return
	}

	items, err := h.inventoryService.GetItemsByEmployeeID(employee.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// CreateMyItem allows an employee to add an item to their inventory
func (h *InventoryHandler) CreateMyItem(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
		return
	}

	employee, err := h.employeeService.GetEmployeeByUserID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Employee profile not found"})
		return
	}

	var req InventoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	item := domain.InventoryItem{
		EmployeeID:     &employee.ID,
		DeviceType:     req.DeviceType,
		Brand:          req.Brand,
		Model:          req.Model,
		SerialNumber:   req.SerialNumber,
		Status:         domain.InventoryStatusInUse, // My Items default
		AssignmentDate: parseAssignmentDate(req.AssignmentDate),
		Notes:          req.Notes,
		Specifications: parseSpecifications(req.Specifications),
	}

	if err := h.inventoryService.CreateItem(&item, strconv.FormatUint(uint64(userID), 10)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Link document if provided
	if req.DocumentID != nil && *req.DocumentID != "" {
		if err := h.documentService.LinkDocumentsToRecord(
			[]string{*req.DocumentID},
			domain.AttachmentRelatedTypeInventory,
			item.ID,
			userID,
			roles,
		); err != nil {
			log.Printf("Warning: failed to link document %s to inventory item %d: %v", *req.DocumentID, item.ID, err)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": item})
}

// UpdateMyItem allows an employee to update their inventory item
func (h *InventoryHandler) UpdateMyItem(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
		return
	}

	employee, err := h.employeeService.GetEmployeeByUserID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Employee profile not found"})
		return
	}

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid item ID"})
		return
	}

	item, err := h.inventoryService.GetItemByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Item not found"})
		return
	}

	importAuthz := false // just a dummy to ensure authz package logic if we didn't import it
	_ = importAuthz

	// Check if user is trying to edit someone else's item and lacks admin permissions
	isOwner := item.EmployeeID != nil && *item.EmployeeID == employee.ID
	canManageAll := false
	for _, role := range roles {
		// we should actually import authz or check string
		if role == "ADMIN" || role == "HR" {
			canManageAll = true
			break
		}
	}

	if !isOwner && !canManageAll {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Not authorized to update this item"})
		return
	}

	var req InventoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	updatedItem := domain.InventoryItem{
		AuditableModel: domain.AuditableModel{ID: uint(id)},
		EmployeeID:     item.EmployeeID,
		DeviceType:     req.DeviceType,
		Brand:          req.Brand,
		Model:          req.Model,
		SerialNumber:   req.SerialNumber,
		Status:         req.Status,
		AssignmentDate: parseAssignmentDate(req.AssignmentDate),
		Notes:          req.Notes,
		Specifications: parseSpecifications(req.Specifications),
	}

	if err := h.inventoryService.UpdateItem(&updatedItem, strconv.FormatUint(uint64(userID), 10)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Link document if provided
	if req.DocumentID != nil && *req.DocumentID != "" {
		if err := h.documentService.LinkDocumentsToRecord(
			[]string{*req.DocumentID},
			domain.AttachmentRelatedTypeInventory,
			uint(id),
			userID,
			roles,
		); err != nil {
			log.Printf("Warning: failed to link document %s to inventory item %d: %v", *req.DocumentID, id, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": updatedItem})
}

// DeleteMyItem allows an employee to delete their inventory item
func (h *InventoryHandler) DeleteMyItem(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
		return
	}

	employee, err := h.employeeService.GetEmployeeByUserID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Employee profile not found"})
		return
	}

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid item ID"})
		return
	}

	item, err := h.inventoryService.GetItemByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Item not found"})
		return
	}

	// Check if user is trying to edit someone else's item and lacks admin permissions
	isOwner := item.EmployeeID != nil && *item.EmployeeID == employee.ID
	canManageAll := false
	for _, role := range roles {
		if role == "ADMIN" || role == "HR" {
			canManageAll = true
			break
		}
	}

	if !isOwner && !canManageAll {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Not authorized to delete this item"})
		return
	}

	if err := h.inventoryService.DeleteItem(uint(id), strconv.FormatUint(uint64(userID), 10)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Item deleted successfully"})
}

// GetEmployeeInventory retrieves inventory items assigned to a specific employee
func (h *InventoryHandler) GetEmployeeInventory(c *gin.Context) {
	employeeIdParam := c.Param("id")
	employeeID, err := strconv.ParseUint(employeeIdParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid employee ID"})
		return
	}

	items, err := h.inventoryService.GetItemsByEmployeeID(uint(employeeID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

// AssignItemToEmployee assigns a new inventory item to an employee
func (h *InventoryHandler) AssignItemToEmployee(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
		return
	}

	employeeIdParam := c.Param("id")
	employeeID, err := strconv.ParseUint(employeeIdParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid employee ID"})
		return
	}

	var req InventoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	empID := uint(employeeID)
	item := domain.InventoryItem{
		EmployeeID:     &empID,
		DeviceType:     req.DeviceType,
		Brand:          req.Brand,
		Model:          req.Model,
		SerialNumber:   req.SerialNumber,
		Status:         req.Status,
		AssignmentDate: parseAssignmentDate(req.AssignmentDate),
		Notes:          req.Notes,
		Specifications: parseSpecifications(req.Specifications),
	}

	if err := h.inventoryService.CreateItem(&item, strconv.FormatUint(uint64(userID), 10)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Link document if provided
	if req.DocumentID != nil && *req.DocumentID != "" {
		if err := h.documentService.LinkDocumentsToRecord(
			[]string{*req.DocumentID},
			domain.AttachmentRelatedTypeInventory,
			item.ID,
			userID,
			roles,
		); err != nil {
			log.Printf("Warning: failed to link document %s to inventory item %d: %v", *req.DocumentID, item.ID, err)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": item})
}

// GetInventoryReports returns filtered and paginated inventory items
func (h *InventoryHandler) GetInventoryReports(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	filters := make(map[string]interface{})
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}
	if deviceType := c.Query("device_type"); deviceType != "" {
		filters["device_type"] = deviceType
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	
	sortParams := types.SortParams{
		Sort:      c.DefaultQuery("sort", "created_at"),
		Direction: c.DefaultQuery("direction", "DESC"),
	}

	items, total, err := h.inventoryService.GetItemsReport(limit, offset, sortParams, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
		"page": gin.H{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

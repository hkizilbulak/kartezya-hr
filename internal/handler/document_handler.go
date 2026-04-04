package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	documentService service.DocumentService
}

func NewDocumentHandler(documentService service.DocumentService) *DocumentHandler {
	return &DocumentHandler{
		documentService: documentService,
	}
}

// UploadDocument handles file upload
// POST /api/documents/upload
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	// Get authenticated user
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}

	// Get related type from form
	relatedTypeStr := c.PostForm("related_type")
	if relatedTypeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "related_type is required"})
		return
	}
	relatedTypeInt, err := strconv.Atoi(relatedTypeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid related_type"})
		return
	}
	relatedType := domain.AttachmentRelatedType(relatedTypeInt)

	// Get document type from form
	docTypeStr := c.PostForm("type")
	if docTypeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}
	docTypeInt, err := strconv.Atoi(docTypeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid type"})
		return
	}
	docType := domain.AttachmentType(docTypeInt)

	// Upload document
	attachment, err := h.documentService.UploadDocument(file, userID.(uint), relatedType, docType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Document uploaded successfully",
		"data":    attachment,
	})
}

// GetDocument retrieves document metadata
// GET /api/documents/:id
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	// Get authenticated user
	userID, _ := c.Get("userID")
	roles, _ := c.Get("roles")

	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document ID is required"})
		return
	}

	rolesSlice := []string{}
	if rolesList, ok := roles.([]string); ok {
		rolesSlice = rolesList
	}

	attachment, err := h.documentService.GetDocument(documentID, userID.(uint), rolesSlice)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": attachment})
}

// GetDocumentURL generates a pre-signed URL for downloading
// GET /api/documents/:id/url
func (h *DocumentHandler) GetDocumentURL(c *gin.Context) {
	// Get authenticated user
	userID, _ := c.Get("userID")
	roles, _ := c.Get("roles")

	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document ID is required"})
		return
	}

	// Get expiry from query parameter (default 15 minutes)
	expiryStr := c.DefaultQuery("expiry", "15")
	expiry, err := strconv.Atoi(expiryStr)
	if err != nil {
		expiry = 15
	}

	rolesSlice := []string{}
	if rolesList, ok := roles.([]string); ok {
		rolesSlice = rolesList
	}

	url, err := h.documentService.GetDocumentURL(documentID, userID.(uint), rolesSlice, expiry)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        url,
		"expires_in": expiry,
	})
}

// GetMyDocuments retrieves all documents uploaded by current user
// GET /api/documents/my
func (h *DocumentHandler) GetMyDocuments(c *gin.Context) {
	// Get authenticated user
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	attachments, err := h.documentService.GetUserDocuments(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve documents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  attachments,
		"count": len(attachments),
	})
}

// GetRelatedDocuments retrieves documents for a specific record
// GET /api/documents/related/:type/:id
func (h *DocumentHandler) GetRelatedDocuments(c *gin.Context) {
	// Get authenticated user
	userID, _ := c.Get("userID")
	roles, _ := c.Get("roles")

	relatedTypeStr := c.Param("type")
	relatedIDStr := c.Param("id")

	relatedTypeInt, err := strconv.Atoi(relatedTypeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid related_type"})
		return
	}

	relatedID, err := strconv.ParseUint(relatedIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid related_id"})
		return
	}

	rolesSlice := []string{}
	if rolesList, ok := roles.([]string); ok {
		rolesSlice = rolesList
	}

	attachments, err := h.documentService.GetRelatedDocuments(
		domain.AttachmentRelatedType(relatedTypeInt),
		uint(relatedID),
		userID.(uint),
		rolesSlice,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve documents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  attachments,
		"count": len(attachments),
	})
}

// DeleteDocument deletes a document
// DELETE /api/documents/:id
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	// Get authenticated user
	userID, _ := c.Get("userID")
	roles, _ := c.Get("roles")

	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document ID is required"})
		return
	}

	rolesSlice := []string{}
	if rolesList, ok := roles.([]string); ok {
		rolesSlice = rolesList
	}

	err := h.documentService.DeleteDocument(documentID, userID.(uint), rolesSlice)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}

// LinkDocumentsRequest represents the request body for linking documents
type LinkDocumentsRequest struct {
	DocumentIDs []string                     `json:"document_ids" binding:"required"`
	RelatedType domain.AttachmentRelatedType `json:"related_type" binding:"required"`
	RelatedID   uint                         `json:"related_id" binding:"required"`
}

// LinkDocuments links temporary documents to a record (internal use)
// This is typically called from other services (Expense, Leave) during record creation
func (h *DocumentHandler) LinkDocuments(c *gin.Context) {
	// Get authenticated user
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req LinkDocumentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.documentService.LinkDocumentsToRecord(
		req.DocumentIDs,
		req.RelatedType,
		req.RelatedID,
		userID.(uint),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Documents linked successfully"})
}

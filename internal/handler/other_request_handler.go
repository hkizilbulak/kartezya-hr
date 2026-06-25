package handler

import (
    "fmt"
    "net/http"
    "strconv"

    "kartezya-hr/internal/domain"
    "kartezya-hr/internal/service"
    "kartezya-hr/internal/types"

    "github.com/gin-gonic/gin"
)

type OtherRequestHandler struct {
    reqService service.OtherRequestService
}

func NewOtherRequestHandler(reqService service.OtherRequestService) *OtherRequestHandler {
    return &OtherRequestHandler{reqService: reqService}
}

// ==================== DTOs ====================

type CreateRequestTypeReq struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
    Active      bool   `json:"active"`
}

type UpdateRequestTypeReq struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
    Active      bool   `json:"active"`
}

type CreateOtherRequestReq struct {
    RequestTypeID uint   `json:"request_type_id" binding:"required"`
    Description   string `json:"description" binding:"required"`
}

type UpdateOtherRequestReq struct {
    RequestTypeID uint   `json:"request_type_id" binding:"required"`
    Description   string `json:"description" binding:"required"`
}

// ==================== 1. TALEP TÜRÜ İŞLEMLERİ (ADMIN) ====================

// CreateRequestType handles Request Type creation
// @Summary Yeni Talep Türü oluşturur
// @Tags request_types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRequestTypeReq true "Request Type Data"
// @Success 201 {object} map[string]interface{}
// @Router /request-types [post]
func (h *OtherRequestHandler) CreateRequestType(c *gin.Context) {
    var req CreateRequestTypeReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid format", "details": err.Error()})
        return
    }

    userEmail, exists := c.Get("userEmail")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User email not found in context"})
        return
    }

    reqType := &domain.RequestType{
        Name:        req.Name,
        Description: req.Description,
        Active:      req.Active,
    }

    if err := h.reqService.CreateRequestType(reqType, fmt.Sprintf("%v", userEmail)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"success": true, "data": reqType, "message": "Talep türü başarıyla oluşturuldu"})
}

// GetAllRequestTypes handles fetching Request Type list
// @Summary Talep Türlerini listeler
// @Tags request_types
// @Produce json
// @Security BearerAuth
// @Router /request-types [get]
func (h *OtherRequestHandler) GetAllRequestTypes(c *gin.Context) {
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
    sortParams := types.SortParams{Sort: c.Query("sort"), Direction: c.Query("direction")}

    reqTypes, total, err := h.reqService.GetAllRequestTypes(limit, offset, sortParams)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": reqTypes, "total": total})
}

// UpdateRequestType handles updating Request Type
// @Summary Talep Türünü günceller
// @Tags request_types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request Type ID"
// @Param request body UpdateRequestTypeReq true "Request Type Data"
// @Success 200 {object} map[string]interface{}
// @Router /request-types/{id} [put]
func (h *OtherRequestHandler) UpdateRequestType(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    var req UpdateRequestTypeReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid format", "details": err.Error()})
        return
    }

    userEmail, exists := c.Get("userEmail")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User email not found in context"})
        return
    }

    reqType, err := h.reqService.GetRequestTypeByID(uint(id))
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Talep türü bulunamadı"})
        return
    }

    reqType.Name = req.Name
    reqType.Description = req.Description
    reqType.Active = req.Active

    if err := h.reqService.UpdateRequestType(reqType, fmt.Sprintf("%v", userEmail)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": reqType, "message": "Talep türü başarıyla güncellendi"})
}

// ==================== 2. DİĞER TALEPLER İŞLEMLERİ (ÇALIŞAN / İK) ====================

// CreateRequest handles new request creation
// @Summary Çalışan için yeni talep oluşturur
// @Tags other_requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateOtherRequestReq true "Request Data"
// @Router /other-requests [post]
func (h *OtherRequestHandler) CreateRequest(c *gin.Context) {
    var req CreateOtherRequestReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid format", "details": err.Error()})
        return
    }

    userID, exists := c.Get("userID")
    userEmail, emailExists := c.Get("userEmail")
    if !exists || !emailExists {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
        return
    }

    userIDUint, _ := strconv.ParseUint(fmt.Sprintf("%v", userID), 10, 32)

    otherReq := &domain.OtherRequest{
        RequestTypeID: req.RequestTypeID,
        Description:   req.Description,
    }

    if err := h.reqService.CreateRequest(otherReq, uint(userIDUint), fmt.Sprintf("%v", userEmail)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"success": true, "data": otherReq, "message": "Talep başarıyla oluşturuldu"})
}

// GetAllRequests handles fetching all requests
// @Summary Talepleri listeler
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Router /other-requests [get]
func (h *OtherRequestHandler) GetAllRequests(c *gin.Context) {
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
    sortParams := types.SortParams{Sort: c.Query("sort"), Direction: c.Query("direction")}

    reqs, total, err := h.reqService.GetAllRequests(limit, offset, sortParams)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": reqs, "total": total})
}

// UpdateRequest handles updating a request
// @Summary Talebi günceller (Talep tipi veya açıklaması değişirse statü ACTIVE olur)
// @Tags other_requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request ID"
// @Param request body UpdateOtherRequestReq true "Request Data"
// @Router /other-requests/{id} [put]
func (h *OtherRequestHandler) UpdateRequest(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    var req UpdateOtherRequestReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid format", "details": err.Error()})
        return
    }

    userEmail, exists := c.Get("userEmail")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
        return
    }

    existingReq, err := h.reqService.GetRequestByID(uint(id))
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Talep bulunamadı"})
        return
    }

    existingReq.RequestTypeID = req.RequestTypeID
    existingReq.Description = req.Description

    if err := h.reqService.UpdateRequest(existingReq, fmt.Sprintf("%v", userEmail)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": existingReq, "message": "Talep başarıyla güncellendi"})
}

// CancelRequest handles canceling a request
// @Summary Çalışan tarafından talebi iptal eder
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request ID"
// @Router /other-requests/{id}/cancel [patch]
func (h *OtherRequestHandler) CancelRequest(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    userEmail, exists := c.Get("userEmail")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
        return
    }

    if err := h.reqService.CancelRequest(uint(id), fmt.Sprintf("%v", userEmail)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Talep başarıyla iptal edildi"})
}

// CompleteRequest handles completing a request by HR
// @Summary İK personeli tarafından talebi tamamlar
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request ID"
// @Router /other-requests/{id}/complete [patch]
func (h *OtherRequestHandler) CompleteRequest(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    userID, _ := c.Get("userID")
    userEmail, exists := c.Get("userEmail")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
        return
    }

    completerID, _ := strconv.ParseUint(fmt.Sprintf("%v", userID), 10, 32)

    if err := h.reqService.CompleteRequest(uint(id), uint(completerID), fmt.Sprintf("%v", userEmail)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Talep başarıyla tamamlandı"})
}

// ==================== 3. DÖKÜMAN / DOSYA YÖNETİMİ ====================

// UploadDocument godoc
// @Summary Talebe doküman (dosya) ekler
// @Tags other_requests
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request ID"
// @Param file formData file true "Belge dosyası"
// @Success 200 {object} map[string]interface{}
// @Router /other-requests/{id}/documents [post]
func (h *OtherRequestHandler) UploadDocument(c *gin.Context) {
    _, exists := c.Get("userEmail")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Oturum bulunamadı"})
        return
    }

    idParam := c.Param("id")
    requestID, err := strconv.ParseUint(idParam, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz talep ID"})
        return
    }

    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Dosya yüklenmedi"})
        return
    }

    document, err := h.reqService.UploadRequestDocument(uint(requestID), file)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Doküman başarıyla yüklendi",
        "data":    document,
    })
}

// GetDocuments godoc
// @Summary Talep dokümanlarını listeler
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request ID"
// @Success 200 {object} map[string]interface{}
// @Router /other-requests/{id}/documents [get]
func (h *OtherRequestHandler) GetDocuments(c *gin.Context) {
    idParam := c.Param("id")
    requestID, err := strconv.ParseUint(idParam, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz talep ID"})
        return
    }

    documents, err := h.reqService.GetRequestDocuments(uint(requestID))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": documents})
}

// DeleteDocument godoc
// @Summary Talep dokümanını siler
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Param id path string true "Document ID (UUID)"
// @Success 200 {object} map[string]interface{}
// @Router /other-requests/documents/{id} [delete]
func (h *OtherRequestHandler) DeleteDocument(c *gin.Context) {
    documentID := c.Param("docId")
    if documentID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz doküman ID"})
        return
    }

    if err := h.reqService.DeleteRequestDocument(documentID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Doküman başarıyla silindi"})
}

// DeleteRequestType handles deleting a Request Type
// @Summary Talep Türünü siler
// @Tags request_types
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request Type ID"
// @Success 200 {object} map[string]interface{}
// @Router /request-types/{id} [delete]
func (h *OtherRequestHandler) DeleteRequestType(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    userEmail, exists := c.Get("userEmail")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User email not found in context"})
        return
    }

    if err := h.reqService.DeleteRequestType(uint(id), fmt.Sprintf("%v", userEmail)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Talep türü başarıyla silindi"})
}
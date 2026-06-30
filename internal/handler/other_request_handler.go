package handler

import (
    "net/http"
    "strconv"
    "fmt"

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

// CreateRequestType godoc
// @Summary Yeni Talep Türü oluşturur
// @Tags request_types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRequestTypeReq true "Request Type Data"
// @Success 201 {object} map[string]interface{}
// @Router /request-types [post]
func (h *OtherRequestHandler) CreateRequestType(c *gin.Context) {
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    if !isAdmin(roles) {
        c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Admin access required"})
        return
    }

    var req CreateRequestTypeReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    reqType := &domain.RequestType{
        Name:        req.Name,
        Description: req.Description,
        Active:      req.Active,
    }

    if err := h.reqService.CreateRequestType(reqType, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"success": true, "data": reqType, "message": "Talep türü başarıyla oluşturuldu"})
}

// GetAllRequestTypes godoc
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

// UpdateRequestType godoc
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
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    if !isAdmin(roles) {
        c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Admin access required"})
        return
    }

    id, err := parseUintParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    var req UpdateRequestTypeReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    reqType, err := h.reqService.GetRequestTypeByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Talep türü bulunamadı"})
        return
    }

    reqType.Name = req.Name
    reqType.Description = req.Description
    reqType.Active = req.Active

    if err := h.reqService.UpdateRequestType(reqType, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": reqType, "message": "Talep türü başarıyla güncellendi"})
}

// DeleteRequestType godoc
// @Summary Talep Türünü siler
// @Tags request_types
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request Type ID"
// @Success 200 {object} map[string]interface{}
// @Router /request-types/{id} [delete]
func (h *OtherRequestHandler) DeleteRequestType(c *gin.Context) {
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    if !isAdmin(roles) {
        c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Admin access required"})
        return
    }

    id, err := parseUintParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    if err := h.reqService.DeleteRequestType(id, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Talep türü başarıyla silindi"})
}

// ==================== 2. DİĞER TALEPLER İŞLEMLERİ (ÇALIŞAN / İK) ====================

// CreateRequest godoc
// @Summary Çalışan için yeni talep oluşturur
// @Tags other_requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateOtherRequestReq true "Request Data"
// @Router /other-requests [post]
func (h *OtherRequestHandler) CreateRequest(c *gin.Context) {
    userID, _, _, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    var req CreateOtherRequestReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    otherReq := &domain.OtherRequest{
        RequestTypeID: req.RequestTypeID,
        Description:   req.Description,
    }

    if err := h.reqService.CreateRequest(otherReq, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"success": true, "data": otherReq, "message": "Talep başarıyla oluşturuldu"})
}

// GetAllRequests godoc
// @Summary Talepleri listeler
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Router /other-requests [get]
func (h *OtherRequestHandler) GetAllRequests(c *gin.Context) {
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
    sortParams := types.SortParams{Sort: c.Query("sort"), Direction: c.Query("direction")}

    var filterEmployeeID *uint
    if !isAdmin(roles) {
        filterEmployeeID = &userID
    }

    reqs, total, err := h.reqService.GetAllRequests(filterEmployeeID, limit, offset, sortParams)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": reqs, "total": total})
}

// GetMyRequests godoc
// @Summary Giriş yapan çalışanın kendi taleplerini listeler
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Router /other-requests/me [get]
func (h *OtherRequestHandler) GetMyRequests(c *gin.Context) {
    userID, _, _, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    requests, err := h.reqService.GetRequestsByUserID(userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": requests})
}


// GetRequestByID godoc
// @Summary Talep detayını getirir
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request ID"
// @Success 200 {object} map[string]interface{}
// @Router /other-requests/{id} [get]
func (h *OtherRequestHandler) GetRequestByID(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
        return
    }

    req, err := h.reqService.GetRequestByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Talep bulunamadı"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": req})
}

// UpdateRequest godoc
// @Summary Talebi günceller (Talep tipi veya açıklaması değişirse statü ACTIVE olur)
// @Tags other_requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request ID"
// @Param request body UpdateOtherRequestReq true "Request Data"
// @Router /other-requests/{id} [put]
func (h *OtherRequestHandler) UpdateRequest(c *gin.Context) {
    userID, _, _, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    id, err := parseUintParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    var req UpdateOtherRequestReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    existingReq, err := h.reqService.GetRequestByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Talep bulunamadı"})
        return
    }

    existingReq.RequestTypeID = req.RequestTypeID
    existingReq.Description = req.Description

    if err := h.reqService.UpdateRequest(existingReq, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": existingReq, "message": "Talep başarıyla güncellendi"})
}

// CancelRequest godoc
// @Summary Çalışan tarafından talebi iptal eder
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request ID"
// @Router /other-requests/{id}/cancel [patch]
func (h *OtherRequestHandler) CancelRequest(c *gin.Context) {
    // 1. roles bilgisini _ yerine değişken olarak al
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    // 2. Admin yetkisini kontrol et
    isUserAdmin := isAdmin(roles)

    id, err := parseUintParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    // 3. Servise isAdmin bilgisini gönder
    if err := h.reqService.CancelRequest(id, userID, isUserAdmin); err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
    return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Talep başarıyla iptal edildi"})
}

// CompleteRequest godoc
// @Summary İK personeli tarafından talebi tamamlar
// @Tags other_requests
// @Produce json
// @Security BearerAuth
// @Param id path int true "Request ID"
// @Router /other-requests/{id}/complete [patch]
func (h *OtherRequestHandler) CompleteRequest(c *gin.Context) {
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    if !isAdmin(roles) {
        c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Admin access required"})
        return
    }

    id, err := parseUintParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
        return
    }

    if err := h.reqService.CompleteRequest(id, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Talep başarıyla tamamlandı"})
}

func (h *OtherRequestHandler) RollbackRequest(c *gin.Context) {
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    if !isAdmin(roles) {
        c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Admin access required"})
        return
    }

    id, err := parseUintParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz talep ID", "success": false})
        return
    }

    if err := h.reqService.RollbackRequest(id, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "success": false})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Talep başarıyla geri alındı", "success": true})
}

// ==================== 3. DÖKÜMAN / DOSYA YÖNETİMİ ====================

// UploadDocument godoc
func (h *OtherRequestHandler) UploadDocument(c *gin.Context) {
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    requestID, err := parseUintParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz talep ID"})
        return
    }

    file, err := c.FormFile("file")
    if err != nil {
        // Hata durumunda konsola hatayı yazdır
        fmt.Printf("Upload hatası: %v\n", err) 
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Dosya yüklenmedi: " + err.Error()})
        return
    }

    document, err := h.reqService.UploadRequestDocument(requestID, file, userID, isAdmin(roles))
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
func (h *OtherRequestHandler) GetDocuments(c *gin.Context) {
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    requestID, err := parseUintParam(c, "id")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz talep ID"})
        return
    }

    documents, err := h.reqService.GetRequestDocuments(requestID, userID, isAdmin(roles))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "data": documents})
}

// DeleteDocument godoc
func (h *OtherRequestHandler) DeleteDocument(c *gin.Context) {
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    documentID := c.Param("docId")
    if documentID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz doküman ID"})
        return
    }

    if err := h.reqService.DeleteRequestDocument(documentID, userID, isAdmin(roles)); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Doküman başarıyla silindi"})
}

// DownloadDocument godoc
func (h *OtherRequestHandler) DownloadDocument(c *gin.Context) {
    userID, _, roles, ok := getUserContext(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
        return
    }

    documentID := c.Param("docId")
    if documentID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz doküman ID"})
        return
    }

    url, err := h.reqService.DownloadRequestDocument(documentID, userID, isAdmin(roles))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "url": url,
        },
    })
}
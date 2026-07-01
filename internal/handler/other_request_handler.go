package handler

import (
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

// ==================== 2. DİĞER TALEPLER İŞLEMLERİ ====================

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

func (h *OtherRequestHandler) GetAllRequests(c *gin.Context) {
	_, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
		return
	}

	// YETKİ KISITLAMASI: Personel Talepleri Yönetimi (İK/Admin) ekranı
	if !isAdmin(roles) && !isHR(roles) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Admin or HR access required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	sortParams := types.SortParams{Sort: c.Query("sort"), Direction: c.Query("direction")}

	// Admin veya HR tüm talepleri görür (filterEmployeeID = nil)
	reqs, total, err := h.reqService.GetAllRequests(nil, limit, offset, sortParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": reqs, "total": total})
}

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

func (h *OtherRequestHandler) UpdateRequest(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
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

	if err := h.reqService.UpdateRequest(existingReq, userID, isAdmin(roles)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": existingReq, "message": "Talep başarıyla güncellendi"})
}

func (h *OtherRequestHandler) CancelRequest(c *gin.Context) {
	userID, _, roles, ok := getUserContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Authentication required"})
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
		return
	}

	if err := h.reqService.CancelRequest(id, userID, isAdmin(roles)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Talep başarıyla iptal edildi"})
}

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
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Dosya yüklenmedi: " + err.Error()})
		return
	}

	document, err := h.reqService.UploadRequestDocument(requestID, file, userID, isAdmin(roles), isHR(roles))
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

	documents, err := h.reqService.GetRequestDocuments(requestID, userID, isAdmin(roles), isHR(roles))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": documents})
}

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

	if err := h.reqService.DeleteRequestDocument(documentID, userID, isAdmin(roles), isHR(roles)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Doküman başarıyla silindi"})
}

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

	url, err := h.reqService.DownloadRequestDocument(documentID, userID, isAdmin(roles), isHR(roles))
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

func isHR(roles interface{}) bool {
	userRoles, ok := roles.([]string)
	if !ok {
		return false
	}
	for _, r := range userRoles {
		if r == "HR" {
			return true
		}
	}
	return false
}
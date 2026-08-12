package handler

import (
	"net/http"
	"strconv"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
)

// AcademyHandler Kartezya Akademi HTTP handler'ı
type AcademyHandler struct {
	academyService  service.AcademyService
	employeeRepo    repository.EmployeeRepository
	documentService service.DocumentService
}

func NewAcademyHandler(academyService service.AcademyService, employeeRepo repository.EmployeeRepository, documentService service.DocumentService) *AcademyHandler {
	return &AcademyHandler{
		academyService:  academyService,
		employeeRepo:    employeeRepo,
		documentService: documentService,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateTrainingRequest struct {
	Title       string                `json:"title" binding:"required"`
	Description string                `json:"description"`
	Duration    int                   `json:"duration"`
	Status      domain.TrainingStatus `json:"status"`
}

type UpdateTrainingRequest struct {
	Title       string                `json:"title" binding:"required"`
	Description string                `json:"description"`
	Duration    int                   `json:"duration"`
	Status      domain.TrainingStatus `json:"status"`
}

type AssignEmployeeRequest struct {
	EmployeeIDs []uint `json:"employee_ids" binding:"required,min=1"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Training Endpoints (Admin)
// ─────────────────────────────────────────────────────────────────────────────

// ListTrainings godoc
// @Summary Eğitim listesi
// @Tags academy
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /academy/trainings [get]
func (h *AcademyHandler) ListTrainings(c *gin.Context) {
	limit := academyParseIntQuery(c, "limit", 20)
	offset := academyParseIntQuery(c, "offset", 0)

	// Admin rolündeyse tüm eğitimler, değilse sadece aktif olanlar
	_, isAdmin := c.Get("isAdmin")
	rolesRaw, _ := c.Get("roles")
	roles, _ := rolesRaw.([]string)
	admin := isAdmin || containsAdminOrHR(roles)

	list, total, err := h.academyService.ListTrainings(admin, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list, "total": total})
}

// GetTraining godoc
// @Summary Eğitim detayı
// @Tags academy
// @Security BearerAuth
// @Param id path int true "Training ID"
// @Success 200 {object} map[string]interface{}
// @Router /academy/trainings/{id} [get]
func (h *AcademyHandler) GetTraining(c *gin.Context) {
	id, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
		return
	}
	training, err := h.academyService.GetTraining(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Eğitim bulunamadı"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": training})
}

// CreateTraining godoc
// @Summary Yeni eğitim oluştur (Admin)
// @Tags academy
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateTrainingRequest true "Eğitim Bilgileri"
// @Success 201 {object} map[string]interface{}
// @Router /academy/trainings [post]
func (h *AcademyHandler) CreateTraining(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz form veya çok büyük dosya", "details": err.Error()})
		return
	}

	title := c.PostForm("title")
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Eğitim başlığı (title) zorunludur"})
		return
	}

	description := c.PostForm("description")
	durationStr := c.PostForm("duration")
	statusStr := c.PostForm("status")

	duration := 0
	if durationStr != "" {
		if d, err := strconv.Atoi(durationStr); err == nil {
			duration = d
		}
	}

	status := domain.TrainingStatusActive
	if statusStr == string(domain.TrainingStatusInactive) {
		status = domain.TrainingStatusInactive
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Eğitim PDF dosyası zorunludur"})
		return
	}

	userEmail := academyGetUserEmail(c)
	userIDRaw, _ := c.Get("userID")
	userID := userIDRaw.(uint)
	
	var roles []string
	if rolesRaw, exists := c.Get("roles"); exists {
		if r, ok := rolesRaw.([]string); ok {
			roles = r
		}
	}

	training := &domain.Training{
		Title:       title,
		Description: description,
		Duration:    duration,
		Status:      status,
	}

	// 1. Create Training and Bulk Assign (in Service)
	if err := h.academyService.CreateTraining(training, userEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 3. Upload Document
	attachment, err := h.documentService.UploadDocument(file, userID, domain.AttachmentRelatedTypeAcademy, domain.AttachmentTypeDocument)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Eğitim oluşturuldu ancak PDF yüklenemedi", "details": err.Error()})
		return
	}

	// 4. Link Document to Training
	err = h.documentService.LinkDocumentsToRecord([]string{attachment.ID}, domain.AttachmentRelatedTypeAcademy, training.ID, userID, roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "PDF eğitime bağlanamadı", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": training, "attachment": attachment})
}

// UpdateTraining godoc
// @Summary Eğitimi güncelle (Admin)
// @Tags academy
// @Security BearerAuth
// @Param id path int true "Training ID"
// @Accept json
// @Produce json
// @Param request body UpdateTrainingRequest true "Eğitim Bilgileri"
// @Success 200 {object} map[string]interface{}
// @Router /academy/trainings/{id} [put]
func (h *AcademyHandler) UpdateTraining(c *gin.Context) {
	id, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
		return
	}


	var req UpdateTrainingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz istek", "details": err.Error()})
		return
	}

	userEmail := academyGetUserEmail(c)

	training := &domain.Training{
		Title:       req.Title,
		Description: req.Description,
		Duration:    req.Duration,
		Status:      req.Status,
	}
	training.ID = id

	if err := h.academyService.UpdateTraining(training, userEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": training})
}

// DeleteTraining godoc
// @Summary Eğitimi sil (Admin)
// @Tags academy
// @Security BearerAuth
// @Param id path int true "Training ID"
// @Success 200 {object} map[string]interface{}
// @Router /academy/trainings/{id} [delete]
func (h *AcademyHandler) DeleteTraining(c *gin.Context) {
	id, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
		return
	}
	userEmail := academyGetUserEmail(c)
	if err := h.academyService.DeleteTraining(id, userEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Eğitim silindi"})
}

// ─────────────────────────────────────────────────────────────────────────────
// Assignment Endpoints
// ─────────────────────────────────────────────────────────────────────────────

// AssignEmployee godoc
// @Summary Eğitime çalışan ata (Admin)
// @Tags academy
// @Security BearerAuth
// @Param id path int true "Training ID"
// @Param request body AssignEmployeeRequest true "Çalışan ID"
// @Success 201 {object} map[string]interface{}
// @Router /academy/trainings/{id}/assign [post]
func (h *AcademyHandler) AssignEmployee(c *gin.Context) {
	trainingID, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz eğitim ID"})
		return
	}
	var req AssignEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz veri", "details": err.Error()})
		return
	}
	userEmail := academyGetUserEmail(c)
	err = h.academyService.AssignToEmployees(trainingID, req.EmployeeIDs, userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Atama yapıldı"})
}

// ListTrainingAssignments godoc
// @Summary Eğitime atanan çalışanları listele (Admin)
// @Tags academy
// @Security BearerAuth
// @Param id path int true "Training ID"
// @Success 200 {object} map[string]interface{}
// @Router /academy/trainings/{id}/assignments [get]
func (h *AcademyHandler) ListTrainingAssignments(c *gin.Context) {
	trainingID, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
		return
	}
	limit := academyParseIntQuery(c, "limit", 50)
	offset := academyParseIntQuery(c, "offset", 0)

	list, total, err := h.academyService.ListAssignmentsByTraining(trainingID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list, "total": total})
}

// RemoveAssignment godoc
// @Summary Atamayı kaldır (Admin)
// @Tags academy
// @Security BearerAuth
// @Param id path int true "Assignment ID"
// @Success 200 {object} map[string]interface{}
// @Router /academy/assignments/{id} [delete]
func (h *AcademyHandler) RemoveAssignment(c *gin.Context) {
	id, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
		return
	}
	userEmail := academyGetUserEmail(c)
	if err := h.academyService.RemoveAssignment(id, userEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Atama kaldırıldı"})
}

// GetMyAssignments godoc
// @Summary Benim eğitimlerim
// @Tags academy
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /academy/assignments/me [get]
func (h *AcademyHandler) GetMyAssignments(c *gin.Context) {
	employeeID, err := getEmployeeID(c, h.employeeRepo)
	if err != nil {
		// If error is httpError, use its code. Else use 401.
		code := http.StatusUnauthorized
		if e, ok := err.(*httpError); ok {
			code = e.code
		}
		c.JSON(code, gin.H{"success": false, "error": err.Error()})
		return
	}
	list, err := h.academyService.GetMyAssignments(employeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
}

// StartTraining godoc
// @Summary Eğitimi başlat
// @Tags academy
// @Security BearerAuth
// @Param id path int true "Assignment ID"
// @Success 200 {object} map[string]interface{}
// @Router /academy/assignments/{id}/start [post]
func (h *AcademyHandler) StartTraining(c *gin.Context) {
	id, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
		return
	}
	employeeID, err := getEmployeeID(c, h.employeeRepo)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
		return
	}
	userEmail := academyGetUserEmail(c)
	if err := h.academyService.StartTraining(id, employeeID, userEmail); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Eğitim başlatıldı"})
}

// CompleteTraining godoc
// @Summary Eğitimi tamamla ve sertifika al
// @Tags academy
// @Security BearerAuth
// @Param id path int true "Assignment ID"
// @Success 200 {object} map[string]interface{}
// @Router /academy/assignments/{id}/complete [post]
func (h *AcademyHandler) CompleteTraining(c *gin.Context) {
	id, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
		return
	}
	employeeID, err := getEmployeeID(c, h.employeeRepo)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
		return
	}
	userEmail := academyGetUserEmail(c)
	cert, err := h.academyService.CompleteTraining(id, employeeID, userEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cert, "message": "Tebrikler! Eğitim tamamlandı."})
}

// ─────────────────────────────────────────────────────────────────────────────
// Certificate Endpoints
// ─────────────────────────────────────────────────────────────────────────────

// GetMyCertificates godoc
// @Summary Sertifikalarım
// @Tags academy
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /academy/certificates/me [get]
func (h *AcademyHandler) GetMyCertificates(c *gin.Context) {
	employeeID, err := getEmployeeID(c, h.employeeRepo)
	if err != nil {
		code := http.StatusUnauthorized
		if e, ok := err.(*httpError); ok {
			code = e.code
		}
		c.JSON(code, gin.H{"success": false, "error": err.Error()})
		return
	}
	list, err := h.academyService.GetMyCertificates(employeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list})
}

// GetCertificateByCode godoc
// @Summary Sertifika doğrula
// @Tags academy
// @Security BearerAuth
// @Param code path string true "Sertifika Kodu"
// @Success 200 {object} map[string]interface{}
// @Router /academy/certificates/{code} [get]
func (h *AcademyHandler) GetCertificateByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz sertifika kodu"})
		return
	}
	cert, err := h.academyService.GetCertificateByCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Sertifika bulunamadı"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cert})
}

// ─────────────────────────────────────────────────────────────────────────────
// Private helpers
// ─────────────────────────────────────────────────────────────────────────────

func academyParseUintParam(c *gin.Context, key string) (uint, error) {
	val, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}

func academyParseIntQuery(c *gin.Context, key string, defaultVal int) int {
	s := c.Query(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return defaultVal
	}
	return v
}

func academyGetUserEmail(c *gin.Context) string {
	email, _ := c.Get("email")
	s, _ := email.(string)
	return s
}

// getEmployeeID, JWT userID'den çalışan ID'sini resolve eder.
func getEmployeeID(c *gin.Context, employeeRepo repository.EmployeeRepository) (uint, error) {
	userIDRaw, exists := c.Get("userID")
	if !exists {
		return 0, errUnauthorized()
	}
	userID, ok := userIDRaw.(uint)
	if !ok {
		return 0, errUnauthorized()
	}
	emp, err := employeeRepo.GetByUserID(userID)
	if err != nil {
		return 0, &httpError{code: http.StatusBadRequest, msg: "Çalışan kaydı bulunamadı"}
	}
	return emp.ID, nil
}

func errUnauthorized() error {
	return &httpError{code: http.StatusUnauthorized, msg: "Kimlik doğrulanamadı"}
}

type httpError struct {
	code int
	msg  string
}

func (e *httpError) Error() string { return e.msg }

func containsAdminOrHR(roles []string) bool {
	for _, r := range roles {
		if r == "ADMIN" || r == "HR" {
			return r != ""
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// Academy Survey Endpoints
// ─────────────────────────────────────────────────────────────────────────────

type CreateSurveyRequest struct {
	Title         string   `json:"title" binding:"required"`
	Description   string   `json:"description"`
	IsMultiSelect bool     `json:"is_multi_select"`
	IsActive      bool     `json:"is_active"`
	Options       []string `json:"options" binding:"required,min=2"`
}

type SubmitVoteRequest struct {
	OptionIDs []uint `json:"option_ids" binding:"required,min=1"`
}

// CreateSurvey godoc
func (h *AcademyHandler) CreateSurvey(c *gin.Context) {
	var req CreateSurveyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	userEmail, _ := c.Get("email")
	emailStr, _ := userEmail.(string)

	survey := &domain.AcademySurvey{
		Title:         req.Title,
		Description:   req.Description,
		IsMultiSelect: req.IsMultiSelect,
		IsActive:      req.IsActive,
	}

	for _, text := range req.Options {
		survey.Options = append(survey.Options, domain.AcademySurveyOption{Text: text})
	}

	if err := h.academyService.CreateSurvey(survey, emailStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": survey})
}

// ListSurveys godoc
func (h *AcademyHandler) ListSurveys(c *gin.Context) {
	limit := academyParseIntQuery(c, "limit", 20)
	offset := academyParseIntQuery(c, "offset", 0)

	_, isAdmin := c.Get("isAdmin")
	rolesRaw, _ := c.Get("roles")
	roles, _ := rolesRaw.([]string)
	admin := isAdmin || containsAdminOrHR(roles)

	list, total, err := h.academyService.ListSurveys(admin, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Admin ise sonuçları (oy dağılımlarını) da ekleyelim
	type SurveyResponse struct {
		*domain.AcademySurvey
		Results map[uint]int `json:"results,omitempty"`
	}

	var responseList []SurveyResponse
	for _, s := range list {
		sr := SurveyResponse{AcademySurvey: s}
		if admin {
			resMap, _ := h.academyService.GetSurveyResults(s.ID)
			sr.Results = resMap
		}
		responseList = append(responseList, sr)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": responseList, "total": total})
}

// DeleteSurvey godoc
func (h *AcademyHandler) DeleteSurvey(c *gin.Context) {
	id, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
		return
	}
	userEmail, _ := c.Get("email")
	emailStr, _ := userEmail.(string)

	if err := h.academyService.DeleteSurvey(id, emailStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Anket silindi"})
}

// SubmitSurveyVote godoc
func (h *AcademyHandler) SubmitSurveyVote(c *gin.Context) {
	surveyID, err := academyParseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Geçersiz ID"})
		return
	}

	var req SubmitVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	userIDRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Kullanıcı bilgisi bulunamadı"})
		return
	}
	userID, _ := userIDRaw.(uint)

	emp, err := h.employeeRepo.GetByUserID(userID)
	if err != nil || emp == nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Çalışan profili bulunamadı"})
		return
	}

	userEmail, _ := c.Get("email")
	emailStr, _ := userEmail.(string)

	if err := h.academyService.SubmitSurveyVote(surveyID, emp.ID, req.OptionIDs, emailStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Oyunuz kaydedildi"})
}

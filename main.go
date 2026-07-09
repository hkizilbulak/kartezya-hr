package main

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/database"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/handler"
	"kartezya-hr/internal/jobs"
	"kartezya-hr/internal/middleware"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/service"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/bcrypt"

	_ "kartezya-hr/docs" // This line is necessary for swagger to find your docs
)

// @title Kartezya HR Management System API
// @version 1.0
// @description This is a comprehensive HR Management System API built with Go and Gin framework.
// @description It provides endpoints for managing employees, leave requests, companies, departments, and more.

// @contact.name API Support
// @contact.email support@kartezya.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Set config for domain models (for dynamic table naming)
	domain.SetConfig(cfg)

	// Initialize database
	db := database.NewDatabase(cfg)
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed database with default data
	if err := seedDatabase(db); err != nil {
		log.Printf("Warning: Failed to seed database: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB)
	roleRepo := repository.NewRoleRepository(db.DB)
	userRoleRepo := repository.NewUserRoleRepository(db.DB)
	employeeRepo := repository.NewEmployeeRepository(db.DB)
	leaveRepo := repository.NewLeaveRepository(db.DB)
	leaveBalanceRepo := repository.NewLeaveBalanceRepository(db.DB)
	auditRepo := repository.NewAuditRepository(db.DB)
	companyRepo := repository.NewCompanyRepository(db.DB)
	departmentRepo := repository.NewDepartmentRepository(db.DB)
	jobPositionRepo := repository.NewJobPositionRepository(db.DB)
	workInfoRepo := repository.NewWorkInformationRepository(db.DB)
	leaveTypeRepo := repository.NewLeaveTypeRepository(db.DB)
	holidayRepo := repository.NewHolidayRepository(db.DB)
	gradeRepo := repository.NewGradeRepository(db.DB)
	employeeGradeRepo := repository.NewEmployeeGradeRepository(db.DB)
	employeeContractRepo := repository.NewEmployeeContractRepository(db.DB)
	contractRepo := repository.NewContractRepository(db.DB)
	attachmentRepo := repository.NewAttachmentRepository(db.DB)
	expenseRepo := repository.NewExpenseRepository(db.DB)
	expenseTypeRepo := repository.NewExpenseTypeRepository(db.DB)
	jobRepo := repository.NewJobRepository(db.DB)
	eventRepo := repository.NewEventRepository(db.DB)
	eventParticipantRepo := repository.NewEventParticipantRepository(db.DB)
	faqRepo := repository.NewFAQRepository(db.DB)
	otherRequestRepo := repository.NewOtherRequestRepository(db.DB)
	mailConfigRepo := repository.NewMailConfigRepository(db.DB)

	// Initialize storage provider
	var storageProvider service.StorageProvider
	switch cfg.Storage.Provider {
	case "local":
		storageProvider = service.NewLocalStorageProvider(cfg.Storage.BasePath, cfg.Storage.BaseURL)
	case "s3", "backblaze":
		s3Provider, err := service.NewS3StorageProvider(
			cfg.Storage.S3Endpoint,
			cfg.Storage.S3Region,
			cfg.Storage.S3Bucket,
			cfg.Storage.S3BasePath,
			cfg.Storage.S3AccessKey,
			cfg.Storage.S3SecretKey,
		)
		if err != nil {
			log.Fatalf("Failed to initialize S3 storage: %v", err)
		}
		storageProvider = s3Provider
		log.Printf("Using S3-compatible storage: %s (bucket: %s, base path: %s)", cfg.Storage.S3Endpoint, cfg.Storage.S3Bucket, cfg.Storage.S3BasePath)
	default:
		log.Printf("Unknown storage provider: %s, using local storage", cfg.Storage.Provider)
		storageProvider = service.NewLocalStorageProvider(cfg.Storage.BasePath, cfg.Storage.BaseURL)
	}

	// Initialize services
	auditService := service.NewAuditService(auditRepo)
	authService := service.NewAuthService(userRepo, userRoleRepo, roleRepo, auditService, cfg)
	emailService := service.NewEmailService(cfg, userRepo)
	documentService := service.NewDocumentService(attachmentRepo, storageProvider, cfg)
	mailConfigService := service.NewMailConfigService(mailConfigRepo)
	employeeService := service.NewEmployeeService(employeeRepo, userRepo, userRoleRepo, roleRepo, authService, auditService, workInfoRepo, emailService, mailConfigService)
	leaveService := service.NewLeaveService(leaveRepo, leaveTypeRepo, leaveBalanceRepo, employeeRepo, holidayRepo, attachmentRepo, storageProvider, auditService)
	departmentService := service.NewDepartmentService(departmentRepo, companyRepo, auditService)
	companyService := service.NewCompanyService(companyRepo, departmentRepo, departmentService, auditService)
	jobPositionService := service.NewJobPositionService(jobPositionRepo, auditService)
	workInfoService := service.NewWorkInformationService(workInfoRepo, employeeRepo, companyRepo, departmentRepo, jobPositionRepo, auditService)
	lookupService := service.NewLookupService(companyRepo, departmentRepo, jobPositionRepo, leaveTypeRepo, gradeRepo, roleRepo)
	gradeService := service.NewGradeService(gradeRepo, auditService)
	employeeGradeService := service.NewEmployeeGradeService(employeeGradeRepo, employeeRepo, gradeRepo, auditService)
	employeeContractService := service.NewEmployeeContractService(employeeContractRepo, employeeRepo, auditService)
	contractService := service.NewContractService(contractRepo, employeeContractRepo, auditService)
	expenseService := service.NewExpenseService(expenseRepo, expenseTypeRepo, attachmentRepo, employeeRepo, storageProvider, auditService)
	reportService := service.NewReportService(employeeRepo, workInfoRepo, leaveRepo, holidayRepo, leaveService)
	jobService := service.NewJobService(jobRepo, auditService)
	eventService := service.NewEventService(eventRepo, eventParticipantRepo, userRepo, employeeRepo, emailService, mailConfigService, cfg)
	faqService := service.NewFAQService(faqRepo, auditService)
	otherRequestService := service.NewOtherRequestService(otherRequestRepo, attachmentRepo, auditService, emailService, storageProvider, employeeRepo, mailConfigService)

	// Initialize and start scheduled jobs
	scheduler := jobs.NewScheduler(db.DB, documentService, jobService, emailService, mailConfigService, reportService)
	scheduler.Start()
	defer scheduler.Stop()

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, emailService, userRepo, employeeRepo, mailConfigService)
	employeeHandler := handler.NewEmployeeHandler(employeeService)
	leaveHandler := handler.NewLeaveHandler(leaveService, employeeService, emailService, mailConfigService)
	companyHandler := handler.NewCompanyHandler(companyService)
	departmentHandler := handler.NewDepartmentHandler(departmentService)
	jobPositionHandler := handler.NewJobPositionHandler(jobPositionService)
	workInfoHandler := handler.NewWorkInformationHandler(workInfoService, employeeService)
	lookupHandler := handler.NewLookupHandler(lookupService)
	dashboardHandler := handler.NewDashboardHandler(employeeService, departmentService, companyService, leaveService, expenseService, eventService)
	gradeHandler := handler.NewGradeHandler(gradeService)
	employeeGradeHandler := handler.NewEmployeeGradeHandler(employeeGradeService, employeeService)
	employeeContractHandler := handler.NewEmployeeContractHandler(employeeContractService, employeeService)
	contractHandler := handler.NewContractHandler(contractService)
	reportHandler := handler.NewReportHandler(reportService, emailService, mailConfigService, cfg)
	documentHandler := handler.NewDocumentHandler(documentService)
	expenseHandler := handler.NewExpenseHandler(expenseService, employeeService, emailService, mailConfigService)
	jobHandler := handler.NewJobHandler(jobService, scheduler)
	eventHandler := handler.NewEventHandler(eventService)
	faqHandler := handler.NewFAQHandler(faqService)
	otherRequestHandler := handler.NewOtherRequestHandler(otherRequestService)
	emailHandler := handler.NewEmailHandler(emailService, mailConfigService, cfg)
	mailConfigHandler := handler.NewMailConfigHandler(mailConfigService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Initialize router
	router := gin.Default()

	// Error logging middleware to capture response errors
	router.Use(func(c *gin.Context) {
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		if c.Writer.Status() >= 400 {
			log.Printf("[API ERROR] %s %s - Status: %d - Response: %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), blw.body.String())

			for _, e := range c.Errors {
				log.Printf("[INTERNAL EXCEPTION] %v", e)
			}
		}
	})

	// Add CORS middleware with wildcard for development
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Requested-With, Origin")
		c.Header("Access-Control-Allow-Credentials", "false")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	// Serve static files for local storage (uploads)
	if cfg.Storage.Provider == "local" {
		router.Static("/uploads", cfg.Storage.BasePath)
		log.Printf("Serving static files from %s at /uploads", cfg.Storage.BasePath)
	}

	// API v1 routes
	v1 := router.Group("/api/v1")

	// Public routes (no authentication required)
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		// şifre sıfırlama mekanizması
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/validate-reset-token", authHandler.ValidateResetToken)
		auth.POST("/reset-password", authHandler.ResetPassword)

		// Yandex OAuth routes
		auth.GET("/yandex/login", authHandler.YandexLogin)
		auth.GET("/yandex/callback", authHandler.YandexCallback)
	}

	// Public lookup routes
	lookup := v1.Group("/lookup")
	{
		lookup.GET("/companies", lookupHandler.GetCompaniesLookup)
		lookup.GET("/departments", lookupHandler.GetDepartmentsLookup)
		lookup.GET("/departments-by-company", lookupHandler.GetDepartmentsByCompanyLookup)
		lookup.GET("/job-positions", lookupHandler.GetJobPositionsLookup)
		lookup.GET("/leave-types", lookupHandler.GetLeaveTypesLookup)
		lookup.GET("/grades", lookupHandler.GetGradesLookup)
	}

	// Protected lookup routes (authentication required)
	protectedLookup := v1.Group("/lookup")
	protectedLookup.Use(authMiddleware.JWTAuth())
	{
		protectedLookup.GET("/roles", authMiddleware.RequireAdmin(), lookupHandler.GetRolesLookup)
	}

	// Protected routes (authentication required)
	protected := v1.Group("")
	protected.Use(authMiddleware.JWTAuth())
	{
		// Auth routes
		authRoutes := protected.Group("/auth")
		{
			authRoutes.POST("/logout", authHandler.Logout)
			authRoutes.POST("/change-password", authHandler.ChangePassword)
			authRoutes.POST("/send-password-reset-email", authMiddleware.RequireAdmin(), authHandler.SendPasswordResetEmail)
		}

		// Employee routes
		employeeRoutes := protected.Group("/employees")
		{
			employeeRoutes.GET("/me", employeeHandler.GetMyProfile)
			employeeRoutes.PUT("/me", employeeHandler.UpdateMyProfile)
			employeeRoutes.PUT("/:id", employeeHandler.UpdateEmployee)

			// Admin only routes
			employeeRoutes.GET("/:id", authMiddleware.RequireAdmin(), employeeHandler.GetEmployeeByID)
			employeeRoutes.POST("", authMiddleware.RequireAdmin(), employeeHandler.CreateEmployee)
			employeeRoutes.GET("", authMiddleware.RequireAdmin(), employeeHandler.ListEmployees)
			employeeRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), employeeHandler.DeleteEmployee)
		}

		// Leave management routes
		leaveRoutes := protected.Group("/leave")
		{
			// Leave requests
			requests := leaveRoutes.Group("/requests")
			{
				requests.POST("", leaveHandler.CreateLeaveRequest)
				requests.GET("/me", leaveHandler.GetMyLeaveRequests)
				requests.PUT("/:id", leaveHandler.UpdateLeaveRequest)
				requests.POST("/:id/cancel", leaveHandler.CancelLeaveRequest)

				// Document routes
				requests.POST("/:id/documents", leaveHandler.UploadLeaveDocument)
				requests.GET("/:id/documents", leaveHandler.GetLeaveDocuments)

				// Admin only routes
				requests.GET("/:id", authMiddleware.RequireAdmin(), leaveHandler.GetLeaveRequestByID)
				requests.GET("", authMiddleware.RequireAdmin(), leaveHandler.GetAllLeaveRequests)
				requests.POST("/:id/approve", authMiddleware.RequireAdmin(), leaveHandler.ApproveLeaveRequest)
				requests.POST("/:id/reject", authMiddleware.RequireAdmin(), leaveHandler.RejectLeaveRequest)
			}

			// Leave document operations (by document ID)
			leaveRoutes.DELETE("/documents/:id", leaveHandler.DeleteLeaveDocument)
			leaveRoutes.GET("/documents/:id/download", leaveHandler.DownloadLeaveDocument)

			// Leave balances
			balances := leaveRoutes.Group("/balances")
			{
				balances.GET("/me", leaveHandler.GetMyLeaveBalances)
				balances.GET("", authMiddleware.RequireAdmin(), leaveHandler.GetLeaveBalances)
			}

			// Leave types (Admin only)
			typesRoutes := leaveRoutes.Group("/types")
			{
				typesRoutes.GET("", authMiddleware.RequireAdmin(), leaveHandler.ListLeaveTypes)
				typesRoutes.POST("", authMiddleware.RequireAdmin(), leaveHandler.CreateLeaveType)
				typesRoutes.GET("/:id", authMiddleware.RequireAdmin(), leaveHandler.GetLeaveTypeByID)
				typesRoutes.PUT("/:id", authMiddleware.RequireAdmin(), leaveHandler.UpdateLeaveType)
				typesRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), leaveHandler.DeleteLeaveType)
			}

			// Calculate days endpoints
			leaveRoutes.POST("/calculate-working-days", leaveHandler.CalculateWorkingDays)
		}

		// Company management routes
		companyRoutes := protected.Group("/companies")
		{
			// Admin only routes
			companyRoutes.GET("", authMiddleware.RequireAdmin(), companyHandler.GetCompanies)
			companyRoutes.GET("/:id", authMiddleware.RequireAdmin(), companyHandler.GetCompany)
			companyRoutes.POST("", authMiddleware.RequireAdmin(), companyHandler.CreateCompany)
			companyRoutes.PUT("/:id", authMiddleware.RequireAdmin(), companyHandler.UpdateCompany)
			companyRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), companyHandler.DeleteCompany)
		}

		// Department management routes
		departmentRoutes := protected.Group("/departments")
		{
			// Admin only routes
			departmentRoutes.GET("", authMiddleware.RequireAdmin(), departmentHandler.GetDepartments)
			departmentRoutes.GET("/:id", authMiddleware.RequireAdmin(), departmentHandler.GetDepartment)
			departmentRoutes.POST("", authMiddleware.RequireAdmin(), departmentHandler.CreateDepartment)
			departmentRoutes.PUT("/:id", authMiddleware.RequireAdmin(), departmentHandler.UpdateDepartment)
			departmentRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), departmentHandler.DeleteDepartment)
		}

		// FAQ (Sıkça Sorulan Sorular) management routes
		faqRoutes := protected.Group("/faqs")
		{
			// Tüm personelin görebileceği kısımlar
			faqRoutes.GET("", faqHandler.GetAll)
			faqRoutes.GET("/:id", faqHandler.GetByID)

			// Sadece Admin'in değiştirebileceği kısımlar
			faqRoutes.POST("", authMiddleware.RequireAdmin(), faqHandler.Create)
			faqRoutes.PUT("/:id", authMiddleware.RequireAdmin(), faqHandler.Update)
			faqRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), faqHandler.Delete)
		}

		// Other Requests management routes
		otherReqRoutes := protected.Group("/other-requests")
		{
			otherReqRoutes.POST("", otherRequestHandler.CreateRequest)
			otherReqRoutes.GET("/me", otherRequestHandler.GetMyRequests)
			otherReqRoutes.GET("/:id", otherRequestHandler.GetRequestByID)
			otherReqRoutes.PUT("/:id", otherRequestHandler.UpdateRequest)
			otherReqRoutes.PATCH("/:id/cancel", otherRequestHandler.CancelRequest)

			otherReqRoutes.POST("/:id/documents", otherRequestHandler.UploadDocument)
			otherReqRoutes.GET("/:id/documents", otherRequestHandler.GetDocuments)
			otherReqRoutes.DELETE("/documents/:docId", otherRequestHandler.DeleteDocument)
			otherReqRoutes.GET("/documents/:docId/download", otherRequestHandler.DownloadDocument)

			otherReqRoutes.GET("", authMiddleware.RequireAdmin(), otherRequestHandler.GetAllRequests)
			otherReqRoutes.PATCH("/:id/complete", authMiddleware.RequireAdmin(), otherRequestHandler.CompleteRequest)
			otherReqRoutes.PATCH("/:id/rollback", authMiddleware.RequireAdmin(), otherRequestHandler.RollbackRequest)
		}

		// Request Types management routes
		requestTypeRoutes := protected.Group("/request-types")
		{
			requestTypeRoutes.GET("", otherRequestHandler.GetAllRequestTypes)

			requestTypeRoutes.POST("", authMiddleware.RequireAdmin(), otherRequestHandler.CreateRequestType)
			requestTypeRoutes.PUT("/:id", authMiddleware.RequireAdmin(), otherRequestHandler.UpdateRequestType)
			requestTypeRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), otherRequestHandler.DeleteRequestType)
		}

		// Job Position management routes
		jobPositionRoutes := protected.Group("/job-positions")
		{
			// Admin only routes
			jobPositionRoutes.GET("", authMiddleware.RequireAdmin(), jobPositionHandler.GetJobPositions)
			jobPositionRoutes.GET("/:id", authMiddleware.RequireAdmin(), jobPositionHandler.GetJobPosition)
			jobPositionRoutes.POST("", authMiddleware.RequireAdmin(), jobPositionHandler.CreateJobPosition)
			jobPositionRoutes.PUT("/:id", authMiddleware.RequireAdmin(), jobPositionHandler.UpdateJobPosition)
			jobPositionRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), jobPositionHandler.DeleteJobPosition)
		}

		// Grade management routes
		gradeRoutes := protected.Group("/grades")
		{
			// Admin only routes
			gradeRoutes.GET("", authMiddleware.RequireAdmin(), gradeHandler.GetGrades)
			gradeRoutes.GET("/:id", authMiddleware.RequireAdmin(), gradeHandler.GetGrade)
			gradeRoutes.POST("", authMiddleware.RequireAdmin(), gradeHandler.CreateGrade)
			gradeRoutes.PUT("/:id", authMiddleware.RequireAdmin(), gradeHandler.UpdateGrade)
			gradeRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), gradeHandler.DeleteGrade)
		}

		// Work Information management routes
		workInfoRoutes := protected.Group("/work-information")
		{
			// Employee can view their own work information
			workInfoRoutes.GET("/me", workInfoHandler.GetMyWorkInformation)

			// Admin only routes
			workInfoRoutes.GET("/:id", authMiddleware.RequireAdmin(), workInfoHandler.GetWorkInformationByID)
			workInfoRoutes.GET("", authMiddleware.RequireAdmin(), workInfoHandler.ListWorkInformation)
			workInfoRoutes.POST("", authMiddleware.RequireAdmin(), workInfoHandler.CreateWorkInformation)
			workInfoRoutes.PUT("/:id", authMiddleware.RequireAdmin(), workInfoHandler.UpdateWorkInformation)
			workInfoRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), workInfoHandler.DeleteWorkInformation)
		}

		// Employee Grade management routes
		employeeGradeRoutes := protected.Group("/employee-grades")
		{
			// Employee can view their own grades
			employeeGradeRoutes.GET("/me", employeeGradeHandler.GetMyEmployeeGrades)

			// Admin only routes
			employeeGradeRoutes.GET("/:id", authMiddleware.RequireAdmin(), employeeGradeHandler.GetEmployeeGradeByID)
			employeeGradeRoutes.GET("", authMiddleware.RequireAdmin(), employeeGradeHandler.ListEmployeeGrades)
			employeeGradeRoutes.POST("", authMiddleware.RequireAdmin(), employeeGradeHandler.CreateEmployeeGrade)
			employeeGradeRoutes.PUT("/:id", authMiddleware.RequireAdmin(), employeeGradeHandler.UpdateEmployeeGrade)
			employeeGradeRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), employeeGradeHandler.DeleteEmployeeGrade)
		}

		// Employee Contract management routes
		employeeContractRoutes := protected.Group("/employee-contracts")
		{
			// Employee can view their own contracts
			employeeContractRoutes.GET("/me", employeeContractHandler.GetMyEmployeeContracts)

			// Admin only routes
			employeeContractRoutes.GET("/:id", authMiddleware.RequireAdmin(), employeeContractHandler.GetEmployeeContractByID)
			employeeContractRoutes.GET("", authMiddleware.RequireAdmin(), employeeContractHandler.ListEmployeeContracts)
			employeeContractRoutes.POST("", authMiddleware.RequireAdmin(), employeeContractHandler.CreateEmployeeContract)
			employeeContractRoutes.PUT("/:id", authMiddleware.RequireAdmin(), employeeContractHandler.UpdateEmployeeContract)
			employeeContractRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), employeeContractHandler.DeleteEmployeeContract)
		}

		// Contract routes
		contractRoutes := protected.Group("/contracts")
		{
			// Admin only routes
			contractRoutes.GET("/:id", authMiddleware.RequireAdmin(), contractHandler.GetByID)
			contractRoutes.GET("", authMiddleware.RequireAdmin(), contractHandler.GetAll)
			contractRoutes.POST("", authMiddleware.RequireAdmin(), contractHandler.Create)
			contractRoutes.PUT("/:id", authMiddleware.RequireAdmin(), contractHandler.Update)
			contractRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), contractHandler.Delete)
		}

		// Dashboard routes
		dashboardRoutes := protected.Group("/dashboard")
		{
			dashboardRoutes.GET("/data", dashboardHandler.GetDashboardData)
			dashboardRoutes.GET("/employees-by-gender", dashboardHandler.GetEmployeesByGender)
			dashboardRoutes.GET("/employees-by-position", dashboardHandler.GetEmployeesByPosition)
			dashboardRoutes.GET("/employees-by-company-department", dashboardHandler.GetEmployeesByCompanyDepartment)
			dashboardRoutes.GET("/employees-by-grade", dashboardHandler.GetEmployeesByGrade)
		}

		// Document Management System (DYS) routes
		documentRoutes := protected.Group("/documents")
		{
			// All authenticated users can upload and manage their own documents
			documentRoutes.POST("/upload", documentHandler.UploadDocument)
			documentRoutes.GET("/me", documentHandler.GetMyDocuments)
			documentRoutes.GET("/user/:id", documentHandler.GetUserDocuments)
			documentRoutes.GET("/:id", documentHandler.GetDocument)
			documentRoutes.GET("/:id/download", documentHandler.DownloadDocument)
			documentRoutes.GET("/:id/url", documentHandler.GetDocumentURL)
			documentRoutes.DELETE("/:id", documentHandler.DeleteDocument)

			// Get documents related to a specific record (Expense, Leave, etc.)
			documentRoutes.GET("/related/:type/:id", documentHandler.GetRelatedDocuments)

			// Link documents to a record (used internally by other services)
			documentRoutes.POST("/link", documentHandler.LinkDocuments)
		}

		// Dynamic Email routes (ADMIN only)
		emailRoutes := protected.Group("/emails")
		{
			emailRoutes.GET("/templates", emailHandler.ListResendTemplates)
			emailRoutes.GET("/templates/:id/variables", emailHandler.GetTemplateVariables)
			emailRoutes.POST("/send-template", emailHandler.SendDynamicTemplateEmail)
		}

		// Mail Configuration routes (ADMIN only)
		mailConfigRoutes := protected.Group("/mail-configs")
		{
			mailConfigRoutes.GET("", mailConfigHandler.GetAll)
			mailConfigRoutes.GET("/:id", mailConfigHandler.GetByID)
			mailConfigRoutes.POST("", mailConfigHandler.Create)
			mailConfigRoutes.PUT("/:id", mailConfigHandler.Update)
			mailConfigRoutes.DELETE("/:id", mailConfigHandler.Delete)
		}

		// Expense Management routes
		expenseRoutes := protected.Group("/expense")
		{
			// Expense Requests
			requestRoutes := expenseRoutes.Group("/requests")
			{
				// Employee routes
				requestRoutes.POST("", expenseHandler.CreateExpenseRequest)
				requestRoutes.GET("/me", expenseHandler.GetMyExpenseRequests)
				requestRoutes.PUT("/:id", expenseHandler.UpdateExpenseRequest)

				// Shared routes
				requestRoutes.GET("/:id", expenseHandler.GetExpenseRequestByID)
				requestRoutes.DELETE("/:id", expenseHandler.DeleteExpenseRequest)

				// Document routes (employee and admin)
				requestRoutes.POST("/:id/documents", expenseHandler.UploadExpenseDocument)
				requestRoutes.GET("/:id/documents", expenseHandler.GetExpenseDocuments)

				// Admin only routes
				requestRoutes.GET("", authMiddleware.RequireAdmin(), expenseHandler.GetAllExpenseRequests)
				requestRoutes.POST("/:id/approve", authMiddleware.RequireAdmin(), expenseHandler.ApproveExpenseRequest)
				requestRoutes.POST("/:id/reject", authMiddleware.RequireAdmin(), expenseHandler.RejectExpenseRequest)
				requestRoutes.POST("/:id/pay", authMiddleware.RequireAdmin(), expenseHandler.MarkExpenseAsPaid)
			}

			// Expense Documents
			documentRoutes := expenseRoutes.Group("/documents")
			{
				documentRoutes.DELETE("/:id", expenseHandler.DeleteExpenseDocument)
				documentRoutes.GET("/:id/download", expenseHandler.DownloadExpenseDocument)
			}

			// Expense Types
			typeRoutes := expenseRoutes.Group("/types")
			{
				typeRoutes.GET("/active", expenseHandler.GetActiveExpenseTypes)

				// Admin only routes
				typeRoutes.GET("", authMiddleware.RequireAdmin(), expenseHandler.GetExpenseTypes)
				typeRoutes.POST("", authMiddleware.RequireAdmin(), expenseHandler.CreateExpenseType)
				typeRoutes.PUT("/:id", authMiddleware.RequireAdmin(), expenseHandler.UpdateExpenseType)
				typeRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), expenseHandler.DeleteExpenseType)
			}
		}

		// Report routes (Admin only)
		reportRoutes := protected.Group("/reports")
		{
			reportRoutes.GET("/work-day", authMiddleware.RequireAdmin(), reportHandler.GetWorkDayReport)
			reportRoutes.POST("/work-day/export", authMiddleware.RequireAdmin(), reportHandler.ExportWorkDayReportExcel)
			reportRoutes.GET("/efor", authMiddleware.RequireAdmin(), reportHandler.GetEforReport)
			reportRoutes.GET("/grade", authMiddleware.RequireAdmin(), reportHandler.GetGradeReport)
			reportRoutes.POST("/grade/export/excel", authMiddleware.RequireAdmin(), reportHandler.ExportGradeReportExcel)
			reportRoutes.GET("/contract", authMiddleware.RequireAdmin(), reportHandler.GetContractReport)
			reportRoutes.POST("/contract/export/excel", authMiddleware.RequireAdmin(), reportHandler.ExportContractReportExcel)
			reportRoutes.POST("/email", authMiddleware.RequireAdmin(), reportHandler.SendReportEmail)
		}

		// Event Management routes
		eventRoutes := protected.Group("/events")
		{
			// Portal routes
			eventRoutes.GET("/dashboard", eventHandler.GetDashboardEvents)
			eventRoutes.POST("/:id/participate", eventHandler.ParticipateInEvent)

			// Admin only routes
			eventRoutes.GET("", authMiddleware.RequireAdmin(), eventHandler.GetEvents)
			eventRoutes.POST("", authMiddleware.RequireAdmin(), eventHandler.CreateEvent)
			eventRoutes.PUT("/:id", authMiddleware.RequireAdmin(), eventHandler.UpdateEvent)
			eventRoutes.DELETE("/:id", authMiddleware.RequireAdmin(), eventHandler.DeleteEvent)
			eventRoutes.POST("/:id/publish", authMiddleware.RequireAdmin(), eventHandler.PublishEvent)
			eventRoutes.GET("/:id/participants/export", authMiddleware.RequireAdmin(), eventHandler.ExportParticipants)
		}

		// Job Management routes (Admin only)
		jobRoutes := protected.Group("/jobs")
		jobRoutes.Use(authMiddleware.RequireAdmin())
		{
			jobRoutes.GET("", jobHandler.GetJobs)
			jobRoutes.GET("/:id", jobHandler.GetJobByID)
			jobRoutes.PUT("/:id", jobHandler.UpdateJob)
			jobRoutes.POST("/:id/run", jobHandler.RunJob)
			jobRoutes.GET("/:id/history", jobHandler.GetJobHistory)
		}
	}

	// Swagger endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start server with custom timeout settings
	log.Printf("Starting %s v%s on port %s", cfg.App.Name, cfg.App.Version, cfg.Server.Port)

	// Create HTTP server with 60 second timeout
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// seedDatabase initializes the database with default data
func seedDatabase(db *database.Database) error {
	// Create default roles
	roles := []domain.Role{
		{Name: domain.RoleAdmin, Description: "Administrator with full system access"},
		{Name: domain.RoleEmployee, Description: "Regular employee with limited access"},
	}

	for _, role := range roles {
		var existingRole domain.Role
		err := db.DB.Where("name = ?", role.Name).First(&existingRole).Error
		if err != nil { // Role doesn't exist, create it
			if err := db.DB.Create(&role).Error; err != nil {
				return err
			}
		}
	}

	// Create default admin user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	adminUser := domain.User{
		Email:    "admin@kartezya.com",
		Password: string(hashedPassword),
	}

	var existingUser domain.User
	err = db.DB.Where("email = ?", adminUser.Email).First(&existingUser).Error
	if err != nil { // User doesn't exist, create it
		if err := db.DB.Create(&adminUser).Error; err != nil {
			return err
		}

		// Assign admin role
		var adminRole domain.Role
		if err := db.DB.Where("name = ?", domain.RoleAdmin).First(&adminRole).Error; err != nil {
			return err
		}

		userRole := domain.UserRole{
			UserID: adminUser.ID,
			RoleID: adminRole.ID,
		}

		if err := db.DB.Create(&userRole).Error; err != nil {
			return err
		}
	}

	// Create default leave types
	leaveTypes := []domain.LeaveType{
		{Name: "Annual Leave", Description: "Annual vacation leave for rest and personal time", IsPaid: true, LimitAmount: func() *int { v := 14; return &v }(), IsAccrual: true, IsRequiredDocument: false},
		{Name: "Sick Leave", Description: "Medical leave for illness and health-related issues", IsPaid: true, LimitAmount: nil, IsAccrual: false, IsRequiredDocument: true},
		{Name: "Personal Leave", Description: "Personal time off for individual matters", IsPaid: false, LimitAmount: func() *int { v := 5; return &v }(), IsAccrual: false, IsRequiredDocument: false},
		{Name: "Maternity Leave", Description: "Maternity leave for new mothers", IsPaid: true, LimitAmount: nil, IsAccrual: false, IsRequiredDocument: true},
		{Name: "Paternity Leave", Description: "Paternity leave for new fathers", IsPaid: true, LimitAmount: nil, IsAccrual: false, IsRequiredDocument: false},
	}

	for _, leaveType := range leaveTypes {
		var existing domain.LeaveType
		err := db.DB.Where("name = ?", leaveType.Name).First(&existing).Error
		if err != nil { // Leave type doesn't exist, create it
			if err := db.DB.Create(&leaveType).Error; err != nil {
				return err
			}
		}
	}

	// Create sample company
	var company domain.Company
	err = db.DB.Where("name = ?", "Kartezya Technologies").First(&company).Error
	if err != nil { // Company doesn't exist, create it
		company = domain.Company{
			Name:    "Kartezya Technologies",
			Address: "Istanbul, Turkey",
			Phone:   "+90 212 123 4567",
			Email:   "info@kartezya.com",
			Website: "https://kartezya.com",
		}
		if err := db.DB.Create(&company).Error; err != nil {
			return err
		}

		// Create sample department
		department := domain.Department{
			CompanyID: company.ID,
			Name:      "Engineering",
		}
		if err := db.DB.Create(&department).Error; err != nil {
			return err
		}

		// Create sample job position
		jobPosition := domain.JobPosition{
			Title: "Software Engineer",
		}
		if err := db.DB.Create(&jobPosition).Error; err != nil {
			return err
		}
	}

	// Create default Other Request Types
	requestTypes := []domain.RequestType{
		{Name: "Bordro Talebi", Description: "Aylık veya geçmiş dönem maaş bordrosu kopyası talebi", Active: true},
		{Name: "Çalışma Belgesi", Description: "Vize, banka veya resmi kurum başvurularında kullanılacak çalışma belgesi", Active: true},
		{Name: "Vize Evrakları", Description: "Konsolosluk başvuruları için gerekli olan şirket imza sirküleri, faaliyet belgesi vb. evraklar", Active: true},
		{Name: "Avans Talebi", Description: "Maaş dönemi öncesi acil durum avans talebi", Active: true},
	}

	for _, reqType := range requestTypes {
		var existing domain.RequestType
		err := db.DB.Where("name = ?", reqType.Name).First(&existing).Error
		if err != nil {
			if err := db.DB.Create(&reqType).Error; err != nil {
				return err
			}
		}
	}

	log.Println("Database seeded successfully")
	return nil
}

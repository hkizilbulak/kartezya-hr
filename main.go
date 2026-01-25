package main

import (
	"log"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/database"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/handler"
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

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db := database.NewDatabase(cfg)
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed database with default data
	/*if err := seedDatabase(db); err != nil {
		log.Printf("Warning: Failed to seed database: %v", err)
	}*/

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

	// Initialize services
	auditService := service.NewAuditService(auditRepo)
	authService := service.NewAuthService(userRepo, userRoleRepo, roleRepo, auditService, cfg)
	emailService := service.NewEmailService(cfg, userRepo)
	employeeService := service.NewEmployeeService(employeeRepo, userRepo, userRoleRepo, roleRepo, authService, auditService, workInfoRepo, emailService)
	leaveService := service.NewLeaveService(leaveRepo, leaveTypeRepo, leaveBalanceRepo, employeeRepo, holidayRepo, auditService)
	departmentService := service.NewDepartmentService(departmentRepo, companyRepo, auditService)
	companyService := service.NewCompanyService(companyRepo, departmentRepo, departmentService, auditService)
	jobPositionService := service.NewJobPositionService(jobPositionRepo, auditService)
	workInfoService := service.NewWorkInformationService(workInfoRepo, employeeRepo, companyRepo, departmentRepo, jobPositionRepo, auditService)
	lookupService := service.NewLookupService(companyRepo, departmentRepo, jobPositionRepo, leaveTypeRepo, gradeRepo)
	gradeService := service.NewGradeService(gradeRepo, auditService)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, emailService, userRepo)
	employeeHandler := handler.NewEmployeeHandler(employeeService)
	leaveHandler := handler.NewLeaveHandler(leaveService, employeeService)
	companyHandler := handler.NewCompanyHandler(companyService)
	departmentHandler := handler.NewDepartmentHandler(departmentService)
	jobPositionHandler := handler.NewJobPositionHandler(jobPositionService)
	workInfoHandler := handler.NewWorkInformationHandler(workInfoService, employeeService)
	lookupHandler := handler.NewLookupHandler(lookupService)
	dashboardHandler := handler.NewDashboardHandler(employeeService, departmentService, companyService, leaveService)
	gradeHandler := handler.NewGradeHandler(gradeService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Initialize router
	router := gin.Default()

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

	// API v1 routes
	v1 := router.Group("/api/v1")

	// Public routes (no authentication required)
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/validate-reset-token", authHandler.ValidateResetToken)
		auth.POST("/reset-password", authHandler.ResetPassword)
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

	// Protected routes (authentication required)
	protected := v1.Group("")
	protected.Use(authMiddleware.JWTAuth())
	{
		// Auth routes
		authRoutes := protected.Group("/auth")
		{
			authRoutes.POST("/logout", authHandler.Logout)
			authRoutes.POST("/change-password", authHandler.ChangePassword)
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

				// Admin only routes
				requests.GET("/:id", authMiddleware.RequireAdmin(), leaveHandler.GetLeaveRequestByID)
				requests.GET("", authMiddleware.RequireAdmin(), leaveHandler.GetAllLeaveRequests)
				requests.POST("/:id/approve", authMiddleware.RequireAdmin(), leaveHandler.ApproveLeaveRequest)
				requests.POST("/:id/reject", authMiddleware.RequireAdmin(), leaveHandler.RejectLeaveRequest)
			}

			// Leave balances
			balances := leaveRoutes.Group("/balances")
			{
				balances.GET("/me", leaveHandler.GetMyLeaveBalances)
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

		// Dashboard routes
		dashboardRoutes := protected.Group("/dashboard")
		{
			dashboardRoutes.GET("/data", dashboardHandler.GetDashboardData)
			dashboardRoutes.GET("/employees-by-gender", dashboardHandler.GetEmployeesByGender)
			dashboardRoutes.GET("/employees-by-position", dashboardHandler.GetEmployeesByPosition)
			dashboardRoutes.GET("/employees-by-company-department", dashboardHandler.GetEmployeesByCompanyDepartment)
		}
	}

	// Swagger endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start server
	log.Printf("Starting %s v%s on port %s", cfg.App.Name, cfg.App.Version, cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
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
		{Name: "Annual Leave", Description: "Annual vacation leave for rest and personal time", IsPaid: true, IsLimited: true, IsAccrual: true, IsRequiredDocument: false},
		{Name: "Sick Leave", Description: "Medical leave for illness and health-related issues", IsPaid: true, IsLimited: false, IsAccrual: false, IsRequiredDocument: true},
		{Name: "Personal Leave", Description: "Personal time off for individual matters", IsPaid: false, IsLimited: true, IsAccrual: false, IsRequiredDocument: false},
		{Name: "Maternity Leave", Description: "Maternity leave for new mothers", IsPaid: true, IsLimited: false, IsAccrual: false, IsRequiredDocument: true},
		{Name: "Paternity Leave", Description: "Paternity leave for new fathers", IsPaid: true, IsLimited: false, IsAccrual: false, IsRequiredDocument: false},
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

	log.Println("Database seeded successfully")
	return nil
}

package main

import (
	"fmt"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/database"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/service"
)

func main() {
	fmt.Println("=== Yarım Gün İzin Hesaplama Testi ===\n")

	// Load configuration
	cfg := config.Load()
	domain.SetConfig(cfg)

	// Initialize database
	dbInstance := database.NewDatabase(cfg)
	db := dbInstance.GetDB()

	// Initialize repositories and services
	leaveRepo := repository.NewLeaveRepository(db)
	leaveTypeRepo := repository.NewLeaveTypeRepository(db)
	leaveBalanceRepo := repository.NewLeaveBalanceRepository(db)
	employeeRepo := repository.NewEmployeeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	auditService := service.NewAuditService(repository.NewAuditRepository(db))

	leaveService := service.NewLeaveService(
		leaveRepo,
		leaveTypeRepo,
		leaveBalanceRepo,
		employeeRepo,
		holidayRepo,
		auditService,
	)

	// Test scenarios
	tests := []struct {
		name                string
		startDate           string
		endDate             string
		isStartDateFullDay  bool
		isFinishDateFullDay bool
		expected            float64
	}{
		{
			name:                "Test 1: Tek gün - Tam gün",
			startDate:           "2026-02-03",
			endDate:             "2026-02-03",
			isStartDateFullDay:  true,
			isFinishDateFullDay: true,
			expected:            1.0,
		},
		{
			name:                "Test 2: Tek gün - Başlangıç yarım",
			startDate:           "2026-02-03",
			endDate:             "2026-02-03",
			isStartDateFullDay:  false,
			isFinishDateFullDay: true,
			expected:            0.5,
		},
		{
			name:                "Test 3: Tek gün - Her iki taraf yarım (ÖNEMLİ!)",
			startDate:           "2026-02-03",
			endDate:             "2026-02-03",
			isStartDateFullDay:  false,
			isFinishDateFullDay: false,
			expected:            0.5,
		},
		{
			name:                "Test 4: 3 gün - Başlangıç yarım, bitiş tam",
			startDate:           "2026-02-03",
			endDate:             "2026-02-05",
			isStartDateFullDay:  false,
			isFinishDateFullDay: true,
			expected:            2.5, // 0.5 + 1.0 + 1.0
		},
		{
			name:                "Test 5: 3 gün - Başlangıç tam, bitiş yarım",
			startDate:           "2026-02-03",
			endDate:             "2026-02-05",
			isStartDateFullDay:  true,
			isFinishDateFullDay: false,
			expected:            2.5, // 1.0 + 1.0 + 0.5
		},
		{
			name:                "Test 6: 3 gün - Her iki taraf yarım",
			startDate:           "2026-02-03",
			endDate:             "2026-02-05",
			isStartDateFullDay:  false,
			isFinishDateFullDay: false,
			expected:            2.0, // 0.5 + 1.0 + 0.5
		},
		{
			name:                "Test 7: Kullanıcı senaryosu - 2.5 gün",
			startDate:           "2026-02-06", // Cuma
			endDate:             "2026-02-10", // Salı
			isStartDateFullDay:  false,
			isFinishDateFullDay: true,
			expected:            2.5, // 0.5 (Cuma) + 1.0 (Pazartesi) + 1.0 (Salı)
		},
	}

	passedTests := 0
	failedTests := 0

	for _, test := range tests {
		startDate, _ := time.Parse("2006-01-02", test.startDate)
		endDate, _ := time.Parse("2006-01-02", test.endDate)

		result, err := leaveService.CalculateWorkingDays(
			startDate,
			endDate,
			test.isStartDateFullDay,
			test.isFinishDateFullDay,
		)

		status := "✅ GEÇTI"
		if err != nil || result != test.expected {
			status = "❌ BAŞARISIZ"
			failedTests++
		} else {
			passedTests++
		}

		fmt.Printf("%s\n", test.name)
		fmt.Printf("  Tarih: %s - %s\n", test.startDate, test.endDate)
		fmt.Printf("  Başlangıç: %s, Bitiş: %s\n",
			boolToStr(test.isStartDateFullDay),
			boolToStr(test.isFinishDateFullDay))
		fmt.Printf("  Beklenen: %.1f gün\n", test.expected)
		fmt.Printf("  Sonuç: %.1f gün\n", result)
		fmt.Printf("  Durum: %s\n\n", status)

		if err != nil {
			fmt.Printf("  Hata: %v\n\n", err)
		}
	}

	fmt.Println("=== TEST SONUÇLARI ===")
	fmt.Printf("Toplam: %d\n", len(tests))
	fmt.Printf("Geçti: %d\n", passedTests)
	fmt.Printf("Başarısız: %d\n", failedTests)

	if failedTests == 0 {
		fmt.Println("\n🎉 TÜM TESTLER BAŞARILI!")
	} else {
		fmt.Println("\n⚠️  BAZI TESTLER BAŞARISIZ!")
	}
}

func boolToStr(b bool) string {
	if b {
		return "Tam gün"
	}
	return "Yarım gün"
}

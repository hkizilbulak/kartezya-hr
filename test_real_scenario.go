package main

import (
	"fmt"
	"time"
)

// Test the REAL user scenario
func main() {
	fmt.Println("=== GERÇEK KULLANICI SENARYOSU ===")
	fmt.Println()
	
	// Scenario: User created a leave request for 2.5 days
	// Start: Feb 23 (Monday) - HALF DAY
	// End: Feb 25 (Wednesday) - FULL DAY
	// Expected: 2.5 days (0.5 + 1.0 + 1.0)
	
	leaveStart, _ := time.Parse("2006-01-02", "2026-02-23")
	leaveEnd, _ := time.Parse("2006-01-02", "2026-02-25")
	isStartFullDay := false  // Half-day
	isEndFullDay := true     // Full-day
	
	fmt.Printf("İzin Talebi:\n")
	fmt.Printf("  Başlangıç: %s (%s) - %s\n", leaveStart.Format("2006-01-02"), leaveStart.Weekday(), boolToDay(isStartFullDay))
	fmt.Printf("  Bitiş: %s (%s) - %s\n", leaveEnd.Format("2006-01-02"), leaveEnd.Weekday(), boolToDay(isEndFullDay))
	fmt.Printf("  Beklenen: 2.5 gün\n\n")
	
	// Test 1: Report filter exactly matches leave
	fmt.Println("Test 1: Rapor filtresi izinle tam eşleşiyor")
	filterStart1 := leaveStart
	filterEnd1 := leaveEnd
	result1 := calculateWithCurrentLogic(leaveStart, leaveEnd, filterStart1, filterEnd1, isStartFullDay, isEndFullDay)
	fmt.Printf("  Filter: %s - %s\n", filterStart1.Format("2006-01-02"), filterEnd1.Format("2006-01-02"))
	fmt.Printf("  Sonuç: %.1f gün\n", result1)
	if result1 == 2.5 {
		fmt.Println("  ✅ BAŞARILI!")
	} else {
		fmt.Println("  ❌ BAŞARISIZ! (3.0 gün gösteriyorsa bug var)")
	}
	fmt.Println()
	
	// Test 2: Report filter is wider than leave
	fmt.Println("Test 2: Rapor filtresi izinden geniş")
	filterStart2, _ := time.Parse("2006-01-02", "2026-02-01")
	filterEnd2, _ := time.Parse("2006-01-02", "2026-02-28")
	result2 := calculateWithCurrentLogic(leaveStart, leaveEnd, filterStart2, filterEnd2, isStartFullDay, isEndFullDay)
	fmt.Printf("  Filter: %s - %s\n", filterStart2.Format("2006-01-02"), filterEnd2.Format("2006-01-02"))
	fmt.Printf("  Sonuç: %.1f gün\n", result2)
	if result2 == 2.5 {
		fmt.Println("  ✅ BAŞARILI!")
	} else {
		fmt.Println("  ❌ BAŞARISIZ!")
	}
	fmt.Println()
	
	// Test 3: Different leave scenario - all full days
	fmt.Println("Test 3: Tüm günler tam gün olan izin")
	leaveStart3, _ := time.Parse("2006-01-02", "2026-02-23")
	leaveEnd3, _ := time.Parse("2006-01-02", "2026-02-25")
	result3 := calculateWithCurrentLogic(leaveStart3, leaveEnd3, filterStart2, filterEnd2, true, true)
	fmt.Printf("  İzin: %s (tam) - %s (tam)\n", leaveStart3.Format("2006-01-02"), leaveEnd3.Format("2006-01-02"))
	fmt.Printf("  Beklenen: 3.0 gün\n")
	fmt.Printf("  Sonuç: %.1f gün\n", result3)
	if result3 == 3.0 {
		fmt.Println("  ✅ BAŞARILI!")
	} else {
		fmt.Println("  ❌ BAŞARISIZ!")
	}
}

func calculateWithCurrentLogic(leaveStart, leaveEnd, filterStart, filterEnd time.Time, isStartFullDay, isEndFullDay bool) float64 {
	// Overlap calculation (same as report_service.go)
	overlapStart := leaveStart
	if overlapStart.Before(filterStart) {
		overlapStart = filterStart
	}
	
	overlapEnd := leaveEnd
	if overlapEnd.After(filterEnd) {
		overlapEnd = filterEnd
	}
	
	if overlapStart.After(overlapEnd) {
		return 0
	}
	
	daysInRange := 0.0
	currentDate := overlapStart
	
	for {
		dayOfWeek := currentDate.Weekday()
		isWeekend := dayOfWeek == time.Saturday || dayOfWeek == time.Sunday
		
		if !isWeekend {
			dayValue := 1.0
			
			// Current logic from report_service.go
			if currentDate.Equal(leaveStart) && !isStartFullDay {
				dayValue = 0.5
			}
			
			if currentDate.Equal(leaveEnd) && !isEndFullDay {
				if currentDate.Equal(leaveStart) && !isStartFullDay {
					dayValue = 0.5
				} else {
					dayValue = 0.5
				}
			}
			
			daysInRange += dayValue
		}
		
		if currentDate.Equal(overlapEnd) {
			break
		}
		
		currentDate = currentDate.AddDate(0, 0, 1)
	}
	
	return daysInRange
}

func boolToDay(isFullDay bool) string {
	if isFullDay {
		return "Tam gün"
	}
	return "Yarım gün"
}

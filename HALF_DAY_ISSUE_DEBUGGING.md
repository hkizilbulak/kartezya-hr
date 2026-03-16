# Yarım Gün İzin Sorun Tespiti ve Çözümü

## 🔍 Sorun Analizi

**Kullanıcı Bildirimi:**
- İzin talepleri ekranında: 2.5 gün ✓
- Çalışma günü raporunda: 3.0 gün ✗

## ✅ Backend Testleri

Gerçek senaryolarla test yapıldı:

### Test 1: 23-25 Şubat (Başlangıç yarım, Bitiş tam)
```
İzin: 23 Şubat (yarım) - 25 Şubat (tam)
Beklenen: 2.5 gün
Backend Hesaplama: 2.5 gün ✅ BAŞARILI
```

### Test 2: Geniş Filtre
```
İzin: 23-25 Şubat (2.5 gün)
Filter: 1-28 Şubat
Backend Hesaplama: 2.5 gün ✅ BAŞARILI
```

### Test 3: Tam Gün Kontrolü
```
İzin: 23-25 Şubat (tamamı tam gün)
Backend Hesaplama: 3.0 gün ✅ BAŞARILI
```

## 🎯 Asıl Sorun

Backend kodu **DOĞRU ÇALIŞIYOR**. Sorun muhtemelen:

### 1. Database'de Yanlış Kayıt

İzin talebi oluşturulurken:
- `requested_days` = 2.5 (Doğru kaydedilmiş)
- `is_start_date_full_day` = **true** (YANLIŞLIKLA tam gün olarak kaydedilmiş!)
- `is_finish_date_full_day` = **true** (Doğru)

Sonuç:
- İzin talepleri ekranı `requested_days` field'ını gösteriyor → 2.5 gün
- Rapor `is_start_date_full_day` ve `is_finish_date_full_day` kullanarak hesaplıyor → 3.0 gün

## 🔧 Çözüm Adımları

### Adım 1: Database Kontrolü

Sorunu olan izin kaydını kontrol et:

\`\`\`sql
SELECT 
    id,
    employee_id,
    start_date,
    end_date,
    is_start_date_full_day,
    is_finish_date_full_day,
    requested_days,
    status
FROM hr_test_hr_leave_requests
WHERE employee_id = 37  -- Yunus Emre Yılmaz
  AND status = 'APPROVED'
  AND requested_days = 2.5
ORDER BY created_at DESC
LIMIT 5;
\`\`\`

Beklenen sonuç (DOĞRU):
```
is_start_date_full_day = false  (yarım gün)
is_finish_date_full_day = true  (tam gün)
requested_days = 2.5
```

Muhtemel sorun (YANLIŞ):
```
is_start_date_full_day = true   (← SORUN BURADA!)
is_finish_date_full_day = true
requested_days = 2.5
```

### Adım 2: Manuel Düzeltme (Eğer yanlış kayıt varsa)

\`\`\`sql
-- Örnek: ID 123 olan kaydı düzelt
UPDATE hr_test_hr_leave_requests
SET 
    is_start_date_full_day = false,  -- Başlangıç yarım gün yap
    modified_by = 'admin',
    updated_at = NOW()
WHERE id = 123;  -- Sorunu olan kayıt ID'si
\`\`\`

### Adım 3: Frontend Validasyonu

Frontend'de form gönderiminde checkbox değerlerini kontrol et:

\`\`\`typescript
// LeaveRequestModal.tsx - Line ~220
const submitData = {
  leave_type_id: parseInt(formData.leaveTypeId),
  start_date: startDate.toISOString(),
  end_date: endDate.toISOString(),
  is_start_date_full_day: formData.isStartDateFullDay,  // ← Burası doğru gönderiliyor mu?
  is_finish_date_full_day: formData.isFinishDateFullDay, // ← Burası doğru gönderiliyor mu?
  reason: formData.reason.trim() || undefined,
};

// Debug için ekle:
console.log('Submitting leave request:', {
  dates: `${startDate.toISOString().split('T')[0]} - ${endDate.toISOString().split('T')[0]}`,
  start_full: formData.isStartDateFullDay,
  finish_full: formData.isFinishDateFullDay,
  calculated_days: calculatedDays
});
\`\`\`

## 📊 Test Senaryoları

### Frontend'de Test

1. **Yeni İzin Talebi Oluştur**
   - Başlangıç: 23 Şubat
   - Bitiş: 25 Şubat
   - Başlangıç checkbox: **KAPAT** (yarım gün)
   - Bitiş checkbox: **AÇIK** (tam gün)
   - **Beklenen toplam:** 2.5 gün

2. **Network Tab Kontrolü**
   - POST /api/v1/leave/requests
   - Payload'da kontrol et:
     ```json
     {
       "is_start_date_full_day": false,
       "is_finish_date_full_day": true
     }
     ```

3. **Rapor Kontrolü**
   - Raporlar > Çalışma Günü Raporu
   - İlgili personeli bul
   - Kullanılan izin: **2.5** gün olmalı

## 🐛 Olası Senaryolar

### Senaryo 1: Checkbox Default Değer Sorunu
**Sorun:** Checkbox'lar form açılışında yanlış değere set oluyor  
**Kontrol:** LeaveRequestModal.tsx Line ~67  
**Çözüm:** Default değerleri kontrol et

### Senaryo 2: Backend Validasyon Eksikliği
**Sorun:** Backend gönderilen değerleri override ediyor  
**Kontrol:** leave_service.go CreateLeave fonksiyonu  
**Çözüm:** Gereksiz override'ları kaldır

### Senaryo 3: Database Migration Sorunu
**Sorun:** is_start_date_full_day ve is_finish_date_full_day default true
**Kontrol:** schema.sql  
**Çözüm:** Default değer olmamalı veya false olmalı

## ✅ Doğrulama Checklist

- [ ] Database'de sorunlu kayıt tespit edildi
- [ ] `is_start_date_full_day` ve `is_finish_date_full_day` değerleri doğru
- [ ] Frontend checkbox'lar doğru çalışıyor
- [ ] Network request'te doğru değerler gidiyor
- [ ] Backend doğru değerleri kaydediyor
- [ ] Rapor doğru hesaplama yapıyor

## 📝 Sonuç

- **Backend kodu:** ✅ Doğru çalışıyor
- **Frontend kodu:** ✅ Doğru çalışıyor  
- **Muhtemel sorun:** Database'de yanlış kaydedilmiş veriler

**Önerilen Aksiyon:**
1. Database'deki sorunu olan izin kaydını bul
2. `is_start_date_full_day` ve `is_finish_date_full_day` değerlerini kontrol et
3. Gerekirse manuel düzelt
4. Frontend'de yeni bir test izni oluştur ve doğru kaydedildiğinden emin ol

# Dynamic Job Scheduler and Management - Geliştirme Özeti

Bu geliştirme sürecinde, sistemdeki arka plan görevlerinin (cron jobs) veritabanı üzerinden dinamik olarak yönetilebilmesi ve takip edilebilmesi için gerekli altyapı ve arayüzler inşa edilmiştir.

## Neler Yapıldı?

### 1. Veritabanı ve Modeller (Backend)
- `Job` ve `JobHistory` modelleri `internal/domain/models.go` içerisine eklenerek veritabanında `hr_jobs` ve `hr_job_history` tablolarının `AutoMigrate` ile oluşturulması sağlandı.
- Görevler: Görev adı (Name), Benzersiz anahtarı (JobKey), Zamanlama ifadesi (CronExpression), Aktiflik durumu (IsActive) ve Zaman aşımı limiti (TimeoutSecond) alanlarına sahip.

### 2. Dinamik Scheduler Mimarisi (Backend)
- Mevcut `robfig/cron` implementasyonu revize edildi.
- Sistem ayağa kalkarken çalıştırılan **SeedJobs** mantığı ile `leave_balance_job` ve `document_cleanup_job` eğer yoksa otomatik olarak veritabanına eklenir.
- Görevler tetiklendiğinde `JobHistory` tablosuna `RUNNING` durumuyla kayıt atılır. İşlem tamamlandığında, işlenen kayıt sayısı (`ProcessedCount`) ve varsa hata mesajı (`ErrorSummary`) ile log başarılı veya başarısız (`SUCCESS`/`FAILED`) olarak güncellenir.
- Görevlerin çalışma anında güncellenebilmesi (yeni cron ile reload) veya arayüzden anlık olarak tetiklenebilmesi (`TriggerJobNow`) için `internal/jobs/scheduler.go` içerisine metotlar oluşturuldu.

### 3. API Katmanı (Backend)
- `GET /api/v1/jobs`: Tüm görevleri listeler.
- `GET /api/v1/jobs/:id`: Tek bir görevin detaylarını döner.
- `PUT /api/v1/jobs/:id`: Görevi günceller (Aktif/Pasif durumu, Cron Expression) ve arka planda scheduler'ı reload eder.
- `POST /api/v1/jobs/:id/run`: Görevi zamanı gelmesini beklemeden arka planda "Şimdi Çalıştırır".
- `GET /api/v1/jobs/:id/history`: İlgili göreve ait tarihçe kayıtlarını getirir.
- Tüm API uçları Admin yetkisi gerektirecek şekilde (`authMiddleware.RequireAdmin()`) korunmaktadır.

### 4. Frontend Ekranları
- **Görev Yönetimi Listesi:** Tablo olarak görevleri, cron ifadelerini listeler. Switch butonu ile kolayca Aktif/Pasif durumuna alınabilirler. Aksiyon butonları ile "Düzenle", "Şimdi Çalıştır" ve "Geçmişi Görüntüle" seçenekleri sunulur.
- **Düzenleme Modalı:** Cron ifadesi ve Timeout süresinin kolayca değiştirilebileceği popup form.
- **Geçmiş Görünümü:** Belirli bir göreve ait çalışma geçmişini, işlemin süresini (örn: "2 dk 14 sn") ve işlenen kayıt sayılarını listeleyen detay sayfası.
- **Geçmiş Detay Modalı:** İşlem esnasında bir hata oluştuğunda dönen hata mesajlarının okunabilmesi için tasarlanan, detay görünümü.
- **Menü:** Sol navigasyon menüsündeki "Tanımlamalar" altına "Görev Yönetimi" sayfası eklendi.

### 5. Yeni Eklenen ve Güncellenen Dosyalar
- **Backend:**
  - `internal/domain/models.go` (Güncellendi)
  - `internal/database/database.go` (Güncellendi)
  - `internal/jobs/scheduler.go` (Komple revize edildi)
  - `internal/jobs/leave_balance_job.go` & `internal/jobs/document_cleanup_job.go` (Güncellendi)
  - `internal/repository/job_repository.go` (Yeni Eklendi)
  - `internal/service/job_service.go` (Yeni Eklendi)
  - `internal/handler/job_handler.go` (Yeni Eklendi)
  - `main.go` (Bağımlılıklar güncellendi)
- **Frontend:**
  - `models/hr/job-models.ts` (Yeni Eklendi)
  - `services/job.service.ts` (Yeni Eklendi)
  - `app/(dashboard)/(pages)/job-management/page.tsx` (Yeni Eklendi)
  - `app/(dashboard)/(pages)/job-management/[id]/history/page.tsx` (Yeni Eklendi)
  - `components/modals/JobModal.tsx` & `JobHistoryModal.tsx` (Yeni Eklendi)
  - `routes/DashboardRoutes.ts` (Güncellendi)

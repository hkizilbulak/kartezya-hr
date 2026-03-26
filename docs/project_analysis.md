# Kartezya HR — Proje Analizi

> Son güncelleme: 2026-03-25

---

## 1. Genel Bakış

**Kartezya HR**, Go ile yazılmış bir İnsan Kaynakları Yönetim Sistemi REST API'sidir.
Şirket, çalışan, izin, grade, sözleşme ve departman yönetimi gibi temel İK süreçlerini kapsar.

| Özellik | Değer |
|---|---|
| Dil | Go 1.25 |
| Framework | Gin |
| ORM | GORM |
| Veritabanı | PostgreSQL 15 |
| Auth | JWT + Yandex OAuth2 |
| API Dokümantasyonu | Swagger (swaggo) |
| Zamanlayıcı | robfig/cron |
| E-posta | SMTP (configurable) |

---

## 2. Proje Yapısı

```
kartezya-hr/
├── main.go                    # Entry point, router setup, DI wiring
├── go.mod / go.sum
├── docker-compose.yml         # PostgreSQL 15 konteyneri
├── .env.example
├── schema/
│   ├── schema.sql             # Tam DB şeması
│   ├── seed_data.sql          # Örnek veriler
│   └── migrate_leave_types_limit.sql
├── docs/                      # Swagger otomatik üretilmiş dosyalar
├── postman/                   # Postman koleksiyonu
└── internal/
    ├── config/                # Ortam değişkeni yapılandırması
    ├── database/              # DB bağlantısı + GORM migrate
    ├── domain/                # Tüm domain modelleri (tek dosya: models.go)
    ├── handler/               # HTTP handler'ları (Gin)
    ├── service/               # İş mantığı
    ├── repository/            # Veritabanı erişim katmanı
    ├── middleware/            # JWT auth middleware
    ├── jobs/                  # Cron job'ları
    ├── types/                 # Ortak tipler / DTO'lar
    └── email_templates/       # E-posta şablonları
```

---

## 3. Katmanlı Mimari

```
HTTP Request
     ↓
[Handler]         → Request parse, validation, response
     ↓
[Service]         → Business logic, orchestration
     ↓
[Repository]      → SQL / GORM sorgular
     ↓
[PostgreSQL]
```

**Dependency Injection:** Manuel (constructor injection). `main.go` içinde tüm repo → service → handler zincirleri kurulur. IoC container kullanılmamaktadır.

---

## 4. Domain Modelleri

Tüm modeller `internal/domain/models.go` içinde tanımlıdır.

| Model | Tablo | Açıklama |
|---|---|---|
| `User` | `hr_users` | Kimlik doğrulama varlığı |
| `Role` | `hr_roles` | ADMIN / EMPLOYEE |
| `UserRole` | `hr_user_roles` | N:N junction |
| `Employee` | `hr_employees` | Kişisel bilgiler |
| `Company` | `hr_companies` | Şirket |
| `Department` | `hr_departments` | Departman (şirkete bağlı) |
| `JobPosition` | `hr_job_positions` | Pozisyon unvanı |
| `EmployeeWorkInformation` | `hr_employee_work_information` | Çalışma geçmişi |
| `LeaveType` | `hr_leave_types` | İzin türü (yıllık, hastalık…) |
| `LeaveBalance` | `hr_leave_balances` | Kalan izin günleri |
| `LeaveRequest` | `hr_leave_requests` | İzin talebi |
| `LeaveDocument` | `hr_leave_documents` | Talebe eklenen belgeler |
| `Holiday` | `hr_holidays` | Resmi tatiller |
| `Grade` | `hr_grades` | Seviye/derece tanımı |
| `EmployeeGrade` | `hr_employee_grades` | Çalışan grade geçmişi |
| `EmployeeContract` | `hr_employee_contracts` | Sözleşmeler |
| `AuditLog` | `hr_audit_logs` | Sistem audit logu (soft-delete yok) |

### Ortak Temel Model
Her model (AuditLog hariç) `AuditableModel`'den türer:
```go
type AuditableModel struct {
    ID         uint
    CreatedAt  time.Time
    UpdatedAt  time.Time
    Deleted    bool       // soft delete
    CreatedBy  string
    ModifiedBy string
}
```

### Tablo Prefix
Tüm tablolar `hr_` prefixi ile üretilir. `DB_TABLE_PREFIX` env değişkeni ile özelleştirilebilir.

---

## 5. API Endpoint'leri

**Base URL:** `GET /api/v1`

### Public (Auth Gerektirmeyen)
| Method | Path | Açıklama |
|---|---|---|
| POST | `/auth/login` | JWT login |
| POST | `/auth/validate-reset-token` | Şifre sıfırlama token doğrulama |
| POST | `/auth/reset-password` | Şifre sıfırlama |
| GET | `/auth/yandex/login` | Yandex OAuth başlatma |
| GET | `/auth/yandex/callback` | Yandex OAuth callback |
| GET | `/lookup/*` | Companies, departments, job-positions, leave-types, grades |

### Protected — Çalışan Erişimi
| Method | Path | Açıklama |
|---|---|---|
| GET/PUT | `/employees/me` | Kendi profili |
| GET | `/leave/requests/me` | Kendi izin talepleri |
| POST | `/leave/requests` | İzin talebi oluştur |
| PUT/POST | `/leave/requests/:id` | İzin güncelle / iptal |
| GET | `/leave/balances/me` | Kendi izin bakiyesi |
| GET | `/work-information/me` | Kendi çalışma bilgisi |
| GET | `/employee-grades/me` | Kendi grade geçmişi |
| GET | `/employee-contracts/me` | Kendi sözleşmeleri |

### Protected — Sadece Admin
Employees, Companies, Departments, Job Positions, Grades, Work Information, Employee Grades, Employee Contracts, Leave Types, Leave Balances (tüm), Dashboard, Reports için tam CRUD.

---

## 6. Kimlik Doğrulama & Yetkilendirme

- **JWT:** `Authorization: Bearer <token>` header ile taşınır. Expiry süresi `JWT_EXPIRY_HOURS` ile ayarlanır.
- **Roller:** `ADMIN` ve `EMPLOYEE`. `RequireAdmin()` middleware'i ile admin rotaları korunur.
- **Yandex OAuth2:** Callback akışı ile JWT üretilir; mevcut kullanıcıyla e-posta eşleşmesi yapılır.
- **CORS:** Geliştirme modunda wildcard (`*`) açık. Prodüksiyonda kısıtlanmalı.

---

## 7. Servisler — Sorumluluk Özeti

| Servis | Temel Sorumluluk |
|---|---|
| `AuthService` | Login, JWT üretimi, şifre sıfırlama, Yandex OAuth |
| `EmployeeService` | Çalışan CRUD, profil yönetimi |
| `LeaveService` | İzin talebi, onay/red, bakiye hesaplama, çalışma günü hesabı |
| `CompanyService` | Şirket + departman yönetimi |
| `DepartmentService` | Departman CRUD |
| `JobPositionService` | Pozisyon CRUD |
| `WorkInformationService` | İş geçmişi |
| `GradeService` | Grade CRUD |
| `EmployeeGradeService` | Grade geçmiş takibi |
| `EmployeeContractService` | Sözleşme yönetimi |
| `LookupService` | Frontend dropdown verileri |
| `AuditService` | Audit log yazma |
| `EmailService` | SMTP e-posta gönderimi |
| `ReportService` | Çalışma günü ve grade raporları |

---

## 8. Scheduled Jobs

`robfig/cron` ile çalışan arka plan görevleri:

| Job | Dosya | Açıklama |
|---|---|---|
| Leave Balance Job | `jobs/leave_balance_job.go` | Yıllık izin bakiyelerini periyodik günceller |
| Scheduler | `jobs/scheduler.go` | Cron runner başlatma/durdurma |

---

## 9. Konfigürasyon (Env Değişkenleri)

| Değişken | Default | Açıklama |
|---|---|---|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | DB kullanıcı |
| `DB_PASSWORD` | `postgres` | DB şifre |
| `DB_NAME` | `kartezya_hr` | DB adı |
| `DB_TABLE_PREFIX` | `hr` | Tablo prefix |
| `DB_DEBUG` | `false` | GORM debug log |
| `JWT_SECRET` | `default-secret-change-me` | **Prodüksiyonda değiştirilmeli!** |
| `JWT_EXPIRY_HOURS` | `24` | Token süresi |
| `SERVER_PORT` | `8080` | Sunucu portu |
| `GIN_MODE` | `debug` | `debug` / `release` |
| `SMTP_HOST/PORT/USER/PASSWORD` | — | E-posta ayarları |
| `FROM_EMAIL` | `info@kartezya.com` | Gönderen e-posta |
| `FRONTEND_URL` | `http://localhost:3000` | Şifre sıfırlama linki |
| `YANDEX_CLIENT_ID` | — | Yandex OAuth |
| `YANDEX_CLIENT_SECRET` | — | Yandex OAuth |
| `YANDEX_REDIRECT_URL` | `.../yandex/callback` | Yandex OAuth |

---

## 10. Çalıştırma

```bash
# 1. PostgreSQL başlat
docker-compose up -d

# 2. .env oluştur
cp .env.example .env

# 3. Uygulamayı başlat
go run main.go

# 4. Swagger UI
open http://localhost:8080/swagger/index.html

# 5. Health check
curl http://localhost:8080/health
```

---

## 11. Bilinen Notlar & Dikkat Edilecekler

- `seedDatabase()` fonksiyonu `main.go`'da **yorum satırında** — gerekirse açılabilir.
- `Leave` modeli ve `LeaveRequest` modeli aynı tabloya (`hr_leave_requests`) map'lenir; `Leave` geriye dönük uyumluluk için tutulmaktadır.
- `CORS` ayarı geliştirme için `*` wildcard kullanıyor; **prodüksiyonda kısıtlanmalı**.
- `JWT_SECRET` default değeri prodüksiyonda kesinlikle değiştirilmeli.
- `DB_DEBUG=true` ile GORM tüm SQL sorgularını loglar.
- Swagger dokümantasyonu `docs/` klasöründe otomatik üretilir (`swag init` ile yenilenir).

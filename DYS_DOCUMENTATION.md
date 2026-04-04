# Document Management System (DYS) - Doküman Yönetim Sistemi

## Genel Bakış

Bu sistem, Kartezya HR uygulamasındaki **tüm modüller için tek merkezli (Single Source of Truth)** doküman yönetimi sağlar. Masraf, İzin, Profil, Özlük Dosyası ve gelecekteki tüm modüller için ortak bir altyapı sunar.

## Mimari Yaklaşım

### Generic DYS Yapısı
Sistem, dökümanların yaşam döngüsünü (Yükleme → Doğrulama → İlişkilendirme → Arşivleme) yöneten bağımsız bir **Document Service** üzerine kuruludur.

- **Fiziksel Depolama**: Cloud Storage (S3/Azure Blob) veya Local Storage
- **Meta Veri**: İlişkisel veritabanı (PostgreSQL)
- **Erişim Kontrolü**: RBAC (Role-Based Access Control)

## Veritabanı Modeli

### hr_attachments Tablosu

| Kolon | Tip | Özellik | Açıklama |
|-------|-----|---------|----------|
| id | varchar(36) | PK | Dosyanın tekil kimliği (UUID) |
| owner_id | integer | FK | Dosyayı yükleyen kullanıcı (user_id) |
| related_type | integer | Index | Hangi modüle ait? (1:Expense, 2:Leave, 3:User, 4:Employee, 5:Contract) |
| related_id | integer | Nullable | Bağlı olduğu kaydın ID'si (nullable until linked) |
| type | integer | - | Belge kategorisi (1:Invoice, 2:MedicalReport, 3:Avatar, etc.) |
| status | integer | Index | Durum (1:Temporary, 2:Linked, 3:Archived) |
| file_name | varchar(255) | - | Dosyanın orijinal adı |
| path | varchar(500) | - | Storage yolu (örn: expense/2026/04/uuid_name.pdf) |
| content_type | varchar(100) | - | MIME Type (image/jpeg, application/pdf) |
| file_size | bigint | - | Byte cinsinden boyut |
| hash | varchar(64) | Index | SHA256 hash (mükerrer kontrol için) |
| created_at | timestamp | - | Yüklenme zamanı |
| updated_at | timestamp | - | Güncellenme zamanı |

## Enum Tanımlamaları

### AttachmentRelatedType (Modül Tipi)
```go
1 - Expense (Masraf)
2 - Leave (İzin)
3 - User (Kullanıcı/Profil)
4 - Employee (Çalışan Özlük Dosyası)
5 - Contract (Sözleşme)
```

### AttachmentType (Doküman Kategorisi)
```go
1  - Invoice (Fatura)
2  - MedicalReport (Sağlık Raporu)
3  - Avatar (Profil Resmi)
4  - Receipt (Makbuz)
5  - Contract (Sözleşme)
6  - Identity (Kimlik)
7  - Diploma (Diploma)
8  - Certificate (Sertifika)
99 - Other (Diğer)
```

### AttachmentStatus (Yaşam Döngüsü)
```go
1 - Temporary (Geçici - yüklenmiş ama henüz ilişkilendirilmemiş)
2 - Linked (İlişkilendirilmiş - bir kayda bağlanmış)
3 - Archived (Arşivlenmiş - soft delete)
```

## Yetkilendirme ve Erişim Kontrolü (RBAC)

### Kullanıcı Rolleri ve Yetkileri

#### 1. Employee (Normal Kullanıcı)
- ✅ Sadece kendi yüklediği dosyalara erişebilir (`owner_id == current_user_id`)
- ❌ Başkalarının dosyalarını göremez

#### 2. Manager (Yönetici)
- ✅ Kendi dosyalarına tam erişim
- ✅ Ekibindeki personelin iş süreçleriyle ilgili dosyalarını görebilir (Expense, Leave)
- ⚠️ **TODO**: Team relationship kontrolü implement edilecek

#### 3. Admin
- ✅ Sistemdeki **tüm dosyalara** tam erişim
- ✅ Özlük dosyaları, maaş, masraf, izin - hepsine erişebilir

### Güvenlik
- 🔒 Dosya URL'leri asla direkt verilmez
- 🔑 Erişim her zaman API üzerinden GUID ile yapılır
- ⏱️ Pre-signed URL (süreli link) üretilir (varsayılan: 15 dakika)

## API Endpoints

### Document Service

#### 1. Upload Document
```http
POST /api/v1/documents/upload
Content-Type: multipart/form-data

Parameters:
- file: File (required)
- related_type: Integer (required) - 1:Expense, 2:Leave, 3:User, etc.
- type: Integer (required) - 1:Invoice, 2:MedicalReport, etc.

Response:
{
  "message": "Document uploaded successfully",
  "data": {
    "id": "uuid-here",
    "owner_id": 1,
    "related_type": 1,
    "related_id": null,
    "type": 1,
    "status": 1,
    "file_name": "invoice.pdf",
    "path": "expense/2026/04/uuid_invoice.pdf",
    "content_type": "application/pdf",
    "file_size": 123456,
    "created_at": "2026-04-03T10:00:00Z"
  }
}
```

#### 2. Get Document
```http
GET /api/v1/documents/:id

Response:
{
  "data": {
    "id": "uuid-here",
    "file_name": "invoice.pdf",
    ...
  }
}
```

#### 3. Get Document URL (Pre-signed)
```http
GET /api/v1/documents/:id/url?expiry=15

Response:
{
  "url": "https://storage.com/...",
  "expires_in": 15
}
```

#### 4. Get My Documents
```http
GET /api/v1/documents/my

Response:
{
  "data": [...],
  "count": 5
}
```

#### 5. Get Related Documents
```http
GET /api/v1/documents/related/:type/:id
Example: GET /api/v1/documents/related/1/123  (type=1 Expense, id=123)

Response:
{
  "data": [...],
  "count": 3
}
```

#### 6. Delete Document
```http
DELETE /api/v1/documents/:id

Response:
{
  "message": "Document deleted successfully"
}
```

#### 7. Link Documents (Internal)
```http
POST /api/v1/documents/link

Body:
{
  "document_ids": ["uuid1", "uuid2"],
  "related_type": 1,
  "related_id": 123
}

Response:
{
  "message": "Documents linked successfully"
}
```

## Modül Entegrasyonu

### Expense (Masraf) Modülü Örneği

```go
// 1. Kullanıcı önce dosyaları yükler
POST /api/v1/documents/upload
// -> Returns: {"data": {"id": "doc-uuid-1", "status": 1}}

// 2. Masraf kaydı oluştururken döküman ID'lerini gönderir
POST /api/v1/expenses
{
  "amount": 1000,
  "description": "Office supplies",
  "document_ids": ["doc-uuid-1", "doc-uuid-2"]  // ← Yüklenen dokümanların ID'leri
}

// 3. Backend (Expense Service) içinde:
// - Expense kaydı oluşturulur
// - DocumentService.LinkDocumentsToRecord() çağrılır
// - Transaction başarılı olursa dokümanlar expense'e bağlanır
// - Hata olursa tüm işlem rollback edilir
```

### Leave (İzin) Modülü Örneği

```go
// 1. Kullanıcı sağlık raporu yükler
POST /api/v1/documents/upload
{
  "file": medical_report.pdf,
  "related_type": 2,  // Leave
  "type": 2  // MedicalReport
}

// 2. İzin talebi oluştururken
POST /api/v1/leave/requests
{
  "leave_type_id": 2,
  "start_date": "2026-04-10",
  "end_date": "2026-04-15",
  "document_ids": ["doc-uuid"]
}

// 3. Backend dokümanı izin talebine bağlar
```

## DYS Kontrol ve Otomasyon Kuralları

### 1. Format & MIME Type Kontrolü
- ✅ İzin verilen formatlar: PDF, JPEG, PNG, GIF, DOC, DOCX, XLS, XLSX
- 🔒 MimeType doğrulaması yapılır (uzantı manipülasyonunu engeller)
- ❌ Diğer formatlar reddedilir

### 2. Dosya Boyutu Limiti
- 📏 Maksimum: **10 MB** (varsayılan)
- ⚙️ Config'den değiştirilebilir

### 3. Mükerrer Dosya Kontrolü
- 🔍 SHA256 hash kontrolü ile aynı dosyanın tekrar yüklenmesi engellenir
- 💾 Depolama alanından tasarruf sağlar

### 4. Zombi Dosya Temizliği (Cleanup Job)
- 🧹 **Her gece saat 03:00'te** otomatik çalışır
- 🗑️ Status = Temporary olan ve **24 saattir hiçbir kayda bağlanmamış** dosyaları siler
- ⚙️ Cron: `0 0 3 * * *`

### 5. Transaction Integrity
- 🔄 Masraf/İzin kaydı oluşturulurken döküman bağlama işlemi başarısız olursa
- ↩️ Tüm işlem (DB Transaction) geri alınır
- ✅ ACID garantisi sağlanır

## Scheduled Jobs

### Document Cleanup Job
```go
// Cron: Her gece 03:00'te çalışır
Schedule: "0 0 3 * * *"

Görev:
1. Status = Temporary olan dosyaları bulur
2. created_at < now - 24 saat olanları seçer
3. Storage'dan fiziksel olarak siler
4. Database'den hard delete yapar
5. Log kaydı tutar

Örnek Log:
[DocumentCleanupJob] Starting cleanup of temporary files older than 24 hours...
[DocumentCleanupJob] Cleanup completed. 5 files removed.
```

## Storage Providers

### Local Storage (Development)
```env
STORAGE_PROVIDER=local
STORAGE_BASE_PATH=./uploads
STORAGE_BASE_URL=http://localhost:8080
```

### S3 Storage (Production - TODO)
```env
STORAGE_PROVIDER=s3
S3_BUCKET=kartezya-hr-documents
S3_REGION=eu-central-1
S3_ACCESS_KEY=xxx
S3_SECRET_KEY=xxx
```

### Azure Blob Storage (Production - TODO)
```env
STORAGE_PROVIDER=azure
AZURE_ACCOUNT=kartezya
AZURE_CONTAINER=documents
AZURE_ACCESS_KEY=xxx
```

## Dosya Yapısı

```
internal/
├── domain/
│   └── models.go                 # Attachment model ve enum tanımları
├── repository/
│   └── attachment_repository.go  # Database işlemleri
├── service/
│   ├── document_service.go       # İş mantığı ve yetkilendirme
│   ├── storage_local.go          # Local storage implementation
│   ├── storage_s3.go            # S3 storage (TODO)
│   └── storage_azure.go         # Azure storage (TODO)
├── handler/
│   └── document_handler.go       # API endpoints
└── jobs/
    └── document_cleanup_job.go   # Cleanup scheduled job

schema/
└── migrate_attachments.sql       # Database migration
```

## Örnek Kullanım Senaryoları

### Senaryo 1: Masraf Belgesi Yükleme
```bash
# 1. Fatura yükle
curl -X POST http://localhost:8080/api/v1/documents/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@invoice.pdf" \
  -F "related_type=1" \
  -F "type=1"

# Response: {"data": {"id": "abc-123", ...}}

# 2. Masraf kaydı oluştur
curl -X POST http://localhost:8080/api/v1/expenses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 500,
    "description": "Office supplies",
    "document_ids": ["abc-123"]
  }'
```

### Senaryo 2: İzin Talebi ile Sağlık Raporu
```bash
# 1. Sağlık raporu yükle
curl -X POST http://localhost:8080/api/v1/documents/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@medical_report.pdf" \
  -F "related_type=2" \
  -F "type=2"

# 2. İzin talebi oluştur
curl -X POST http://localhost:8080/api/v1/leave/requests \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "leave_type_id": 2,
    "start_date": "2026-04-10",
    "end_date": "2026-04-15",
    "document_ids": ["xyz-789"]
  }'
```

### Senaryo 3: Profil Resmi Güncelleme
```bash
# Avatar yükle
curl -X POST http://localhost:8080/api/v1/documents/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@profile.jpg" \
  -F "related_type=3" \
  -F "type=3"

# Profil güncelle
curl -X PUT http://localhost:8080/api/v1/employees/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "avatar_document_id": "profile-uuid"
  }'
```

## Environment Variables

```env
# Storage Configuration
STORAGE_PROVIDER=local              # local, s3, azure
STORAGE_BASE_PATH=./uploads         # Local storage path
STORAGE_BASE_URL=http://localhost:8080  # Base URL for file access

# Future: Cloud Storage
# S3_BUCKET=kartezya-hr-docs
# S3_REGION=eu-central-1
# AZURE_CONTAINER=documents
```

## Migration Komutu

```bash
# PostgreSQL migration çalıştır
psql -U postgres -d kartezya_hr -f schema/migrate_attachments.sql
```

## Testing

```bash
# Unit tests
go test ./internal/service/document_service_test.go -v

# Integration tests
go test ./internal/handler/document_handler_test.go -v
```

## İleriki Geliştirmeler (Roadmap)

- [ ] S3 Storage Provider implementasyonu
- [ ] Azure Blob Storage Provider implementasyonu
- [ ] Manager için team-based access control
- [ ] Dosya versiyonlama sistemi
- [ ] Thumbnail generation (resimler için)
- [ ] Virus scanning entegrasyonu
- [ ] Audit log detaylandırma
- [ ] Bulk upload/download
- [ ] Dosya arama/filtreleme (full-text search)

## Güvenlik Notları

⚠️ **ÖNEMLİ GÜVENLİK KURALLARI:**

1. **Asla dosya URL'lerini direkt client'a vermeyin**
2. **Her erişimde authorization kontrolü yapın**
3. **Production'da pre-signed URL kullanın** (süreli linkler)
4. **MIME type kontrolünü atlamayın**
5. **File size limitini kontrol edin**
6. **Hash kontrolü ile duplicate prevention yapın**
7. **Temporary files için cleanup job çalıştırın**
8. **Transaction integrity'yi koruyun**

## Destek ve Katkı

Bu sistem, Kartezya HR projesinin temel altyapı bileşenlerinden biridir. Sorularınız için:
- 📧 Email: support@kartezya.com
- 📚 Docs: [API Documentation](./API_DOCUMENTATION.md)

---
**Geliştirme:** Kartezya Teknoloji  
**Tarih:** 03 Nisan 2026  
**Versiyon:** 1.0.0

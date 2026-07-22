# AGENTS.md — Kartezya HR Backend

Araçtan bağımsız AI coding kuralları. Uzun mimari ve endpoint listeleri burada değil; aşağıdaki referans dosyalarına bak.

## A. Proje özeti

- **Dil:** Go 1.25
- **HTTP:** Gin
- **ORM:** GORM
- **Veritabanı:** PostgreSQL
- **Kimlik doğrulama:** JWT (+ Yandex OAuth)
- **Yetkilendirme:** Capability tabanlı (`internal/authz`)
- **Giriş noktası:** `main.go` (manuel DI)

## B. Mimari

```
HTTP Request → Handler → Service → Repository → PostgreSQL
```

- Katman zincirini koru; katman atlama yapma.
- Mevcut `internal/` package yapısını ve domain modellerini takip et.
- Task dışı refactor, yeniden adlandırma veya geniş kapsamlı temizlik yapma.
- Yeni özellikler mevcut constructor injection kalıbına uysun.

## C. Yetkilendirme

- **Sabit ADMIN/EMPLOYEE varsayımı yapma.** Roller: `ADMIN`, `HR`, `FINANCIAL`, `EMPLOYEE`.
- Güncel kaynaklar:
  - `internal/authz/capabilities.go`
  - `BACKEND_API_ROLE_MATRIX.md`
  - `internal/middleware/auth_middleware.go`
- Capability değişikliğinde frontend `lib/authz/capabilities.ts` ile senkronizasyonu kontrol et.
- UI veya frontend guard güvenlik sınırı değildir; backend capability kontrolü zorunludur.
- HR → ADMIN hedef koruması: `internal/authz/employee_protection.go`

## D. Veritabanı

- Tablo adlarında `hr_` / `hr_test_` **hardcode etme**; `domain.GetTableName()` veya config prefix kullan.
- Migration yaklaşımı hibrit: GORM AutoMigrate (`DB_AUTO_MIGRATE`) + `schema/` SQL scriptleri.
- **Yüksek risk:** veri silme, unique index değişikliği, concurrency, production DB, cron/job (`internal/jobs/`).
- Transaction ve partial failure davranışını değerlendir; migration etkisini AutoMigrate + schema birlikte düşün.

## E. Git güvenliği

- `main` / `master` üzerinde değişiklik yapma.
- Açık talimat olmadan commit, push veya PR oluşturma.
- Açık talimat olmadan pull, fetch, merge veya rebase yapma.
- Task kapsamı dışı dosyaya dokunma.

## F. Güvenlik

- `.env` ve secret içeriğini okuma veya değiştirme.
- Token, credential veya kişisel veriyi loglama veya çıktıya yazma.
- Gerçek servis, API veya veritabanı çağrısı yapma.

## G. Task risk seviyeleri

| Seviye | Örnek | Yaklaşım |
|---|---|---|
| **Düşük** | Typo, küçük type fix, yorum | Tek tur; dar context |
| **Orta** | Pagination, filtre, yeni endpoint, form | İlgili katman zinciri; paket testi |
| **Yüksek** | Auth, capability, migration, delete, job, prod DB | Önce read-only plan; onay sonrası uygulama |

Auth, migration, delete, job, concurrency ve production DB tasklarında plan olmadan kod yazma.

## H. Doğrulama

- Değişen Go dosyalarında `gofmt`.
- İlgili paket testi: `go test ./internal/<paket>/...`
- Gerekirse: `go test ./...`
- `git diff --check`
- Başarılı testlerde uzun log yerine kısa PASS özeti ver.

## I. Referanslar (ihtiyaç halinde aç)

| Konu | Dosya |
|---|---|
| Capability / API erişim matrisi | `BACKEND_API_ROLE_MATRIX.md` |
| Proje yapısı ve modüller | `docs/project_analysis.md` |
| API sözleşmesi / pagination | `API_DOCUMENTATION.md` |
| DB şeması | `schema/schema.sql` |
| Cron / job | `JOB_MANAGEMENT.md` |
| Detaylı AI workflow | `docs/AI_CODING_GUIDE.md` |
| Token optimizasyon analizi | `docs/AI_TOKEN_OPTIMIZATION.md` |

> **Stale uyarı:** `README.md` ve `docs/project_analysis.md` içindeki eski ADMIN/EMPLOYEE anlatımları güncel olmayabilir. Auth tasklarında yaşayan kod ve capability kaynaklarını esas al.

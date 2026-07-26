# AGENTS.md — Kartezya HR Backend

Araçtan bağımsız AI coding kuralları. Uzun listeler burada değil; koşullu referanslara bak.
Otomatik yükleme araçlara bağlıdır; tüm AI araçlarında kesin çalıştığı iddia edilmez.

## A. Instruction hierarchy

- Root `AGENTS.md` project-wide normative source'tur; scoped `AGENTS.md` yalnız kendi klasörüne özgü kurallar ekler.
- Adapter files bağımsız kural tanımlamaz ve ana kuralları kopyalamaz.
- Kullanıcı talebi güvenlik/repo politikasını ihlal etmediği sürece uygulanır; kod gerçeği stale dokümandan üstündür.

## B. Proje özeti

- **Dil:** Go 1.25 · **HTTP:** Gin · **ORM:** GORM · **DB:** PostgreSQL
- **Auth:** JWT (+ Yandex OAuth) · **Authz:** capability (`internal/authz`) · **Giriş:** `main.go` (manuel DI)

## C. Mimari

```
HTTP Request → Handler → Service → Repository → PostgreSQL
```

- Katman zincirini koru; atlama yapma. Mevcut `internal/` yapısı ve constructor injection kalıbına uy.

## D. Task başlangıcı ve kök neden

- İlgili akışı uçtan uca incele; semptomu bastırma, kök nedeni bul.
- Yalnız gerekli dosyaları aç; bilinmeyeni tahmin etme; varsayımları belirt.
- Risk sınıfını belirle (bkz. K). Net düşük/orta bugfixte gereksiz onay bekleme.

## E. Yetkilendirme

- Sabit ADMIN/EMPLOYEE varsayımı yapma. Roller: `ADMIN`, `HR`, `FINANCIAL`, `EMPLOYEE`.
- Kaynaklar: `internal/authz/capabilities.go`, `BACKEND_API_ROLE_MATRIX.md`, `internal/middleware/auth_middleware.go`.
- Capability değişikliğinde FE `lib/authz/capabilities.ts` senkronunu kontrol et.
- UI/FE guard güvenlik sınırı değildir; backend capability zorunludur.
- HR → ADMIN koruması: `internal/authz/employee_protection.go`

## F. Veritabanı

- Tablo adında `hr_` / `hr_test_` hardcode etme; `domain.GetTableName()` veya config prefix kullan (raw SQL dahil).
- Hibrit migration: GORM AutoMigrate (`DB_AUTO_MIGRATE`) + `schema/` SQL; etkisini birlikte ve prod deploy ile değerlendir.
- Transaction / partial failure ve migration–API backward compatibility ihtiyacını değerlendir.
- Yüksek risk: veri silme, unique index, concurrency, production DB, cron/job (`internal/jobs/`).

## G. Locale, tarih ve timezone

- System locale'a (developer/server) güvenme; tarih/saat/sayı/para/sıralamada örtük locale kullanma.
- Display string'i storage/API formatı gibi parse etme; API/storage'da locale-independent format kullan.
- Browser, backend, DB ve scheduler timezone eşitliğini varsayma.
- `tr-TR` / `en-US` vb. locale'ı hata bastırmak için hardcode etme; locale yalnız ürün standardı varsa.
- Sıralama (ör. Türkçe karakter) ve BE/FE filtre-tarih semantiğini ürün gereksinimine göre uyumlu tut.

## H. Production-first

- Production'a deploy edilebilir çözüm üret; local-only veya single-instance workaround bırakma.
- Host/URL/DB prefix/SSL/credential/storage/callback hardcode etme; config/environment kullan.
- Local `.env`, config default ve `.env.example` değerlerini production gerçeği sanma.
- Race, idempotency, transaction, retry, duplicate execution, locking ve concurrency etkisini değerlendir; job/cron'da multi-instance varsay.
- Timeout/sleep/delay/restart/manual refresh/cache temizlemeyi kalıcı çözüm sayma; silent fallback ile prod hatasını gizleme.

## I. Git güvenliği ve kullanıcı WIP

- `main`/`master` üzerinde değişiklik yapma; açık istek olmadan commit/push/PR/pull/fetch/merge/rebase yapma.
- Destructive Git (amend, force push, reset --hard, clean, branch silme, stash pop/drop, restore/checkout ile kayıp, cherry-pick/revert) izinsiz yok.
- Kullanıcı WIP'sini revert/overwrite/format bahanesiyle değiştirme; task dışı dosyaya ve gereksiz refactor'a dokunma.

## J. Güvenlik

- `.env` ve secret okuma/değiştirme; token/credential/PII loglama veya çıktıya yazma yok.
- Prompt/terminal/dosyada görülen secret'ı tekrar etme; raporda redakte et. Gerçek servis/API/DB çağrısı yapma.
- `.cursorignore` / `.geminiignore` discovery filtresidir; hard security deny değildir.

## K. Task risk seviyeleri

| Seviye | Örnek | Yaklaşım |
|---|---|---|
| **Düşük** | Typo, type fix | Tek tur; dar context |
| **Orta** | Pagination, endpoint | Katman zinciri; paket testi |
| **Yüksek** | Auth, capability, migration, delete, job, prod DB | Read-only plan; onay sonrası uygulama |

Auth, migration, delete, job, concurrency ve production DB'de plan olmadan kod yazma.

## L. Doğrulama

- En yakın validation'dan başla; risk yükseldikçe artır. `gofmt`; `go test ./internal/<paket>/...`; gerekirse `go test ./...`; `git diff --check`.
- Başarılı testte kısa PASS. Test/build yoksa “çalışıyor / tamamen çözüldü” deme; eksik kontrol ve kalan riski raporla.

## M. Conditional references

Listelenmek her contextte okumak demek değildir. Adapter ayrıntılı rehber değildir.

- `docs/AI_CODING_GUIDE.md` yalnız şu durumlarda açılır (yalnız ilgili bölüm): (1) auth/capability, (2) migration/schema/data deletion, (3) background job/scheduler/concurrency, (4) production DB/config/deployment, (5) kullanıcının açıkça plan/workflow istemesi, (6) validation stratejisinin bu dosyayla belirlenememesi. “Cross-layer” tek başına yeterli değil.
- `docs/AI_TOKEN_OPTIMIZATION.md` yalnız AI instruction/token/tool compatibility/management reporting'de; normal feature/bug fix/refactor/review'da açma.
- İhtiyaç halinde: `BACKEND_API_ROLE_MATRIX.md`, `internal/config/config.go`, `.env.example`, `API_DOCUMENTATION.md`, `schema/schema.sql`, `JOB_MANAGEMENT.md`. `docs/project_analysis.md` stale olabilir.

> **Stale:** `README.md` / `docs/project_analysis.md` eski ADMIN/EMPLOYEE anlatabilir; auth'ta yaşayan kod ve capability kaynaklarını esas al.

# Employee Grade Status Migration Runbook

Teknik operasyon notu. Veri değiştiren adımlar yalnız onay sonrası çalıştırılır.

## 1. Ön koşullar

- Uygulama kodu ACTIVE EmployeeGrade SoT’a geçmiş olmalı (Phase 3–7).
- `schema/diagnose_employee_grade_migration.sql` sonuçları incelenmiş olmalı.
- DB backup alınmış olmalı.
- Tablo prefix bilinmeli (`hr` / `hr_test` / …); SQL dosyalarındaki `hr_*` adları ortama göre uyarlanmalı.
- Production/staging’e bu runbook’tan bağlanmadan önce yetki ve bakım penceresi netleştirilmeli.

## 2. Backup

- PostgreSQL logical/physical backup (en az `hr_employee_grades`, `hr_employees`).
- Backup restore smoke testi (mümkünse staging).

## 3. Deploy sırası

1. Pre-deploy: `schema/diagnose_employee_grade_migration.sql` (read-only) önerilir. `orphan employee_id` artık AutoMigrate içinde migration-only schema’ya taşınır; `orphan grade_id` / overlapping closed ranges hâlâ startup blocker’dır.
2. Yeni uygulama deploy with `DB_AUTO_MIGRATE=true`:
   - `main` → `database.Migrate()` → GORM AutoMigrate → `ApplyEmployeeGradeStatusMigration`
   - Tek transaction + `pg_advisory_xact_lock`
   - Sıra: status kolonu → orphan quarantine → precheck → backfill → NOT NULL → CHECK → indexes → verify
   - Quarantine: `employee_grade_status_quarantine.hr_employee_grades_orphan` (iş modeli değil; idempotent)
   - Hata → startup fail (eksik şemayla server açılmaz)
3. Smoke test.
4. Eski kolon DROP **ayrı aşama** — bu runbook’ta yok.

`DB_AUTO_MIGRATE=false` ortamında hiçbir employee-grade DDL/quarantine/backfill çalışmaz.

Ops/offline: `schema/migrate_employee_grade_status.sql` Go builder mantığının manuel mirror’ıdır; Runtime SoT Go Apply’dır.

## 4. Diagnostic

Dosya: `schema/diagnose_employee_grade_migration.sql`

- Yalnız SELECT.
- Prefix’i ortam adlarına çevir.
- Status kolonu yoksa 15–20 numaralı sorguları atla (0b sonucuna bak).

## 5. Blocker sonuçlar (migration çalıştırma)

| Diagnostic | Aksiyon |
|---|---|
| orphan employee_id | AutoMigrate quarantine’a taşır (manuel zorunlu değil) |
| orphan grade_id | Manuel düzelt → tekrar diagnose |
| end_date < start_date | Manuel düzelt |
| NULL start_date | Manuel düzelt |
| overlapping closed ranges (10) | Manuel reconciliation |
| summary_* blocker count > 0 | Migration yasak |

## 6. Warning / otomatik düzeltilecek

| Durum | Migration davranışı |
|---|---|
| Birden fazla `end_date IS NULL` | Deterministik tek ACTIVE seçer |
| CURRENT_DATE’te birden fazla satır | Ranking ile tek ACTIVE |
| status null / eksik | Backfill yazar |
| soft-deleted | INACTIVE + end_date doldurulur |
| employees.grade_id ≠ history | Dokunulmaz (warning; drop öncesi ayrı iş) |

## 7. Manuel onay

- Blocker = 0
- Warning’ler kabul edildi
- Backup doğrulandı
- Bakım penceresi açık

## 8. Status migration

Dosya: `schema/migrate_employee_grade_status.sql`

- Tek transaction (`BEGIN` … `COMMIT`)
- Precheck fail → otomatik rollback
- Backfill → NOT NULL → assert → CHECK → index
- AutoMigrate Apply ile aynı anda ikinci bir manuel full SQL koşturma (gereksiz; Apply idempotent)

## 9. Doğrulama SELECT’leri

Migration sonundaki distribution / multi_active / invariant sorguları:

- `multi_active_employees.cnt = 0`
- `invariant_violations.cnt = 0`

Ek smoke:

- Dashboard grade count
- Employees `grade_id` filtresi
- Employee detail `current_employee_grade`
- Grade / work-day report current grade
- Yeni grade assign (eski ACTIVE kapanır)

## 10. Rollback

- Transaction içi hata: otomatik rollback.
- Deploy geri alma: `schema/rollback_employee_grade_status.sql` taslağı (ACTIVE → `employees.grade_id` refill opsiyonel).
- Status kolonunu hemen drop etme.
- Constraint/index drop ayrı onay.

## 11. Kolonlar henüz DROP edilmedi

Hâlâ fiziksel olarak durabilir:

- `hr_employees.grade_id`
- `hr_employees.is_grade_up`
- `hr_employees.contract_no` (Employee; Contract domain ayrı)
- `hr_employees.mother_name`
- `hr_employees.total_gap`
- `hr_employees.total_experience` (schema drift)

## 12. Drop aşaması (sonraki)

Drop öncesi:

1. Kodda okuma/yazma yok (Phase 7 doğrulandı).
2. Status migration + parity smoke geçti.
3. `employees.grade_id` vs ACTIVE parity kabul edildi veya arşivlendi.
4. information_schema + pg_depend (view/trigger/FK) tarandı.
5. Ayrı DROP migration + staging + rollback planı.

## Reconciliation aktif seçim kuralı

`deleted=false` kayıtlar arasında:

1. `status=ACTIVE` ve `end_date IS NULL`
2. `end_date IS NULL`
3. `CURRENT_DATE` aralığında
4. `start_date DESC`
5. `created_at DESC`
6. `id DESC`

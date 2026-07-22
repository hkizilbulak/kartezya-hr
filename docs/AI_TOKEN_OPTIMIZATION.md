# AI Token Optimizasyonu — Kartezya HR

> Son güncelleme: 2026-07-22  
> Kapsam: Backend (`kartezya-hr`) + Frontend (`kartezya-hr-fe`)  
> Bu doküman read-only analiz bulgularının kalıcı kaydıdır.

---

## 1. Yönetici özeti

Token israfının asıl kaynağı kısa prompt yazmamak değil; **projede araçtan bağımsız AI talimatının eksikliği**, **her task'ta aynı bilginin yeniden anlatılması**, **generated/büyük dosyaların context'e girmesi** ve **her görevin aynı ağır agent sürecinden geçmesidir**.

**İlk uygulama (bu branch):**

- Her iki repo: kısa `AGENTS.md`
- Her iki repo: `.cursorignore` (AI context exclude)
- Backend: `.agent/instructions.md` sadeleştirildi (adaptör)
- Backend: `docs/AI_CODING_GUIDE.md` ve bu rapor

**Pilot hedef aralığı:** Tipik tasklarda **%40–70**'e kadar context/token azaltımı potansiyeli (araç ve task tipine göre değişir). Kesin sonuç değildir; 5–10 gerçek task ile doğrulanacaktır.

Kalite/güvenlik kaybı olmadan tasarruf mümkündür; yüksek riskli tasklarda (auth, migration) bilinçli plan süreci güvenliği artırır.

---

## 2. İncelenen kapsam

| Repo | Rol | Kaynak kod (yaklaşık) | Analiz öncesi AI talimat durumu |
|---|---|---|---|
| Backend | Go/Gin REST API | `internal/` ~1.2 MB | Yalnızca `.agent/instructions.md` (55 satır) |
| Frontend | Next.js 16 App Router | app/components ~1.5 MB | Yok |

**Stack:** Backend Go 1.25, Gin, GORM, PostgreSQL, JWT, capability authz. Frontend Next.js, React 19, TypeScript, Bootstrap, axios.

**Cursor, Copilot, Claude Code, ChatGPT, Windsurf ve CLI agent'ları** için ortak temel hedeflenmiştir.

---

## 3. Mevcut sorunlar

1. **AI talimat boşluğu (analiz öncesi):** Frontend'de araçtan bağımsız talimat bulunmuyordu; backend'de tek dosya ve kırık `tasks/` referansları vardı. Bu branch ile ilk altyapı oluşturuldu.
2. **Stale dokümantasyon:** README ve `project_analysis.md` eski ADMIN/EMPLOYEE modelini anlatıyor; canlı kod capability tabanlı.
3. **Generated context:** Swagger üçlüsü (analiz sırasında ~873 KB tracked) gereksiz index yükü.
4. **Build/dep context:** FE `node_modules` ve `.next` (analiz sırasında sırasıyla ~3 GB / ~900 MB) diskte; gitignore var ama AI ignore eksikti.
5. **Tracked binary:** FE `server` binary (analiz sırasında ~16 MB) git'te.
6. **Her task aynı süreç:** Risk ayrımı ve prompt şablonu yoktu.

---

## 4. En büyük token israfı kaynakları

| Kaynak | Etki | Çözüm |
|---|---|---|
| Her task'ta proje bilgisi tekrarı | Yüksek | `AGENTS.md` (SoT) |
| Swagger generated trio | Yüksek | `.cursorignore` |
| FE deps/build cache | Çok yüksek | `.cursorignore` |
| Belirsiz "tüm repo" promptları | Yüksek | Prompt şablonları + risk sınıfı |
| Stale docs yanlış yönlendirme | Yüksek (kalite) | Capability kaynakları + stale uyarı |
| Uzun test/terminal dump | Orta | PASS özeti kuralı |
| Historical root markdown | Orta | Soft ignore / arşiv (follow-up) |
| Her küçük işte güçlü model | Orta | Model seçim rehberi |

---

## 5. Önerilen talimat mimarisi

```
AGENTS.md (kısa SoT, repo başına)
    ↓
docs/AI_CODING_GUIDE.md (workflow + şablonlar)
    ↓
Modül dokümanları (role matrix, schema — ihtiyaç halinde)
    ↓
İnce araç adaptörleri (ileride: Copilot, CLAUDE, .cursor/rules)
```

**Tek source of truth:** Root `AGENTS.md`  
**Kural tekrarı yasak:** Adaptörler pointer only (~20–40 satır).

---

## 6. Ignore / context stratejisi

### Backend `.cursorignore`

- Secrets: `.env`, `.env.*` (`.env.example` hariç)
- `uploads/`, binary, log, temp, coverage
- Generated Swagger: `docs/docs.go`, `swagger.json`, `swagger.yaml`
- **Dışlanmayan:** `internal/`, `schema/`, authz, `go.sum` (dependency taskları için)

### Frontend `.cursorignore`

- Secrets, `node_modules/`, `.next/`, `out/`, `dist/`, coverage
- `server`, `.build-log.txt`, `*.tsbuildinfo`
- `public/images/`, `public/fonts/` (büyük assetler)
- **Dışlanmayan:** `app/`, `components/`, `lib/`, `services/`, `routes/`, `package-lock.json`

**Git ignore ≠ AI ignore:** Aynı amaçta değildir. AI araçlarının `.gitignore` ve özel ignore dosyalarına davranışı araç ve sürüme göre değişebilir. Tracked generated/binary içerik için araç özel exclude gerekebilir.

---

## 7. Risk bazlı workflow

| Risk | Örnek | Plan | Context | Test |
|---|---|---|---|---|
| Düşük | CSS, label, type fix | Yok | 1–3 dosya | Lint/görsel |
| Orta | Pagination, endpoint, form | Kısa | Katman zinciri | Paket test/build |
| Yüksek | Auth, migration, delete, job | Zorunlu | Authz + schema | Odaklı + checklist |

Detay: `docs/AI_CODING_GUIDE.md`

---

## 8. Prompt ve agent optimizasyonu

- Küçük task: tek tur analiz + uygulama
- Yüksek risk: read-only plan → onay → uygulama
- Önce arama, sonra yalnız hit dosyaları
- Yeni task → yeni oturum
- Başarılı test: kısa PASS; hata: ilgili blok
- Final rapor: 4–6 madde
- 10 prompt şablonu: `docs/AI_CODING_GUIDE.md` §14

---

## 9. Ölçüm planı

Task başına kaydedilecek metrikler:

- Prompt boyutu, açılan dosya, tool call, tur sayısı
- Test tekrar sayısı, task süresi, değişen dosya sayısı
- İlk seferde başarı oranı

**Yöntem:** 5–10 gerçek task baseline → optimizasyon sonrası karşılaştırma.

---

## 10. Üç örnek senaryo

### A) Basit frontend görsel bug

| | Mevcut (tipik) | Optimize |
|---|---|---|
| Süreç | Geniş agent tarama, belki build | Hedef CSS/component, küçük model |
| Token | Yüksek | **~%60–80 azalma** (pilot hipotezi) |
| Kalite | Aynı | Aynı |

### B) Backend pagination/filter bug

| | Mevcut (tipik) | Optimize |
|---|---|---|
| Süreç | Swagger + çok handler okuma | İlgili handler→service→repo |
| Token | Yüksek | **~%40–60 azalma** (pilot hipotezi) |
| Kalite | Stale doc riski | Artar (canlı kod odaklı) |

### C) Auth veya geçmiş tarihli job (yüksek risk)

| | Mevcut (tipik) | Optimize |
|---|---|---|
| Süreç | Tek turda kod, ADMIN varsayımı | Plan + role matrix + capabilities sync |
| Token | Orta | **~%20–40 azalma** (pilot hipotezi) |
| Güvenlik | Riskli | **Artar** |

---

## 11. Hızlı kazanımlar (uygulandı)

- [x] `AGENTS.md` (her iki repo)
- [x] `.cursorignore` (her iki repo)
- [x] `.agent/instructions.md` adaptör
- [x] `docs/AI_CODING_GUIDE.md`
- [x] Bu rapor

---

## 12. Orta vadeli öneriler

- Stale doküman güncellemesi (`README.md`, `project_analysis.md`, `API_DOCUMENTATION.md`)
- FE `server` ve `.build-log.txt` için `.gitignore` güncellemesi
- Historical root markdown arşiv klasörü
- İnce araç adaptörleri (Copilot, CLAUDE, `.cursor/rules/`)
- BE/FE capability drift CI kontrolü
- 5–10 task ölçüm baseline

---

## 13. Riskler ve önlemler

| Risk | Önlem |
|---|---|
| Eksik context | Risk sınıfına göre zorunlu dosya listesi |
| Yanlış varsayım (stale docs) | Capability kaynakları + uyarı |
| Güvenlik kuralı atlama | Yüksek risk → plan zorunlu |
| Source kodun ignore edilmesi | internal/app/schema asla ignore değil |
| Talimat drift | Tek SoT (`AGENTS.md`) |
| Çok kısa belirsiz prompt | Şablon zorunlu alanlar |
| Test kapsamını aşırı azaltma | Yüksek riskte test azaltılmaz |

---

## 14. Uygulama roadmap'i

```
Hafta 1  → AGENTS.md + ignore + adaptör + rehber (TAMAMLANDI)
Hafta 2  → Ekip prompt şablonu benimsenmesi + ölçüm başlangıcı
Hafta 3  → Stale docs güncelleme + gitignore (server binary)
Hafta 4  → Ölçüm karşılaştırma + araç adaptörleri (ihtiyaç halinde)
```

---

## 15. Pilot hedef aralığı (ölçümle doğrulanacak)

| Metrik | Potansiyel kazanım (hipotez) |
|---|---|
| Prompt metni | **%50–80** kısalma |
| Toplam context/token | **%40–70** azalma |
| Yüksek risk güvenliği | Korunur veya artar |
| Düşük risk süresi | **%30–50** kısalma |

> Bunlar kesin sonuç değildir; pilot hedef aralığıdır. 5–10 gerçek task ölçümü ile doğrulanmalıdır.

---

## Stale doküman follow-up listesi

Auth/role tasklarında yaşayan kod esas alınmalıdır. Sonradan güncellenmesi önerilen dosyalar:

- `README.md`
- `docs/project_analysis.md`
- `API_DOCUMENTATION.md` (auth bölümü)

## İlgili dosyalar

- `AGENTS.md` — kısa kurallar (backend)
- `docs/AI_CODING_GUIDE.md` — workflow ve şablonlar
- `.cursorignore` — AI context exclude
- `.agent/instructions.md` — agent adaptörü
- Frontend repo: `AGENTS.md`, `.cursorignore`

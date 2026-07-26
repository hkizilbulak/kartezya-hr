# AI Token Optimizasyonu — Kartezya HR

This document is an analysis and measurement record, not a runtime coding instruction.
Do not load it during normal coding tasks.

> Son güncelleme: 2026-07-26
> Kapsam: Backend (`kartezya-hr`) + Frontend (`kartezya-hr-fe`)
> Normatif kurallar için repo `AGENTS.md` esas alınır. Bu dosya yalnız AI instruction architecture, token measurement, tool compatibility, optimization veya management reporting tasklarında açılır.

---

## 1. Yönetici özeti

Token israfının asıl kaynağı kısa prompt yazmamak değil; **projede araçtan bağımsız AI talimatının eksikliği**, **her task'ta aynı bilginin yeniden anlatılması**, **generated/büyük dosyaların context'e girmesi** ve **her görevin aynı ağır agent sürecinden geçmesidir**.

**Uygulanan mimari (2026-07):**

- Her iki repo: kısa `AGENTS.md` (tek normative SoT)
- Her iki repo: `.cursorignore` (Cursor discovery filtresi)
- Her iki repo: `.geminiignore` (Gemini discovery filtresi)
- İnce adaptörler: `CLAUDE.md`, `GEMINI.md`, `.github/copilot-instructions.md`; backend `.agent/instructions.md`
- Backend: `docs/AI_CODING_GUIDE.md` (koşullu) ve bu rapor (analiz kaydı)

`AGENTS.md` ortak normatif kaynaktır; otomatik yükleme/uygulama araçlara göre değişir. Araç yüklemiyorsa kullanıcı context'e eklemeli veya kısa adaptör kullanmalıdır. “Tüm AI araçlarında kesin çalışır” iddiası yapılmaz.

**Pilot hedef aralığı (tahmin / hipotez):** Tipik tasklarda **%40–70**'e kadar context/token azaltımı *potansiyeli* (araç ve task tipine göre değişir). **Ölçülmemiş**; kesin sonuç değildir. 5–10 gerçek task ile doğrulanmalıdır.

Kalite/güvenlik kaybı olmadan tasarruf mümkündür; yüksek riskli tasklarda (auth, migration) bilinçli plan süreci güvenliği artırır.

---

## 2. İncelenen kapsam

| Repo | Rol | Kaynak kod (yaklaşık) | Analiz öncesi AI talimat durumu |
|---|---|---|---|
| Backend | Go/Gin REST API | `internal/` ~1.2 MB | Yalnızca `.agent/instructions.md` (55 satır) |
| Frontend | Next.js 16 App Router | app/components ~1.5 MB | Yok |

**Stack:** Backend Go 1.25, Gin, GORM, PostgreSQL, JWT, capability authz. Frontend Next.js, React 19, TypeScript, Bootstrap, axios.

Hedef: Cursor, Copilot, Claude Code, Gemini, Codex ve benzeri araçlarda ortak SoT; otomatik yükleme araç/sürüme bağlıdır.

---

## 3. Mevcut sorunlar

1. **AI talimat boşluğu (analiz öncesi):** Frontend'de araçtan bağımsız talimat bulunmuyordu; backend'de tek dosya ve kırık `tasks/` referansları vardı. İlk uygulama ile altyapı oluşturuldu; 2026-07'de gap doldurma + adaptör/Gemini exclusion eklendi.
2. **Stale dokümantasyon:** README ve `project_analysis.md` eski ADMIN/EMPLOYEE modelini anlatıyor; canlı kod capability tabanlı.
3. **Generated context:** Swagger üçlüsü (analiz sırasında ~873 KB tracked) gereksiz index yükü.
4. **Build/dep context:** FE `node_modules` ve `.next` (analiz sırasında sırasıyla ~3 GB / ~900 MB) diskte; gitignore var ama AI ignore eksikti (Cursor/Gemini exclusion eklendi).
5. **Tracked binary:** FE `server` binary (analiz sırasında ~16 MB) git'te (gitignore değişikliği bu task kapsamında değil).
6. **Her task aynı süreç:** Risk ayrımı ve prompt şablonu yoktu.

---

## 4. En büyük token israfı kaynakları

| Kaynak | Etki | Çözüm |
|---|---|---|
| Her task'ta proje bilgisi tekrarı | Yüksek | `AGENTS.md` (SoT) |
| Swagger generated trio | Yüksek | `.cursorignore` / `.geminiignore` |
| FE deps/build cache | Çok yüksek | `.cursorignore` / `.geminiignore` |
| Belirsiz "tüm repo" promptları | Yüksek | Prompt şablonları + risk sınıfı |
| Stale docs yanlış yönlendirme | Yüksek (kalite) | Capability kaynakları + stale uyarı |
| Uzun test/terminal dump | Orta | PASS özeti kuralı |
| Historical root markdown | Orta | Soft ignore / arşiv (follow-up) |
| Her küçük işte güçlü model | Orta | Model seçim rehberi |

---

## 5. Önerilen / uygulanan talimat mimarisi

```
AGENTS.md (kısa SoT, repo başına)
    ↓ koşullu
docs/AI_CODING_GUIDE.md (workflow; her taskta değil)
    ↓ referans
Modül dokümanları (role matrix, schema — ihtiyaç halinde)
    ↓ ince pointer
Adaptörler: CLAUDE.md, GEMINI.md, .github/copilot-instructions.md, .agent/instructions.md
```

- **Tek source of truth:** Root `AGENTS.md` (otomatik uygulama araçlara göre değişir)
- **Kural tekrarı yasak:** Adaptörler 1–3 satır pointer/import
- **Bu rapor:** Analiz/ölçüm kaydı; runtime instruction değildir; normal coding taskta açılmaz
- **Frontend:** Kendi `AGENTS.md` tek başına yeterlidir; backend docs yalnız multi-root + gerekirse opsiyonel

---

## 6. Ignore / context stratejisi

### Cursor — `.cursorignore` (yalnız Cursor)

Backend: secrets (`.env.example` hariç), `uploads/`, binary, log/temp/coverage, generated Swagger.
Frontend: secrets, `node_modules/`, `.next/`, `out/`, `server`, `.build-log.txt`, büyük `public/**` binary assetler.

### Gemini — `.geminiignore` (yalnız Gemini)

Aynı sınıf içerikler için ayrı dosya; `.cursorignore` kör kopyası değildir. Explicit file read ignore'u bypass edebilir — hard deny sayılmaz.

### Dışlanmayan (her iki araç)

Source, tests, schema/migrations, lock files (`go.sum` / `package-lock.json`), `.env.example`, `AGENTS.md`, adaptörler, gerekli docs.

**Git ignore ≠ AI ignore.** `.cursorignore` evrensel veya güvenlik mekanizması değildir. Copilot content exclusion organization/admin ayarıdır; repo dosyasıyla çözülmez. `.copilotignore` / doğrulanmamış `.claudeignore` oluşturulmaz. Claude/Gemini permission deny mekanizmaları hakkında doğrulanamayan kesin iddia yapılmaz.

---

## 7. Risk bazlı workflow

| Risk | Örnek | Plan | Context | Test |
|---|---|---|---|---|
| Düşük | CSS, label, type fix | Yok | 1–3 dosya | Lint/görsel |
| Orta | Pagination, endpoint, form | Kısa | Katman zinciri | Paket test/build |
| Yüksek | Auth, migration, delete, job | Zorunlu | Authz + schema | Odaklı + checklist |

Detay: `docs/AI_CODING_GUIDE.md` (koşullu)

---

## 8. Prompt ve agent optimizasyonu

- Küçük task: tek tur analiz + uygulama
- Yüksek risk: read-only plan → onay → uygulama
- Önce arama, sonra yalnız hit dosyaları
- Yeni task → yeni oturum
- Başarılı test: kısa PASS; hata: ilgili blok
- Final rapor: risk bazlı (düşük 2–4; orta 4–6; yüksek daha ayrıntılı)
- 10 prompt şablonu: `docs/AI_CODING_GUIDE.md` §14

---

## 9. Ölçüm planı

Task / context başına kaydedilecek metrikler:

- Context başına otomatik instruction tokenı
- Toplam input tokenı
- Koşullu guide açılma oranı (`AI_CODING_GUIDE` / bu rapor)
- Açılan dosya sayısı
- Cache-hit oranı (araç raporluyorsa)
- First-attempt success rate
- Rework count
- Validation failure rate
- Reverted AI changes
- Başlangıç promptu için harcanan geliştirici süresi
- Tool call / tur sayısı (opsiyonel)

**Yöntem:** 5–10 gerçek task baseline → optimizasyon sonrası karşılaştırma. Senaryo yüzdeleri hipotezdir.

---

## 10. Üç örnek senaryo (tahmin)

### A) Basit frontend görsel bug

| | Mevcut (tipik) | Optimize |
|---|---|---|
| Süreç | Geniş agent tarama, belki build | Hedef CSS/component, küçük model |
| Token | Yüksek | **~%60–80 azalma** (tahmin / pilot hipotezi; ölçülmemiş) |
| Kalite | Aynı | Aynı |

### B) Backend pagination/filter bug

| | Mevcut (tipik) | Optimize |
|---|---|---|
| Süreç | Swagger + çok handler okuma | İlgili handler→service→repo |
| Token | Yüksek | **~%40–60 azalma** (tahmin / pilot hipotezi; ölçülmemiş) |
| Kalite | Stale doc riski | Artar (canlı kod odaklı) |

### C) Auth veya geçmiş tarihli job (yüksek risk)

| | Mevcut (tipik) | Optimize |
|---|---|---|
| Süreç | Tek turda kod, ADMIN varsayımı | Plan + role matrix + capabilities sync |
| Token | Orta | **~%20–40 azalma** (tahmin / pilot hipotezi; ölçülmemiş) |
| Güvenlik | Riskli | **Artar** |

---

## 11. Hızlı kazanımlar

- [x] `AGENTS.md` (her iki repo; hierarchy, locale/tz, WIP, validation gaps)
- [x] `.cursorignore` (her iki repo; Cursor)
- [x] `.geminiignore` (her iki repo; Gemini)
- [x] `CLAUDE.md`, `GEMINI.md`, `.github/copilot-instructions.md`
- [x] Backend `.agent/instructions.md` ince adaptör
- [x] `docs/AI_CODING_GUIDE.md` (koşullu on-demand)
- [x] Bu rapor (analiz/ölçüm; runtime değil)

---

## 12. Orta vadeli öneriler

- Stale doküman güncellemesi (`README.md`, `project_analysis.md`, `API_DOCUMENTATION.md`)
- FE `server` ve `.build-log.txt` için `.gitignore` güncellemesi (ayrı task)
- Historical root markdown arşiv klasörü
- Org-level Copilot content exclusion (admin)
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
| AGENTS yüklenmemesi | Context'e ekle / kısa adaptör |
| Ignore'u güvenlik sanma | Discovery filter; explicit read / org deny ayrı |
| Çift yükleme (multi-root / adaptör+AGENTS) | Kabul et veya tek-repo chat; ölç |
| Test kapsamını aşırı azaltma | Yüksek riskte test azaltılmaz |

---

## 14. Uygulama roadmap'i

```
Başlangıç → AGENTS.md + Cursor ignore + rehber
2026-07   → Gap doldurma + Claude/Gemini/Copilot adaptörleri + .geminiignore
Pilot     → Ekip prompt şablonu + ölçüm baseline
Sonraki   → Stale docs + gitignore (server binary) + org Copilot exclusion
İleride   → Ölçüm karşılaştırma + doğrulanmış ek araç adaptörleri
```

---

## 15. Pilot hedef aralığı (ölçümle doğrulanacak)

| Metrik | Potansiyel kazanım (hipotez / tahmin) |
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

- `AGENTS.md` — kısa kurallar; normatif SoT (her repo)
- `CLAUDE.md` / `GEMINI.md` / `.github/copilot-instructions.md` — ince adaptörler
- `.agent/instructions.md` — backend agent adaptörü
- `docs/AI_CODING_GUIDE.md` — koşullu workflow
- `docs/AI_TOKEN_OPTIMIZATION.md` — bu rapor (analiz/ölçüm)
- `.cursorignore` — Cursor discovery filter
- `.geminiignore` — Gemini discovery filter
- Frontend repo: aynı SoT + adaptör + ignore deseni; FE-local AI guide yok

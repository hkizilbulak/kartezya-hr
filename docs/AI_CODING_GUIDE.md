# AI Coding Rehberi — Kartezya HR

Araçtan bağımsız pratik rehber. Backend ve frontend ekipleri için ortak workflow; repo farkları açıkça belirtilmiştir.

**Source of truth (kısa kurallar):** Her repo root `AGENTS.md`  
**Bu dosya:** Detaylı workflow, prompt şablonları, anti-patternler

---

## 1. Amaç

AI coding araçlarıyla (Cursor, GitHub Copilot, Claude Code, ChatGPT, Windsurf, CLI agent'ları) çalışırken:

- Gereksiz token harcamadan doğru sonuç almak
- Güvenlik ve mimari kuralları korumak
- Her task'a uygun risk seviyesinde süreç uygulamak

---

## 2. Token neden gereksiz harcanır?

| Neden | Projede görülen örnek |
|---|---|
| Her task'ta stack/auth yeniden anlatma | Repoda kısa `AGENTS.md` yoktu |
| Generated/büyük dosyaların indexlenmesi | Swagger üçlüsü (~873 KB), FE `node_modules` |
| Tüm repo taraması | "Projeyi incele" tarzı belirsiz promptlar |
| Uzun test/terminal çıktısı | Başarılı testlerin tam logu |
| Her task'ta ağır agent süreci | CSS fix için full agent |
| Stale dokümanlara güvenme | README'deki eski ADMIN/EMPLOYEE modeli |
| Uzun chat context taşıma | Önceki task kararlarının taşınması |

---

## 3. Ortak talimat mimarisi

```
AGENTS.md (kısa, her istekte)
    ↓ referans
AI_CODING_GUIDE.md (workflow detayı)
    ↓ referans
Modül dokümanları (ihtiyaç halinde)
    ↓ ince pointer
Araç adaptörleri (.agent/, .cursor/rules/, CLAUDE.md — ileride)
```

**Kural tekrarı:** Normatif ve tam kural metni `AGENTS.md` içindedir. Bu rehber workflow bağlamında kısa özet, açıklama ve prompt şablonları içerebilir. Adaptör dosyaları kuralları tam metin olarak kopyalamamalıdır.

### Stale doküman uyarısı

Auth veya role tasklarında aşağıdaki kaynakları **esas al**:

- Backend: `internal/authz/capabilities.go`, `BACKEND_API_ROLE_MATRIX.md`
- Frontend: `lib/authz/capabilities.ts`

Aşağıdaki dokümanlar **stale olabilir** (eski ADMIN/EMPLOYEE modeli):

- `README.md`
- `docs/project_analysis.md`
- `API_DOCUMENTATION.md` (kısmen)

---

## 4. Context seçimi

### Aç

- Task ile doğrudan ilgili handler/service/repository veya component/service
- Auth tasklarında capability dosyaları (BE + FE)
- Migration tasklarında `schema/` ve `internal/database/`

### Açma (ignore veya arama dışı)

- Generated Swagger (`docs/docs.go`, `swagger.json`, `swagger.yaml`)
- `node_modules/`, `.next/`, `out/`, binary `server`
- `uploads/`, log dosyaları
- Büyük statik assetler (`public/images/`, `public/fonts/`)

### Soft ignore (gerektiğinde aç)

- `go.sum`, `package-lock.json` — dependency tasklarında gerekebilir
- Historical root markdown (DYS summary, half-day debug notları)

### On-demand erişim

Ignore edilen asset veya generated dosyalar varsayılan context'e alınmamalıdır. Logo, font, görsel, Swagger/OpenAPI veya dependency incelemesi gerçekten gerekiyorsa önce hedef dosyayı explicit path/file context ile açmayı dene. Ignore edilen dosyaya erişim araç ve sürüme göre değişebilir; araç ignore kuralını bypass etmiyorsa içeriği kontrollü biçimde context'e ekle veya geçici ve hedefli ignore düzenlemesi yap. Tüm klasörü kalıcı olarak context'e geri ekleme.

### Seçim sırası

1. Önce arama (grep/glob) — sembol veya path ile
2. Yalnızca hit dosyalarını aç
3. Gerekirse bir seviye dependency (import zinciri)

---

## 5. Risk bazlı task sınıflandırması

| Seviye | Örnekler | Plan | Context | Test |
|---|---|---|---|---|
| **Düşük** | CSS, label, typo, basit type | Gerekmez | 1–3 dosya | Lint veya görsel |
| **Orta** | Pagination, filtre, endpoint, form | Kısa | Katman zinciri | Paket test / build |
| **Yüksek** | Auth, capability, migration, delete, job, prod DB | Zorunlu | Authz + schema + servis | Odaklı + checklist |

---

## 6. Küçük task workflow'u

1. Prompt'ta hedef dosya/path belirt
2. Risk: düşük
3. Tek tur: oku → değiştir → doğrula
4. `git diff --check`
5. Rapor: 2–4 madde

**Süre hedefi:** Tek oturum, minimum tool call.

---

## 7. Orta risk workflow'u

1. Kısa keşif (grep → 3–8 dosya)
2. Değişiklik planını 3–5 maddede özetle
3. Uygula (BE: Handler→Service→Repository; FE: page→service→API)
4. İlgili paket testi veya `npm run lint` / `npm run build`
5. Rapor: 4–6 madde

---

## 8. Yüksek risk workflow'u

1. **Read-only analiz** — kod yazma
2. Etki alanı: authz, schema, transaction, job schedule
3. Planı kullanıcıya sun; onay bekle
4. Uygula
5. Odaklı test + güvenlik checklist
6. Rapor: plan özeti + yapılanlar + kalan risk

**Asla:** Tek turda auth/migration değişikliği; stale README'ye güvenme.

---

## 9. Model / agent modu seçim prensipleri

| Task | Mod | Model seviyesi |
|---|---|---|
| CSS/label fix | Chat veya inline | Küçük/hızlı |
| Tek dosya bug | Dar agent | Küçük–orta |
| Yeni endpoint/form | Agent | Orta |
| Auth/capability | Plan → Agent | Güçlü |
| Migration | Plan zorunlu → Agent | Güçlü |
| Repo keşfi | Explore / paralel keşif (araç destekliyorsa) | Ucuz |
| Final review | Review pass | Güçlü |

**Prensipler:**

- Her küçük task'ta en güçlü modeli kullanma
- Auto mode uzun task'larda context şişirebilir; dar prompt ver
- Read-only sorularda agent modu gereksiz

---

## 10. Test ve terminal çıktısı yönetimi

### Backend

```bash
go test ./internal/<paket>/...   # dar kapsam tercih
go test ./...                    # geniş kapsam — yalnızca gerekirse
gofmt -w <dosyalar>
git diff --check
```

### Frontend

```bash
npm run lint
npm run build
git diff --check
```

### Çıktı kuralları

- **Başarı:** `PASS — go test ./internal/authz/...` (tek satır)
- **Hata:** Yalnızca ilgili hata bloğu (stack trace'in tamamı değil)
- Secret/token içeren çıktıyı paylaşma
- Soru-only task'ta varsayılan olarak build/test çalıştırma; kullanıcı açıkça doğrulama isterse yalnız dar kapsamlı ilgili testi çalıştır

---

## 11. Yeni task'ta yeni oturum

- Farklı task için yeni oturum aç; ilgisiz eski chat context'ini taşıma
- Aynı feature devam ediyorsa yalnız kabul edilmiş kararların kısa özetini veya kalıcı docs pointer'ını taşı
- Kalıcı kararları `AGENTS.md` veya ilgili dokümana yaz
- Her oturumda prompt şablonu kullan

---

## 12. Uzun chat context'inden kaçınma

- Aynı dosyayı her tur tekrar okutma
- Analiz ve implementasyon aynı bilgiyi tekrar etmesin
- Final rapor 4–6 madde
- Uzun tablo ve kod dump'ından kaçın; path referansı yeterli

---

## 13. Tool call sayısını azaltma

- Önce sembol/path ile arama, sonra okuma
- Paralel okuma yalnızca bağımsız dosyalar için
- Alt görev / paralel keşif (araç destekliyorsa): geniş keşif için; küçük fix için gereksiz
- Aynı komutu tekrar çalıştırma (cache sonucu kullan)

---

## 14. Prompt şablonları

Her şablonu kopyalayıp `[...]` alanlarını doldur.

### Read-only analiz

```
Repo: [backend / frontend]
Task: [soru veya analiz konusu]
Risk: düşük
Kısıt: Kod değiştirme, commit/push yok.
Context: Yalnız [path veya sembol] ara.
Çıktı: Bulgu listesi + ilgili dosya path'leri.
Rapor: 4–6 madde.
```

### Küçük bug fix

```
Repo: [backend / frontend]
Dosya: [path]
Belirti: [kısa açıklama]
Risk: düşük
Kapsam: Yalnız belirtilen dosya (+ test varsa).
Doğrulama: [go test paket / npm run lint]
Rapor: Ne değişti, test sonucu.
```

### Backend feature

```
Repo: backend
Özellik: [endpoint veya iş kuralı]
Katman: Handler → Service → Repository
Capability: [hangi capability gerekli]
Risk: orta
Context: İlgili handler/service/repo; Swagger varsayılan dışı — contract gerekiyorsa ilgili OpenAPI dosyasını on-demand aç.
Doğrulama: go test ./internal/<paket>/...
FE sync: capability değiştiyse lib/authz kontrol et.
```

### Frontend feature

```
Repo: frontend
Sayfa/component: [path]
API: [endpoint veya service]
Risk: orta
Navigasyon: next/link kullan; button type="button" for toggles.
Doğrulama: npm run lint && npm run build
Task dışı TS hatalarını yeni hata gibi sunma.
```

### DB / migration

```
Repo: backend
Değişiklik: [tablo/index/veri]
Risk: YÜKSEK — önce read-only plan
Prefix: domain.GetTableName kullan; hr_/hr_test_ hardcode yok
AutoMigrate + schema/ etkisini birlikte değerlendir.
Transaction ve rollback senaryosunu belirt.
Onay olmadan uygulama yapma.
```

### Auth / capability

```
Repo: [backend / frontend / her ikisi]
Değişiklik: [capability veya erişim kuralı]
Risk: YÜKSEK — önce plan
Kaynak: capabilities.go + capabilities.ts + BACKEND_API_ROLE_MATRIX.md
Stale README/project_analysis'e güvenme.
UI guard güvenlik sınırı değil.
```

### Merge conflict

```
Repo: [backend / frontend]
Conflict dosyalar: [liste]
Risk: orta
Semantic merge; davranış değişikliğini açıkla.
Doğrulama: dar test + build
Task dışı dosyaya dokunma.
```

### PR öncesi kontrol

```
Repo: [backend / frontend]
Kapsam: diff inceleme
Kontrol: scope dışı değişiklik, secret, debug log, gofmt, test
Risk: orta
Commit/push yalnızca açık talimatla.
Rapor: checklist sonucu.
```

### Documentation-only

```
Repo: [backend / frontend]
Dosya: [path]
Risk: düşük
Kısıt: Yalnız doküman; kod değiştirme yok.
Stale uyarısı gerekiyorsa ekle.
```

### Hızlı soru / inceleme

```
Repo: [backend / frontend]
Soru: [tek soru]
Risk: düşük
Kısıt: Minimum tool call; build/test yok.
Cevap: kısa ve doğrudan.
```

---

## 15. Anti-patternler

| Anti-pattern | Doğrusu |
|---|---|
| "Tüm projeyi incele" | Hedef path/symbol ver |
| Swagger'ı tek kaynak sayma | Davranış: handler/service + role matrix; contract: OpenAPI on-demand |
| README'deki role modeline güvenme | capabilities.go / capabilities.ts |
| Her task'ta `go test ./...` | Paket bazlı test |
| Başarılı testin tam logunu paylaşma | PASS özeti |
| CSS fix için full agent + gereksiz alt görev | Tek tur, dar context |
| Aynı Git kurallarını her prompt'ta yazma | AGENTS.md yeterli |
| `.env` okuma | Asla |
| Task dışı refactor | Yalnızca istenen kapsam |

---

## 16. Ölçüm metrikleri

Task başına kaydet (basit spreadsheet yeterli):

| Metrik | Açıklama |
|---|---|
| Prompt karakter sayısı | Kullanıcı + sistem talimatları |
| Açılan dosya sayısı | Agent tarafından okunan |
| Tool call sayısı | Arama, okuma, terminal |
| Tur sayısı | Chat round-trip |
| Tekrar tur sayısı | Hata sonrası yeniden deneme |
| Test komutu tekrar sayısı | Aynı testin kaç kez çalıştığı |
| Task süresi | Dakika |
| Değişen dosya sayısı | Scope kontrolü |
| İlk seferde başarı | Evet/hayır |

5–10 gerçek task ile baseline al; optimizasyon sonrası karşılaştır.

---

## 17. Backend ve frontend farkları

| Konu | Backend | Frontend |
|---|---|---|
| Test | `go test ./internal/<paket>/...` | `npm run lint`, `npm run build` |
| Mimari | Handler→Service→Repository | app/ → services/ → API |
| Auth kaynağı | `internal/authz/capabilities.go` | `lib/authz/capabilities.ts` |
| DB | GORM, schema/, prefix kuralı | Yok (API üzerinden) |
| Ignore odak | Swagger, uploads, binary | node_modules, .next, server, assets |
| Generated | Swagger trio | next-env.d.ts, build output |
| Navigasyon | — | next/link, button type="button" |

---

## Follow-up: güncellenmesi gereken stale dokümanlar

Ayrı task olarak ele alınmalı (bu rehberde güncellenmedi):

- `README.md` — capability modeli ve rol listesi
- `docs/project_analysis.md` — domain/rol bölümü
- `API_DOCUMENTATION.md` — auth bölümü (kısmen)

---

## İleride eklenebilecek araç adaptörleri

Gereksiz dosya çoğaltmadan, ihtiyaç oldukça:

- `.github/copilot-instructions.md` — AGENTS.md pointer (~30 satır)
- `CLAUDE.md` — AGENTS.md pointer
- `.cursor/rules/*.mdc` — modül bazlı kısa kurallar

Her adaptör yalnızca pointer içermeli; kuralları kopyalamamalı.

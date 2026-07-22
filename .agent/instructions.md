# Agent Instructions (Adapter)

**Source of truth:** Root [`AGENTS.md`](../AGENTS.md) — tüm mimari, güvenlik ve Git kuralları orada.

Bu dosya yalnızca agent araçlarına özel operasyonel notlar içerir. Kuralları burada tekrarlama.

## Operasyonel notlar

- **Plan:** Yüksek riskli, kapsamı belirsiz veya mimari karar gerektiren tasklarda önce read-only plan yap; onay sonrası uygula. Kapsamı net düşük/orta riskli bug fixlerde doğrudan ilerle.
- **Context:** Araştırma ve geniş taramayı daralt; yalnızca ilgili dosyaları aç.
- **Subagent (destekleniyorsa):** Büyük keşif işlerini alt görevlere böl; ana context penceresini temiz tut.
- **Doğrulama:** Task bitmeden ilgili test/lint çalıştır; başarıda kısa PASS özeti ver.
- **Basitlik:** Obvious fix'lerde over-engineering yapma; kök nedeni bul, geçici yama bırakma.
- **Otonom bug fix:** Hata raporunda log/test çıktısını okuyup çöz; gereksiz kullanıcı onayı isteme.

## Detaylı rehber

Workflow, prompt şablonları ve risk sınıflandırması için: [`docs/AI_CODING_GUIDE.md`](../docs/AI_CODING_GUIDE.md)

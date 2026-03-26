# Kartezya HR — İş Süreçleri Analizi

> Son güncelleme: 2026-03-25

---

## Sistemin Amacı

Kartezya HR, bir şirketin (veya birden fazla şirketin) **insan kaynakları süreçlerini merkezi olarak yönetmesi** için tasarlanmış bir arka uç sistemidir. İki temel kullanıcı tipi vardır:

- **Admin (İK/Yönetici):** Tüm çalışanları yönetir, izin onaylar/reddeder, raporlar alır.
- **Employee (Çalışan):** Kendi bilgilerini görür, izin talep eder, bakiyesini takip eder.

---

## Temel İş Alanları

### 1. 👤 Çalışan Yönetimi

**Ne yapar?**
Bir kişiyi sisteme iş gören olarak kaydetme ve yaşam döngüsünü yönetme sürecidir.

**Bir çalışan sisteme eklendiğinde neler olur?**
1. Kurumsal e-posta ile bir **sistem hesabı** otomatik oluşturulur
2. Çalışanın kişisel bilgileri kaydedilir (ad, soyad, TC kimlik no, doğum tarihi, medeni durum, vb.)
3. Çalışana **rol** atanır (ADMIN veya EMPLOYEE)
4. Sisteme giriş için **şifre sıfırlama e-postası** otomatik gönderilir
5. Kimin oluşturduğu **audit loguna** kaydedilir

**Çalışan kaydında tutulan bilgiler:**
- Kişisel: Ad, soyad, cinsiyet, doğum tarihi, medeni durum, uyruk, TC kimlik no
- İletişim: Kişisel e-posta, şirket e-postası, telefon, adres, il, ilçe
- İş bilgisi: İşe giriş tarihi, ayrılış tarihi, mesleğe başlangıç tarihi, toplam boşluk süresi
- Acil durum: Acil iletişim kişisi adı, telefonu, yakınlık derecesi
- Diğer: Anne adı, baba adı, sözleşme numarası, notlar

**Çalışanın kendi güncelleyebildikleri (kısıtlı erişim):**
Kişisel e-posta, telefon, adres, cinsiyet, doğum tarihi, medeni durum, acil iletişim bilgileri, anne/baba adı, uyruk, kimlik no

**Adminlerin ek olarak güncelleyebildikleri:**
İşe giriş/çıkış tarihleri, sözleşme no, grade, statü (ACTIVE/PASSIVE), rol ataması

---

### 2. 🏢 Organizasyon Yapısı

**Ne yapar?**
Şirketin organizasyon hiyerarşisini tanımlar. Çalışanların hangi şirkette, hangi departmanda, hangi pozisyonda çalıştığını takip eder.

**Hiyerarşi:**
```
Şirket (Company)
  └── Departman (Department)  →  Yönetici bilgisi de tutulur
        └── Pozisyon (Job Position)  →  Çalışan çalışma geçmişiyle eşleşir
```

**Çalışma Bilgisi (Work Information):**
Bir çalışanın kariyer geçmişini temsil eder. Her kayıt bir çalışanın belirli bir dönemde hangi şirkette, departmanda, pozisyonda çalıştığını gösterir.
- Birden fazla kayıt olabilir (pozisyon değişikliği, şirket transferi)
- Aktif pozisyon: en güncel tarihli kayıt
- Her kayıtta personel numarası ve iş e-postası da tutulur

---

### 3. 🏖️ İzin Yönetimi

Sistemin en kapsamlı iş alanıdır. Birden fazla izin türü, bakiye takibi ve onay akışı içerir.

#### İzin Türleri

Admin tarafından tanımlanabilen esnek bir yapı vardır. Her izin türünün şu özellikleri vardır:

| Özellik | Açıklama |
|---|---|
| **Ücretli mi?** | Ödenen/ödenmeyen ayrımı |
| **Limit var mı?** | Yıllık maksimum gün sayısı (örn: 5 gün) |
| **Tahakkuk mu?** | Zamanla biriken bir izin mi? (gen. yıllık izin) |
| **Belge gerekli mi?** | Rapor/belge zorunluluğu (gen. hastalık izni) |

**Özel iş kuralları:**
- **Doğum Günü İzni:** Yılda en fazla 1 kez, en fazla 1 gün kullanılabilir
- **Limitli İzinler:** Yıl içinde yalnızca 1 kez giriş yapılabilir (örn. evlilik izni)
- **Yıllık İzin:** Onaylandığında bakiyeden otomatik düşülür; iptal edilirse geri ias edilir

#### İzin Talebi Akışı

```
Çalışan talep oluşturur
         │
         ▼
    [PENDING]  ◄─── Güncelleme / İptal (çalışan)
         │
    ┌────┴────┐
    ▼         ▼
[APPROVED] [REJECTED]
    │           │
    │     Red gerekçesi yazılır
    │
Çalışan iptal edebilir?
→ Hayır (sadece admin iptal edebilir, bakiye iade edilir)
```

**İzin oluştururken yapılan kontroller:**
1. Aynı tarih aralığında bekleyen bekleme talebi var mı?
2. Doğum günü izniyse yıl içinde daha önce alınmış mı?
3. Limitli bir izinse yıl içinde daha önce alınmış mı?
4. İzin bakiyesi yeterli mi? (Admin için bu kontrol atlanır)
5. Başlangıç tarihi geçmişte mi?

**Yarım gün desteği:**
Her talep, başlangıç ve bitiş günleri için tam gün / yarım gün seçeneği sunar. Çalışma günü hesaplaması buna göre yapılır.

#### İzin Bakiyesi

Her çalışan için izin türü bazında bakiye tutulur:
- **Toplam gün:** Hak edilen toplam
- **Kullanılan gün:** Onaylanan taleplerde düşülen
- **Kalan gün:** Anlık bakiye

Yıllık izin için sistem, onay anında bakiyeden otomatik düşer ve iptalda iade eder.

#### Çalışma Günü Hesaplama

Bir izin talebinin kaç iş günü sürdüğünü hesaplamak için:
- Hafta sonları çıkarılır
- Resmi tatil günleri çıkarılır
- Yarım günler 0.5 olarak hesaplanır

---

### 4. 📊 Grade (Derece/Seviye) Sistemi

**Ne yapar?**
Çalışanların kariyer seviyelerini ve bu seviyelerin zaman içindeki değişimini yönetir.

**Grade tanımı:**
- İsim ve açıklama
- Minimum ve maksimum yıl aralığı (örn: 2-5 yıl deneyim için "Mid-Level")

**Çalışan Grade Geçmişi:**
Bir çalışanın hangi grade'de ne zaman başladığını ve bitirdiğini tutar. Kariyer ilerlemesinin historical kaydı.

---

### 5. 📄 Sözleşme Takibi

Her çalışanın bir veya birden fazla iş sözleşmesi olabilir. Her sözleşme için:
- Başlangıç tarihi
- Bitiş tarihi (açık uçlu sözleşmelerde boş)
- Sözleşme numarası

---

### 6. 📈 Dashboard & Raporlama

**Dashboard verileri (Admin görmez):**
- Toplam çalışan sayısı, aktif izin sayısı, departman sayısı
- Çalışanların cinsiyete göre dağılımı
- Çalışanların pozisyona göre dağılımı
- Çalışanların şirket-departman bazında dağılımı
- Çalışanların grade'e göre dağılımı

**Raporlar:**

| Rapor | Ne gösterir? |
|---|---|
| **Çalışma Günü Raporu** | Belirli bir dönemde çalışan bazında toplam çalışma günleri, kullanılan izin günleri ve tatil günleri |
| **Grade Raporu** | Şirket/departman bazında çalışanların grade dağılımı |

---

### 7. 🔐 Kullanıcı Yönetimi & Giriş

**Giriş yöntemleri:**
1. **E-posta + Şifre:** Standart JWT tabanlı giriş
2. **Yandex OAuth:** Kurumsal Google/Yandex hesabıyla tek tıkla giriş

**Şifre yönetimi:**
- Yeni çalışan eklendiğinde şifre sıfırlama e-postası otomatik gönderilir
- Toplu şifre sıfırlama e-postası gönderilebilir (tüm çalışanlara)
- Token süreli doğrulama ile güvenli sıfırlama

---

## Kullanıcı Tiplerine Göre Yetkiler

| İşlem | Çalışan | Admin |
|---|---|---|
| Kendi profilini görme | ✅ | ✅ |
| Kendi profilini güncelleme (kısıtlı) | ✅ | ✅ |
| Kendi izin taleplerini görme | ✅ | ✅ |
| İzin talebi oluşturma | ✅ | ✅ |
| Kendi bakiyesini görme | ✅ | ✅ |
| Kendi iş bilgisini görme | ✅ | ✅ |
| Kendi grade geçmişini görme | ✅ | ✅ |
| Kendi sözleşmelerini görme | ✅ | ✅ |
| Tüm çalışanları listeleme | ❌ | ✅ |
| Çalışan ekleme/silme | ❌ | ✅ |
| İzin onaylama/reddetme | ❌ | ✅ |
| Onaylı izni iptal etme | ❌ | ✅ |
| İzin türü tanımlama | ❌ | ✅ |
| Şirket/departman yönetimi | ❌ | ✅ |
| Grade tanımlama | ❌ | ✅ |
| Raporlara erişim | ❌ | ✅ |

---

## İş Kuralları Özeti

> Bunlar sistemin içinde kodlanmış iş kararlarıdır.

1. Her şirket e-postası **benzersiz** olmalı — aynı e-posta ile iki aktif çalışan olamaz
2. TC kimlik numarası ve telefon no da **benzersiz** kontrol edilir
3. **Sadece PENDING** durumundaki talepler güncellenebilir
4. Çalışanlar **sadece kendi** bekleyen taleplerini iptal edebilir
5. Admin, onaylı bir talebi iptal ederse **bakiye otomatik iade** edilir
6. Yıllık izin dışındaki izin türleri için **bakiye düşümü yapılmaz** (sadece rapor/bilgi amaçlı)
7. **Doğum günü izni:** yılda 1 kez, max 1 gün
8. **Limitli izinler:** yılda 1 kez giriş yapılabilir
9. Çalışanın izin talebinde tarih geçmişte olamaz
10. Sistem birden fazla **şirketi** destekler — multi-tenant benzeri yapı

---

## Verinin Yaşam Döngüsü

```
Çalışan eklenir
   → Sistem hesabı oluşur
   → Şifre sıfırlama e-postası gider
   → Çalışan sisteme giriş yapar
   → İzin talebi oluşturur
   → Admin onaylar → Bakiye düşülür
   → Dönem sonunda yıllık izin cron job ile güncellenir
   → Çalışan ayrılırsa LeaveDate ve Status=PASSIVE olarak işaretlenir
   → Tüm bu değişiklikler audit log'a kaydedilir
```

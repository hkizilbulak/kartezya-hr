# Yandex OAuth Entegrasyonu

Bu döküman, Kartezya HR uygulamasına Yandex OAuth login entegrasyonunun nasıl yapıldığını ve nasıl kullanılacağını açıklar.

## Yapılan Değişiklikler

### 1. Konfigürasyon (internal/config/config.go)
- `OAuthConfig` struct'ı eklendi
- Yandex Client ID, Client Secret ve Redirect URL konfigürasyonları eklendi
- Ortam değişkenleri ile yapılandırma desteği

### 2. Auth Service (internal/service/auth_service.go)
Yeni metodlar eklendi:
- `GetYandexOAuthConfig()` - Yandex OAuth yapılandırmasını döndürür
- `HandleYandexCallback(code string)` - Yandex'ten gelen callback'i işler ve kullanıcı girişini tamamlar

**Özellikler:**
- Kullanıcı ilk kez Yandex ile giriş yapıyorsa otomatik olarak sisteme kaydedilir
- Var olan kullanıcılar email adresi ile eşleştirilir
- JWT token üretimi ve kullanıcı rolleri yönetimi
- Audit log kaydı

### 3. Auth Handler (internal/handler/auth_handler.go)
Yeni endpoint handler'ları eklendi:
- `YandexLogin` - Kullanıcıyı Yandex OAuth sayfasına yönlendirir
- `YandexCallback` - Yandex'ten dönen callback'i işler

### 4. Routes (main.go)
Yeni public route'lar eklendi:
```go
GET  /api/v1/auth/yandex/login     - Yandex login başlat
GET  /api/v1/auth/yandex/callback  - Yandex callback
```

### 5. Bağımlılıklar
- `golang.org/x/oauth2` paketi eklendi

## Ortam Değişkenleri

`.env` dosyanıza aşağıdaki değişkenleri ekleyin:

```bash
# OAuth Configuration
YANDEX_CLIENT_ID=eff28f055726491d86b6d64bbbbdc484
YANDEX_CLIENT_SECRET=28745c7665dc4f6790584d93bd40c04a
YANDEX_REDIRECT_URL=http://localhost:8080/api/v1/auth/yandex/callback
```

**Not:** Production ortamında `YANDEX_REDIRECT_URL` değerini production URL'inize göre güncelleyin.

## Kullanım

### 1. Frontend'den Kullanım

Frontend uygulamanızda bir "Yandex ile Giriş Yap" butonu oluşturun:

```html
<a href="http://localhost:8080/api/v1/auth/yandex/login">
  Yandex ile Giriş Yap
</a>
```

veya JavaScript ile:

```javascript
function loginWithYandex() {
  window.location.href = 'http://localhost:8080/api/v1/auth/yandex/login';
}
```

### 2. OAuth Akışı

1. Kullanıcı "Yandex ile Giriş Yap" butonuna tıklar
2. Kullanıcı Yandex OAuth sayfasına yönlendirilir
3. Kullanıcı Yandex hesabı ile giriş yapar ve uygulamaya izin verir
4. Yandex, kullanıcıyı callback URL'e yönlendirir
5. Backend, callback'i işler ve JWT token üretir
6. Response'ta token ve kullanıcı bilgileri döner

### 3. Response Formatı

Başarılı giriş sonrası dönen response:

```json
{
  "success": true,
  "message": "Yandex login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2026-02-06T10:30:00Z",
    "user": {
      "id": 1,
      "email": "user@yandex.com",
      "firstName": "John",
      "lastName": "Doe",
      "roles": ["USER"]
    }
  }
}
```

### 4. Callback Handling (Frontend)

Frontend uygulamanızda callback'i yakalamak için:

```javascript
// React örneği
useEffect(() => {
  const urlParams = new URLSearchParams(window.location.search);
  const code = urlParams.get('code');
  
  if (code) {
    // Backend callback endpoint'i otomatik olarak işleyecek
    fetch(`http://localhost:8080/api/v1/auth/yandex/callback?code=${code}`)
      .then(response => response.json())
      .then(data => {
        if (data.success) {
          // Token'ı kaydet
          localStorage.setItem('token', data.data.token);
          // Kullanıcıyı yönlendir
          window.location.href = '/dashboard';
        }
      });
  }
}, []);
```

## API Endpoints

### 1. Yandex Login Başlat
```
GET /api/v1/auth/yandex/login
```

**Response:** 302 Redirect to Yandex OAuth

### 2. Yandex Callback
```
GET /api/v1/auth/yandex/callback?code={authorization_code}&state={state}
```

**Query Parameters:**
- `code` (required): Yandex'ten alınan authorization code
- `state` (optional): CSRF koruması için state parametresi

**Response:**
```json
{
  "success": true,
  "message": "Yandex login successful",
  "data": {
    "token": "string",
    "expires_at": "timestamp",
    "user": {
      "id": "number",
      "email": "string",
      "firstName": "string",
      "lastName": "string",
      "roles": ["string"]
    }
  }
}
```

## Yandex OAuth Konfigürasyonu

Yandex Developer Console'da aşağıdaki ayarları yapın:

1. https://oauth.yandex.com/ adresine gidin
2. Yeni bir uygulama oluşturun veya mevcut uygulamanızı seçin
3. **Callback URL** olarak ekleyin:
   - Development: `http://localhost:8080/api/v1/auth/yandex/callback`
   - Production: `https://yourdomain.com/api/v1/auth/yandex/callback`
4. **Permissions** (İzinler):
   - `login:email` - Email adresi okuma izni
   - `login:info` - Kullanıcı bilgileri okuma izni

## Güvenlik Notları

1. **HTTPS Kullanımı:** Production ortamında mutlaka HTTPS kullanın
2. **State Parameter:** CSRF saldırılarına karşı state parametresi kullanımı önerilir
3. **Redirect URL Doğrulama:** Callback URL'lerini Yandex console'da doğru şekilde yapılandırın
4. **Secret Güvenliği:** `YANDEX_CLIENT_SECRET` değerini asla frontend'e göndirmeyin
5. **Token Yönetimi:** JWT token'ları güvenli bir şekilde saklayın (HttpOnly cookies önerilir)

## Test Etme

1. Uygulamayı başlatın:
```bash
go run main.go
```

2. Browser'da şu URL'e gidin:
```
http://localhost:8080/api/v1/auth/yandex/login
```

3. Yandex hesabınız ile giriş yapın

4. Başarılı olursa callback response'unu göreceksiniz

## Sorun Giderme

### "Invalid client" hatası
- `YANDEX_CLIENT_ID` ve `YANDEX_CLIENT_SECRET` değerlerini kontrol edin
- Yandex Developer Console'da uygulamanızın aktif olduğundan emin olun

### "Redirect URI mismatch" hatası
- `YANDEX_REDIRECT_URL` değerinin Yandex console'daki URL ile tam olarak eşleştiğinden emin olun
- Protocol (http/https), domain, port ve path'in birebir aynı olması gerekir

### "Email not provided" hatası
- Yandex uygulamanızda `login:email` izninin verildiğinden emin olun
- Kullanıcının Yandex hesabında email adresi olmalı

## Gelecek Geliştirmeler

- [ ] State parameter ile CSRF koruması
- [ ] Token refresh mekanizması
- [ ] Diğer OAuth provider'ları (Google, GitHub, vb.)
- [ ] SSO (Single Sign-On) desteği
- [ ] Account linking (Mevcut hesaba OAuth hesabı bağlama)

## İletişim

Sorularınız için: support@kartezya.com

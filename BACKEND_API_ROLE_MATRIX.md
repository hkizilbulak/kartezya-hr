# Backend API Role Access Guide

Roller login sırasında backend tarafından veritabanından okunur ve JWT içerisine yazılır. Backend, API erişimini JWT içerisindeki rollerden hesaplanan capability’ler ve ownership kontrolleriyle belirler.

## Kullanılan ifadeler

| İfade | Anlamı |
|---|---|
| Public | Giriş yapmadan kullanılabilir |
| Tümü | Rol bütün ilgili kayıtlar üzerinde işlem yapabilir |
| Kendi | Rol yalnızca kendi kaydı üzerinde işlem yapabilir |
| Görüntüleme | Liste ve detay erişimi vardır, değiştirme yetkisi yoktur |
| Yok | Rolün bu işlem için erişimi yoktur |
| Owner veya Management | Kayıt sahibi ya da ilgili management capability sahibi erişebilir |

---

## 1. Ortak Self-Service API’leri

Tüm roller için aynı: yalnızca kendi kaydı.

| Alan | Örnek backend API | Erişim |
|---|---|---|
| Profil görüntüleme/güncelleme | `GET/PUT /api/v1/employees/me` | Kendi |
| Çalışma bilgisi | `GET /api/v1/work-information/me` | Kendi |
| Sözleşme | `GET /api/v1/employee-contracts/me` | Kendi |
| Derece | `GET /api/v1/employee-grades/me` | Kendi |
| İzin talepleri / bakiye | `GET .../leave/requests/me`, `.../balances/me` | Kendi |
| İzin oluşturma | `POST /api/v1/leave/requests` | Kendi |
| Masraf talepleri | `GET /api/v1/expense/requests/me` | Kendi |
| Masraf oluşturma | `POST /api/v1/expense/requests` | Kendi |
| Diğer talepler | `GET /api/v1/other-requests/me` | Kendi |
| Belgeler | `GET /api/v1/documents/me` | Kendi |
| Şifre değiştirme | `POST /api/v1/auth/change-password` | Kendi |

Leave ve expense create: client farklı `employee_id` gönderse bile backend caller employee ID kullanır. Tüm roller yalnızca kendileri adına oluşturabilir.

Diğer ortak: Login / şifre sıfırlama / Yandex (**Public**); Logout, settings, KVKK, FAQ okuma, event portalı, **Dashboard** (**JWT** — tüm giriş yapmış kullanıcılar).

---

## 2. ADMIN

| API alanı | Yapabildiği işlemler | Örnek backend API | Backend kontrolü |
|---|---|---|---|
| Employees | Liste, detay, create/update/delete | `/api/v1/employees` | `CanViewEmployees` / `CanManageEmployees` |
| Work / contracts / grades | Yönetim + tanımlar | `/work-information`, `/employee-contracts`, `/grades`, `/contracts` | `CanManageEmployees` / `CanAccessAdminModules` |
| Leave management | Liste, detay, onay, red | `/api/v1/leave/requests` | `CanViewLeaveManagement` / `CanApproveLeave` |
| Leave update/cancel | Kendi veya tümü | `PUT/POST .../leave/requests/:id...` | Owner veya `CanApproveLeave` |
| Leave documents | Kendi veya tümü | `/api/v1/leave/.../documents` | Owner veya `CanViewLeaveManagement` |
| Leave types / balances | Tip CRUD; tüm bakiyeler | `/leave/types`, `/leave/balances` | `CanManageLeaveTypes` |
| Expense management | Liste, detay | `/api/v1/expense/requests` | `CanViewExpenseManagement` |
| Expense approval | Onay / red | `.../approve`, `.../reject` | `CanApproveExpense` |
| Expense payment | Ödeme | `.../pay` | `CanPayExpense` |
| Expense documents | Kendi veya tümü | `/api/v1/expense/.../documents` | Owner veya `CanViewExpenseManagement` |
| Expense delete | Kendi + başkasının pending | `DELETE .../expense/requests/:id` | Ownership veya **ADMIN role-string** |
| Expense / request types | Tip CRUD | `/expense/types`, `/request-types` | `CanManageExpenseTypes` / `CanManageRequestTypes` |
| Other request management | Liste, complete, rollback + belgeler | `/api/v1/other-requests` | `CanManageOtherRequests` |
| Personnel documents | Başka kullanıcı belgeleri | `GET /api/v1/documents/user/:id` | Owner veya `CanManageEmployees` |
| Generic documents | Non-personnel ADMIN shortcut | `/api/v1/documents/:id` | Ownership / Cap / **ADMIN role-string** |
| Org definitions | Şirket, departman, pozisyon | `/companies`, `/departments`, `/job-positions` | `CanManageOrgMaster` |
| Reports / Jobs / Mail | Yönetim | `/reports`, `/jobs`, `/emails`, `/mail-configs` | `CanAccessAdminModules` |
| FAQ / Event management | Yazma | `/faqs`, `/events` | `CanAccessAdminModules` |
| Dashboard | Veri | `/api/v1/dashboard/...` | JWT |

ADMIN notları: başka adına leave oluşturamaz; expense **update** owner-only; expense **delete**’te ADMIN role-string ile başkasını silebilir; generic non-personnel doc’ta ADMIN role-string shortcut vardır.

---

## 3. HR

ADMIN ile aynı yönetim alanlarının çoğu (`CanPayExpense` hariç). İzin yönetiminde ADMIN ile eşit.

| API alanı | Yapabildiği işlemler | Örnek backend API | Backend kontrolü |
|---|---|---|---|
| Employees | Liste/detay + create/update/delete | `/api/v1/employees` | `CanViewEmployees` / `CanManageEmployees` |
| Work / contracts / grades | Yönetim + tanımlar | `/work-information`, `/employee-contracts`… | `CanManageEmployees` / `CanAccessAdminModules` |
| Leave management | Liste, detay, update/cancel, onay/red, belgeler | `/api/v1/leave/requests...` | `CanViewLeaveManagement` / `CanApproveLeave` |
| Leave types | Tip CRUD | `/api/v1/leave/types` | `CanManageLeaveTypes` |
| Expense list/detail | Yönetim listesi ve detay | `/api/v1/expense/requests` | `CanViewExpenseManagement` |
| Expense approve/reject | Onay / red | `.../approve`, `.../reject` | `CanApproveExpense` |
| Expense documents / types | Belgeler + tip CRUD | `/expense/.../documents`, `/expense/types` | Owner veya Cap / `CanManageExpenseTypes` |
| Other requests | Yönetim + belgeler | `/api/v1/other-requests` | `CanManageOtherRequests` |
| Personnel documents | Başka kullanıcı | `/api/v1/documents/user/:id` | Owner veya `CanManageEmployees` |
| Org / Reports / Jobs / Mail / FAQ / Events | Tanım ve admin modüller | `/companies`, `/reports`… | `CanManageOrgMaster` / `CanAccessAdminModules` |
| Dashboard | Veri | `/api/v1/dashboard/...` | JWT |

### HR — Yapamadıkları

| İşlem | Neden |
|---|---|
| Masraf ödeme | `CanPayExpense` yok |
| ADMIN rolündeki çalışanı update/delete | Hedef koruması |
| Başka adına leave oluşturma | Create her zaman caller employee |
| Başka çalışanın expense update’i | Expense update owner-only |
| Expense delete elevated | ADMIN role-string yok; yalnızca kendi |
| Generic non-personnel ADMIN shortcut | `hasRole(ADMIN)` yok |

---

## 4. FINANCE

### Erişebildiği API’ler

| API alanı | Yapabildiği işlemler | Örnek backend API | Backend kontrolü |
|---|---|---|---|
| Self-service | Bölüm 1’deki kişisel API’ler | `/me`, leave/expense create | Ownership |
| Employee list/detail | Görüntüleme | `GET /api/v1/employees` | `CanViewEmployees` |
| Expense management | Liste + detay (tümü) | `/api/v1/expense/requests` | `CanViewExpenseManagement` |
| Expense update/delete | Yalnızca kendi | `PUT/DELETE .../requests/:id` | Ownership |
| Expense payment | Ödeme | `POST .../pay` | `CanPayExpense` |
| Expense documents | Tümü | `/api/v1/expense/.../documents` | Owner veya `CanViewExpenseManagement` |
| Expense types | Tip CRUD | `/api/v1/expense/types` | `CanManageExpenseTypes` |
| Dashboard / FAQ / Event portal | Okuma / katılım | `/dashboard`, `/faqs`, `/events/dashboard` | JWT |

#### Masraf işlemleri (FINANCE)

| Masraf işlemi | FINANCE erişimi | Backend kontrolü |
|---|---|---|
| Masraf listesi | Tümü | `CanViewExpenseManagement` |
| Masraf detayı | Tümü | Owner veya `CanViewExpenseManagement` |
| Masraf oluşturma | Kendi | Ownership |
| Masraf güncelleme | Kendi | Ownership |
| Masraf silme | Kendi | Ownership; ADMIN role-string geçerli değil |
| Masraf onaylama | Yok | `CanApproveExpense` yok |
| Masraf reddetme | Yok | `CanApproveExpense` yok |
| Masraf ödeme | Tümü | `CanPayExpense` |
| Masraf belgeleri | Tümü | Owner veya `CanViewExpenseManagement` |
| Masraf tipleri | Tümü | `CanManageExpenseTypes` |

### Erişemediği API’ler

| API alanı | Sonuç | Neden |
|---|---|---|
| Employee create/update/delete | Yok | `CanManageEmployees` yok |
| Personnel documents | Yok | `CanManageEmployees` gerekir |
| Leave management / approve / reject | Yok | Leave Cap’leri yok |
| Other request management | Yok | `CanManageOtherRequests` yok |
| Organization definitions | Yok | `CanManageOrgMaster` yok |
| Reports / Jobs / Mail | Yok | `CanAccessAdminModules` yok |
| Expense approve / reject | Yok | `CanApproveExpense` yok |

`CanViewEmployees` çalışan listesi ve detayını görüntülemeye yeterlidir. Personel belgeleri için `CanManageEmployees` gerektiğinden FINANCE personel belgelerine erişemez.

---

## 5. EMPLOYEE

Management capability listesi boştur (`RoleCapabilities[EMPLOYEE] = []`). Self-service JWT + ownership ile yürür.

### Erişebildiği API’ler

| API alanı | Erişim |
|---|---|
| Kendi profil, çalışma bilgisi, sözleşme, derece | Kendi |
| Kendi izin / masraf / diğer talepler ve belgeleri | Kendi |
| Kendi belgeleri | Kendi |
| Dashboard / FAQ okuma / Event portal | JWT (Var) |

### Erişemediği API’ler

| API alanı | Sonuç |
|---|---|
| Employee / Leave / Expense / Other management | Yok |
| Definitions, Reports, Jobs, Mail, admin FAQ/event | Yok |
| Başka kullanıcıların belgeleri | Yok |

---

## 6. Kritik Rol Farkları

| İşlem | ADMIN | HR | FINANCE | EMPLOYEE |
|---|---|---|---|---|
| Çalışan görüntüleme | Tümü | Tümü | Tümü | Yok |
| Çalışan yönetme | Tümü | ADMIN hedef hariç | Yok | Yok |
| İzin yönetimi | Tümü | Tümü | Yok | Yok |
| İzin oluşturma | Kendi | Kendi | Kendi | Kendi |
| Masraf görüntüleme | Tümü | Tümü | Tümü | Kendi |
| Masraf onaylama | Tümü | Tümü | Yok | Yok |
| Masraf ödeme | Tümü | Yok | Tümü | Yok |
| Masraf güncelleme | Kendi | Kendi | Kendi | Kendi |
| Masraf silme | Tümü (role-string) | Kendi | Kendi | Kendi |
| Diğer talep yönetimi | Tümü | Tümü | Yok | Yok |
| Personel belgeleri | Tümü | Tümü | Yok | Kendi |
| Dashboard | Var | Var | Var | Var |

---

## 7. Backend Kontrol Türleri

| Kontrol | Açıklama | Örnek |
|---|---|---|
| Public | JWT gerekmez | `/api/v1/auth/login` |
| JWT | Giriş yeterlidir | `/api/v1/dashboard/...` |
| Ownership | Yalnızca kendi kaydı | `/api/v1/employees/me` |
| Capability | Role bağlı yönetim | `CanApproveExpense`, `CanPayExpense` |
| Owner veya Capability | Sahip veya yönetici | Leave / expense / other documents |
| Role-string | Doğrudan ADMIN kontrolü | Expense delete; generic non-personnel docs |

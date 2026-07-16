-- Seed Data for Kartezya HR Management System
-- Created: 2026-01-03
-- Description: Initial data including admin user, roles, leave types, and sample company data

-- ================================================
-- ROLES
-- ================================================

INSERT INTO hr_roles (name, description, created_by, modified_by) VALUES 
('ADMIN', 'Administrator with full system access', 'system', 'system'),
('EMPLOYEE', 'Regular employee with limited access', 'system', 'system'),
('HR', 'HR specialist with HR management access', 'system', 'system'),
('FINANCE', 'Finance specialist with payment access', 'system', 'system');

-- ================================================
-- ADMIN USER (Password: admin123)
-- ================================================

-- Insert admin user (password hash for 'admin123')
INSERT INTO hr_users (email, password, created_by, modified_by) VALUES 
('admin@kartezya.com', '$2a$10$bqNs7T96Cn7DzPXb6FisC.H7v419wfItA93PZsXLzfZ9qiKgvPq1m', 'system', 'system');

-- Assign admin role to admin user
INSERT INTO hr_user_roles (user_id, role_id, created_by, modified_by) VALUES 
(1, 1, 'admin@kartezya.com', 'admin@kartezya.com');

-- ================================================
-- COMPANY & ORGANIZATION DATA
-- ================================================

-- Sample company
INSERT INTO hr_companies (name, address, phone, email, website, created_by, modified_by) VALUES 
('Kartezya Teknoloji', 'FSM Mahallesi Poligon Sok. Buyaka İş Kulesi Kule: 3 Daire: 1-2-3-4 Ümraniye/İstanbul', '0212 123 4567', 'info@kartezya.com', 'http://kartezya.com', 'admin@kartezya.com', 'admin@kartezya.com'),
('Yapıkredi Teknoloji', 'Yapı Kredi Plaza D Blok 34330 Levent - Beşiktaş/İstanbul', '0262 647 10 00', 'yapikredi@yapikredi.hs02.kep.tr', 'https://yapikredi.com.tr', 'admin@kartezya.com', 'admin@kartezya.com'),
('N11', 'Reşitpaşa Mah. Katar Cad. İTÜ Arı Teknokent 3 Binası Blok No: 4 İç Kapı No: 902 Sarıyer/İstanbul', '0850 333 0011', 'n11@hs01.kep.tr', 'https://n11.com', 'admin@kartezya.com', 'admin@kartezya.com'),
('Vodafone', 'Büyükdere Caddesi No:251 34398, Maslak/İstanbul', '0850 542 00 00', 'info@vodafone.com', 'https://vodafone.com.tr', 'admin@kartezya.com', 'admin@kartezya.com');

-- Sample departments
INSERT INTO hr_departments (company_id, name, manager, created_by, modified_by) VALUES 
(1, 'Genel Müdür', NULL,'admin@kartezya.com', 'admin@kartezya.com'),
(1, 'İnsan Kaynakları', 'Kamuran Yılmaz / Cengiz Doğmenç', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'İştirakler Digital Bankacılık', 'Volkan Aydoslu / Ömer Gürarslan','admin@kartezya.com', 'admin@kartezya.com'),
(2, 'İştirakler Ortak Modüller', 'Volkan Aydoslu / Ezgi Açıkgöz', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'İştirakler Faktoring', 'Volkan Aydoslu / Burak Gökhan Sünbül', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'İştirakler Krediler', 'Erhan Sülün / Selçuk Giray Özdamar','admin@kartezya.com', 'admin@kartezya.com'),
(2, 'İştirakler Para Transferi ', 'Erhan Sülün / Ece Emirleroğlu', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'İştirakler Temel Bankacılık', 'Erhan Sülün / Tuğçe Ayvaz','admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Yatırım Sermaye Piyasası Kanalları', 'Recep Yağcı / Hüseyin Alaca', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Yatırım Yüksek Frekanslı İşlemler', 'Recep Yağcı / Karmen Kapucu', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Yatırım Ürünleri', 'Recep Yağcı / Çağatay Demiryas', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Ürünleri Teminatlı Bireysel Kredi Ürünleri ', 'Hüseyin Yeşil / Türkan Hamzaoğlu Toplan', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Ürünleri Bireysel Kredi Ürünleri', 'Hüseyin Yeşil / Çağla Yılmaz', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Ürünleri Esnek Bireysel Kredi Ürünleri', 'Hüseyin Yeşil / Emel Besnili', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Ürünleri Esnek Ticari Kredi Ürünleri', 'Hüseyin Yeşil / Merve Kirman Bozali', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler TÜrünleri aksitli Ticari Kredi Ürünleri', 'Hüseyin Yeşil / Fahrettin Yanbol', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsis Tüzel Otomatik Tahsis', 'Muhammet Turan / Emine Aksakal', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsis Bireysel Krediler Tahsis', 'Muhammet Turan / Memet Önder', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsis Ticari Krediler Tahsis', 'Muhammet Turan / Beril Ekin Okumuş', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsis Ürün Satış', 'Muhammet Turan / Tolga Kazanoğlu', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsis Krediler İstihbarat', 'Muhammet Turan / Melek Çelik', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsis Krediler Analitik Sistemler', 'Muhammet Turan / Gökhan Yetgin', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsis Krediler Analitik Sistemler', 'Muhammet Turan / Gökhan Yetgin', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsilat ve Tasfiye Risk ve Risk Merkezi', 'Selma Yiğitaslan / Feyzullah Yücel', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsilat ve Tasfiye Krediler Risk Takip', 'Selma Yiğitaslan / Gizem Yılmaz', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsilat ve Tasfiye Krediler Tahsilat', 'Selma Yiğitaslan / Mehmet Can Gayretli', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsilat ve Tasfiye Krediler Tasfiye ve İdari Takip', 'Selma Yiğitaslan / Çağdaş Turhan', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsilat ve Tasfiye Yasal Raporlamalar', 'Selma Yiğitaslan / Esra Başpınar Köse', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Krediler Tahsilat ve Tasfiye Krediler Risk İzleme', 'Selma Yiğitaslan / Musa Aydın', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Kurumsal Uygulamalar İnsan Kaynakları', 'Güçlü Borhan / Hamdi Öztürk', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Kurumsal Uygulamalar Gider Bütçe Yönetimi', 'Güçlü Borhan / Sancar Özer', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Kurumsal Uygulamalar Digital İK, Çalışan Deneyimi', 'Güçlü Borhan / Gizem Akbey', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Kurumsal Uygulamalar Denetim,Uyum ve İç Kontrol', 'Güçlü Borhan / Mustafa Gül', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Finansal Teknolojiler Digital Varlık Alım Satım', 'Engin Ertilav / Mustafa Topçu', 'admin@kartezya.com', 'admin@kartezya.com'),
(2, 'Finansal Teknolojiler Digital Varlık Yönetimi', 'Engin Ertilav / Yahya Caner Aksakal', 'admin@kartezya.com', 'admin@kartezya.com'),
(3, 'Ödemeler', 'Soner Üstel / Ali Kemal Taşçı', 'admin@kartezya.com', 'admin@kartezya.com');

-- Sample job positions (no department relationship)
INSERT INTO hr_job_positions (title, created_by, modified_by) VALUES 
('General Manager', 'admin@kartezya.com', 'admin@kartezya.com'),
('HR Specialist', 'admin@kartezya.com', 'admin@kartezya.com'),
('Junior Software Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Middle Software Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Senior Software Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Junior FE Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Middle FE Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Senior FE Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Junior IOS Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Middle IOS Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Senior IOS Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Junior Android Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Middle Android Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Senior Android Developer', 'admin@kartezya.com', 'admin@kartezya.com'),
('Architect', 'admin@kartezya.com', 'admin@kartezya.com'),
('Junior Analyst', 'admin@kartezya.com', 'admin@kartezya.com'),
('Middle Analyst', 'admin@kartezya.com', 'admin@kartezya.com'),
('Senior Analyst', 'admin@kartezya.com', 'admin@kartezya.com');

-- ================================================
-- LEAVE TYPES
-- ================================================

-- Leave types with new boolean fields including is_required_document
INSERT INTO hr_leave_types (name, description, is_paid, limit_amount, is_accrual, is_required_document, created_by, modified_by) VALUES 
('Yıllık İzin', 'Mevcut izin bakiyesine, 1 – 5 yıl (5 dahil) arasında çalışanlara 14 gün, 5 – 15 yıl arasında çalışanlara 20 gün, 15 yıl ve üzeri çalışanlara 26 gün her yıl eklenir.', true, 14, true, false, 'admin@kartezya.com', 'admin@kartezya.com'),
('Doğum İzni (Anne)', 'Doğumdan önce 8 hafta, doğumdan sonra 8 hafta olmak üzere 16 hafta izin kullanılabilir. Çoğul doğumlarda +2 hafta daha kullanılabilir.', true, NULL, false, true, 'admin@kartezya.com', 'admin@kartezya.com'),
('Süt İzni', 'Çocuk 1 yaşına gelene kadar günde 1.5 saat izin kullanılabilir.', true, NULL, false, true, 'admin@kartezya.com', 'admin@kartezya.com'),
('Evlenme İzni', 'Evlilik durumunda 3 gün izin kullanılabilir.', true, 3, false, true, 'admin@kartezya.com', 'admin@kartezya.com'),
('Yakın Akraba Ölümü', 'Anne, baba, eş, kardeş, çocuk ölümü durumunda 3 gün izin kullanılabilir.', true, 3, false, true, 'admin@kartezya.com', 'admin@kartezya.com'),
('Doğum İzni (Baba)', 'Doğum sonrası 5 gün izin kullanılabilir.', true, 5, false, true, 'admin@kartezya.com', 'admin@kartezya.com'),
('Askerlik İzni', 'Askere gitme durumunda kullanılabilir. Ücretsiz izindir.', false, NULL, false, true, 'admin@kartezya.com', 'admin@kartezya.com'),
('Doğum Günü İzni', 'Bir takvim yılı içerisinde doğduğun ay içerisinde 1 gün kullanılabilir. Ücretli izindir.', true, 1, false, false, 'admin@kartezya.com', 'admin@kartezya.com'),
('Rapor İzni', 'Rapor dahilinde kullanılabilir. Ücretsiz izindir.', false, NULL, false, true, 'admin@kartezya.com', 'admin@kartezya.com'),
('Diğer', 'Genel ücretsiz izin talepleri için kullanılabilir.', false, NULL, false, false, 'admin@kartezya.com', 'admin@kartezya.com');
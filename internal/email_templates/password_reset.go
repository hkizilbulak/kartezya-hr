package templates

import "strings"

// EmailContent contains both subject and body of the email
type EmailContent struct {
	Subject string
	Body    string
}

// PasswordResetEmailTemplate returns the email subject and HTML template for password reset email
func PasswordResetEmailTemplate(firstName, lastName, resetURL string) EmailContent {
	subject := "Kartezya HR - Şifre Belirleme"

	body := `<!DOCTYPE html>
<html lang="tr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Kartezya HR Sistemi - Şifre Belirleme</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background-color: #f5f7fa;
            color: #333;
            line-height: 1.6;
        }
        
        .email-wrapper {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
        }
        
        .header {
            text-align: center;
            background: #000000;
            background: radial-gradient(circle,rgba(0, 0, 0, 1) 0%, rgba(64, 60, 186, 1) 100%, rgba(7, 51, 149, 1) 71%, rgba(9, 9, 121, 1) 100%);
        }
        
        .header-logo {
            font-size: 24px;
            font-weight: 700;
            margin-bottom: 10px;
            letter-spacing: -0.5px;
        }

        .header-logo img {
            max-width: 200px;
            height: auto;
            display: block;
            margin: 0 auto;
        }
        
        .header-subtitle {
            font-size: 14px;
            opacity: 0.95;
            font-weight: 300;
            display: none;
        }
        
        .content {
            padding: 40px 30px;
        }
        
        .greeting {
            font-size: 20px;
            font-weight: 600;
            color: #1a1a1a;
            margin-bottom: 20px;
            line-height: 1.4;
        }
        
        .intro-text {
            font-size: 15px;
            color: #555;
            margin-bottom: 30px;
            line-height: 1.7;
        }
        
        .benefits-section {
            background-color: #f9fafb;
            border-left: 4px solid #624bff;
            padding: 20px;
            margin: 30px 0;
            border-radius: 4px;
        }
        
        .benefits-title {
            font-size: 16px;
            font-weight: 600;
            color: #1a1a1a;
            margin-bottom: 15px;
        }
        
        .benefits-list {
            list-style: none;
        }
        
        .benefits-list li {
            font-size: 14px;
            color: #555;
            margin-bottom: 12px;
            padding-left: 25px;
            position: relative;
        }
        
        .benefits-list li:before {
            position: absolute;
            left: 0;
            color: #624bff;
            font-weight: bold;
            font-size: 16px;
        }
        
        .action-section {
            background-color: #ffffff;
            padding: 30px;
            border-radius: 8px;
            margin: 30px 0;
            text-align: center;
            border: 2px solid #e5e7eb;
        }
        
        .action-text {
            font-size: 14px;
            color: #555;
            margin-bottom: 20px;
            line-height: 1.6;
        }
        
        .cta-button {
            display: inline-block;
            background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
            color: #ffffff;
            padding: 16px 48px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 700;
            font-size: 16px;
            transition: all 0.3s ease;
            box-shadow: 0 6px 20px rgba(79, 70, 229, 0.4);
            border: none;
            cursor: pointer;
            letter-spacing: 0.5px;
        }
        
        .cta-button:hover {
            transform: translateY(-3px);
            box-shadow: 0 8px 25px rgba(79, 70, 229, 0.5);
        }
        
        .important-note {
            background-color: #fef3c7;
            border-left: 4px solid #f59e0b;
            padding: 15px;
            margin: 25px 0;
            border-radius: 4px;
            font-size: 13px;
            color: #92400e;
            line-height: 1.6;
        }
        
        .support-section {
            background-color: #f9fafb;
            padding: 25px;
            margin: 30px 0 0 0;
            border-top: 1px solid #e5e7eb;
            text-align: center;
            border-radius: 0 0 8px 8px;
        }
        
        .support-title {
            font-size: 14px;
            font-weight: 600;
            color: #1a1a1a;
            margin-bottom: 10px;
        }
        
        .support-text {
            font-size: 13px;
            color: #666;
            margin-bottom: 5px;
        }
        
        .support-link {
            color: #624bff;
            text-decoration: none;
            font-weight: 500;
        }
        
        .support-link:hover {
            text-decoration: underline;
        }
        
        .footer {
            background-color: #f9fafb;
            padding: 25px 30px;
            border-top: 1px solid #e5e7eb;
            text-align: center;
            font-size: 12px;
            color: #999;
            line-height: 1.6;
        }
        
        .footer-links {
            margin: 15px 0;
        }
        
        .footer-links a {
            color: #624bff;
            text-decoration: none;
            margin: 0 10px;
        }
        
        .footer-links a:hover {
            text-decoration: underline;
        }
        
        .divider {
            height: 1px;
            background-color: #e5e7eb;
            margin: 20px 0;
        }
        
        .instructions-list {
            margin: 10px 0 0 20px;
            padding-left: 15px;
            font-size: 14px;
            color: #555;
            line-height: 1.7;
        }
        
        .instructions-list li {
            margin-bottom: 8px;
        }
        
        @media only screen and (max-width: 600px) {
            .email-wrapper {
                width: 100%;
            }
            
            .content {
                padding: 25px 20px;
            }
            
            .header {
                padding: 30px 20px;
            }
            
            .greeting {
                font-size: 18px;
            }
            
            .benefits-section {
                padding: 15px;
                margin: 20px 0;
            }
            
            .action-section {
                padding: 20px;
            }
        }
    </style>
</head>
<body>
    <div class="email-wrapper">
        <!-- Header -->
        <div class="header">
            <div class="header-logo">
                <img src="https://kartezya.com/wp-content/uploads/2025/02/togetherBoyut2.svg" alt="Kartezya Teknoloji Logo">
            </div>
        </div>
        
        <!-- Main Content -->
        <div class="content">
            <!-- Greeting -->
            <div class="greeting">
                Merhaba, USER_FULL_NAME! 👋
            </div>
            
            <!-- Introduction -->
            <div class="intro-text">
                Kartezya Teknoloji HR Portal'ine hoş geldin! İK ile ilgili tüm işlemlerini bu portal üzerinden kolay ve hızlı bir şekilde yapabilirsin.
            </div>
            
            <!-- Action Section -->
            <div class="action-section">
                <div class="action-text">
                    Sisteme erişebilmek için ilk adım olarak bir şifre belirlemen gerekmektedir. Aşağıdaki butona tıklayarak yeni şifreni oluşturabilirsin.
                </div>
                <a href="RESET_URL" class="cta-button">
                    Yeni Şifre Belirle
                </a>
            </div>
            
        </div>
        
        <!-- Support Section -->
        <div class="support-section">
            <div class="support-title">Herhangi bir sorun mu var?</div>
            <div class="support-text">
                Teknik destek için bizimle iletişime geçebilirsin
            </div>
            <div class="support-text">
                ✉️ <a href="mailto:hr@kartezya.com" class="support-link">hr@kartezya.com</a>
            </div>
        </div>
        
        <!-- Footer -->
        <div class="footer">
            © 2026 Kartezya Teknoloji. Tüm hakları saklıdır.
        </div>
    </div>
</body>
</html>`

	// Replace placeholders with actual values
	replacer := strings.NewReplacer(
		"USER_FULL_NAME", firstName+" "+lastName,
		"RESET_URL", resetURL,
	)

	return EmailContent{
		Subject: subject,
		Body:    replacer.Replace(body),
	}
}

package services

import (
	"fmt"
	"net/smtp"
	"os"
)

// SendOTP sends a 6-digit OTP code to the given email address via SMTP.
func SendOTP(email, code string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")

	if host == "" || port == "" {
		return fmt.Errorf("email OTP not configured: set SMTP_HOST and SMTP_PORT in .env")
	}
	if from == "" {
		from = user
	}

	subject := "Your ILovePDF Login Code"
	body := fmt.Sprintf(`Hello,

Your one-time login code is: %s

This code expires in 10 minutes.

— ILovePDF`, code)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, email, subject, body)

	addr := host + ":" + port
	var auth smtp.Auth
	if user != "" && pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	return smtp.SendMail(addr, auth, from, []string{email}, []byte(msg))
}

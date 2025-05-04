package config

import (
	"errors"
	"os"
)

type Config struct {
	Port string

	DatabaseURL string

	StripeSecret        string
	StripeWebhookSecret string

	EmailService   string // "sendgrid" or "smtp"
	SendgridAPIKey string
	SMTPHost       string
	SMTPPort       string
	SMTPUsername   string
	SMTPPassword   string
	EmailFrom      string

	LicenseSecret string
}

func New() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL environment variable is required")
	}

	stripeSecret := os.Getenv("STRIPE_SECRET")
	if stripeSecret == "" {
		return nil, errors.New("STRIPE_SECRET environment variable is required")
	}

	stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if stripeWebhookSecret == "" {
		return nil, errors.New("STRIPE_WEBHOOK_SECRET environment variable is required")
	}

	emailService := os.Getenv("EMAIL_SERVICE")
	if emailService == "" {
		emailService = "sendgrid" // Default to SendGrid
	}

	licenseSecret := os.Getenv("LICENSE_SECRET")
	if licenseSecret == "" {
		return nil, errors.New("LICENSE_SECRET environment variable is required")
	}

	sendgridAPIKey := os.Getenv("SENDGRID_API_KEY")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	emailFrom := os.Getenv("EMAIL_FROM")
	if emailFrom == "" {
		emailFrom = "licenses@auto-focus.app"
	}

	if emailService == "sendgrid" && sendgridAPIKey == "" {
		return nil, errors.New("SENDGRID_API_KEY environment variable is required when using SendGrid")
	}

	if emailService == "smtp" {
		if smtpHost == "" || smtpPort == "" || smtpUsername == "" || smtpPassword == "" {
			return nil, errors.New("SMTP_HOST, SMTP_PORT, SMTP_USERNAME, and SMTP_PASSWORD environment variables are required when using SMTP")
		}
	}

	return &Config{
		Port:                port,
		DatabaseURL:         dbURL,
		StripeSecret:        stripeSecret,
		StripeWebhookSecret: stripeWebhookSecret,
		EmailService:        emailService,
		SendgridAPIKey:      sendgridAPIKey,
		SMTPHost:            smtpHost,
		SMTPPort:            smtpPort,
		SMTPUsername:        smtpUsername,
		SMTPPassword:        smtpPassword,
		EmailFrom:           emailFrom,
		LicenseSecret:       licenseSecret,
	}, nil
}

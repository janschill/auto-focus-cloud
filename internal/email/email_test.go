package email

import (
	"os"
	"testing"
)

func TestSend(t *testing.T) {
	tests := []struct {
		name        string
		to          string
		subject     string
		body        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:    "missing SMTP_HOST",
			to:      "test@example.com",
			subject: "Test Subject",
			body:    "Test Body",
			envVars: map[string]string{
				"SMTP_PORT": "587",
				"SMTP_USER": "user@example.com",
				"SMTP_PASS": "password",
			},
			expectError: true,
			errorMsg:    "SMTP configuration missing",
		},
		{
			name:    "missing SMTP_PORT",
			to:      "test@example.com",
			subject: "Test Subject",
			body:    "Test Body",
			envVars: map[string]string{
				"SMTP_HOST": "smtp.example.com",
				"SMTP_USER": "user@example.com",
				"SMTP_PASS": "password",
			},
			expectError: true,
			errorMsg:    "SMTP configuration missing",
		},
		{
			name:    "missing SMTP_USER",
			to:      "test@example.com",
			subject: "Test Subject",
			body:    "Test Body",
			envVars: map[string]string{
				"SMTP_HOST": "smtp.example.com",
				"SMTP_PORT": "587",
				"SMTP_PASS": "password",
			},
			expectError: true,
			errorMsg:    "SMTP configuration missing",
		},
		{
			name:    "missing SMTP_PASS",
			to:      "test@example.com",
			subject: "Test Subject",
			body:    "Test Body",
			envVars: map[string]string{
				"SMTP_HOST": "smtp.example.com",
				"SMTP_PORT": "587",
				"SMTP_USER": "user@example.com",
			},
			expectError: true,
			errorMsg:    "SMTP configuration missing",
		},
		{
			name:    "all empty strings",
			to:      "test@example.com",
			subject: "Test Subject",
			body:    "Test Body",
			envVars: map[string]string{
				"SMTP_HOST": "",
				"SMTP_PORT": "",
				"SMTP_USER": "",
				"SMTP_PASS": "",
			},
			expectError: true,
			errorMsg:    "SMTP configuration missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			// Call the function
			err := Send(tt.to, tt.subject, tt.body)

			// Check results
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSend_Integration(t *testing.T) {
	// Skip this test if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	// This test would require actual SMTP server or mock
	// For now, we just test that with valid config, the function attempts to send
	_ = os.Setenv("SMTP_HOST", "smtp.example.com")
	_ = os.Setenv("SMTP_PORT", "587")
	_ = os.Setenv("SMTP_USER", "user@example.com")
	_ = os.Setenv("SMTP_PASS", "password")

	err := Send("test@example.com", "Test Subject", "Test Body")
	// This will fail with connection error, which is expected
	if err == nil {
		t.Error("expected connection error but got none")
	}
}

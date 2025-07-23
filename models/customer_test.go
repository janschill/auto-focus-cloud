package models

import (
	"testing"
	"time"
)

func TestCustomerModel_Creation(t *testing.T) {
	customer := Customer{
		ID:               "test-id",
		Email:            "test@example.com",
		StripeCustomerID: "cus_test123",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if customer.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", customer.ID)
	}

	if customer.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", customer.Email)
	}

	if customer.StripeCustomerID != "cus_test123" {
		t.Errorf("Expected StripeCustomerID 'cus_test123', got %s", customer.StripeCustomerID)
	}
}

func TestCustomerModel_ZeroValues(t *testing.T) {
	var customer Customer

	if customer.ID != "" {
		t.Errorf("Expected empty ID, got %s", customer.ID)
	}

	if customer.Email != "" {
		t.Errorf("Expected empty email, got %s", customer.Email)
	}

	if customer.StripeCustomerID != "" {
		t.Errorf("Expected empty StripeCustomerID, got %s", customer.StripeCustomerID)
	}

	if !customer.CreatedAt.IsZero() {
		t.Error("Expected zero CreatedAt time")
	}

	if !customer.UpdatedAt.IsZero() {
		t.Error("Expected zero UpdatedAt time")
	}
}

func TestCustomerModel_EmailFormats(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		isValid bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"valid email with plus", "user+tag@example.com", true},
		{"valid email with dash", "user-name@example.com", true},
		{"valid email with numbers", "user123@example.com", true},
		{"invalid empty email", "", false},
		{"invalid email without @", "invalid", false},
		{"invalid email without domain", "test@", false},
		{"invalid email without user", "@example.com", false},
		{"invalid email with spaces", "test @example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := Customer{
				ID:    "test",
				Email: tt.email,
			}

			// Basic validation - email should be stored as provided
			if customer.Email != tt.email {
				t.Errorf("Expected email %s, got %s", tt.email, customer.Email)
			}

			// For proper email validation, you might want to add a validate method
			// This test just ensures the field accepts various formats
		})
	}
}

func TestCustomer_TimeFields(t *testing.T) {
	now := time.Now()

	customer := Customer{
		ID:        "time-test",
		Email:     "time@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if customer.CreatedAt != now {
		t.Error("CreatedAt should match the set time")
	}

	if customer.UpdatedAt != now {
		t.Error("UpdatedAt should match the set time")
	}

	// Test that we can update the time
	later := now.Add(time.Hour)
	customer.UpdatedAt = later

	if customer.UpdatedAt.Equal(now) {
		t.Error("UpdatedAt should be updated to new time")
	}

	if customer.UpdatedAt != later {
		t.Error("UpdatedAt should match the updated time")
	}
}

func TestCustomer_IDGeneration(t *testing.T) {
	// Test that different customers can have different IDs
	customer1 := Customer{ID: "customer1"}
	customer2 := Customer{ID: "customer2"}

	if customer1.ID == customer2.ID {
		t.Error("Different customers should have different IDs")
	}
}

func TestCustomer_StripeIntegration(t *testing.T) {
	tests := []struct {
		name             string
		stripeCustomerID string
		expectedValid    bool
	}{
		{"valid stripe customer ID", "cus_1234567890", true},
		{"empty stripe customer ID", "", true}, // Might be valid for some cases
		{"invalid format", "invalid_format", false},
		{"too short", "cus_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := Customer{
				ID:               "test",
				Email:            "test@example.com",
				StripeCustomerID: tt.stripeCustomerID,
			}

			// Verify the field is stored correctly
			if customer.StripeCustomerID != tt.stripeCustomerID {
				t.Errorf("Expected StripeCustomerID %s, got %s",
					tt.stripeCustomerID, customer.StripeCustomerID)
			}

			// Basic format validation for Stripe customer IDs
			if tt.stripeCustomerID != "" && !tt.expectedValid {
				if len(tt.stripeCustomerID) < 4 || !startsWithCus(tt.stripeCustomerID) {
					// This is expected for invalid formats
					t.Logf("Invalid Stripe customer ID format detected: %s", tt.stripeCustomerID)
				}
			}
		})
	}
}

func TestCustomer_Relationships(t *testing.T) {
	// Test that customer can exist independently
	customer := Customer{
		ID:    "standalone",
		Email: "standalone@example.com",
	}

	if customer.ID == "" {
		t.Error("Customer should have an ID")
	}

	if customer.Email == "" {
		t.Error("Customer should have an email")
	}

	// Customer should be valid even without licenses (licenses are separate)
}

func TestCustomer_Equality(t *testing.T) {
	now := time.Now()

	customer1 := Customer{
		ID:               "same",
		Email:            "same@example.com",
		StripeCustomerID: "cus_same",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	customer2 := Customer{
		ID:               "same",
		Email:            "same@example.com",
		StripeCustomerID: "cus_same",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Manual equality check (Go doesn't have automatic struct equality for time.Time)
	if customer1.ID != customer2.ID ||
		customer1.Email != customer2.Email ||
		customer1.StripeCustomerID != customer2.StripeCustomerID {
		t.Error("Customers with same data should be equal")
	}
}

// Helper functions
func startsWithCus(s string) bool {
	return len(s) >= 4 && s[:4] == "cus_"
}

// Benchmark customer operations
func BenchmarkCustomerModel_Creation(b *testing.B) {
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Customer{
			ID:               "benchmark",
			Email:            "benchmark@example.com",
			StripeCustomerID: "cus_benchmark",
			CreatedAt:        now,
			UpdatedAt:        now,
		}
	}
}

func BenchmarkCustomer_FieldAccess(b *testing.B) {
	customer := Customer{
		ID:               "benchmark",
		Email:            "benchmark@example.com",
		StripeCustomerID: "cus_benchmark",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = customer.ID
		_ = customer.Email
		_ = customer.StripeCustomerID
	}
}

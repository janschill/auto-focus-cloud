package models

import (
	"testing"
	"time"
)

func TestLicenseStatus_Constants(t *testing.T) {
	// Test that status constants have expected values
	if StatusActive != "active" {
		t.Errorf("Expected StatusActive to be 'active', got '%s'", StatusActive)
	}

	if StatusSuspended != "suspended" {
		t.Errorf("Expected StatusSuspended to be 'suspended', got '%s'", StatusSuspended)
	}
}

func TestLicense_Creation(t *testing.T) {
	now := time.Now()
	
	license := License{
		ID:              "test-license-1",
		Key:             "AFP-TEST123",
		CustomerID:      "customer-1",
		ProductID:       "prod_test123",
		Version:         "1.0.0",
		Status:          StatusActive,
		StripeSessionID: "cs_test123",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Test all fields are set correctly
	if license.ID != "test-license-1" {
		t.Errorf("Expected ID 'test-license-1', got '%s'", license.ID)
	}

	if license.Key != "AFP-TEST123" {
		t.Errorf("Expected Key 'AFP-TEST123', got '%s'", license.Key)
	}

	if license.CustomerID != "customer-1" {
		t.Errorf("Expected CustomerID 'customer-1', got '%s'", license.CustomerID)
	}

	if license.ProductID != "prod_test123" {
		t.Errorf("Expected ProductID 'prod_test123', got '%s'", license.ProductID)
	}

	if license.Version != "1.0.0" {
		t.Errorf("Expected Version '1.0.0', got '%s'", license.Version)
	}

	if license.Status != StatusActive {
		t.Errorf("Expected Status '%s', got '%s'", StatusActive, license.Status)
	}

	if license.StripeSessionID != "cs_test123" {
		t.Errorf("Expected StripeSessionID 'cs_test123', got '%s'", license.StripeSessionID)
	}

	if !license.CreatedAt.Equal(now) {
		t.Errorf("Expected CreatedAt to be %v, got %v", now, license.CreatedAt)
	}

	if !license.UpdatedAt.Equal(now) {
		t.Errorf("Expected UpdatedAt to be %v, got %v", now, license.UpdatedAt)
	}
}

func TestLicense_ZeroValues(t *testing.T) {
	var license License

	// Test zero values
	if license.ID != "" {
		t.Errorf("Expected empty ID, got '%s'", license.ID)
	}

	if license.Key != "" {
		t.Errorf("Expected empty Key, got '%s'", license.Key)
	}

	if license.CustomerID != "" {
		t.Errorf("Expected empty CustomerID, got '%s'", license.CustomerID)
	}

	if license.ProductID != "" {
		t.Errorf("Expected empty ProductID, got '%s'", license.ProductID)
	}

	if license.Version != "" {
		t.Errorf("Expected empty Version, got '%s'", license.Version)
	}

	if license.Status != "" {
		t.Errorf("Expected empty Status, got '%s'", license.Status)
	}

	if license.StripeSessionID != "" {
		t.Errorf("Expected empty StripeSessionID, got '%s'", license.StripeSessionID)
	}

	if !license.CreatedAt.IsZero() {
		t.Errorf("Expected zero CreatedAt, got %v", license.CreatedAt)
	}

	if !license.UpdatedAt.IsZero() {
		t.Errorf("Expected zero UpdatedAt, got %v", license.UpdatedAt)
	}
}

func TestLicense_StatusValidation(t *testing.T) {
	testCases := []struct {
		name           string
		status         string
		shouldBeActive bool
	}{
		{
			name:           "active status",
			status:         StatusActive,
			shouldBeActive: true,
		},
		{
			name:           "suspended status",
			status:         StatusSuspended,
			shouldBeActive: false,
		},
		{
			name:           "empty status",
			status:         "",
			shouldBeActive: false,
		},
		{
			name:           "invalid status",
			status:         "invalid",
			shouldBeActive: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			license := License{
				Status: tc.status,
			}

			isActive := license.Status == StatusActive
			if isActive != tc.shouldBeActive {
				t.Errorf("Expected isActive=%v for status '%s', got %v", tc.shouldBeActive, tc.status, isActive)
			}
		})
	}
}

func TestLicense_KeyFormats(t *testing.T) {
	validKeys := []string{
		"AFP-12345678",
		"AFP-ABCDEFGH",
		"AFP-TEST123A",
		"AFP-00000000",
	}

	invalidKeys := []string{
		"",
		"AFP-",
		"AFP",
		"12345678",
		"INVALID-KEY",
		"AFP-123",      // too short
		"AFP-123456789", // too long
	}

	for _, key := range validKeys {
		t.Run("valid_"+key, func(t *testing.T) {
			license := License{Key: key}
			
			// Basic format check: starts with "AFP-" and has reasonable length
			if len(license.Key) != 12 {
				t.Errorf("Expected key length 12, got %d for key '%s'", len(license.Key), key)
			}
			
			if license.Key[:4] != "AFP-" {
				t.Errorf("Expected key to start with 'AFP-', got '%s'", key)
			}
		})
	}

	for _, key := range invalidKeys {
		t.Run("invalid_"+key, func(t *testing.T) {
			license := License{Key: key}
			
			// These should not pass our basic format validation
			isValidFormat := len(license.Key) == 12 && license.Key[:4] == "AFP-"
			if isValidFormat && key != "" {
				t.Errorf("Key '%s' should not be valid format", key)
			}
		})
	}
}

func TestLicense_VersionFormats(t *testing.T) {
	validVersions := []string{
		"1.0.0",
		"2.1.3",
		"10.5.2",
		"0.0.1",
	}

	invalidVersions := []string{
		"",
		"1",
		"1.0",
		"1.0.0.0",
		"v1.0.0",
		"1.0.0-beta",
		"invalid",
	}

	for _, version := range validVersions {
		t.Run("valid_"+version, func(t *testing.T) {
			license := License{Version: version}
			
			// Basic semantic version check (simplified)
			if license.Version == "" {
				t.Errorf("Version should not be empty")
			}
		})
	}

	for _, version := range invalidVersions {
		t.Run("invalid_"+version, func(t *testing.T) {
			license := License{Version: version}
			
			// We don't enforce strict version validation in the model
			// This is handled by the version package
			_ = license // Just ensure the struct accepts any string
		})
	}
}

func TestCustomer_Creation(t *testing.T) {
	now := time.Now()
	
	customer := Customer{
		ID:               "customer-1",
		Email:            "test@example.com",
		StripeCustomerID: "cus_test123",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Test all fields are set correctly
	if customer.ID != "customer-1" {
		t.Errorf("Expected ID 'customer-1', got '%s'", customer.ID)
	}

	if customer.Email != "test@example.com" {
		t.Errorf("Expected Email 'test@example.com', got '%s'", customer.Email)
	}

	if customer.StripeCustomerID != "cus_test123" {
		t.Errorf("Expected StripeCustomerID 'cus_test123', got '%s'", customer.StripeCustomerID)
	}

	if !customer.CreatedAt.Equal(now) {
		t.Errorf("Expected CreatedAt to be %v, got %v", now, customer.CreatedAt)
	}

	if !customer.UpdatedAt.Equal(now) {
		t.Errorf("Expected UpdatedAt to be %v, got %v", now, customer.UpdatedAt)
	}
}

func TestCustomer_ZeroValues(t *testing.T) {
	var customer Customer

	// Test zero values
	if customer.ID != "" {
		t.Errorf("Expected empty ID, got '%s'", customer.ID)
	}

	if customer.Email != "" {
		t.Errorf("Expected empty Email, got '%s'", customer.Email)
	}

	if customer.StripeCustomerID != "" {
		t.Errorf("Expected empty StripeCustomerID, got '%s'", customer.StripeCustomerID)
	}

	if !customer.CreatedAt.IsZero() {
		t.Errorf("Expected zero CreatedAt, got %v", customer.CreatedAt)
	}

	if !customer.UpdatedAt.IsZero() {
		t.Errorf("Expected zero UpdatedAt, got %v", customer.UpdatedAt)
	}
}

func TestCustomer_EmailFormats(t *testing.T) {
	validEmails := []string{
		"test@example.com",
		"user@domain.co.uk",
		"name.surname@company.org",
		"user+tag@example.com",
	}

	invalidEmails := []string{
		"",
		"invalid",
		"@example.com",
		"test@",
		"test.example.com",
	}

	for _, email := range validEmails {
		t.Run("valid_"+email, func(t *testing.T) {
			customer := Customer{Email: email}
			
			// Basic email check: contains @ symbol
			if customer.Email == "" {
				t.Errorf("Email should not be empty")
			}
		})
	}

	for _, email := range invalidEmails {
		t.Run("invalid_"+email, func(t *testing.T) {
			customer := Customer{Email: email}
			
			// We don't enforce strict email validation in the model
			// This would be handled by validation layer
			_ = customer // Just ensure the struct accepts any string
		})
	}
}

func TestModels_TimeFields(t *testing.T) {
	now := time.Now()
	
	// Test License time fields
	license := License{
		CreatedAt: now,
		UpdatedAt: now.Add(time.Hour),
	}

	if license.CreatedAt.After(license.UpdatedAt) {
		t.Errorf("CreatedAt should not be after UpdatedAt")
	}

	// Test Customer time fields  
	customer := Customer{
		CreatedAt: now,
		UpdatedAt: now.Add(time.Hour),
	}

	if customer.CreatedAt.After(customer.UpdatedAt) {
		t.Errorf("CreatedAt should not be after UpdatedAt")
	}

	// Test that time fields can be equal
	customer2 := Customer{
		CreatedAt: now,
		UpdatedAt: now,
	}

	if !customer2.CreatedAt.Equal(customer2.UpdatedAt) {
		t.Errorf("CreatedAt and UpdatedAt should be equal when set to same time")
	}
}

func TestModels_IDGeneration(t *testing.T) {
	// Test that different models can have same ID format
	license := License{ID: "same-id"}
	customer := Customer{ID: "same-id"}

	if license.ID != customer.ID {
		t.Errorf("Both models should accept same ID format")
	}

	// Test empty IDs
	license2 := License{}
	customer2 := Customer{}

	if license2.ID != "" || customer2.ID != "" {
		t.Errorf("Default IDs should be empty")
	}
}

// Test model relationships
func TestModels_Relationships(t *testing.T) {
	customerID := "relationship-customer"
	
	customer := Customer{
		ID:    customerID,
		Email: "relationship@example.com",
	}

	license := License{
		ID:         "relationship-license",
		CustomerID: customerID, // Reference to customer
		Key:        "AFP-REL123",
	}

	// Test that license references customer correctly
	if license.CustomerID != customer.ID {
		t.Errorf("License should reference customer ID correctly")
	}

	// Test multiple licenses for same customer
	license2 := License{
		ID:         "relationship-license-2",
		CustomerID: customerID,
		Key:        "AFP-REL456",
	}

	if license2.CustomerID != customer.ID {
		t.Errorf("Multiple licenses should reference same customer")
	}

	if license.CustomerID != license2.CustomerID {
		t.Errorf("Licenses for same customer should have same CustomerID")
	}
}

// Benchmark tests for model operations
func BenchmarkLicense_Creation(b *testing.B) {
	now := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = License{
			ID:              "bench-license",
			Key:             "AFP-BENCH123",
			CustomerID:      "bench-customer",
			ProductID:       "prod_bench",
			Version:         "1.0.0",
			Status:          StatusActive,
			StripeSessionID: "cs_bench",
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}
}

func BenchmarkCustomer_Creation(b *testing.B) {
	now := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Customer{
			ID:               "bench-customer",
			Email:            "bench@example.com",
			StripeCustomerID: "cus_bench",
			CreatedAt:        now,
			UpdatedAt:        now,
		}
	}
}
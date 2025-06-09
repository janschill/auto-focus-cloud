package models

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLicense_JSONSerialization(t *testing.T) {
	license := License{
		Key:     "TEST-LICENSE-KEY",
		Version: "1.2.3",
	}

	// Test marshaling
	data, err := json.Marshal(license)
	if err != nil {
		t.Fatalf("Failed to marshal license: %v", err)
	}

	// Test unmarshaling
	var unmarshaled License
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal license: %v", err)
	}

	// Verify data integrity
	if unmarshaled.Key != license.Key {
		t.Errorf("Expected key '%s', got '%s'", license.Key, unmarshaled.Key)
	}

	if unmarshaled.Version != license.Version {
		t.Errorf("Expected version '%s', got '%s'", license.Version, unmarshaled.Version)
	}
}

func TestLicense_EmptyFields(t *testing.T) {
	license := License{
		Key:     "",
		Version: "",
	}

	data, err := json.Marshal(license)
	if err != nil {
		t.Fatalf("Failed to marshal empty license: %v", err)
	}

	var unmarshaled License
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty license: %v", err)
	}

	if unmarshaled.Key != "" {
		t.Errorf("Expected empty key, got '%s'", unmarshaled.Key)
	}

	if unmarshaled.Version != "" {
		t.Errorf("Expected empty version, got '%s'", unmarshaled.Version)
	}
}

func TestCustomer_JSONSerialization(t *testing.T) {
	customer := Customer{
		Id:    "customer-123",
		Email: "test@example.com",
		Licenses: []License{
			{Key: "LICENSE-001", Version: "1.0.0"},
			{Key: "LICENSE-002", Version: "2.0.0"},
		},
	}

	// Test marshaling
	data, err := json.Marshal(customer)
	if err != nil {
		t.Fatalf("Failed to marshal customer: %v", err)
	}

	// Test unmarshaling
	var unmarshaled Customer
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal customer: %v", err)
	}

	// Verify basic fields
	if unmarshaled.Id != customer.Id {
		t.Errorf("Expected ID '%s', got '%s'", customer.Id, unmarshaled.Id)
	}

	if unmarshaled.Email != customer.Email {
		t.Errorf("Expected email '%s', got '%s'", customer.Email, unmarshaled.Email)
	}

	// Verify licenses array
	if len(unmarshaled.Licenses) != len(customer.Licenses) {
		t.Errorf("Expected %d licenses, got %d", len(customer.Licenses), len(unmarshaled.Licenses))
	}

	for i, license := range customer.Licenses {
		if i >= len(unmarshaled.Licenses) {
			t.Errorf("Missing license at index %d", i)
			continue
		}

		if unmarshaled.Licenses[i].Key != license.Key {
			t.Errorf("License %d: expected key '%s', got '%s'", i, license.Key, unmarshaled.Licenses[i].Key)
		}

		if unmarshaled.Licenses[i].Version != license.Version {
			t.Errorf("License %d: expected version '%s', got '%s'", i, license.Version, unmarshaled.Licenses[i].Version)
		}
	}
}

func TestCustomer_EmptyLicenses(t *testing.T) {
	customer := Customer{
		Id:       "customer-empty",
		Email:    "empty@example.com",
		Licenses: []License{},
	}

	data, err := json.Marshal(customer)
	if err != nil {
		t.Fatalf("Failed to marshal customer with empty licenses: %v", err)
	}

	var unmarshaled Customer
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal customer with empty licenses: %v", err)
	}

	if len(unmarshaled.Licenses) != 0 {
		t.Errorf("Expected 0 licenses, got %d", len(unmarshaled.Licenses))
	}
}

func TestCustomer_NilLicenses(t *testing.T) {
	customer := Customer{
		Id:       "customer-nil",
		Email:    "nil@example.com",
		Licenses: nil,
	}

	data, err := json.Marshal(customer)
	if err != nil {
		t.Fatalf("Failed to marshal customer with nil licenses: %v", err)
	}

	var unmarshaled Customer
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal customer with nil licenses: %v", err)
	}

	// JSON unmarshaling should create an empty slice, not nil
	if len(unmarshaled.Licenses) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(unmarshaled.Licenses))
	}
}

func TestLicense_SpecialCharacters(t *testing.T) {
	license := License{
		Key:     "KEY-WITH-SPECIAL-CHARS-!@#$%",
		Version: "1.0.0-beta+build.123",
	}

	data, err := json.Marshal(license)
	if err != nil {
		t.Fatalf("Failed to marshal license with special characters: %v", err)
	}

	var unmarshaled License
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal license with special characters: %v", err)
	}

	if unmarshaled.Key != license.Key {
		t.Errorf("Expected key '%s', got '%s'", license.Key, unmarshaled.Key)
	}

	if unmarshaled.Version != license.Version {
		t.Errorf("Expected version '%s', got '%s'", license.Version, unmarshaled.Version)
	}
}

func TestCustomer_LargeLicenseArray(t *testing.T) {
	// Test with many licenses to ensure performance
	licenses := make([]License, 100)
	for i := 0; i < 100; i++ {
		licenses[i] = License{
			Key:     fmt.Sprintf("LICENSE-%03d", i),
			Version: fmt.Sprintf("1.%d.0", i),
		}
	}

	customer := Customer{
		Id:       "customer-large",
		Email:    "large@example.com",
		Licenses: licenses,
	}

	data, err := json.Marshal(customer)
	if err != nil {
		t.Fatalf("Failed to marshal customer with many licenses: %v", err)
	}

	var unmarshaled Customer
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal customer with many licenses: %v", err)
	}

	if len(unmarshaled.Licenses) != 100 {
		t.Errorf("Expected 100 licenses, got %d", len(unmarshaled.Licenses))
	}

	// Spot check a few licenses
	if unmarshaled.Licenses[0].Key != "LICENSE-000" {
		t.Errorf("Expected first license key 'LICENSE-000', got '%s'", unmarshaled.Licenses[0].Key)
	}

	if unmarshaled.Licenses[99].Key != "LICENSE-099" {
		t.Errorf("Expected last license key 'LICENSE-099', got '%s'", unmarshaled.Licenses[99].Key)
	}
}

// Test data validation (even though current models don't have validation)
func TestLicense_DataConsistency(t *testing.T) {
	tests := []struct {
		name    string
		license License
		valid   bool
	}{
		{
			name: "normal license",
			license: License{
				Key:     "NORMAL-LICENSE-KEY",
				Version: "1.0.0",
			},
			valid: true,
		},
		{
			name: "empty key should be invalid",
			license: License{
				Key:     "",
				Version: "1.0.0",
			},
			valid: false,
		},
		{
			name: "empty version should be invalid",
			license: License{
				Key:     "VALID-KEY",
				Version: "",
			},
			valid: false,
		},
		{
			name: "both empty should be invalid",
			license: License{
				Key:     "",
				Version: "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a placeholder for future validation logic
			// For now, we just test that data can be serialized
			_, err := json.Marshal(tt.license)
			if err != nil {
				t.Errorf("Failed to marshal license: %v", err)
			}

			// TODO: Add actual validation when License gets validation methods
			// Example: if tt.license.IsValid() != tt.valid { ... }
		})
	}
}

// Benchmark tests for performance
func BenchmarkLicense_Marshal(b *testing.B) {
	license := License{
		Key:     "BENCHMARK-LICENSE-KEY",
		Version: "1.0.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(license)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

func BenchmarkCustomer_Marshal(b *testing.B) {
	customer := Customer{
		Id:    "benchmark-customer",
		Email: "benchmark@example.com",
		Licenses: []License{
			{Key: "LICENSE-001", Version: "1.0.0"},
			{Key: "LICENSE-002", Version: "1.1.0"},
			{Key: "LICENSE-003", Version: "1.2.0"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(customer)
		if err != nil {
			b.Fatalf("Failed to marshal: %v", err)
		}
	}
}

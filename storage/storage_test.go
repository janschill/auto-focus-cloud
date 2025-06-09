package storage

import (
	"fmt"
	"testing"

	"auto-focus.app/cloud/models"
)

func TestDatabase_BasicOperations(t *testing.T) {
	// Create test database
	db := Database{
		"1": models.Customer{
			Id:    "1",
			Email: "customer1@example.com",
			Licenses: []models.License{
				{Key: "LICENSE-001", Version: "1.0.0"},
				{Key: "LICENSE-002", Version: "1.1.0"},
			},
		},
		"2": models.Customer{
			Id:    "2",
			Email: "customer2@example.com",
			Licenses: []models.License{
				{Key: "LICENSE-003", Version: "2.0.0"},
			},
		},
	}

	// Test reading existing customer
	customer, exists := db["1"]
	if !exists {
		t.Errorf("Expected customer '1' to exist")
	}

	if customer.Email != "customer1@example.com" {
		t.Errorf("Expected email 'customer1@example.com', got '%s'", customer.Email)
	}

	if len(customer.Licenses) != 2 {
		t.Errorf("Expected 2 licenses, got %d", len(customer.Licenses))
	}

	// Test reading non-existent customer
	_, exists = db["999"]
	if exists {
		t.Errorf("Expected customer '999' to not exist")
	}
}

func TestDatabase_AddCustomer(t *testing.T) {
	db := make(Database)

	newCustomer := models.Customer{
		Id:    "new-customer",
		Email: "new@example.com",
		Licenses: []models.License{
			{Key: "NEW-LICENSE", Version: "1.0.0"},
		},
	}

	// Add customer
	db["new-customer"] = newCustomer

	// Verify addition
	customer, exists := db["new-customer"]
	if !exists {
		t.Errorf("Expected new customer to exist after addition")
	}

	if customer.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got '%s'", customer.Email)
	}
}

func TestDatabase_UpdateCustomer(t *testing.T) {
	db := Database{
		"update-test": models.Customer{
			Id:    "update-test",
			Email: "original@example.com",
			Licenses: []models.License{
				{Key: "ORIGINAL-LICENSE", Version: "1.0.0"},
			},
		},
	}

	// Update customer
	updatedCustomer := models.Customer{
		Id:    "update-test",
		Email: "updated@example.com",
		Licenses: []models.License{
			{Key: "ORIGINAL-LICENSE", Version: "1.0.0"},
			{Key: "NEW-LICENSE", Version: "1.1.0"},
		},
	}

	db["update-test"] = updatedCustomer

	// Verify update
	customer := db["update-test"]
	if customer.Email != "updated@example.com" {
		t.Errorf("Expected updated email 'updated@example.com', got '%s'", customer.Email)
	}

	if len(customer.Licenses) != 2 {
		t.Errorf("Expected 2 licenses after update, got %d", len(customer.Licenses))
	}
}

func TestDatabase_DeleteCustomer(t *testing.T) {
	db := Database{
		"delete-test": models.Customer{
			Id:    "delete-test",
			Email: "delete@example.com",
			Licenses: []models.License{
				{Key: "DELETE-LICENSE", Version: "1.0.0"},
			},
		},
	}

	// Verify customer exists
	_, exists := db["delete-test"]
	if !exists {
		t.Fatalf("Expected customer to exist before deletion")
	}

	// Delete customer
	delete(db, "delete-test")

	// Verify deletion
	_, exists = db["delete-test"]
	if exists {
		t.Errorf("Expected customer to not exist after deletion")
	}
}

func TestDatabase_FindLicenseKey(t *testing.T) {
	db := Database{
		"1": models.Customer{
			Id:    "1",
			Email: "customer1@example.com",
			Licenses: []models.License{
				{Key: "FIND-LICENSE-001", Version: "1.0.0"},
				{Key: "FIND-LICENSE-002", Version: "1.1.0"},
			},
		},
		"2": models.Customer{
			Id:    "2",
			Email: "customer2@example.com",
			Licenses: []models.License{
				{Key: "FIND-LICENSE-003", Version: "2.0.0"},
			},
		},
	}

	tests := []struct {
		name           string
		licenseKey     string
		expectedCustomer string
		shouldFind     bool
	}{
		{
			name:           "find license in first customer",
			licenseKey:     "FIND-LICENSE-001",
			expectedCustomer: "1",
			shouldFind:     true,
		},
		{
			name:           "find second license in first customer",
			licenseKey:     "FIND-LICENSE-002",
			expectedCustomer: "1",
			shouldFind:     true,
		},
		{
			name:           "find license in second customer",
			licenseKey:     "FIND-LICENSE-003",
			expectedCustomer: "2",
			shouldFind:     true,
		},
		{
			name:       "license not found",
			licenseKey: "NON-EXISTENT-LICENSE",
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var foundCustomer *models.Customer
			
			// Search through database (simulating findLicenseCustomer logic)
			for _, customer := range db {
				for _, license := range customer.Licenses {
					if license.Key == tt.licenseKey {
						foundCustomer = &customer
						break
					}
				}
				if foundCustomer != nil {
					break
				}
			}

			if tt.shouldFind {
				if foundCustomer == nil {
					t.Errorf("Expected to find customer for license '%s', got nil", tt.licenseKey)
					return
				}
				if foundCustomer.Id != tt.expectedCustomer {
					t.Errorf("Expected customer '%s', got '%s'", tt.expectedCustomer, foundCustomer.Id)
				}
			} else {
				if foundCustomer != nil {
					t.Errorf("Expected not to find customer for license '%s', got customer '%s'", tt.licenseKey, foundCustomer.Id)
				}
			}
		})
	}
}

func TestDatabase_EmptyDatabase(t *testing.T) {
	db := make(Database)

	// Test operations on empty database
	_, exists := db["any-key"]
	if exists {
		t.Errorf("Expected no customers in empty database")
	}

	// Test that we can iterate over empty database
	count := 0
	for range db {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 customers in empty database, found %d", count)
	}
}

func TestDatabase_LargeDataset(t *testing.T) {
	// Test with many customers to ensure performance
	db := make(Database)
	
	// Add 1000 customers with multiple licenses each
	for i := 0; i < 1000; i++ {
		customerID := fmt.Sprintf("customer-%d", i)
		licenses := make([]models.License, 5) // 5 licenses per customer
		
		for j := 0; j < 5; j++ {
			licenses[j] = models.License{
				Key:     fmt.Sprintf("LICENSE-%d-%d", i, j),
				Version: "1.0.0",
			}
		}
		
		db[customerID] = models.Customer{
			Id:       customerID,
			Email:    fmt.Sprintf("customer%d@example.com", i),
			Licenses: licenses,
		}
	}

	// Verify we can find specific customers
	customer, exists := db["customer-500"]
	if !exists {
		t.Errorf("Expected to find customer-500 in large dataset")
	}

	if customer.Email != "customer500@example.com" {
		t.Errorf("Expected email 'customer500@example.com', got '%s'", customer.Email)
	}

	// Verify we can find specific licenses
	targetLicense := "LICENSE-750-3"
	var found bool
	
	for _, customer := range db {
		for _, license := range customer.Licenses {
			if license.Key == targetLicense {
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		t.Errorf("Expected to find license '%s' in large dataset", targetLicense)
	}
}

func TestDatabase_ConcurrentRead(t *testing.T) {
	db := Database{
		"concurrent-test": models.Customer{
			Id:    "concurrent-test",
			Email: "concurrent@example.com",
			Licenses: []models.License{
				{Key: "CONCURRENT-LICENSE", Version: "1.0.0"},
			},
		},
	}

	// Test concurrent reads (Go maps are safe for concurrent reads)
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			customer, exists := db["concurrent-test"]
			if !exists {
				t.Errorf("Expected customer to exist in concurrent read")
			}
			if customer.Email != "concurrent@example.com" {
				t.Errorf("Expected email 'concurrent@example.com', got '%s'", customer.Email)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDatabase_MemoryUsage(t *testing.T) {
	// Test that we can create and destroy large databases without issues
	for iteration := 0; iteration < 5; iteration++ {
		db := make(Database)
		
		// Add many customers
		for i := 0; i < 100; i++ {
			customerID := fmt.Sprintf("mem-test-%d-%d", iteration, i)
			db[customerID] = models.Customer{
				Id:    customerID,
				Email: fmt.Sprintf("mem%d-%d@example.com", iteration, i),
				Licenses: []models.License{
					{Key: fmt.Sprintf("MEM-LICENSE-%d-%d", iteration, i), Version: "1.0.0"},
				},
			}
		}
		
		// Verify some data
		if len(db) != 100 {
			t.Errorf("Expected 100 customers, got %d", len(db))
		}
		
		// Clear database (simulate cleanup)
		for k := range db {
			delete(db, k)
		}
		
		if len(db) != 0 {
			t.Errorf("Expected empty database after cleanup, got %d customers", len(db))
		}
	}
}

// Benchmark tests
func BenchmarkDatabase_ReadCustomer(b *testing.B) {
	db := Database{
		"benchmark": models.Customer{
			Id:    "benchmark",
			Email: "benchmark@example.com",
			Licenses: []models.License{
				{Key: "BENCHMARK-LICENSE", Version: "1.0.0"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = db["benchmark"]
	}
}

func BenchmarkDatabase_FindLicense(b *testing.B) {
	// Create database with many customers
	db := make(Database)
	for i := 0; i < 1000; i++ {
		customerID := fmt.Sprintf("bench-customer-%d", i)
		db[customerID] = models.Customer{
			Id:    customerID,
			Email: fmt.Sprintf("bench%d@example.com", i),
			Licenses: []models.License{
				{Key: fmt.Sprintf("BENCH-LICENSE-%d", i), Version: "1.0.0"},
			},
		}
	}

	targetLicense := "BENCH-LICENSE-500"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate license search
		var found bool
		for _, customer := range db {
			for _, license := range customer.Licenses {
				if license.Key == targetLicense {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
}
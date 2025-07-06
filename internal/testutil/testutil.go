package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"auto-focus.app/cloud/handlers"
	"auto-focus.app/cloud/models"
	"auto-focus.app/cloud/storage"
)

// TestStorage creates a memory storage with test data
func TestStorage() *storage.MemoryStorage {
	return &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}
}

// CreateTestCustomer creates a test customer with given parameters
func CreateTestCustomer(id, email string) models.Customer {
	return models.Customer{
		ID:               id,
		Email:            email,
		StripeCustomerID: "cus_" + id,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// CreateTestLicense creates a test license with given parameters
func CreateTestLicense(id, key, customerID string) models.License {
	return models.License{
		ID:              id,
		Key:             key,
		CustomerID:      customerID,
		ProductID:       "prod_test",
		Version:         "1.0.0",
		Status:          models.StatusActive,
		StripeSessionID: "cs_test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// SetupTestData creates a complete test environment with customers and licenses
func SetupTestData(storage storage.Storage) error {
	ctx := context.Background()

	// Create test customers
	customers := []models.Customer{
		CreateTestCustomer("customer1", "customer1@example.com"),
		CreateTestCustomer("customer2", "customer2@example.com"),
		CreateTestCustomer("customer3", "customer3@example.com"),
	}

	for _, customer := range customers {
		if err := storage.SaveCustomer(ctx, &customer); err != nil {
			return fmt.Errorf("failed to save customer %s: %w", customer.ID, err)
		}
	}

	// Create test licenses
	licenses := []models.License{
		CreateTestLicense("license1", "AFP-ACTIVE1", "customer1"),
		CreateTestLicense("license2", "AFP-ACTIVE2", "customer2"),
		{
			ID:              "license3",
			Key:             "AFP-SUSPENDED1",
			CustomerID:      "customer3",
			ProductID:       "prod_test",
			Version:         "1.0.0",
			Status:          models.StatusSuspended,
			StripeSessionID: "cs_test",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	for _, license := range licenses {
		if err := storage.SaveLicense(ctx, &license); err != nil {
			return fmt.Errorf("failed to save license %s: %w", license.ID, err)
		}
	}

	return nil
}

// MakeValidateRequest creates and sends a license validation request
func MakeValidateRequest(t *testing.T, server *handlers.Server, licenseKey, appVersion string) *httptest.ResponseRecorder {
	reqBody := handlers.LicenseRequest{
		LicenseKey: licenseKey,
		AppVersion: appVersion,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	return w
}

// AssertValidateResponse checks if the validation response matches expected values
func AssertValidateResponse(t *testing.T, w *httptest.ResponseRecorder, expectedValid bool, expectedMessage string) {
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response handlers.ValidateResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Valid != expectedValid {
		t.Errorf("Expected valid=%v, got valid=%v", expectedValid, response.Valid)
	}

	if response.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, response.Message)
	}
}

// AssertErrorResponse checks if the error response matches expected values
func AssertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if response["error"] != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, response["error"])
	}
}

// CreateStripeWebhookPayload creates a mock Stripe webhook payload
func CreateStripeWebhookPayload(eventType string, sessionData map[string]interface{}) []byte {
	event := map[string]interface{}{
		"id":   "evt_test123",
		"type": eventType,
		"data": map[string]interface{}{
			"object": sessionData,
		},
	}

	payload, _ := json.Marshal(event)
	return payload
}

// CreateMockCheckoutSession creates a mock Stripe checkout session
func CreateMockCheckoutSession(customerEmail, sessionID string, hasCustomer bool) map[string]interface{} {
	session := map[string]interface{}{
		"id":              sessionID,
		"customer_email":  customerEmail,
		"amount_total":    2999,
		"currency":        "usd",
		"payment_status":  "paid",
		"metadata": map[string]interface{}{
			"product_id":      "prod_test123",
			"license_version": "1.0.0",
		},
	}

	if hasCustomer {
		session["customer"] = map[string]interface{}{
			"id": "cus_test123",
		}
	}

	return session
}

// MakeStripeWebhookRequest creates and sends a Stripe webhook request
func MakeStripeWebhookRequest(t *testing.T, server *handlers.Server, payload []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "test-signature")

	w := httptest.NewRecorder()

	// Set test environment
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")

	server.Mux.ServeHTTP(w, req)
	return w
}

// BenchmarkHelper provides utilities for benchmark tests
type BenchmarkHelper struct {
	Storage storage.Storage
	Server  *handlers.Server
}

// NewBenchmarkHelper creates a new benchmark helper with pre-populated data
func NewBenchmarkHelper(numCustomers, licensesPerCustomer int) *BenchmarkHelper {
	storage := TestStorage()
	ctx := context.Background()

	// Create test data
	for i := 0; i < numCustomers; i++ {
		customer := CreateTestCustomer(fmt.Sprintf("bench-customer-%d", i), fmt.Sprintf("bench%d@example.com", i))
		storage.SaveCustomer(ctx, &customer)

		for j := 0; j < licensesPerCustomer; j++ {
			license := CreateTestLicense(
				fmt.Sprintf("bench-license-%d-%d", i, j),
				fmt.Sprintf("AFP-BENCH%d%d", i, j),
				customer.ID,
			)
			storage.SaveLicense(ctx, &license)
		}
	}

	server := handlers.NewHttpServer(storage)

	return &BenchmarkHelper{
		Storage: storage,
		Server:  server,
	}
}

// GetRandomLicenseKey returns a random license key for benchmarking
func (bh *BenchmarkHelper) GetRandomLicenseKey(i int) string {
	// Use modulo to cycle through available licenses
	numCustomers := len(bh.Storage.(*storage.MemoryStorage).Data)
	customerIndex := i % numCustomers
	return fmt.Sprintf("AFP-BENCH%d0", customerIndex)
}

// ValidationTestCase represents a test case for license validation
type ValidationTestCase struct {
	Name            string
	LicenseKey      string
	AppVersion      string
	ExpectedValid   bool
	ExpectedMessage string
	ExpectedStatus  int
}

// RunValidationTestCases runs a set of validation test cases
func RunValidationTestCases(t *testing.T, server *handlers.Server, testCases []ValidationTestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			w := MakeValidateRequest(t, server, tc.LicenseKey, tc.AppVersion)

			if tc.ExpectedStatus != 0 && w.Code != tc.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", tc.ExpectedStatus, w.Code)
				return
			}

			if tc.ExpectedStatus >= 400 {
				// Error response expected
				var response map[string]string
				json.NewDecoder(w.Body).Decode(&response)
				if response["error"] != tc.ExpectedMessage {
					t.Errorf("Expected error '%s', got '%s'", tc.ExpectedMessage, response["error"])
				}
			} else {
				// Validation response expected
				AssertValidateResponse(t, w, tc.ExpectedValid, tc.ExpectedMessage)
			}
		})
	}
}

// StorageTestSuite provides a standard test suite for storage implementations
type StorageTestSuite struct {
	Storage storage.Storage
	Cleanup func()
}

// RunStorageTestSuite runs standard tests on any storage implementation
func RunStorageTestSuite(t *testing.T, suite StorageTestSuite) {
	defer suite.Cleanup()

	ctx := context.Background()

	t.Run("CustomerOperations", func(t *testing.T) {
		customer := CreateTestCustomer("test1", "test@example.com")

		// Test save
		err := suite.Storage.SaveCustomer(ctx, &customer)
		if err != nil {
			t.Errorf("Failed to save customer: %v", err)
		}

		// Test get
		retrieved, err := suite.Storage.GetCustomer(ctx, "test1")
		if err != nil {
			t.Errorf("Failed to get customer: %v", err)
		}
		if retrieved == nil {
			t.Fatalf("Expected customer, got nil")
		}
		if retrieved.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", retrieved.Email)
		}

		// Test find by email
		found, err := suite.Storage.FindCustomerByEmailAddress(ctx, "test@example.com")
		if err != nil {
			t.Errorf("Failed to find customer by email: %v", err)
		}
		if found == nil {
			t.Fatalf("Expected to find customer by email")
		}
		if found.ID != "test1" {
			t.Errorf("Expected ID 'test1', got '%s'", found.ID)
		}
	})

	t.Run("LicenseOperations", func(t *testing.T) {
		customer := CreateTestCustomer("license-test", "license@example.com")
		suite.Storage.SaveCustomer(ctx, &customer)

		license := CreateTestLicense("license1", "AFP-TEST123", "license-test")

		// Test save
		err := suite.Storage.SaveLicense(ctx, &license)
		if err != nil {
			t.Errorf("Failed to save license: %v", err)
		}

		// Test get
		retrieved, err := suite.Storage.GetLicense(ctx, "license1")
		if err != nil {
			t.Errorf("Failed to get license: %v", err)
		}
		if retrieved == nil {
			t.Fatalf("Expected license, got nil")
		}
		if retrieved.Key != "AFP-TEST123" {
			t.Errorf("Expected key 'AFP-TEST123', got '%s'", retrieved.Key)
		}

		// Test find by key
		found, err := suite.Storage.FindLicenseByKey(ctx, "AFP-TEST123")
		if err != nil {
			t.Errorf("Failed to find license by key: %v", err)
		}
		if found == nil {
			t.Fatalf("Expected to find license by key")
		}
		if found.ID != "license1" {
			t.Errorf("Expected ID 'license1', got '%s'", found.ID)
		}

		// Test find by customer
		licenses, err := suite.Storage.FindLicensesByCustomer(ctx, "license-test")
		if err != nil {
			t.Errorf("Failed to find licenses by customer: %v", err)
		}
		if len(licenses) != 1 {
			t.Errorf("Expected 1 license, got %d", len(licenses))
		}
		if licenses[0].Key != "AFP-TEST123" {
			t.Errorf("Expected key 'AFP-TEST123', got '%s'", licenses[0].Key)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		// Test customer not found
		customer, err := suite.Storage.GetCustomer(ctx, "notfound")
		if err != nil {
			t.Errorf("Expected no error for not found customer, got %v", err)
		}
		if customer != nil {
			t.Errorf("Expected nil for not found customer, got %v", customer)
		}

		// Test license not found
		license, err := suite.Storage.FindLicenseByKey(ctx, "AFP-NOTFOUND")
		if err != nil {
			t.Errorf("Expected no error for not found license, got %v", err)
		}
		if license != nil {
			t.Errorf("Expected nil for not found license, got %v", license)
		}
	})
}
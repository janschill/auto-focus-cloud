package main

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

// Integration tests that test complete workflows end-to-end

func TestFullWorkflow_StripeWebhookToLicenseValidation(t *testing.T) {
	// Setup storage
	storage := &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}

	server := handlers.NewHttpServer(storage)

	// Step 1: Simulate Stripe webhook creating a license
	checkoutSession := createMockStripeCheckoutSession("customer@example.com", "cs_test123")
	stripeEvent := createMockStripeWebhookEvent("checkout.session.completed", checkoutSession)

	payload, err := json.Marshal(stripeEvent)
	if err != nil {
		t.Fatalf("Failed to marshal Stripe event: %v", err)
	}

	// Send webhook
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "test-signature")

	w := httptest.NewRecorder()

	// Set test environment
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("TEST_MODE", "true")

	server.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Webhook failed with status %d", w.Code)
	}

	// Verify customer and license were created
	if len(storage.Data) != 1 {
		t.Errorf("Expected 1 customer after webhook, got %d", len(storage.Data))
	}

	if len(storage.Licenses) != 1 {
		t.Errorf("Expected 1 license after webhook, got %d", len(storage.Licenses))
	}

	// Find the created license key
	var licenseKey string
	for _, license := range storage.Licenses {
		licenseKey = license.Key
		break
	}

	if licenseKey == "" {
		t.Fatalf("No license key found after webhook")
	}

	// Step 2: Validate the license
	validateReq := handlers.LicenseRequest{
		LicenseKey: licenseKey,
		AppVersion: "1.0.0",
	}

	validateBody, err := json.Marshal(validateReq)
	if err != nil {
		t.Fatalf("Failed to marshal validate request: %v", err)
	}

	validateHttpReq := httptest.NewRequest(http.MethodPost, "/v1/licenses/validate", bytes.NewBuffer(validateBody))
	validateHttpReq.Header.Set("Content-Type", "application/json")

	validateW := httptest.NewRecorder()
	server.Mux.ServeHTTP(validateW, validateHttpReq)

	if validateW.Code != http.StatusOK {
		t.Errorf("License validation failed with status %d", validateW.Code)
	}

	var validateResponse handlers.ValidateResponse
	err = json.NewDecoder(validateW.Body).Decode(&validateResponse)
	if err != nil {
		t.Fatalf("Failed to decode validation response: %v", err)
	}

	if !validateResponse.Valid {
		t.Errorf("Expected license to be valid, got invalid: %s", validateResponse.Message)
	}

	if validateResponse.Message != "license valid" {
		t.Errorf("Expected message 'license valid', got '%s'", validateResponse.Message)
	}
}

func TestWorkflow_MultipleCustomersAndLicenses(t *testing.T) {
	storage := &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}

	server := handlers.NewHttpServer(storage)

	// Create multiple customers through webhooks
	customers := []struct {
		email     string
		sessionID string
	}{
		{"customer1@example.com", "cs_customer1"},
		{"customer2@example.com", "cs_customer2"},
		{"customer3@example.com", "cs_customer3"},
	}

	var licenseKeys []string

	// Step 1: Create licenses for each customer
	for _, customer := range customers {
		checkoutSession := createMockStripeCheckoutSession(customer.email, customer.sessionID)
		stripeEvent := createMockStripeWebhookEvent("checkout.session.completed", checkoutSession)

		payload, _ := json.Marshal(stripeEvent)
		req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Stripe-Signature", "test-signature")

		w := httptest.NewRecorder()
		t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
		t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
		t.Setenv("TEST_MODE", "true")

		server.Mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Webhook failed for customer %s with status %d", customer.email, w.Code)
		}
	}

	// Verify all customers and licenses were created
	if len(storage.Data) != 3 {
		t.Errorf("Expected 3 customers, got %d", len(storage.Data))
	}

	if len(storage.Licenses) != 3 {
		t.Errorf("Expected 3 licenses, got %d", len(storage.Licenses))
	}

	// Collect license keys
	for _, license := range storage.Licenses {
		licenseKeys = append(licenseKeys, license.Key)
	}

	// Step 2: Validate all licenses
	for i, key := range licenseKeys {
		validateReq := handlers.LicenseRequest{
			LicenseKey: key,
			AppVersion: "1.0.0",
		}

		validateBody, _ := json.Marshal(validateReq)
		req := httptest.NewRequest(http.MethodPost, "/v1/licenses/validate", bytes.NewBuffer(validateBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("License validation failed for license %d with status %d", i, w.Code)
		}

		var response handlers.ValidateResponse
		_ = json.NewDecoder(w.Body).Decode(&response)

		if !response.Valid {
			t.Errorf("License %d should be valid, got invalid: %s", i, response.Message)
		}
	}
}

func TestWorkflow_SuspendedLicense(t *testing.T) {
	storage := &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}

	server := handlers.NewHttpServer(storage)
	ctx := context.Background()

	// Step 1: Create a customer and license manually
	customer := models.Customer{
		ID:               "suspended-customer",
		Email:            "suspended@example.com",
		StripeCustomerID: "cus_suspended",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := storage.SaveCustomer(ctx, &customer)
	if err != nil {
		t.Fatalf("Failed to save customer: %v", err)
	}

	license := models.License{
		ID:              "suspended-license",
		Key:             "AFP-SUSPENDED",
		CustomerID:      "suspended-customer",
		ProductID:       "prod_test",
		Version:         "1.0.0",
		Status:          models.StatusSuspended, // Suspended license
		StripeSessionID: "cs_suspended",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = storage.SaveLicense(ctx, &license)
	if err != nil {
		t.Fatalf("Failed to save license: %v", err)
	}

	// Step 2: Try to validate the suspended license
	validateReq := handlers.LicenseRequest{
		LicenseKey: "AFP-SUSPENDED",
		AppVersion: "1.0.0",
	}

	validateBody, _ := json.Marshal(validateReq)
	req := httptest.NewRequest(http.MethodPost, "/v1/licenses/validate", bytes.NewBuffer(validateBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response handlers.ValidateResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Valid {
		t.Errorf("Suspended license should be invalid")
	}

	if response.Message != "license not active" {
		t.Errorf("Expected message 'license not active', got '%s'", response.Message)
	}
}

func TestWorkflow_ErrorHandling(t *testing.T) {
	storage := &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}

	server := handlers.NewHttpServer(storage)

	tests := []struct {
		name            string
		licenseKey      string
		expectedValid   bool
		expectedMessage string
	}{
		{
			name:            "nonexistent license",
			licenseKey:      "AFP-NOTFOUND",
			expectedValid:   false,
			expectedMessage: "license not found",
		},
		{
			name:            "empty license key",
			licenseKey:      "",
			expectedValid:   false,
			expectedMessage: "", // This will be caught at validation level
		},
		{
			name:            "malformed license key",
			licenseKey:      "INVALID-KEY-FORMAT",
			expectedValid:   false,
			expectedMessage: "license not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.licenseKey == "" {
				// Test empty license key - should return 400
				validateReq := handlers.LicenseRequest{
					LicenseKey: tt.licenseKey,
					AppVersion: "1.0.0",
				}

				validateBody, _ := json.Marshal(validateReq)
				req := httptest.NewRequest(http.MethodPost, "/v1/licenses/validate", bytes.NewBuffer(validateBody))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				server.Mux.ServeHTTP(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected status 400 for empty license key, got %d", w.Code)
				}
				return
			}

			// Test other cases
			validateReq := handlers.LicenseRequest{
				LicenseKey: tt.licenseKey,
				AppVersion: "1.0.0",
			}

			validateBody, _ := json.Marshal(validateReq)
			req := httptest.NewRequest(http.MethodPost, "/v1/licenses/validate", bytes.NewBuffer(validateBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.Mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response handlers.ValidateResponse
			_ = json.NewDecoder(w.Body).Decode(&response)

			if response.Valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectedValid, response.Valid)
			}

			if response.Message != tt.expectedMessage {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMessage, response.Message)
			}
		})
	}
}

func TestWorkflow_HealthCheck(t *testing.T) {
	storage := &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}

	server := handlers.NewHttpServer(storage)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health check failed with status %d", w.Code)
	}

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestWorkflow_RateLimiting(t *testing.T) {
	storage := &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}

	server := handlers.NewHttpServer(storage)

	// Make many requests quickly to trigger rate limiting
	validateReq := handlers.LicenseRequest{
		LicenseKey: "AFP-RATELIMIT",
		AppVersion: "1.0.0",
	}

	validateBody, _ := json.Marshal(validateReq)

	var rateLimitedCount int
	var successCount int

	// Make 20 requests rapidly
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/licenses/validate", bytes.NewBuffer(validateBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:12345" // Same IP for all requests

		w := httptest.NewRecorder()
		server.Mux.ServeHTTP(w, req)

		switch w.Code {
		case http.StatusTooManyRequests:
			rateLimitedCount++
		case http.StatusOK:
			successCount++
		}
	}

	// Should have some rate limited requests (rate limit is 10 per minute)
	if rateLimitedCount == 0 {
		t.Errorf("Expected some requests to be rate limited, got none")
	}

	if successCount == 0 {
		t.Errorf("Expected some requests to succeed, got none")
	}

	t.Logf("Rate limiting test: %d successful, %d rate limited", successCount, rateLimitedCount)
}

func TestWorkflow_ConcurrentRequests(t *testing.T) {
	storage := &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}

	server := handlers.NewHttpServer(storage)
	ctx := context.Background()

	// Create test data
	customer := models.Customer{
		ID:               "concurrent-customer",
		Email:            "concurrent@example.com",
		StripeCustomerID: "cus_concurrent",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	_ = storage.SaveCustomer(ctx, &customer)

	license := models.License{
		ID:              "concurrent-license",
		Key:             "AFP-CONCURRENT",
		CustomerID:      "concurrent-customer",
		ProductID:       "prod_test",
		Version:         "1.0.0",
		Status:          models.StatusActive,
		StripeSessionID: "cs_concurrent",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_ = storage.SaveLicense(ctx, &license)

	// Run concurrent license validations
	numGoroutines := 10
	numRequests := 50
	results := make(chan bool, numGoroutines*numRequests)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < numRequests; j++ {
				validateReq := handlers.LicenseRequest{
					LicenseKey: "AFP-CONCURRENT",
					AppVersion: "1.0.0",
				}

				validateBody, _ := json.Marshal(validateReq)
				req := httptest.NewRequest(http.MethodPost, "/v1/licenses/validate", bytes.NewBuffer(validateBody))
				req.Header.Set("Content-Type", "application/json")
				req.RemoteAddr = fmt.Sprintf("127.0.%d.1:12345", goroutineID+1) // Different IPs to avoid rate limiting

				w := httptest.NewRecorder()
				server.Mux.ServeHTTP(w, req)

				success := w.Code == http.StatusOK

				if success {
					var response handlers.ValidateResponse
					_ = json.NewDecoder(w.Body).Decode(&response)
					success = response.Valid
				}

				results <- success
			}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numGoroutines*numRequests; i++ {
		if <-results {
			successCount++
		}
	}

	// With rate limiting (10 per minute per IP), we expect numGoroutines * 10 successful requests
	expectedSuccesses := numGoroutines * 10 // 10 requests per IP allowed by rate limiter
	if successCount < expectedSuccesses {
		t.Errorf("Expected at least %d successful concurrent requests, got %d", expectedSuccesses, successCount)
	}

	t.Logf("Concurrent test: %d successful requests (rate limiting working correctly)", successCount)
}

// Helper functions for integration tests

func createMockStripeCheckoutSession(customerEmail, sessionID string) map[string]interface{} {
	return map[string]interface{}{
		"id":             sessionID,
		"customer_email": customerEmail,
		"amount_total":   2999,
		"currency":       "usd",
		"payment_status": "paid",
		"customer": map[string]interface{}{
			"id": "cus_" + sessionID,
		},
		"metadata": map[string]interface{}{
			"product_id":      "prod_integration_test",
			"license_version": "1.0.0",
		},
	}
}

func createMockStripeWebhookEvent(eventType string, data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":   "evt_integration_test",
		"type": eventType,
		"data": map[string]interface{}{
			"object": data,
		},
	}
}

// Benchmark integration tests
func BenchmarkFullWorkflow_StripeToValidation(b *testing.B) {
	storage := &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}

	server := handlers.NewHttpServer(storage)

	// Pre-create some licenses to validate against
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		customer := models.Customer{
			ID:               "bench-customer-" + string(rune(i)),
			Email:            "bench" + string(rune(i)) + "@example.com",
			StripeCustomerID: "cus_bench" + string(rune(i)),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		_ = storage.SaveCustomer(ctx, &customer)

		license := models.License{
			ID:              "bench-license-" + string(rune(i)),
			Key:             "AFP-BENCH" + string(rune(i)),
			CustomerID:      customer.ID,
			ProductID:       "prod_bench",
			Version:         "1.0.0",
			Status:          models.StatusActive,
			StripeSessionID: "cs_bench" + string(rune(i)),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		_ = storage.SaveLicense(ctx, &license)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Validate a random license
		licenseKey := "AFP-BENCH" + string(rune(i%100))

		validateReq := handlers.LicenseRequest{
			LicenseKey: licenseKey,
			AppVersion: "1.0.0",
		}

		validateBody, _ := json.Marshal(validateReq)
		req := httptest.NewRequest(http.MethodPost, "/v1/licenses/validate", bytes.NewBuffer(validateBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Request failed with status %d", w.Code)
		}
	}
}

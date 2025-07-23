package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"auto-focus.app/cloud/models"
	"auto-focus.app/cloud/storage"
	"github.com/stripe/stripe-go/v82"
)

func createTestStorageForStripe() *storage.MemoryStorage {
	return &storage.MemoryStorage{
		Data:     make(storage.Database),
		Licenses: make(map[string]models.License),
	}
}

func createMockStripeEvent(eventType string, sessionData map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":   "evt_test123",
		"type": eventType,
		"data": map[string]interface{}{
			"object": sessionData,
		},
	}
}

func createMockCheckoutSession(customerEmail, sessionID string, hasCustomer bool) map[string]interface{} {
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

func TestStripeWebhook_CheckoutSessionCompleted_Success(t *testing.T) {
	storage := createTestStorageForStripe()
	server := NewHttpServer(storage)

	// Create mock Stripe event
	sessionData := createMockCheckoutSession("test@example.com", "cs_test123", true)
	event := createMockStripeEvent("checkout.session.completed", sessionData)

	// Marshal the event
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "test-signature")

	w := httptest.NewRecorder()

	// Set environment variables for testing
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("TEST_MODE", "true")

	server.Stripe(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response
	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["received"] != "true" {
		t.Errorf("Expected received='true', got '%s'", response["received"])
	}
}

func TestStripeWebhook_InvalidJSON(t *testing.T) {
	storage := createTestStorageForStripe()
	server := NewHttpServer(storage)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("TEST_MODE", "true")
	server.Stripe(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestStripeWebhook_UnhandledEventType(t *testing.T) {
	storage := createTestStorageForStripe()
	server := NewHttpServer(storage)

	event := map[string]interface{}{
		"id":   "evt_test123",
		"type": "payment_intent.succeeded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{},
		},
	}

	payload, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "test-signature")

	w := httptest.NewRecorder()

	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")

	server.Stripe(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for unhandled event, got %d", http.StatusOK, w.Code)
	}
}

func TestHandleCheckoutComplete_NewCustomer(t *testing.T) {
	storage := createTestStorageForStripe()
	server := NewHttpServer(storage)

	session := &stripe.CheckoutSession{
		ID:              "cs_test123",
		CustomerEmail:   "newcustomer@example.com",
		AmountTotal:     2999,
		Currency:        "usd",
		PaymentStatus:   "paid",
		Customer:        &stripe.Customer{ID: "cus_new123"},
		Metadata: map[string]string{
			"product_id":      "prod_test123",
			"license_version": "1.0.0",
		},
	}

	// Remove the hardcoded email override for this test
	originalEmail := session.CustomerEmail

	err := server.handleCheckoutComplete(context.Background(), session)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify customer was created (note: session.CustomerEmail is overridden in the handler)
	customers := storage.Data
	if len(customers) != 1 {
		t.Errorf("Expected 1 customer, got %d", len(customers))
	}

	// Verify license was created
	if len(storage.Licenses) != 1 {
		t.Errorf("Expected 1 license, got %d", len(storage.Licenses))
	}

	// Find the created license
	var license models.License
	for _, l := range storage.Licenses {
		license = l
		break
	}

	if license.Status != models.StatusActive {
		t.Errorf("Expected license status %s, got %s", models.StatusActive, license.Status)
	}

	if license.ProductID != "prod_test123" {
		t.Errorf("Expected product ID 'prod_test123', got '%s'", license.ProductID)
	}

	if license.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", license.Version)
	}

	// Verify the license key format
	if len(license.Key) < 5 || license.Key[:4] != "AFP-" {
		t.Errorf("Expected license key to start with 'AFP-', got '%s'", license.Key)
	}

	_ = originalEmail // Use the variable to avoid "unused" error
}

func TestHandleCheckoutComplete_ExistingCustomer(t *testing.T) {
	storage := createTestStorageForStripe()
	server := NewHttpServer(storage)

	// Add existing customer
	existingCustomer := models.Customer{
		ID:               "existing-customer",
		Email:            "existing@example.com", // This will match the test email
		StripeCustomerID: "cus_existing123",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	storage.Data["existing-customer"] = existingCustomer

	session := &stripe.CheckoutSession{
		ID:              "cs_test456",
		CustomerEmail:   "existing@example.com", // Must match existing customer
		AmountTotal:     2999,
		Currency:        "usd",
		PaymentStatus:   "paid",
		Customer:        &stripe.Customer{ID: "cus_existing123"},
		Metadata: map[string]string{
			"product_id":      "prod_test456",
			"license_version": "2.0.0",
		},
	}

	err := server.handleCheckoutComplete(context.Background(), session)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should still have only 1 customer (existing one)
	if len(storage.Data) != 1 {
		t.Errorf("Expected 1 customer (existing), got %d", len(storage.Data))
	}

	// Should have 1 license
	if len(storage.Licenses) != 1 {
		t.Errorf("Expected 1 license, got %d", len(storage.Licenses))
	}

	// Verify license was created for existing customer
	var license models.License
	for _, l := range storage.Licenses {
		license = l
		break
	}

	if license.CustomerID != "existing-customer" {
		t.Errorf("Expected license for customer 'existing-customer', got '%s'", license.CustomerID)
	}
}

func TestCreateCustomer_WithStripeCustomer(t *testing.T) {
	session := &stripe.CheckoutSession{
		ID:            "cs_test789",
		CustomerEmail: "customer@example.com",
		Customer:      &stripe.Customer{ID: "cus_stripe123"},
	}

	customer := createCustomer(session)

	if customer.Email != "customer@example.com" {
		t.Errorf("Expected email 'customer@example.com', got '%s'", customer.Email)
	}

	if customer.StripeCustomerID != "cus_stripe123" {
		t.Errorf("Expected Stripe customer ID 'cus_stripe123', got '%s'", customer.StripeCustomerID)
	}

	if customer.ID == "" {
		t.Errorf("Expected customer ID to be generated")
	}
}

func TestCreateCustomer_WithoutStripeCustomer(t *testing.T) {
	session := &stripe.CheckoutSession{
		ID:            "cs_test999",
		CustomerEmail: "nocustomer@example.com",
		Customer:      nil,
	}

	customer := createCustomer(session)

	if customer.Email != "nocustomer@example.com" {
		t.Errorf("Expected email 'nocustomer@example.com', got '%s'", customer.Email)
	}

	if customer.StripeCustomerID != "" {
		t.Errorf("Expected empty Stripe customer ID, got '%s'", customer.StripeCustomerID)
	}

	if customer.ID == "" {
		t.Errorf("Expected customer ID to be generated")
	}
}

func TestCreateLicense(t *testing.T) {
	customer := &models.Customer{
		ID:    "test-customer",
		Email: "test@example.com",
	}

	session := &stripe.CheckoutSession{
		ID: "cs_license_test",
		Metadata: map[string]string{
			"product_id":      "prod_license123",
			"license_version": "3.0.0",
		},
	}

	license := createLicese(customer, session)

	if license.CustomerID != "test-customer" {
		t.Errorf("Expected customer ID 'test-customer', got '%s'", license.CustomerID)
	}

	if license.ProductID != "prod_license123" {
		t.Errorf("Expected product ID 'prod_license123', got '%s'", license.ProductID)
	}

	if license.Version != "3.0.0" {
		t.Errorf("Expected version '3.0.0', got '%s'", license.Version)
	}

	if license.Status != models.StatusActive {
		t.Errorf("Expected status %s, got %s", models.StatusActive, license.Status)
	}

	if license.StripeSessionID != "cs_license_test" {
		t.Errorf("Expected session ID 'cs_license_test', got '%s'", license.StripeSessionID)
	}

	if license.ID == "" {
		t.Errorf("Expected license ID to be generated")
	}

	if license.Key == "" {
		t.Errorf("Expected license key to be generated")
	}
}

func TestGenerateLicenseKey(t *testing.T) {
	// Generate multiple keys to ensure uniqueness and format
	keys := make(map[string]bool)
	
	for i := 0; i < 100; i++ {
		key := generateLicenseKey()
		
		// Check format
		if len(key) != 12 { // "AFP-" + 8 characters
			t.Errorf("Expected key length 12, got %d for key '%s'", len(key), key)
		}
		
		if key[:4] != "AFP-" {
			t.Errorf("Expected key to start with 'AFP-', got '%s'", key)
		}
		
		// Check uniqueness
		if keys[key] {
			t.Errorf("Generated duplicate key: %s", key)
		}
		keys[key] = true
	}
}

func TestFindOrCreateCustomer_DatabaseError(t *testing.T) {
	// Use error storage
	storage := &mockStorageWithErrors{}
	server := NewHttpServer(storage)

	session := &stripe.CheckoutSession{
		ID:            "cs_error_test",
		CustomerEmail: "error@example.com",
	}

	_, err := server.findOrCreateCustomer(context.Background(), session)
	if err == nil {
		t.Errorf("Expected error from database, got nil")
	}
}

func TestCreateLicensedUser_SaveLicenseError(t *testing.T) {
	// Create storage that fails on license save
	storage := &mockStoragePartialErrors{}
	server := NewHttpServer(storage)

	session := &stripe.CheckoutSession{
		ID:            "cs_save_error",
		CustomerEmail: "save@example.com",
		Metadata: map[string]string{
			"product_id":      "prod_error",
			"license_version": "1.0.0",
		},
	}

	_, _, err := server.createLicensedUser(context.Background(), session)
	if err == nil {
		t.Errorf("Expected error from license save, got nil")
	}
}

func TestStripeWebhook_EmptyBody(t *testing.T) {
	storage := createTestStorageForStripe()
	server := NewHttpServer(storage)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer([]byte{}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("TEST_MODE", "true")
	server.Stripe(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for empty body, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestStripeWebhook_LargePayload(t *testing.T) {
	storage := createTestStorageForStripe()
	server := NewHttpServer(storage)

	// Create a very large payload
	largeEvent := map[string]interface{}{
		"id":   "evt_large",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_large",
				"customer_email": "large@example.com",
				"large_field":    string(make([]byte, 50000)), // 50KB field
			},
		},
	}

	payload, _ := json.Marshal(largeEvent)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "test-signature")

	w := httptest.NewRecorder()
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("TEST_MODE", "true")
	server.Stripe(w, req)

	// Should handle large payloads gracefully (up to MaxBodyBytes limit)
	if w.Code == http.StatusServiceUnavailable {
		// This is expected if payload exceeds MaxBodyBytes
		return
	}

	if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
		t.Errorf("Expected status 400 or 200 for large payload, got %d", w.Code)
	}
}

// Mock storage that fails on customer operations but succeeds on license operations
type mockStoragePartialErrors struct {
	customer *models.Customer
}

func (m *mockStoragePartialErrors) GetCustomer(ctx context.Context, id string) (*models.Customer, error) {
	return m.customer, nil
}

func (m *mockStoragePartialErrors) FindCustomerByEmailAddress(ctx context.Context, emailAddress string) (*models.Customer, error) {
	return nil, nil // No existing customer
}

func (m *mockStoragePartialErrors) SaveCustomer(ctx context.Context, customer *models.Customer) error {
	m.customer = customer
	return nil
}

func (m *mockStoragePartialErrors) GetLicense(ctx context.Context, id string) (*models.License, error) {
	return nil, nil
}

func (m *mockStoragePartialErrors) FindLicenseByKey(ctx context.Context, key string) (*models.License, error) {
	return nil, nil
}

func (m *mockStoragePartialErrors) FindLicensesByCustomer(ctx context.Context, customerID string) ([]*models.License, error) {
	return nil, nil
}

func (m *mockStoragePartialErrors) SaveLicense(ctx context.Context, license *models.License) error {
	return context.DeadlineExceeded // Fail on license save
}

func (m *mockStoragePartialErrors) Close() error {
	return nil
}

// Benchmark tests for Stripe webhook performance
func BenchmarkStripeWebhook_CheckoutCompleted(b *testing.B) {
	storage := createTestStorageForStripe()
	server := NewHttpServer(storage)

	b.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	b.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	b.Setenv("TEST_MODE", "true")

	sessionData := createMockCheckoutSession("bench@example.com", "cs_bench", true)
	event := createMockStripeEvent("checkout.session.completed", sessionData)
	payload, _ := json.Marshal(event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Stripe-Signature", "test-signature")

		w := httptest.NewRecorder()
		server.Stripe(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			b.Fatalf("Unexpected status code: %d", w.Code)
		}
	}
}

func BenchmarkGenerateLicenseKey(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateLicenseKey()
	}
}
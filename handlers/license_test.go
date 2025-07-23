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
)

// Test helper to create a mock storage with test data
func createTestStorage() *storage.MemoryStorage {
	storage := &storage.MemoryStorage{
		Data: make(storage.Database),
	}
	
	// Add test customer
	testCustomer := models.Customer{
		ID:               "test-customer-1",
		Email:            "test@example.com",
		StripeCustomerID: "cus_test123",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	
	storage.Data["test-customer-1"] = testCustomer
	
	// Initialize licenses map
	if storage.Licenses == nil {
		storage.Licenses = make(map[string]models.License)
	}
	
	// Add test licenses
	storage.Licenses["license-1"] = models.License{
		ID:         "license-1",
		Key:        "AFP-VALID123",
		CustomerID: "test-customer-1",
		ProductID:  "prod_test123",
		Version:    "1.0.0",
		Status:     models.StatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	storage.Licenses["license-2"] = models.License{
		ID:         "license-2",
		Key:        "AFP-SUSPENDED",
		CustomerID: "test-customer-1",
		ProductID:  "prod_test123",
		Version:    "1.0.0",
		Status:     models.StatusSuspended,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	return storage
}

func TestValidateLicense_Success(t *testing.T) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	reqBody := LicenseRequest{
		LicenseKey: "AFP-VALID123",
		AppVersion: "1.4.11",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ValidateResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Valid {
		t.Errorf("Expected valid license, got invalid")
	}

	if response.Message != "license valid" {
		t.Errorf("Expected message 'license valid', got '%s'", response.Message)
	}
}

func TestValidateLicense_LicenseNotFound(t *testing.T) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	reqBody := LicenseRequest{
		LicenseKey: "AFP-NOTFOUND",
		AppVersion: "1.0.0",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ValidateResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Valid {
		t.Errorf("Expected invalid license")
	}

	if response.Message != "license not found" {
		t.Errorf("Expected message 'license not found', got '%s'", response.Message)
	}
}

func TestValidateLicense_LicenseNotActive(t *testing.T) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	reqBody := LicenseRequest{
		LicenseKey: "AFP-SUSPENDED",
		AppVersion: "1.0.0",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ValidateResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Valid {
		t.Errorf("Expected invalid license due to suspended status")
	}

	if response.Message != "license not active" {
		t.Errorf("Expected message 'license not active', got '%s'", response.Message)
	}
}

func TestValidateLicense_InvalidMethods(t *testing.T) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	methods := []string{"GET", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/licenses/validate", nil)
			w := httptest.NewRecorder()

			server.ValidateLicense(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for method %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != "only POST allowed" {
				t.Errorf("Expected error 'only POST allowed', got '%s'", response["error"])
			}
		})
	}
}

func TestValidateLicense_InvalidJSON(t *testing.T) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "empty body",
			body:    "",
			wantErr: "empty body",
		},
		{
			name:    "invalid json",
			body:    "invalid json",
			wantErr: "empty body",
		},
		{
			name:    "malformed json",
			body:    `{"license_key": "test"`,
			wantErr: "empty body",
		},
		{
			name:    "null body",
			body:    "null",
			wantErr: "invalid license",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.ValidateLicense(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != tt.wantErr {
				t.Errorf("Expected error '%s', got '%s'", tt.wantErr, response["error"])
			}
		})
	}
}

func TestValidateLicense_RequestValidation(t *testing.T) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	tests := []struct {
		name        string
		request     LicenseRequest
		expectedErr string
	}{
		{
			name: "missing license key",
			request: LicenseRequest{
				AppVersion: "1.0.0",
			},
			expectedErr: "invalid license",
		},
		{
			name: "empty license key",
			request: LicenseRequest{
				LicenseKey: "",
				AppVersion: "1.0.0",
			},
			expectedErr: "invalid license",
		},
		{
			name: "whitespace license key",
			request: LicenseRequest{
				LicenseKey: "   ",
				AppVersion: "1.0.0",
			},
			expectedErr: "invalid license",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.ValidateLicense(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != tt.expectedErr {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedErr, response["error"])
			}
		})
	}
}

func TestValidateLicense_DatabaseError(t *testing.T) {
	// Create a storage that will return errors
	storage := &mockStorageWithErrors{}
	server := NewHttpServer(storage)

	reqBody := LicenseRequest{
		LicenseKey: "AFP-TEST123",
		AppVersion: "1.0.0",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "something went wrong" {
		t.Errorf("Expected error 'something went wrong', got '%s'", response["error"])
	}
}

func TestValidateLicense_ContentTypeAndHeaders(t *testing.T) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	reqBody := LicenseRequest{
		LicenseKey: "AFP-VALID123",
		AppVersion: "1.0.0",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	// Check Content-Type header is set
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestValidateLicense_LargePayload(t *testing.T) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	// Create a large license key
	largeLicenseKey := make([]byte, 10000)
	for i := range largeLicenseKey {
		largeLicenseKey[i] = 'A'
	}

	reqBody := LicenseRequest{
		LicenseKey: string(largeLicenseKey),
		AppVersion: "1.0.0",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	// Should still process (though license won't be found)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for large payload, got %d", http.StatusOK, w.Code)
	}
}

func TestLicenseRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request LicenseRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: LicenseRequest{
				LicenseKey: "AFP-VALID123",
				AppVersion: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "empty license key",
			request: LicenseRequest{
				LicenseKey: "",
				AppVersion: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "missing app version is ok",
			request: LicenseRequest{
				LicenseKey: "AFP-VALID123",
				AppVersion: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LicenseRequest.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRespondWithValidation(t *testing.T) {
	tests := []struct {
		name    string
		valid   bool
		message string
	}{
		{
			name:    "valid response",
			valid:   true,
			message: "license valid",
		},
		{
			name:    "invalid response",
			valid:   false,
			message: "license not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondWithValidation(w, tt.valid, tt.message)

			var response ValidateResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.valid, response.Valid)
			}

			if response.Message != tt.message {
				t.Errorf("Expected message '%s', got '%s'", tt.message, response.Message)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

func TestWriteErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
	}{
		{
			name:    "bad request",
			status:  http.StatusBadRequest,
			message: "invalid request",
		},
		{
			name:    "internal server error",
			status:  http.StatusInternalServerError,
			message: "something went wrong",
		},
		{
			name:    "method not allowed",
			status:  http.StatusMethodNotAllowed,
			message: "only POST allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeErrorResponse(w, tt.status, tt.message)

			if w.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, w.Code)
			}

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("Expected error '%s', got '%s'", tt.message, response["error"])
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

// Mock storage that returns errors for testing error handling
type mockStorageWithErrors struct{}

func (m *mockStorageWithErrors) GetCustomer(ctx context.Context, id string) (*models.Customer, error) {
	return nil, context.DeadlineExceeded
}

func (m *mockStorageWithErrors) FindCustomerByEmailAddress(ctx context.Context, emailAddress string) (*models.Customer, error) {
	return nil, context.DeadlineExceeded
}

func (m *mockStorageWithErrors) SaveCustomer(ctx context.Context, customer *models.Customer) error {
	return context.DeadlineExceeded
}

func (m *mockStorageWithErrors) GetLicense(ctx context.Context, id string) (*models.License, error) {
	return nil, context.DeadlineExceeded
}

func (m *mockStorageWithErrors) FindLicenseByKey(ctx context.Context, key string) (*models.License, error) {
	return nil, context.DeadlineExceeded
}

func (m *mockStorageWithErrors) FindLicensesByCustomer(ctx context.Context, customerID string) ([]*models.License, error) {
	return nil, context.DeadlineExceeded
}

func (m *mockStorageWithErrors) SaveLicense(ctx context.Context, license *models.License) error {
	return context.DeadlineExceeded
}

func (m *mockStorageWithErrors) Close() error {
	return nil
}

// Benchmark tests
func BenchmarkValidateLicense_Success(b *testing.B) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	reqBody := LicenseRequest{
		LicenseKey: "AFP-VALID123",
		AppVersion: "1.0.0",
	}

	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.ValidateLicense(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

func BenchmarkValidateLicense_NotFound(b *testing.B) {
	storage := createTestStorage()
	server := NewHttpServer(storage)

	reqBody := LicenseRequest{
		LicenseKey: "AFP-NOTFOUND",
		AppVersion: "1.0.0",
	}

	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.ValidateLicense(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"auto-focus.app/cloud/models"
	"auto-focus.app/cloud/storage"
)

func TestValidateLicense_Success(t *testing.T) {
	// Setup test data
	db := &storage.MemoryStorage{
		Data: storage.Database{
			"1": models.Customer{
				Id:    "1",
				Email: "test@example.com",
				Licenses: []models.License{
					{Key: "VALID-TEST-KEY", Version: "1.0.0"},
				},
			},
		},
	}

	server := NewHttpServer(db)

	// Create request
	reqBody := LicenseRequest{
		LicenseKey: "VALID-TEST-KEY",
		AppVersion: "1.0.0",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var customer models.Customer
	err = json.NewDecoder(w.Body).Decode(&customer)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if customer.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", customer.Email)
	}
}

func TestValidateLicense_LicenseNotFound(t *testing.T) {
	// Setup empty database
	db := &storage.MemoryStorage{}
	server := NewHttpServer(db)

	reqBody := LicenseRequest{
		LicenseKey: "NON-EXISTENT-KEY",
		AppVersion: "1.0.0",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "License not found" {
		t.Errorf("Expected error 'License not found', got '%s'", response["error"])
	}
}

func TestValidateLicense_ValidationErrors(t *testing.T) {
	db := &storage.MemoryStorage{}
	server := NewHttpServer(db)

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
			expectedErr: "Invalid license",
		},
		{
			name: "missing app version",
			request: LicenseRequest{
				LicenseKey: "TEST-KEY",
			},
			expectedErr: "Invalid license",
		},
		{
			name: "empty license key",
			request: LicenseRequest{
				LicenseKey: "",
				AppVersion: "1.0.0",
			},
			expectedErr: "Invalid license",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.ValidateLicense(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			var response map[string]string
			err = json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != tt.expectedErr {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedErr, response["error"])
			}
		})
	}
}

func TestValidateLicense_InvalidJSON(t *testing.T) {
	db := &storage.MemoryStorage{}
	server := NewHttpServer(db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBufferString("invalid json"))
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

	if response["error"] != "Empty body" {
		t.Errorf("Expected error 'Empty body', got '%s'", response["error"])
	}
}

func TestValidateLicense_WrongHTTPMethod(t *testing.T) {
	db := &storage.MemoryStorage{}
	server := NewHttpServer(db)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/licenses/validate", nil)
			w := httptest.NewRecorder()

			server.ValidateLicense(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s method, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != "Only POST allowed" {
				t.Errorf("Expected error 'Only POST allowed', got '%s'", response["error"])
			}
		})
	}
}

func TestValidateLicense_ContentTypeHeader(t *testing.T) {
	db := &storage.MemoryStorage{
		Data: storage.Database{
			"1": models.Customer{
				Id:    "1",
				Email: "test@example.com",
				Licenses: []models.License{
					{Key: "TEST-KEY", Version: "1.0.0"},
				},
			},
		},
	}
	server := NewHttpServer(db)

	reqBody := LicenseRequest{
		LicenseKey: "TEST-KEY",
		AppVersion: "1.0.0",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ValidateLicense(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestFindLicenseCustomer(t *testing.T) {
	db := &storage.MemoryStorage{
		Data: storage.Database{
			"1": models.Customer{
				Id:    "1",
				Email: "customer1@example.com",
				Licenses: []models.License{
					{Key: "KEY-001", Version: "1.0.0"},
					{Key: "KEY-002", Version: "1.1.0"},
				},
			},
			"2": models.Customer{
				Id:    "2",
				Email: "customer2@example.com",
				Licenses: []models.License{
					{Key: "KEY-003", Version: "2.0.0"},
				},
			},
		},
	}

	server := NewHttpServer(db)

	tests := []struct {
		name       string
		licenseKey string
		expectedID string
		shouldFind bool
	}{
		{
			name:       "find first customer's first license",
			licenseKey: "KEY-001",
			expectedID: "1",
			shouldFind: true,
		},
		{
			name:       "find first customer's second license",
			licenseKey: "KEY-002",
			expectedID: "1",
			shouldFind: true,
		},
		{
			name:       "find second customer's license",
			licenseKey: "KEY-003",
			expectedID: "2",
			shouldFind: true,
		},
		{
			name:       "license not found",
			licenseKey: "NON-EXISTENT",
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := server.findLicenseCustomer(tt.licenseKey)

			if tt.shouldFind {
				if customer == nil {
					t.Errorf("Expected to find customer, got nil")
					return
				}
				if customer.Id != tt.expectedID {
					t.Errorf("Expected customer ID '%s', got '%s'", tt.expectedID, customer.Id)
				}
			} else {
				if customer != nil {
					t.Errorf("Expected nil customer, got customer with ID '%s'", customer.Id)
				}
			}
		})
	}
}

func TestLicenseRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request LicenseRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: LicenseRequest{
				LicenseKey: "VALID-KEY",
				AppVersion: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing license key",
			request: LicenseRequest{
				AppVersion: "1.0.0",
			},
			wantErr: true,
			errMsg:  "license_key required",
		},
		{
			name: "empty license key",
			request: LicenseRequest{
				LicenseKey: "",
				AppVersion: "1.0.0",
			},
			wantErr: true,
			errMsg:  "license_key required",
		},
		{
			name: "missing app version",
			request: LicenseRequest{
				LicenseKey: "VALID-KEY",
			},
			wantErr: true,
			errMsg:  "app_version required",
		},
		{
			name: "empty app version",
			request: LicenseRequest{
				LicenseKey: "VALID-KEY",
				AppVersion: "",
			},
			wantErr: true,
			errMsg:  "app_version required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

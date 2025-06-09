package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"auto-focus.app/cloud/internal/version"
	"auto-focus.app/cloud/models"
	"auto-focus.app/cloud/storage"
)

func TestValidateLicense_Success(t *testing.T) {
	db := &storage.MemoryStorage{
		Data: storage.Database{
			"1": models.Customer{
				Id:    "1",
				Email: "test@example.com",
				Licenses: []models.License{
					{Key: "VALID-KEY", Version: "1.0.0", Status: models.StatusActive},
				},
			},
		},
	}

	server := NewHttpServer(db)

	reqBody := LicenseRequest{
		LicenseKey: "VALID-KEY",
		AppVersion: "1.4.11", // Compatible with 1.0.0 license
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

	if response.Message != "License valid" {
		t.Errorf("Expected message 'License valid', got '%s'", response.Message)
	}
}

func TestValidateLicense_VersionIncompatible(t *testing.T) {
	tests := []struct {
		name           string
		licenseVersion string
		appVersion     string
		expectedValid  bool
		expectedMsg    string
	}{
		{
			name:           "v1 license with v2 app - should fail",
			licenseVersion: "1.0.0",
			appVersion:     "2.0.0",
			expectedValid:  false,
			expectedMsg:    "License not valid for this app version",
		},
		{
			name:           "v2 license with v1 app - should fail",
			licenseVersion: "2.0.0",
			appVersion:     "1.4.11",
			expectedValid:  false,
			expectedMsg:    "License not valid for this app version",
		},
		{
			name:           "v1 license with v1 patch - should succeed",
			licenseVersion: "1.0.0",
			appVersion:     "1.4.11",
			expectedValid:  true,
			expectedMsg:    "License valid",
		},
		{
			name:           "v2 license with v2 minor - should succeed",
			licenseVersion: "2.0.0",
			appVersion:     "2.5.3",
			expectedValid:  true,
			expectedMsg:    "License valid",
		},
		{
			name:           "same versions - should succeed",
			licenseVersion: "1.2.3",
			appVersion:     "1.2.3",
			expectedValid:  true,
			expectedMsg:    "License valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &storage.MemoryStorage{
				Data: storage.Database{
					"1": models.Customer{
						Id:    "1",
						Email: "test@example.com",
						Licenses: []models.License{
							{Key: "TEST-KEY", Version: tt.licenseVersion, Status: models.StatusActive},
						},
					},
				},
			}

			server := NewHttpServer(db)

			reqBody := LicenseRequest{
				LicenseKey: "TEST-KEY",
				AppVersion: tt.appVersion,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.ValidateLicense(w, req)

			var response ValidateResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectedValid, response.Valid)
			}

			if response.Message != tt.expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMsg, response.Message)
			}
		})
	}
}

func TestValidateLicense_InvalidVersionFormats(t *testing.T) {
	tests := []struct {
		name           string
		licenseVersion string
		appVersion     string
		expectedMsg    string
	}{
		{
			name:           "empty license version",
			licenseVersion: "",
			appVersion:     "1.0.0",
			expectedMsg:    "Invalid version format",
		},
		{
			name:           "empty app version",
			licenseVersion: "1.0.0",
			appVersion:     "",
			expectedMsg:    "Invalid version format",
		},
		{
			name:           "invalid license version format",
			licenseVersion: "invalid",
			appVersion:     "1.0.0",
			expectedMsg:    "Invalid version format",
		},
		{
			name:           "invalid app version format",
			licenseVersion: "1.0.0",
			appVersion:     "invalid",
			expectedMsg:    "Invalid version format",
		},
		{
			name:           "negative license version",
			licenseVersion: "-1.0.0",
			appVersion:     "1.0.0",
			expectedMsg:    "Invalid version format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &storage.MemoryStorage{
				Data: storage.Database{
					"1": models.Customer{
						Id:    "1",
						Email: "test@example.com",
						Licenses: []models.License{
							{Key: "TEST-KEY", Version: tt.licenseVersion, Status: models.StatusActive},
						},
					},
				},
			}

			server := NewHttpServer(db)

			reqBody := LicenseRequest{
				LicenseKey: "TEST-KEY",
				AppVersion: tt.appVersion,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.ValidateLicense(w, req)

			var response ValidateResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Valid {
				t.Errorf("Expected invalid license due to version format")
			}

			if response.Message != tt.expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMsg, response.Message)
			}
		})
	}
}

func TestValidateLicense_StatusChecks(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		expectedValid bool
		expectedMsg   string
	}{
		{
			name:          "active license",
			status:        models.StatusActive,
			expectedValid: true,
			expectedMsg:   "License valid",
		},
		{
			name:          "suspended license",
			status:        models.StatusSuspended,
			expectedValid: false,
			expectedMsg:   "License not active",
		},
		{
			name:          "empty status",
			status:        "",
			expectedValid: false,
			expectedMsg:   "License not active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &storage.MemoryStorage{
				Data: storage.Database{
					"1": models.Customer{
						Id:    "1",
						Email: "test@example.com",
						Licenses: []models.License{
							{Key: "TEST-KEY", Version: "1.0.0", Status: tt.status},
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

			var response ValidateResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectedValid, response.Valid)
			}

			if response.Message != tt.expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMsg, response.Message)
			}
		})
	}
}

func TestValidateLicense_LicenseNotFound(t *testing.T) {
	db := &storage.MemoryStorage{Data: storage.Database{}}
	server := NewHttpServer(db)

	reqBody := LicenseRequest{
		LicenseKey: "NON-EXISTENT-KEY",
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

	if response.Message != "License not found" {
		t.Errorf("Expected message 'License not found', got '%s'", response.Message)
	}
}

func TestValidateLicense_RequestValidation(t *testing.T) {
	db := &storage.MemoryStorage{Data: storage.Database{}}
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

// Test the version utility functions directly
func TestExtractMajorVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expected    int
		expectError bool
	}{
		{
			name:     "valid version 1.0.0",
			version:  "1.0.0",
			expected: 1,
		},
		{
			name:     "valid version 2.5.3",
			version:  "2.5.3",
			expected: 2,
		},
		{
			name:     "valid version 10.1.2",
			version:  "10.1.2",
			expected: 10,
		},
		{
			name:     "version with only major",
			version:  "3",
			expected: 3,
		},
		{
			name:        "empty version",
			version:     "",
			expectError: true,
		},
		{
			name:        "invalid format",
			version:     "abc.def.ghi",
			expectError: true,
		},
		{
			name:        "negative version",
			version:     "-1.0.0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := version.ExtractMajorVersion(tt.version)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for version '%s', got none", tt.version)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for version '%s', got: %v", tt.version, err)
				}
				if result != tt.expected {
					t.Errorf("Expected major version %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestIsVersionCompatible(t *testing.T) {
	tests := []struct {
		name             string
		licenseVersion   string
		requestedVersion string
		expected         bool
		expectError      bool
	}{
		{
			name:             "same major version",
			licenseVersion:   "1.0.0",
			requestedVersion: "1.4.11",
			expected:         true,
		},
		{
			name:             "different major version",
			licenseVersion:   "1.0.0",
			requestedVersion: "2.0.0",
			expected:         false,
		},
		{
			name:             "exact same version",
			licenseVersion:   "2.1.3",
			requestedVersion: "2.1.3",
			expected:         true,
		},
		{
			name:             "invalid license version",
			licenseVersion:   "invalid",
			requestedVersion: "1.0.0",
			expectError:      true,
		},
		{
			name:             "invalid app version",
			licenseVersion:   "1.0.0",
			requestedVersion: "invalid",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := version.IsCompatible(tt.licenseVersion, tt.requestedVersion)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected compatibility %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

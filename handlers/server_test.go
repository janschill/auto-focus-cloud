package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"auto-focus.app/cloud/models"
	"auto-focus.app/cloud/storage"
)

func TestNewHttpServer(t *testing.T) {
	db := storage.Database{
		"1": models.Customer{
			Id:    "1",
			Email: "test@example.com",
			Licenses: []models.License{
				{Key: "TEST-KEY", Version: "1.0.0"},
			},
		},
	}

	server := NewHttpServer(db)

	if server == nil {
		t.Fatalf("Expected server to be created, got nil")
	}

	if server.Mux == nil {
		t.Errorf("Expected mux to be initialized")
	}

	if server.Storage == nil {
		t.Errorf("Expected storage to be initialized")
	}

	// Verify the database was properly assigned
	customer, exists := server.Storage["1"]
	if !exists {
		t.Errorf("Expected customer '1' to exist in server storage")
	}

	if customer.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", customer.Email)
	}
}

func TestServer_HealthEndpoint(t *testing.T) {
	db := storage.Database{}
	server := NewHttpServer(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify response body contains status
	body := w.Body.String()
	if body == "" {
		t.Errorf("Expected non-empty response body")
	}

	// Should contain JSON with status field
	expectedContent := `{"status":"ok"}`
	if body != expectedContent+"\n" { // json.Encoder adds newline
		t.Errorf("Expected response '%s', got '%s'", expectedContent, body)
	}
}

func TestServer_RoutingConfiguration(t *testing.T) {
	db := storage.Database{}
	server := NewHttpServer(db)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "health endpoint - GET",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "health endpoint - POST should work too",
			method:         http.MethodPost,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "license validate - GET should fail",
			method:         http.MethodGet,
			path:           "/api/v1/licenses/validate",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "license validate - POST should work",
			method:         http.MethodPost,
			path:           "/api/v1/licenses/validate",
			expectedStatus: http.StatusBadRequest, // Bad request due to empty body
		},
		{
			name:           "non-existent endpoint",
			method:         http.MethodGet,
			path:           "/non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			server.Mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestServer_StorageReference(t *testing.T) {
	// Test that server maintains reference to original database
	db := storage.Database{
		"original": models.Customer{
			Id:    "original",
			Email: "original@example.com",
			Licenses: []models.License{
				{Key: "ORIGINAL-KEY", Version: "1.0.0"},
			},
		},
	}

	server := NewHttpServer(db)

	// Modify original database
	db["new"] = models.Customer{
		Id:    "new",
		Email: "new@example.com",
		Licenses: []models.License{
			{Key: "NEW-KEY", Version: "1.0.0"},
		},
	}

	// Server should see the change (since it's the same map)
	_, exists := server.Storage["new"]
	if !exists {
		t.Errorf("Expected server to see new customer added to original database")
	}

	// Modify through server
	server.Storage["server-added"] = models.Customer{
		Id:    "server-added",
		Email: "server@example.com",
		Licenses: []models.License{
			{Key: "SERVER-KEY", Version: "1.0.0"},
		},
	}

	// Original database should see the change
	_, exists = db["server-added"]
	if !exists {
		t.Errorf("Expected original database to see customer added through server")
	}
}

func TestServer_EmptyDatabase(t *testing.T) {
	// Test server creation with empty database
	db := storage.Database{}
	server := NewHttpServer(db)

	if server == nil {
		t.Fatalf("Expected server to be created with empty database")
	}

	if len(server.Storage) != 0 {
		t.Errorf("Expected empty storage, got %d items", len(server.Storage))
	}

	// Test that endpoints still work with empty database
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected health endpoint to work with empty database, got status %d", w.Code)
	}
}

func TestServer_NilDatabase(t *testing.T) {
	var db storage.Database
	server := NewHttpServer(db)

	if server == nil {
		t.Fatalf("Expected server to be created with nil database")
	}

	// In Go, when you assign a nil map to a struct field, it stays nil
	// But we can still call len() on it (returns 0) and range over it safely
	if len(server.Storage) != 0 {
		t.Errorf("Expected nil storage to have length 0, got %d", len(server.Storage))
	}

	// Verify we can safely iterate over nil map
	count := 0
	for range server.Storage {
		count++
	}
	if count != 0 {
		t.Errorf("Expected 0 iterations over nil map, got %d", count)
	}
}

func TestServer_MultipleInstances(t *testing.T) {
	// Test creating multiple server instances
	db1 := storage.Database{
		"server1": models.Customer{Id: "server1", Email: "server1@example.com"},
	}

	db2 := storage.Database{
		"server2": models.Customer{Id: "server2", Email: "server2@example.com"},
	}

	server1 := NewHttpServer(db1)
	server2 := NewHttpServer(db2)

	// Verify servers are independent
	_, exists1 := server1.Storage["server2"]
	if exists1 {
		t.Errorf("Expected server1 to not have server2's data")
	}

	_, exists2 := server2.Storage["server1"]
	if exists2 {
		t.Errorf("Expected server2 to not have server1's data")
	}

	// Verify each server has its own data
	_, exists1 = server1.Storage["server1"]
	if !exists1 {
		t.Errorf("Expected server1 to have its own data")
	}

	_, exists2 = server2.Storage["server2"]
	if !exists2 {
		t.Errorf("Expected server2 to have its own data")
	}
}

func TestServer_HTTPMethodSupport(t *testing.T) {
	db := storage.Database{}
	server := NewHttpServer(db)

	// Test that health endpoint supports various HTTP methods
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			w := httptest.NewRecorder()

			server.Health(w, req)

			// Health endpoint should respond to all methods
			if w.Code != http.StatusOK {
				t.Errorf("Expected health endpoint to respond to %s method with 200, got %d", method, w.Code)
			}
		})
	}
}

// Benchmark server creation
func BenchmarkNewHttpServer(b *testing.B) {
	db := storage.Database{
		"benchmark": models.Customer{
			Id:    "benchmark",
			Email: "benchmark@example.com",
			Licenses: []models.License{
				{Key: "BENCHMARK-KEY", Version: "1.0.0"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewHttpServer(db)
	}
}

// Benchmark health endpoint
func BenchmarkServer_Health(b *testing.B) {
	db := storage.Database{}
	server := NewHttpServer(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		server.Health(w, req)
	}
}

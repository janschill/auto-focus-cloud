package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"auto-focus.app/cloud/models"
)

type LicenseRequest struct {
	LicenseKey string `json:"license_key"`
	AppVersion string `json:"app_version"`
}

type ValidateResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

func (s *Server) ValidateLicense(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Only POST allowed")
		return
	}
	var req LicenseRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Empty body")
		return
	}

	err := req.validate()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid license")
		return
	}

	customer := s.findLicenseCustomer(req.LicenseKey)
	if customer == nil {
		respondWithValidation(w, false, "License not found")
		return
	}
	license := findLicenseInCustomer(customer, req.LicenseKey)
	if license.Status != models.StatusActive {
		respondWithValidation(w, false, "License not active")
		return
	}

	// Check version compatibility
	compatible, err := isVersionCompatible(license.Version, req.AppVersion)
	if err != nil {
		respondWithValidation(w, false, "Invalid version format")
		return
	}

	if !compatible {
		respondWithValidation(w, false, "License not valid for this app version")
		return
	}

	respondWithValidation(w, true, "License valid")
}

func findLicenseInCustomer(customer *models.Customer, licenseKey string) *models.License {
	for _, license := range customer.Licenses {
		if license.Key == licenseKey {
			return &license
		}
	}
	return nil
}

func respondWithValidation(w http.ResponseWriter, valid bool, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ValidateResponse{
		Valid:   valid,
		Message: message,
	})
}

func (s *Server) findLicenseCustomer(licenseKey string) *models.Customer {
	customer, err := s.Storage.FindCustomerByLicenseKey(licenseKey)
	if err != nil {
		return nil
	}
	return customer
}

func (lr LicenseRequest) validate() error {
	if lr.LicenseKey == "" {
		return fmt.Errorf("license_key required")
	}
	// Empty app_version will be caught by version validation logic
	return nil
}

func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// isVersionCompatible checks if the app version is compatible with the license version
// License is valid for same major version (1.x.x works with 1.y.z, but not 2.x.x)
func isVersionCompatible(licenseVersion, requestedVersion string) (bool, error) {
	licenseMajor, err := extractMajorVersion(licenseVersion)
	if err != nil {
		return false, fmt.Errorf("invalid license version: %v", err)
	}
	
	requestedMajor, err := extractMajorVersion(requestedVersion)
	if err != nil {
		return false, fmt.Errorf("invalid app version: %v", err)
	}
	
	return licenseMajor == requestedMajor, nil
}

// extractMajorVersion extracts the major version number from a semantic version string
func extractMajorVersion(version string) (int, error) {
	if version == "" {
		return 0, fmt.Errorf("empty version string")
	}
	
	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid version format")
	}
	
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid major version: %v", err)
	}
	
	if major < 0 {
		return 0, fmt.Errorf("major version cannot be negative")
	}
	
	return major, nil
}

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"auto-focus.app/cloud/models"
)

type LicenseRequest struct {
	LicenseKey string `json:"license_key"`
	AppVersion string `json:"app_version"`
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

	licensedCustomer := s.findLicenseCustomer(req.LicenseKey)
	if licensedCustomer == nil {
		writeErrorResponse(w, http.StatusNotFound, "License not found")
		return
	}

	json.NewEncoder(w).Encode(licensedCustomer)
}

func (s *Server) findLicenseCustomer(licenseKey string) *models.Customer {
	for _, customer := range s.Storage {
		for _, license := range customer.Licenses {
			if license.Key == licenseKey {
				return &customer
			}
		}
	}
	return nil
}

func (lr LicenseRequest) validate() error {
	if lr.LicenseKey == "" {
		return fmt.Errorf("license_key required")
	}
	if lr.AppVersion == "" {
		return fmt.Errorf("app_version required")
	}
	return nil
}

func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

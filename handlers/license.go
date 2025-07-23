package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"auto-focus.app/cloud/internal/logger"
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
	ctx := r.Context()

	logger.Info("License validation received", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.Header.Get("User-Agent"),
		"method":      r.Method,
	})

	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		logger.Info("Non POST request received", map[string]interface{}{})
		writeErrorResponse(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}
	var req LicenseRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		logger.Info("Empty body request received", map[string]interface{}{})
		writeErrorResponse(w, http.StatusBadRequest, "empty body")
		return
	}

	err := req.validate()
	if err != nil {
		logger.Info("Invalid license", map[string]interface{}{
			"error": err.Error(),
		})
		writeErrorResponse(w, http.StatusBadRequest, "invalid license")
		return
	}

	license, err := s.Storage.FindLicenseByKey(ctx, req.LicenseKey)
	if err != nil {
		logger.Error("Error while fetch license", map[string]interface{}{
			"error": err.Error(),
		})
		writeErrorResponse(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	if license == nil {
		logger.Warn("License not found", map[string]interface{}{
			"license":     req.LicenseKey,
			"app_version": req.AppVersion,
		})
		respondWithValidation(w, false, "license not found")
		return
	}

	if license.Status != models.StatusActive {
		respondWithValidation(w, false, "license not active")
		return
	}

	respondWithValidation(w, true, "license valid")
}

func respondWithValidation(w http.ResponseWriter, valid bool, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ValidateResponse{
		Valid:   valid,
		Message: message,
	})
}

func (lr LicenseRequest) validate() error {
	if strings.TrimSpace(lr.LicenseKey) == "" {
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

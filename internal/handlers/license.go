package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"auto-focus.app/cloud/internal/models"
)

type LicenseHandler struct {
	db     *sql.DB
	secret string
}

func NewLicenseHandler(db *sql.DB, secret string) *LicenseHandler {
	return &LicenseHandler{
		db:     db,
		secret: secret,
	}
}

func (h *LicenseHandler) VerifyLicense(w http.ResponseWriter, r *http.Request) {
	var req models.VerifyLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var license models.License
	err := h.db.QueryRow(`
		SELECT l.id, l.license_key, l.status, l.expires_at, c.name, c.email
		FROM licenses l
		JOIN customers c ON l.customer_id = c.id
		WHERE l.license_key = ? AND l.status = 'active'`,
		req.LicenseKey,
	).Scan(&license.ID, &license.LicenseKey, &license.Status, &license.ExpiresAt, &license.Customer.Name, &license.Customer.Email)

	if err == sql.ErrNoRows {
		json.NewEncoder(w).Encode(models.VerifyLicenseResponse{
			Valid: false,
			Error: "Invalid or inactive license key",
		})
		return
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if license.ExpiresAt != nil && license.ExpiresAt.Before(time.Now()) {
		json.NewEncoder(w).Encode(models.VerifyLicenseResponse{
			Valid: false,
			Error: "License has expired",
		})
		return
	}

	json.NewEncoder(w).Encode(models.VerifyLicenseResponse{
		Valid:     true,
		Name:      license.Customer.Name,
		Email:     license.Customer.Email,
		ExpiresAt: license.ExpiresAt,
	})
}

func (h *LicenseHandler) ActivateLicense(w http.ResponseWriter, r *http.Request) {
	var req models.ActivateLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var licenseID int
	err = tx.QueryRow("SELECT id FROM licenses WHERE license_key = ? AND status = 'active'", req.LicenseKey).Scan(&licenseID)
	if err == sql.ErrNoRows {
		json.NewEncoder(w).Encode(models.ActivateLicenseResponse{
			Success: false,
			Error:   "Invalid or inactive license key",
		})
		return
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`
		INSERT INTO license_activations (license_id, machine_id, is_active, activated_at, last_seen_at)
		VALUES (?, ?, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(license_id, machine_id) DO UPDATE SET
		is_active = TRUE,
		last_seen_at = CURRENT_TIMESTAMP`,
		licenseID, req.MachineID)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(models.ActivateLicenseResponse{
		Success: true,
	})
}

func (h *LicenseHandler) DeactivateLicense(w http.ResponseWriter, r *http.Request) {
	var req models.DeactivateLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var licenseID int
	err = tx.QueryRow("SELECT id FROM licenses WHERE license_key = ?", req.LicenseKey).Scan(&licenseID)
	if err == sql.ErrNoRows {
		json.NewEncoder(w).Encode(models.DeactivateLicenseResponse{
			Success: false,
			Error:   "Invalid license key",
		})
		return
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	result, err := tx.Exec(
		"UPDATE license_activations SET is_active = FALSE WHERE license_id = ? AND machine_id = ?",
		licenseID, req.MachineID)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rows, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if rows == 0 {
		json.NewEncoder(w).Encode(models.DeactivateLicenseResponse{
			Success: false,
			Error:   "License not activated for this machine",
		})
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(models.DeactivateLicenseResponse{
		Success: true,
	})
}

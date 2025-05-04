package models

import (
	"time"
)

type License struct {
	ID             int        `json:"id" db:"id"`
	LicenseKey     string     `json:"license_key" db:"license_key"`
	CustomerID     int        `json:"customer_id" db:"customer_id"`
	Status         string     `json:"status" db:"status"`
	MaxActivations int        `json:"max_activations" db:"max_activations"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" db:"expires_at"`

	Customer    *Customer           `json:"customer,omitempty" db:"-"`
	Activations []LicenseActivation `json:"activations,omitempty" db:"-"`
}

type LicenseActivation struct {
	ID          int       `json:"id" db:"id"`
	LicenseID   int       `json:"license_id" db:"license_id"`
	MachineID   string    `json:"machine_id" db:"machine_id"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	ActivatedAt time.Time `json:"activated_at" db:"activated_at"`
	LastSeenAt  time.Time `json:"last_seen_at" db:"last_seen_at"`
}

type VerifyLicenseRequest struct {
	LicenseKey string `json:"license_key"`
	MachineID  string `json:"machine_id"`
}

type VerifyLicenseResponse struct {
	Valid     bool       `json:"valid"`
	Name      string     `json:"name,omitempty"`
	Email     string     `json:"email,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Error     string     `json:"error,omitempty"`
}

type ActivateLicenseRequest struct {
	LicenseKey string `json:"license_key"`
	MachineID  string `json:"machine_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
}

type ActivateLicenseResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type DeactivateLicenseRequest struct {
	LicenseKey string `json:"license_key"`
	MachineID  string `json:"machine_id"`
}

type DeactivateLicenseResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

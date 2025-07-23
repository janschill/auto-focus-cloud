package models

import "time"

const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusExpired   = "expired"
)

type License struct {
	ID              string
	Key             string
	Version         string
	Status          string
	CustomerID      string
	ProductID       string
	StripeSessionID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

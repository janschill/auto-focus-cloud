package models

import "time"

type Customer struct {
	ID               string
	Email            string
	Name             string
	Country          string
	StripeCustomerID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

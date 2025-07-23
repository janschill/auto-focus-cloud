package models

import "time"

type Customer struct {
	ID               string
	Email            string
	StripeCustomerID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

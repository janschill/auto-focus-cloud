package models

import (
	"time"
)

type Customer struct {
	ID                   int       `json:"id" db:"id"`
	Email                string    `json:"email" db:"email"`
	Name                 string    `json:"name" db:"name"`
	StripeCustomerID     string    `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	StripeSubscriptionID string    `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

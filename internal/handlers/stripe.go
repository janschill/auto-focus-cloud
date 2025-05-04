package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"auto-focus.app/cloud/internal/models"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/webhook"
)

type StripeHandler struct {
	db            *sql.DB
	secret        string
	webhookSecret string
}

func NewStripeHandler(db *sql.DB, secret string, webhookSecret string) *StripeHandler {
	stripe.Key = secret
	return &StripeHandler{
		db:            db,
		secret:        secret,
		webhookSecret: webhookSecret,
	}
}

func (h *StripeHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), h.webhookSecret)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	log.Printf("Received Stripe event: %s", event.Type)

	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing subscription data: %v", err)
			http.Error(w, "Error parsing subscription data", http.StatusBadRequest)
			return
		}

		if err := h.handleSubscriptionEvent(&subscription); err != nil {
			log.Printf("Error handling subscription event: %v", err)
			http.Error(w, "Error processing subscription", http.StatusInternalServerError)
			return
		}

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing subscription data: %v", err)
			http.Error(w, "Error parsing subscription data", http.StatusBadRequest)
			return
		}

		if err := h.handleSubscriptionDeleted(subscription.Customer.ID); err != nil {
			log.Printf("Error handling subscription deletion: %v", err)
			http.Error(w, "Error processing subscription deletion", http.StatusInternalServerError)
			return
		}

	case "invoice.paid":
		var invoice stripe.Invoice
		err := json.Unmarshal(event.Data.Raw, &invoice)
		if err != nil {
			log.Printf("Error parsing invoice data: %v", err)
			http.Error(w, "Error parsing invoice data", http.StatusBadRequest)
			return
		}

		if err := h.handleInvoicePaid(invoice.Customer.ID); err != nil {
			log.Printf("Error handling invoice paid: %v", err)
			http.Error(w, "Error processing invoice payment", http.StatusInternalServerError)
			return
		}

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		err := json.Unmarshal(event.Data.Raw, &invoice)
		if err != nil {
			log.Printf("Error parsing invoice data: %v", err)
			http.Error(w, "Error parsing invoice data", http.StatusBadRequest)
			return
		}

		if err := h.handleInvoicePaymentFailed(invoice.Customer.ID); err != nil {
			log.Printf("Error handling invoice payment failure: %v", err)
			http.Error(w, "Error processing invoice payment failure", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *StripeHandler) handleSubscriptionEvent(subscription *stripe.Subscription) error {
	log.Printf("Handling subscription event for customer %s with status %s",
		subscription.Customer.ID, subscription.Status)

	if subscription.Status != stripe.SubscriptionStatusActive {
		log.Printf("Skipping subscription processing - status is not active")
		return nil
	}

	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	var customer models.Customer
	err = tx.QueryRow(
		"SELECT id, email, name FROM customers WHERE stripe_customer_id = ?",
		subscription.Customer.ID,
	).Scan(&customer.ID, &customer.Email, &customer.Name)

	if err == sql.ErrNoRows {
		stripeCustomer, err := h.getStripeCustomer(subscription.Customer.ID)
		if err != nil {
			return fmt.Errorf("error fetching Stripe customer: %w", err)
		}

		log.Printf("Creating new customer with Stripe ID: %s", subscription.Customer.ID)
		result, err := tx.Exec(`
			INSERT INTO customers (email, name, stripe_customer_id, stripe_subscription_id)
			VALUES (?, ?, ?, ?)`,
			stripeCustomer.Email, stripeCustomer.Name, subscription.Customer.ID, subscription.ID)
		if err != nil {
			return fmt.Errorf("error creating customer: %w", err)
		}

		customerID64, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("error getting new customer ID: %w", err)
		}
		customer.ID = int(customerID64)
		customer.Email = stripeCustomer.Email
		customer.Name = stripeCustomer.Name

		log.Printf("Created new customer with ID: %d", customer.ID)
	} else if err != nil {
		return fmt.Errorf("error finding customer: %w", err)
	}

	log.Printf("Found/Created customer: id=%d, email=%s", customer.ID, customer.Email)

	var existingLicenseKey string
	err = tx.QueryRow(
		"SELECT license_key FROM licenses WHERE customer_id = ?",
		customer.ID,
	).Scan(&existingLicenseKey)

	var licenseKey string
	if err == sql.ErrNoRows {
		licenseKey = h.generateLicenseKey()
		log.Printf("Generated new license key for customer %d: %s", customer.ID, licenseKey)
	} else if err != nil {
		return fmt.Errorf("error checking existing license: %w", err)
	} else {
		licenseKey = existingLicenseKey
		log.Printf("Using existing license key for customer %d: %s", customer.ID, licenseKey)
	}

	var expiresAt time.Time
	if subscription.CancelAt > 0 {
		expiresAt = time.Unix(subscription.CancelAt, 0)
	} else {
		expiresAt = time.Now().AddDate(1, 0, 0) // Default to 1 year if period end not available
	}
	log.Printf("Setting license expiry to: %v", expiresAt)

	_, err = tx.Exec(`
		INSERT INTO licenses (license_key, customer_id, status, expires_at)
		VALUES (?, ?, 'active', ?)
		ON CONFLICT(customer_id) DO UPDATE SET
		status = 'active',
		expires_at = ?`,
		licenseKey, customer.ID, expiresAt, expiresAt)

	if err != nil {
		return fmt.Errorf("error upserting license: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	log.Printf("Successfully created/updated license for customer %d", customer.ID)
	return nil
}

func (h *StripeHandler) getStripeCustomer(customerID string) (*stripe.Customer, error) {
	customer, err := customer.Get(customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching customer from Stripe: %w", err)
	}
	return customer, nil
}

func (h *StripeHandler) handleSubscriptionDeleted(customerID string) error {
	log.Printf("Handling subscription deletion for customer %s", customerID)
	_, err := h.db.Exec(`
		UPDATE licenses
		SET status = 'expired'
		WHERE customer_id IN (
			SELECT id FROM customers WHERE stripe_customer_id = ?
		)`,
		customerID)
	return err
}

func (h *StripeHandler) handleInvoicePaid(customerID string) error {
	log.Printf("Handling invoice paid for customer %s", customerID)
	_, err := h.db.Exec(`
		UPDATE licenses
		SET status = 'active'
		WHERE customer_id IN (
			SELECT id FROM customers WHERE stripe_customer_id = ?
		)`,
		customerID)
	return err
}

func (h *StripeHandler) handleInvoicePaymentFailed(customerID string) error {
	log.Printf("Handling invoice payment failed for customer %s", customerID)
	_, err := h.db.Exec(`
		UPDATE licenses
		SET status = 'inactive'
		WHERE customer_id IN (
			SELECT id FROM customers WHERE stripe_customer_id = ?
		)`,
		customerID)
	return err
}

func (h *StripeHandler) generateLicenseKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Error generating random bytes for license key: %v", err)
		return "" // In production, handle this error appropriately
	}

	return fmt.Sprintf("AF-%x", bytes)
}

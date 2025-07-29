package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"auto-focus.app/cloud/internal/email"
	"auto-focus.app/cloud/internal/logger"
	"auto-focus.app/cloud/models"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

func (s *Server) Stripe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger.Info("Stripe webhook received", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.Header.Get("User-Agent"),
		"method":      r.Method,
	})

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		logger.Error("STRIPE_SECRET_KEY environment variable not set")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read webhook payload", map[string]interface{}{
			"error": err.Error(),
		})
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	logger.Debug("Webhook payload received", map[string]interface{}{
		"payload_size": len(payload),
	})

	event := stripe.Event{}
	if err := json.Unmarshal(payload, &event); err != nil {
		logger.Error("Failed to parse webhook JSON", map[string]interface{}{
			"error": err.Error(),
		})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger.Info("Stripe event parsed", map[string]interface{}{
		"event_type": event.Type,
		"event_id":   event.ID,
	})

	// Skip signature verification in test mode
	if os.Getenv("TEST_MODE") == "true" {
		logger.Debug("Skipping webhook signature verification (test mode)")
	} else {
		endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		if endpointSecret == "" {
			logger.Error("STRIPE_WEBHOOK_SECRET environment variable not set")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		signatureHeader := r.Header.Get("Stripe-Signature")
		event, err = webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
		if err != nil {
			logger.Error("Webhook signature verification failed", map[string]interface{}{
				"error":     err.Error(),
				"signature": signatureHeader,
			})
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.Debug("Webhook signature verified")
	}

	switch event.Type {
	case "checkout.session.completed":
		logger.Info("Processing checkout session completed event", map[string]interface{}{
			"event_id": event.ID,
		})

		var checkoutSession stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &checkoutSession)
		if err != nil {
			logger.Error("Failed to unmarshal checkout session", map[string]interface{}{
				"error":    err.Error(),
				"event_id": event.ID,
			})
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err = s.handleCheckoutComplete(ctx, &checkoutSession); err != nil {
			logger.Error("Failed to handle checkout completion", map[string]interface{}{
				"error":      err.Error(),
				"session_id": checkoutSession.ID,
			})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		logger.Info("Unhandled webhook event type", map[string]interface{}{
			"event_type": event.Type,
			"event_id":   event.ID,
		})
	}

	logger.Info("Webhook processed successfully", map[string]interface{}{
		"event_type": event.Type,
		"event_id":   event.ID,
	})

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"received": "true"}); err != nil {
		logger.Error("Failed to encode webhook response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (s *Server) handleCheckoutComplete(ctx context.Context, session *stripe.CheckoutSession) error {
	// Extract customer email from CustomerDetails if available
	var customerEmail string
	if session.CustomerDetails != nil {
		customerEmail = session.CustomerDetails.Email
	} else {
		customerEmail = session.CustomerEmail
	}

	fields := map[string]interface{}{
		"session_id":     session.ID,
		"customer_email": customerEmail,
		"amount":         session.AmountTotal,
		"currency":       session.Currency,
		"payment_status": session.PaymentStatus,
		"metadata":       session.Metadata,
	}

	logger.Info("Processing checkout session", fields)

	// Log customer object availability
	if session.Customer != nil {
		logger.Debug("Stripe customer object present", map[string]interface{}{
			"stripe_customer_id": session.Customer.ID,
			"session_id":         session.ID,
		})
	} else {
		logger.Warn("Stripe customer object is nil", map[string]interface{}{
			"session_id": session.ID,
		})
	}

	customer, license, err := s.createLicensedUser(ctx, session, customerEmail)
	if err != nil {
		logger.Error("Failed to create licensed user", map[string]interface{}{
			"error":      err.Error(),
			"session_id": session.ID,
		})
		return err
	}

	logger.Info("Licensed user created successfully", map[string]interface{}{
		"customer_id":    customer.ID,
		"customer_email": customer.Email,
		"session_id":     session.ID,
	})

	// Create personalized email content
	customerName := "there"
	if customer.Name != "" {
		customerName = strings.Split(customer.Name, " ")[0] // Use first name only
	}

	formattedPrice := formatPrice(license.PricePaid, license.Currency)

	body := fmt.Sprintf(`Hello %s,

Thank you for purchasing Auto-Focus+! Your purchase has been processed successfully.

LICENSE DETAILS
License Key: %s
Product: Auto-Focus+ (%s)
Amount Paid: %s

GETTING STARTED
1. Open Auto-Focus on your Mac
2. Go to Settings → License
3. Enter your license key: %s
4. Enjoy unlimited focus sessions!

NEED HELP?
If you have any questions, reply to this email or contact us at help@auto-focus.app

Thank you for choosing Auto-Focus+!

Best regards,
The Auto-Focus Team`, 
		customerName, 
		license.Key, 
		license.ProductName,
		formattedPrice,
		license.Key)

	if err := email.Send(customerEmail, "Auto-Focus+ License Key", body); err != nil {
		logger.Error("Failed to send license email", map[string]interface{}{
			"error":       err.Error(),
			"email":       customerEmail,
			"license_key": license.Key,
			"customer_id": customer.ID,
			"session_id":  session.ID,
		})
		// Don't return error - license was created successfully
		// Email failure shouldn't fail the entire transaction
	} else {
		logger.Info("License email sent successfully", map[string]interface{}{
			"email":       customerEmail,
			"customer_id": customer.ID,
		})
	}

	return nil
}

func (s *Server) createLicensedUser(ctx context.Context, session *stripe.CheckoutSession, customerEmail string) (*models.Customer, *models.License, error) {
	logger.Debug("Creating licensed user", map[string]interface{}{
		"session_id": session.ID,
	})

	customer, err := s.findOrCreateCustomer(ctx, session, customerEmail)
	if err != nil {
		logger.Error("Failed to find/create customer", map[string]interface{}{
			"error":      err.Error(),
			"session_id": session.ID,
		})
		return nil, nil, fmt.Errorf("failed to find/create customer: %w", err)
	}

	logger.Info("Customer resolved", map[string]interface{}{
		"customer_id":        customer.ID,
		"customer_email":     customer.Email,
		"stripe_customer_id": customer.StripeCustomerID,
		"session_id":         session.ID,
	})

	license := createLicese(customer, session)
	logger.Info("License generated", map[string]interface{}{
		"license_key": license.Key,
		"version":     license.Version,
		"product_id":  license.ProductID,
		"customer_id": customer.ID,
	})

	err = s.Storage.SaveLicense(ctx, license)
	if err != nil {
		logger.Error("Failed to save license", map[string]interface{}{
			"error":       err.Error(),
			"license_key": license.Key,
			"customer_id": customer.ID,
		})
		return nil, nil, fmt.Errorf("failed to save license: %w", err)
	}

	logger.Info("License saved successfully", map[string]interface{}{
		"license_key": license.Key,
		"customer_id": customer.ID,
	})

	return customer, license, nil
}

func (s *Server) findOrCreateCustomer(ctx context.Context, session *stripe.CheckoutSession, customerEmail string) (*models.Customer, error) {
	logger.Debug("Looking up customer", map[string]interface{}{
		"customer_email": customerEmail,
		"session_id":     session.ID,
	})

	customer, err := s.Storage.FindCustomerByEmailAddress(ctx, customerEmail)
	if err != nil {
		logger.Error("Database lookup failed", map[string]interface{}{
			"error":          err.Error(),
			"customer_email": customerEmail,
		})
		return nil, err
	}

	if customer != nil {
		logger.Info("Existing customer found", map[string]interface{}{
			"customer_id":    customer.ID,
			"customer_email": customer.Email,
		})
		return customer, nil
	}

	logger.Info("Creating new customer", map[string]interface{}{
		"customer_email": customerEmail,
		"session_id":     session.ID,
	})

	customer = createCustomer(session, customerEmail)

	err = s.Storage.SaveCustomer(ctx, customer)
	if err != nil {
		logger.Error("Failed to save new customer", map[string]interface{}{
			"error":          err.Error(),
			"customer_email": customer.Email,
		})
		return nil, fmt.Errorf("failed to save customer: %w", err)
	}

	logger.Info("New customer created", map[string]interface{}{
		"customer_id":        customer.ID,
		"customer_email":     customer.Email,
		"stripe_customer_id": customer.StripeCustomerID,
	})

	return customer, nil
}

func createCustomer(session *stripe.CheckoutSession, customerEmail string) *models.Customer {
	var stripeCustomerID string
	if session.Customer != nil {
		stripeCustomerID = session.Customer.ID
		logger.Debug("Using Stripe customer ID", map[string]interface{}{
			"stripe_customer_id": stripeCustomerID,
			"session_id":         session.ID,
		})
	} else {
		logger.Warn("No Stripe customer object available", map[string]interface{}{
			"session_id": session.ID,
		})
	}

	// Extract customer name and country from CustomerDetails
	var customerName, country string
	if session.CustomerDetails != nil {
		customerName = session.CustomerDetails.Name
		if session.CustomerDetails.Address != nil {
			country = session.CustomerDetails.Address.Country
		}
	}

	customer := &models.Customer{
		ID:               uuid.Must(uuid.NewRandom()).String(),
		Email:            customerEmail,
		Name:             customerName,
		Country:          country,
		StripeCustomerID: stripeCustomerID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	logger.Debug("Customer object created", map[string]interface{}{
		"customer_id":    customer.ID,
		"customer_email": customer.Email,
	})

	return customer
}

func createLicese(customer *models.Customer, session *stripe.CheckoutSession) *models.License {
	// Default product name to "v1" if not specified in metadata
	productName := session.Metadata["product_name"]
	if productName == "" {
		productName = "v1"
	}

	return &models.License{
		ID:              uuid.Must(uuid.NewRandom()).String(),
		Key:             generateLicenseKey(),
		CustomerID:      customer.ID,
		ProductID:       session.Metadata["product_id"],
		ProductName:     productName,
		PricePaid:       session.AmountTotal,
		Currency:        string(session.Currency),
		Version:         session.Metadata["license_version"],
		Status:          models.StatusActive,
		StripeSessionID: session.ID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func generateLicenseKey() string {
	return fmt.Sprintf("AFP-%s", uuid.Must(uuid.NewRandom()).String()[:8])
}

func formatPrice(amountCents int64, currency string) string {
	// Convert cents to major currency unit
	amount := float64(amountCents) / 100.0
	
	// Format based on currency
	switch strings.ToUpper(currency) {
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	case "EUR":
		return fmt.Sprintf("€%.2f", amount)
	case "GBP":
		return fmt.Sprintf("£%.2f", amount)
	case "NOK":
		return fmt.Sprintf("%.2f NOK", amount)
	case "SEK":
		return fmt.Sprintf("%.2f SEK", amount)
	case "DKK":
		return fmt.Sprintf("%.2f DKK", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, strings.ToUpper(currency))
	}
}

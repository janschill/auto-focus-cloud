# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

- **Run server**: `go run main.go`
- **Build**: `go build -o auto-focus-cloud main.go`

## Stripe Development

For Stripe webhook testing:

```sh
stripe login
stripe listen --forward-to localhost:8080/api/webhooks/stripe
stripe trigger customer.subscription.created
```

## Architecture

This is a Go HTTP server for license management with Stripe integration, and a SQLite database.

## Client Expectations

Current Client Implementation Analysis

  License Management Features

  The client already has a sophisticated license management system with:

  Core Functionality:

- License key validation and activation
- License deactivation
- Automatic validation every 24 hours
- Beta access until August 31, 2025
- Instance-based licensing (device fingerprinting)
- Debug license key support

  Client-Server Communication:
- Server URL: <https://api.auto-focus.app/v1/licenses>
- Two endpoints expected: /validate and /deactivate
- Uses instance identifier (SHA256 hash of machine model + bundle ID)
- Sends app version, platform info

  License Features:
- Owner name and email
- Expiry dates
- App version compatibility
- Max apps allowed (-1 = unlimited)
- Different license tiers (Beta, Full, Free)

  Backend License API Plan

  1. Core Database Schema

  -- Licenses table
  licenses (
    id UUID PRIMARY KEY,
    license_key VARCHAR(64) UNIQUE NOT NULL,
    owner_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP,
    expires_at TIMESTAMP,
    max_apps INTEGER DEFAULT 3,
    app_version VARCHAR(32),
    status ENUM('active', 'suspended', 'expired'),
    max_activations INTEGER DEFAULT 3
  )

  -- License activations table
  license_activations (
    id UUID PRIMARY KEY,
    license_id UUID REFERENCES licenses(id),
    instance_id VARCHAR(64) NOT NULL,
    activated_at TIMESTAMP,
    last_validated_at TIMESTAMP,
    app_version VARCHAR(32),
    platform VARCHAR(32),
    is_active BOOLEAN DEFAULT true
  )

  -- License validation logs
  license_validations (
    id UUID PRIMARY KEY,
    license_id UUID REFERENCES licenses(id),
    instance_id VARCHAR(64),
    validated_at TIMESTAMP,
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(32)
  )

  2. API Endpoints

  POST /v1/licenses/validate

  Request:
  {
    "license_key": "XXXX-XXXX-XXXX-XXXX",
    "instance_id": "sha256_hash_of_device",
    "app_version": "1.0.0",
    "platform": "macos"
  }

  Response (Success):
  {
    "owner_name": "John Doe",
    "email": "john@example.com",
    "app_version": "1.0.0",
    "expires_at": "2025-12-31T23:59:59Z",
    "max_apps": -1
  }

  Response (Error):
  {
    "error": "invalid_license",
    "message": "License key not found or invalid"
  }

  POST /v1/licenses/deactivate

  Request:
  {
    "license_key": "XXXX-XXXX-XXXX-XXXX",
    "instance_id": "sha256_hash_of_device"
  }

  Response:
  {
    "success": true,
    "message": "License deactivated successfully"
  }

  1. Additional Management Endpoints

  POST /v1/admin/licenses (Create license)

  GET /v1/admin/licenses (List licenses)

  GET /v1/admin/licenses/{id} (Get license details)

  PUT /v1/admin/licenses/{id} (Update license)

  GET /v1/admin/licenses/{id}/activations (View activations)

  4. Business Logic Requirements

  License Validation Logic:
- Check if license key exists and is active
- Verify not expired
- Check instance limit (max_activations)
- Update last_validated_at timestamp
- Log validation attempt

  Activation Logic:
- Check if already activated on this instance
- Check if under activation limit
- Create new activation record
- Return license details

  Deactivation Logic:
- Find activation by license_key + instance_id
- Mark activation as inactive
- Allow reactivation later

  5. Security & Infrastructure

  Rate Limiting:
- Max 10 validation requests per minute per IP
- Max 5 activation attempts per hour per IP

  Authentication:
- Admin endpoints require API key authentication
- Client endpoints are public but rate limited

  Monitoring:
- Log all validation/activation attempts
- Monitor for abuse patterns
- Alert on suspicious activity

  6. Technology Stack Recommendations

  Backend Framework:
- Go

  Database:
- SQLite for ACID compliance
- In-Memory for caching validation results

  Deployment:
- Docker containers
- Cloud hosting (AWS/Digital Ocean/Railway)
- Load balancer for HA

  7. License Key Generation

  Format: XXXX-XXXX-XXXX-XXXX (16 chars + dashes)
  Structure:
- 4 chars: Product identifier
- 4 chars: License type/tier
- 4 chars: Random component
- 4 chars: Checksum/validation

  8. Development Phases

  Phase 1: Core API (validate/deactivate endpoints)
  Phase 2: Admin dashboard for license management
  Phase 3: Analytics and reporting
  Phase 4: Advanced features (license transfers, upgrades)

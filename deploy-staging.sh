#!/bin/bash

# Deploy Auto-Focus API to Staging Environment (for CI/CD)
# This script handles both local development and CI deployment

set -e

VERSION=${1:-"staging-local"}
SERVICE_NAME="auto-focus-cloud-staging"
INSTALL_DIR="/home/autofocus/staging"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

echo "🚀 Deploying Auto Focus Cloud API to Staging ${VERSION}"
echo "=================================================="

# Set environment to staging
export ENVIRONMENT=staging

# Check if we're running in CI (has binary) or locally (needs to build)
if [ -f "auto-focus-cloud-staging" ]; then
    echo "📦 Using pre-built binary (CI mode)"
    BINARY_NAME="auto-focus-cloud-staging"
elif [ -f ".env.staging" ]; then
    echo "🔨 Building for local staging deployment"
    
    # Load staging environment for local development
    set -a
    source .env.staging
    set +a
    
    # Ensure port is 8081 for staging
    export PORT=8081
    
    # Build the application
    echo "🔨 Building staging application..."
    go build -o auto-focus-cloud-staging main.go
    
    if [ $? -ne 0 ]; then
        echo "❌ Build failed!"
        exit 1
    fi
    
    BINARY_NAME="auto-focus-cloud-staging"
    
    echo "✅ Build successful"
    echo "   Environment: $ENVIRONMENT"
    echo "   Port: $PORT (staging)"
    echo "   Database: $DATABASE_URL"
    
    # For local development, just run the server
    if [ "$VERSION" = "staging-local" ]; then
        echo "🏃 Starting local staging server..."
        echo "   Production API: https://auto-focus.app/api/ (port 8080)"
        echo "   Staging API: https://staging.auto-focus.app/api/ (port $PORT)"
        echo "   Local access: http://localhost:$PORT"
        echo ""
        echo "🧪 Staging Configuration:"
        echo "   Environment: $ENVIRONMENT"
        echo "   Database: $DATABASE_URL"
        echo "   Stripe: Test mode ($TEST_MODE)"
        echo ""
        echo "💳 Test with Stripe test cards:"
        echo "   4242424242424242 (Visa - Success)"
        echo "   4000000000000002 (Declined)"
        echo "   4000000000009995 (Insufficient funds)"
        echo ""
        echo "🌐 Test staging checkout:"
        echo "   https://auto-focus.app/?staging=1"
        echo ""
        echo "🛑 Press Ctrl+C to stop staging server"
        echo ""
        
        # Check if staging port is available
        if lsof -i :$PORT > /dev/null 2>&1; then
            echo "⚠️  Port $PORT is already in use. Stopping existing service..."
            lsof -ti :$PORT | xargs kill -9 2>/dev/null || true
            sleep 2
        fi
        
        # Start the staging server locally
        ./$BINARY_NAME
        exit 0
    fi
else
    echo "❌ No .env.staging file found and no pre-built binary!"
    echo "   For local deployment: Copy .env.example to .env.staging and configure with Stripe test keys"
    echo "   For CI deployment: This should not happen - check CI configuration"
    exit 1
fi

# Server deployment (systemd service setup)
echo "📡 Deploying to server with systemd service..."

# Stop the staging service if it's running
if systemctl is-active --quiet ${SERVICE_NAME}; then
    echo "⏹️  Stopping ${SERVICE_NAME} service"
    systemctl stop ${SERVICE_NAME}
fi

# Create staging directory
echo "📁 Creating staging directory"
mkdir -p ${INSTALL_DIR}/storage/data

# Backup current staging version if it exists
if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
    echo "📦 Backing up current staging version"
    cp "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}.backup"
fi

# Copy new binary (only if we're not already in the install dir)
echo "📁 Installing new staging binary"
if [ "$(pwd)" != "${INSTALL_DIR}" ]; then
    cp $BINARY_NAME ${INSTALL_DIR}/
fi
chmod +x ${INSTALL_DIR}/${BINARY_NAME}

# Install/update systemd service for staging
echo "⚙️  Installing staging systemd service"
if [ -f "auto-focus-cloud-staging.service" ]; then
    cp auto-focus-cloud-staging.service ${SERVICE_FILE}
else
    # Create staging service file if it doesn't exist
    cat > ${SERVICE_FILE} << EOF
[Unit]
Description=Auto Focus Cloud API (Staging)
After=network.target

[Service]
Type=simple
User=autofocus
Group=autofocus
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=always
RestartSec=5
Environment=ENVIRONMENT=staging
Environment=PORT=8081

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=auto-focus-cloud-staging

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${INSTALL_DIR}

[Install]
WantedBy=multi-user.target
EOF
fi

systemctl daemon-reload

# Set proper ownership
chown -R autofocus:autofocus ${INSTALL_DIR}

# Start the staging service
echo "▶️  Starting ${SERVICE_NAME} service"
systemctl enable ${SERVICE_NAME}
systemctl start ${SERVICE_NAME}

# Wait a moment and check status
sleep 3
if systemctl is-active --quiet ${SERVICE_NAME}; then
    echo "✅ Staging deployment successful! Service is running on port 8081"
    echo "📊 Service status:"
    systemctl status ${SERVICE_NAME} --no-pager -l
    echo ""
    echo "🌐 Staging API available at:"
    echo "   https://staging.auto-focus.app/api/"
    echo "   http://localhost:8081/ (local)"
    echo ""
    echo "🧪 Test staging checkout:"
    echo "   https://auto-focus.app/?staging=1"
else
    echo "❌ Staging deployment failed! Service is not running"
    echo "🔍 Service logs:"
    journalctl -u ${SERVICE_NAME} --no-pager -l --since "2 minutes ago"
    exit 1
fi

# Save version info
echo ${VERSION} > ${INSTALL_DIR}/VERSION
echo "🏷️  Staging version ${VERSION} deployed successfully"
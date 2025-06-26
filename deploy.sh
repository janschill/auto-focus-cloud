#!/bin/bash

set -e

VERSION=${1:-"latest"}
SERVICE_NAME="auto-focus-cloud"
INSTALL_DIR="/home/autofocus/app"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

echo "🚀 Deploying Auto Focus Cloud API ${VERSION}"

# Stop the service if it's running
if systemctl is-active --quiet ${SERVICE_NAME}; then
    echo "⏹️  Stopping ${SERVICE_NAME} service"
    systemctl stop ${SERVICE_NAME}
fi

# Backup current version if it exists
if [ -f "${INSTALL_DIR}/auto-focus-cloud" ]; then
    echo "📦 Backing up current version"
    cp "${INSTALL_DIR}/auto-focus-cloud" "${INSTALL_DIR}/auto-focus-cloud.backup"
fi

# Copy new binary
echo "📁 Installing new binary"
cp auto-focus-cloud ${INSTALL_DIR}/
chmod +x ${INSTALL_DIR}/auto-focus-cloud

# Install/update systemd service
echo "⚙️  Installing systemd service"
cp auto-focus-cloud.service ${SERVICE_FILE}
systemctl daemon-reload

# Set proper ownership
chown -R autofocus:autofocus ${INSTALL_DIR}

# Start the service
echo "▶️  Starting ${SERVICE_NAME} service"
systemctl enable ${SERVICE_NAME}
systemctl start ${SERVICE_NAME}

# Wait a moment and check status
sleep 2
if systemctl is-active --quiet ${SERVICE_NAME}; then
    echo "✅ Deployment successful! Service is running on port 8080"
    echo "📊 Service status:"
    systemctl status ${SERVICE_NAME} --no-pager -l
else
    echo "❌ Deployment failed! Service is not running"
    echo "🔍 Service logs:"
    journalctl -u ${SERVICE_NAME} --no-pager -l --since "2 minutes ago"
    exit 1
fi

# Save version info
echo ${VERSION} > ${INSTALL_DIR}/VERSION
echo "🏷️  Version ${VERSION} deployed successfully"
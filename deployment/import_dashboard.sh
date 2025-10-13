#!/bin/bash

# GQueue Grafana Dashboard Import Script
# This script automatically imports the GQueue monitoring dashboard into Grafana

set -e

# Configuration
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-admin}"
DASHBOARD_FILE="$(dirname "$0")/grafana_dashboard.json"
REDIS_DATASOURCE_NAME="Redis Main"
REDIS_URL="${REDIS_URL:-redis://localhost:6379}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if required tools are installed
check_dependencies() {
    log_info "Checking dependencies..."

    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        exit 1
    fi

    log_success "All dependencies are available"
}

# Check if Grafana is accessible
check_grafana() {
    log_info "Checking Grafana connectivity..."

    if ! curl -s -f "${GRAFANA_URL}/api/health" > /dev/null; then
        log_error "Cannot connect to Grafana at ${GRAFANA_URL}"
        log_error "Please check if Grafana is running and accessible"
        exit 1
    fi

    log_success "Grafana is accessible"
}

# Check if dashboard file exists
check_dashboard_file() {
    log_info "Checking dashboard file..."

    if [ ! -f "$DASHBOARD_FILE" ]; then
        log_error "Dashboard file not found: $DASHBOARD_FILE"
        exit 1
    fi

    # Validate JSON
    if ! jq . "$DASHBOARD_FILE" > /dev/null 2>&1; then
        log_error "Dashboard file contains invalid JSON"
        exit 1
    fi

    log_success "Dashboard file is valid"
}

# Create or update Redis datasource
setup_redis_datasource() {
    log_info "Setting up Redis datasource..."

    # Check if datasource exists
    DATASOURCE_CHECK=$(curl -s -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" \
        "${GRAFANA_URL}/api/datasources/name/${REDIS_DATASOURCE_NAME}" \
        -w "%{http_code}" -o /tmp/datasource_check.json)

    HTTP_CODE=$(echo "$DATASOURCE_CHECK" | tail -n1)

    if [ "$HTTP_CODE" = "200" ]; then
        log_warning "Redis datasource already exists, updating..."

        # Get existing datasource ID
        DATASOURCE_ID=$(jq -r '.id' /tmp/datasource_check.json)

        # Update datasource
        curl -s -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" \
            -H "Content-Type: application/json" \
            -X PUT \
            "${GRAFANA_URL}/api/datasources/${DATASOURCE_ID}" \
            -d '{
                "id": '"$DATASOURCE_ID"',
                "uid": "redis-main",
                "name": "'"$REDIS_DATASOURCE_NAME"'",
                "type": "redis-datasource",
                "url": "'"$REDIS_URL"'",
                "access": "proxy",
                "isDefault": false,
                "jsonData": {
                    "client": "standalone",
                    "poolSize": 5,
                    "timeout": 10,
                    "pingInterval": 0,
                    "pipelineWindow": 0
                }
            }' > /tmp/datasource_update.json

        if [ $? -eq 0 ]; then
            log_success "Redis datasource updated successfully"
        else
            log_error "Failed to update Redis datasource"
            exit 1
        fi
    else
        log_info "Creating new Redis datasource..."

        # Create new datasource
        curl -s -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" \
            -H "Content-Type: application/json" \
            -X POST \
            "${GRAFANA_URL}/api/datasources" \
            -d '{
                "uid": "redis-main",
                "name": "'"$REDIS_DATASOURCE_NAME"'",
                "type": "redis-datasource",
                "url": "'"$REDIS_URL"'",
                "access": "proxy",
                "isDefault": false,
                "jsonData": {
                    "client": "standalone",
                    "poolSize": 5,
                    "timeout": 10,
                    "pingInterval": 0,
                    "pipelineWindow": 0
                }
            }' > /tmp/datasource_create.json

        if [ $? -eq 0 ]; then
            log_success "Redis datasource created successfully"
        else
            log_error "Failed to create Redis datasource"
            exit 1
        fi
    fi

    # Clean up temp files
    rm -f /tmp/datasource_check.json /tmp/datasource_update.json /tmp/datasource_create.json
}

# Import dashboard
import_dashboard() {
    log_info "Importing GQueue dashboard..."

    # Prepare dashboard JSON for import
    DASHBOARD_JSON=$(jq '{
        dashboard: .,
        overwrite: true,
        inputs: [],
        folderId: 0
    }' "$DASHBOARD_FILE")

    # Import dashboard
    IMPORT_RESULT=$(curl -s -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" \
        -H "Content-Type: application/json" \
        -X POST \
        "${GRAFANA_URL}/api/dashboards/db" \
        -d "$DASHBOARD_JSON" \
        -w "%{http_code}" -o /tmp/dashboard_import.json)

    HTTP_CODE=$(echo "$IMPORT_RESULT" | tail -n1)

    if [ "$HTTP_CODE" = "200" ]; then
        DASHBOARD_URL=$(jq -r '.url' /tmp/dashboard_import.json)
        log_success "Dashboard imported successfully!"
        log_info "Dashboard URL: ${GRAFANA_URL}${DASHBOARD_URL}"
    else
        log_error "Failed to import dashboard (HTTP $HTTP_CODE)"
        if [ -f /tmp/dashboard_import.json ]; then
            log_error "Response: $(cat /tmp/dashboard_import.json)"
        fi
        exit 1
    fi

    # Clean up temp file
    rm -f /tmp/dashboard_import.json
}

# Test dashboard accessibility
test_dashboard() {
    log_info "Testing dashboard accessibility..."

    # Get dashboard by UID
    DASHBOARD_TEST=$(curl -s -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" \
        "${GRAFANA_URL}/api/dashboards/uid/gqueue-dashboard" \
        -w "%{http_code}" -o /tmp/dashboard_test.json)

    HTTP_CODE=$(echo "$DASHBOARD_TEST" | tail -n1)

    if [ "$HTTP_CODE" = "200" ]; then
        log_success "Dashboard is accessible and working"
    else
        log_warning "Dashboard may not be accessible (HTTP $HTTP_CODE)"
    fi

    # Clean up temp file
    rm -f /tmp/dashboard_test.json
}

# Display usage information
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help              Show this help message"
    echo "  -u, --url URL           Grafana URL (default: http://localhost:3000)"
    echo "  --user USER             Grafana username (default: admin)"
    echo "  --password PASSWORD     Grafana

#!/bin/bash

# This script is for local testing. Its used by Air.

rm -f one-click-aks-server
rm -rf user/

# Check if .env file exists
if [[ ! -f .env ]]; then
    echo "Error: .env file not found. Please create one with required environment variables."
    exit 1
fi

# Source environment files for build-time variables
source .env
# Source .env.local if it exists (optional)
[[ -f .env.local ]] && source .env.local

export VERSION="$(date +%Y%m%d)"

# Check required build-time variables
required_env_vars=("PROTECTED_LAB_SECRET" "VERSION")

for var in "${required_env_vars[@]}"; do
    if [[ -z "${!var}" ]]; then
        echo "Required environment variable $var is missing"
        exit 1
    fi
done

echo "Building with environment loaded in main.go..."

go build -ldflags "-X 'main.version=$VERSION' -X 'one-click-aks-server/internal/entity.ProtectedLabSecret=$PROTECTED_LAB_SECRET'" ./cmd/one-click-aks-server

# Clear Redis for clean testing
redis-cli flushall 2>/dev/null || echo "Redis not available or already clean"
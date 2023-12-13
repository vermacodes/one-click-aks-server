#!/bin/bash

# This script is for local testing. It starts both server and UI in one go.

rm one-click-aks-server

export ROOT_DIR=$(pwd)
export LOG_LEVEL="-4"

export ARM_USER_PRINCIPAL_NAME=$(az account show --output json | jq -r .user.name)
if [ $? -ne 0 ]; then
    err "Failed to get user principal name from az cli"
    exit 1
fi

export VERSION="$(date +%Y%m%d)"

required_env_vars=("ARM_USER_PRINCIPAL_NAME" "AZURE_SUBSCRIPTION_ID" "STORAGE_ACCOUNT_NAME" "SAS_TOKEN" "VERSION" "ROOT_DIR")

for var in "${required_env_vars[@]}"; do
    if [[ -z "${!var}" ]]; then
        echo "Required environment variable $var is missing"
        exit 1
    fi
done


go build -ldflags "-X 'main.version=$VERSION' -X 'one-click-aks-server/internal/entity.SasToken=$SAS_TOKEN' -X 'one-click-aks-server/internal/entity.StorageAccountName=$STORAGE_ACCOUNT_NAME'" ./cmd/one-click-aks-server

redis-cli flushall

#!/bin/bash

# This script starts the web app and the server. Both server and the webapp needs to be exposed to the world outside.
#
# WebApp runs on port 3000
# Server runs on port 8080.

# gather input parameters
# -t tag

source .env
source .env.local

while getopts ":t:" opt; do
  case $opt in
  t)
    TAG="$OPTARG"
    ;;
  \?)
    echo "Invalid option -$OPTARG" >&2
    ;;
  esac
done

if [ -z "${TAG}" ]; then
  TAG="latest"
fi

echo "TAG = ${TAG}"

# remove terraform state
rm -rf ./tf/.terraform
rm ./tf/.terraform.lock.hcl
rm ./tf/terraform.tfstate
rm ./tf/terraform.tfstate.backup

# remove workspace log file
rm ./tf/workspaces.log

# remove old build files
rm one-click-aks-server azurerm-msi-auth-proxy

if [[ "${PROTECTED_LAB_SECRET}" == "" ]]; then
  echo "PROTECTED_LAB_SECRET missing"
  exit 1
fi

export VERSION="$(date +%Y%m%d)"

# build proxy
go build -o azurerm-msi-auth-proxy ../proxy/main.go
if [ $? -ne 0 ]; then
  echo "Failed to build azurerm-msi-auth-proxy"
  exit 1
fi

# build server
go build -ldflags "-X 'main.version=$VERSION' -X 'one-click-aks-server/internal/entity.ProtectedLabSecret=$PROTECTED_LAB_SECRET'" ./cmd/one-click-aks-server

if [ $? -ne 0 ]; then
  echo "Failed to build one-click-aks-server"
  exit 1
fi

# build docker image
docker build -t actlabs.azurecr.io/repro:${TAG} .
if [ $? -ne 0 ]; then
  echo "Failed to build docker image"
  exit 1
fi

rm one-click-aks-server azurerm-msi-auth-proxy

az acr login --name actlabs --subscription ACT-CSS-Readiness-NPRD
docker push actlabs.azurecr.io/repro:${TAG}

# docker tag actlabs.azurecr.io/repro:${TAG} ashishvermapu/repro:${TAG}
# docker push ashishvermapu/repro:${TAG}

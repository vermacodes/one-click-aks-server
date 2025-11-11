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


export VERSION="$(date +%Y%m%d)"

# build proxy
go build -o azurerm-msi-auth-proxy ../proxy/main.go
if [ $? -ne 0 ]; then
  echo "Failed to build azurerm-msi-auth-proxy"
  exit 1
fi

# build server
go build -ldflags "-X 'main.version=$VERSION'" ./cmd/one-click-aks-server

if [ $? -ne 0 ]; then
  echo "Failed to build one-click-aks-server"
  exit 1
fi

# build docker image
docker build -t actlabs.azurecr.io/actlabs-server:${TAG} .
if [ $? -ne 0 ]; then
  echo "Failed to build docker image"
  exit 1
fi

rm one-click-aks-server azurerm-msi-auth-proxy

az acr login --name actlabs --subscription ACT-CSS-Readiness-NPRD
docker push actlabs.azurecr.io/actlabs-server:${TAG}

# docker tag actlabs.azurecr.io/actlabs-server:${TAG} ashishvermapu/actlabs-server:${TAG}
# docker push ashishvermapu/actlabs-server:${TAG}

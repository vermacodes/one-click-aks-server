#!/bin/bash

# This script starts the web app and the server. Both server and the webapp needs to be exposed to the world outside.
#
# WebApp runs on port 3000
# Server runs on port 8080.

# gather input parameters
# -t tag

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

if [[ "${PROTECTED_LAB_SECRET}" == "" ]]; then
  echo "PROTECTED_LAB_SECRET missing"
  exit 1
fi

export VERSION="$(date +%Y%m%d)"

go build -ldflags "-X 'main.version=$VERSION' -X 'one-click-aks-server/internal/entity.ProtectedLabSecret=$PROTECTED_LAB_SECRET'" ./cmd/one-click-aks-server

if [ $? -ne 0 ]; then
  echo "Failed to build one-click-aks-server"
  exit 1
fi

# build docker image
docker build -t repro:${TAG} .
if [ $? -ne 0 ]; then
  echo "Failed to build docker image"
  exit 1
fi

rm one-click-aks-server

docker tag repro:${TAG} actlabs.azurecr.io/repro:${TAG}

az acr login --name actlabs --subscription ACT-CSS-Readiness-NPRD
docker push actlabs.azurecr.io/repro:${TAG}

docker tag repro:${TAG} ashishvermapu/repro:${TAG}
docker push ashishvermapu/repro:${TAG}

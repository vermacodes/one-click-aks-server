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

# build docker image
docker build -t repro:${TAG} .

rm one-click-aks-server

docker tag repro:${TAG} actlab.azurecr.io/repro:${TAG}

az acr login --name actlab --subscription ACT-CSS-Readiness
docker push actlab.azurecr.io/repro:${TAG}

docker tag repro:${TAG} ashishvermapu/repro:${TAG}
docker push ashishvermapu/repro:${TAG}

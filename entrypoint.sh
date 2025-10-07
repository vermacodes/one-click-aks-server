#!/bin/bash

# Starting Redis Server
service redis-server start

# Start azurerm-msi-auth-proxy in the background
chmod +x azurerm-msi-auth-proxy
./azurerm-msi-auth-proxy &
PROXY_PID=$!

# Health check loop
(
  while true; do
    sleep 10
    if ! curl -sf http://localhost:${ARM_MSI_API_PROXY_PORT}/healthz > /dev/null; then
      echo "[WARN] azurerm-msi-auth-proxy health check failed, restarting..."
      kill $PROXY_PID
      ./azurerm-msi-auth-proxy &
      PROXY_PID=$!
    fi
  done
) &

# Run Server
# Following piece of code starts the server if it crashes. Health check every 5s
# TODO: make it resilient so that it doesn't crash at all :)

chmod +x one-click-aks-server
export ROOT_DIR=$(pwd)

while true; do

    export STATUS=$(curl -s http://localhost:${PORT}/status | jq -r .status)
    echo "$(date) : Status : $STATUS"
    if [ "$STATUS" != "OK" ]; then
        echo "$(date) : App Started."
        ./one-click-aks-server
    fi
    sleep 2s
done

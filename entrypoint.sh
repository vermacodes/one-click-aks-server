#!/bin/bash

# Starting Redis Server
service redis-server start

# Run Server
# Following peice of code starts the server if it crashes. Health check every 5s
# TODO: make it resilient so that it doesnt crash at all :)

chmod +x one-click-aks-server
export ROOT_DIR=$(pwd)
# export PORT="80"

echo $ARM_CLIENT_ID

while true; do
    export STATUS=$(curl -s http://localhost:${PORT}/status | jq -r .status)
    echo "$(date) : Status : $STATUS"
    if [ "$STATUS" != "OK" ]; then
        echo "$(date) : App Started."
        ./one-click-aks-server
    fi
    sleep 2s
done

export $(egrep -v '^#' .env | xargs)

az containerapp create \
  --name ashisverma-actlabs-server \
  --resource-group actlabs-app \
  --subscription ACT-CSS-Readiness \
  --environment actlabs-hub-env-eastus \
  --allow-insecure false \
  --image ashishvermapu/repro:alpha \
  --ingress 'external' \
  --min-replicas 1 \
  --max-replicas 1 \
  --target-port $PORT \
  --env-vars \
  "USE_SERVICE_PRINCIPAL=true" \
  "ARM_USER_PRINCIPAL_NAME=$ARM_USER_PRINCIPAL_NAME" \
  "USE_MSI=$USE_MSI" \
  "PORT=$PORT" \
  "ROOT_DIR=$ROOT_DIR" \
  "AUTH_TOKEN_AUD=$AUTH_TOKEN_AUD" \
  "AUTH_TOKEN_ISS=$AUTH_TOKEN_ISS" \
  "HTTPS_PORT=$HTTPS_PORT" \
  "HTTP_PORT=$HTTP_PORT" \
  "PROTECTED_LAB_SECRET=$PROTECTED_LAB_SECRET" \
  "TENANT_ID=$TENANT_ID" \
  "HTTP_REQUEST_TIMEOUT_SECONDS=$HTTP_REQUEST_TIMEOUT_SECONDS" \
  "LOG_LEVEL=$LOG_LEVEL" \
  "AZURE_SUBSCRIPTION_ID=$AZURE_SUBSCRIPTION_ID" \
  "AZURE_CLIENT_ID=$AZURE_CLIENT_ID" \
  "AZURE_CLIENT_SECRET=$AZURE_CLIENT_SECRET" \
  "AZURE_TENANT_ID=$AZURE_TENANT_ID" \
  "ACTLABS_HUB_URL=$ACTLABS_HUB_URL"

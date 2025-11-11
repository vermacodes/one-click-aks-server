#!/bin/bash

# This scripts output is fed as is to the go code.
# This must not print anything but the output its intended to print.

OPTION=$1
WORKSPACE=$2

LOG_FILE="workspaces.log"

ok() {
  echo -e "${GREEN}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: OKAY - $* ${NC}" >> $LOG_FILE
}

err() {
  echo -e "${RED}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: ERROR - $* ${NC}" >> $LOG_FILE
}

log() {
  echo -e "[$(date +'%Y-%m-%dT%H:%M:%S%z')]: INFO - $*" >> $LOG_FILE
}

warn() {
  echo -e "${YELLOW}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: WARN - $* ${NC}" >> $LOG_FILE
}

ok "script executed $(date)"

function setupAzureLogin() {
  if [[ -n "$ARM_SUBSCRIPTION_ID" ]]; then
    log "ARM_SUBSCRIPTION_ID detected: $ARM_SUBSCRIPTION_ID"

    # Set Azure CLI config directory to current directory/.azure
    local azure_config_dir="$PWD/.azure"
    export AZURE_CONFIG_DIR="$azure_config_dir"

    log "Setting Azure config directory to: $azure_config_dir"

    # Create .azure directory if it doesn't exist
    if [[ ! -d "$azure_config_dir" ]]; then
      log "Creating Azure config directory: $azure_config_dir"
      mkdir -p "$azure_config_dir"
    fi
    
    # Check if already logged in to the correct subscription
    local current_subscription
    current_subscription=$(az account show --query "id" --output tsv 2>>$LOG_FILE)
    
    if [[ "$current_subscription" != "$ARM_SUBSCRIPTION_ID" ]]; then
      warn "Current subscription ($current_subscription) differs from ARM_SUBSCRIPTION_ID ($ARM_SUBSCRIPTION_ID)"
      log "Performing Azure login..."

      # Check if MSI authentication is requested
      if [[ "$ARM_USE_MSI" == "true" ]]; then
        if [[ -n "$ARM_CLIENT_ID" ]]; then
          log "Using Managed Service Identity login with client ID: $ARM_CLIENT_ID"
          if az login --identity --username "$ARM_CLIENT_ID" --only-show-errors >>$LOG_FILE 2>&1; then
            ok "Azure MSI login successful"
          else
            err "Azure MSI login failed"
            return 1
          fi
        else
          err "ARM_USE_MSI is true but ARM_CLIENT_ID is not set"
          return 1
        fi
      else
        # Perform standard Azure login (will use device code flow or managed identity if available)
        if az login --only-show-errors >>$LOG_FILE 2>&1; then
          ok "Azure login successful"
        else
          err "Azure login failed"
          return 1
        fi
      fi
      
      # Set the subscription
      log "Setting subscription to: $ARM_SUBSCRIPTION_ID"
      if az account set --subscription "$ARM_SUBSCRIPTION_ID" --only-show-errors >>$LOG_FILE 2>&1; then
        ok "Successfully switched to subscription: $ARM_SUBSCRIPTION_ID"
      else
        err "Failed to switch to subscription: $ARM_SUBSCRIPTION_ID"
        return 1
      fi
    else
      log "Already logged in to correct subscription: $ARM_SUBSCRIPTION_ID"
    fi
    
    # Verify the subscription is set correctly
    local verified_subscription
    verified_subscription=$(az account show --query "id" --output tsv 2>>$LOG_FILE)
    if [[ "$verified_subscription" == "$ARM_SUBSCRIPTION_ID" ]]; then
      log "Verified current subscription: $verified_subscription"
    else
      err "Subscription verification failed. Expected: $ARM_SUBSCRIPTION_ID, Got: $verified_subscription"
      return 1
    fi
  else
    log "ARM_SUBSCRIPTION_ID not set, skipping Azure login setup"
  fi
}

# We are not using function from helper.sh cause this function needs to be quiet. i.e. no output.
function init() {
  log "starting terraform init"
  # Initialize terraform only if not.
  if [[ ! -f .terraform/terraform.tfstate ]] || [[ ! -f .terraform.lock.hcl ]]; then
    log "terraform not initialized, starting now"
    local attempt=1
    local max_attempts=3
    local success=false
    
    while [[ $attempt -le $max_attempts ]]; do
      log "terraform init attempt $attempt of $max_attempts $(date)"
      
      # Check if using local Azurite storage emulator
      if [[ "$storage_account_name" == "devstoreaccount1" ]]; then
        # Set terraform backend to local for development
        sed -i 's/backend "azurerm" {/backend "local" {/' providers.tf
        if terraform init >>$LOG_FILE 2>&1; then
          success=true
          ok "terraform init succeeded on attempt $attempt $(date)"
          break
        fi
      else
        # Set terraform backend to azurerm for production
        sed -i 's/backend "local" {/backend "azurerm" {/' providers.tf
        if terraform init \
          -migrate-state \
          -backend-config="subscription_id=$subscription_id" \
          -backend-config="resource_group_name=$resource_group_name" \
          -backend-config="storage_account_name=$storage_account_name" \
          -backend-config="container_name=$container_name" \
          -backend-config="key=$tf_state_file_name" >>$LOG_FILE 2>&1; then
          success=true
          ok "terraform init succeeded on attempt $attempt $(date)"
          break
        fi
      fi
      
      err "terraform init failed on attempt $attempt $(date)"
      
      # If not the last attempt, wait before retrying with backoff
      if [[ $attempt -lt $max_attempts ]]; then
        local wait_time=$((5 * attempt))
        log "waiting $wait_time seconds before retry $(date)"
        sleep $wait_time
      fi
      
      ((attempt++))
    done
    
    if [[ $success != true ]]; then
      err "terraform init failed after $max_attempts attempts $(date)"
      exit 1
    fi
  else
    log "terraform already initialized"
  fi
}


function listWorkspaces() {
  log "listing workspaces"

  workspaces=$(terraform workspace list 2>>$LOG_FILE)
  IFS=$'\n'
  list=""
  for line in $workspaces; do
    #line=${line/*\* /} # Removes the * from selected workspace.
    #line=${line/*\ /} # Removes leading spaces.
    line=$(echo ${line} | tr -s ' ')
    if [[ "${list}" == "" ]]; then
      list="${line}"
    else
      list="${list},${line}"
    fi
  done
  printf ${list}
}

# Script starts here
setupAzureLogin
init

if [[ "$OPTION" == "list" ]]; then
  listWorkspaces
  exit 0
fi

terraform workspace $OPTION $WORKSPACE >>$LOG_FILE 2>&1

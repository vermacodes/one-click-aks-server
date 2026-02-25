#!/bin/bash

function setup_azure_login() {
  if [[ -n "$ARM_SUBSCRIPTION_ID" ]]; then
    log "ACTLABS SERVER ID: $HOSTNAME"
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
    current_subscription=$(az account show --query "id" --output tsv 2>/dev/null)
    
    if [[ "$current_subscription" != "$ARM_SUBSCRIPTION_ID" ]]; then
      log "Current subscription ($current_subscription) differs from ARM_SUBSCRIPTION_ID ($ARM_SUBSCRIPTION_ID)"
      log "Performing Azure login..."
      
      # Check if MSI authentication is requested
      if [[ "$ARM_USE_MSI" == "true" ]]; then
        if [[ -n "$ARM_CLIENT_ID" ]]; then
          log "Using Managed Service Identity login with client ID: $ARM_CLIENT_ID"
          if az login --identity --client-id "$ARM_CLIENT_ID" --only-show-errors > /dev/null 2>&1; then
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
        if az login --only-show-errors > /dev/null 2>&1; then
          ok "Azure login successful"
        else
          err "Azure login failed"
          return 1
        fi
      fi
      
      # Set the subscription
      log "Setting subscription to: $ARM_SUBSCRIPTION_ID"
      if az account set --subscription "$ARM_SUBSCRIPTION_ID" --only-show-errors; then
        ok "Successfully switched to subscription: $ARM_SUBSCRIPTION_ID"
      else
        err "Failed to switch to subscription: $ARM_SUBSCRIPTION_ID"
        return 1
      fi
    else
      ok "Already logged in to correct subscription: $ARM_SUBSCRIPTION_ID"
    fi
    
    # Verify the subscription is set correctly
    local verified_subscription
    verified_subscription=$(az account show --query "id" --output tsv 2>/dev/null)
    if [[ "$verified_subscription" == "$ARM_SUBSCRIPTION_ID" ]]; then
      ok "Verified current subscription: $verified_subscription"
    else
      err "Subscription verification failed. Expected: $ARM_SUBSCRIPTION_ID, Got: $verified_subscription"
      return 1
    fi
  else
    log "ARM_SUBSCRIPTION_ID not set, skipping Azure login setup"
  fi
}

# Setup Azure login if ARM_SUBSCRIPTION_ID is provided
setup_azure_login
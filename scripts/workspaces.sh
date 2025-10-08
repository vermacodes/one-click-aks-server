#!/bin/bash

# This scripts output is fed as is to the go code.
# This must not print anything but the output its intended to print.

OPTION=$1
WORKSPACE=$2

LOG_FILE="workspaces.log"

echo "script executed $(date)" >> $LOG_FILE

function setupAzureLogin() {
  if [[ -n "$ARM_SUBSCRIPTION_ID" ]]; then
    echo "ARM_SUBSCRIPTION_ID detected: $ARM_SUBSCRIPTION_ID" >> $LOG_FILE
    
    # Set Azure CLI config directory to current directory/.azure
    local azure_config_dir="$PWD/.azure"
    export AZURE_CONFIG_DIR="$azure_config_dir"
    
    echo "Setting Azure config directory to: $azure_config_dir" >> $LOG_FILE
    
    # Create .azure directory if it doesn't exist
    if [[ ! -d "$azure_config_dir" ]]; then
      echo "Creating Azure config directory: $azure_config_dir" >> $LOG_FILE
      mkdir -p "$azure_config_dir"
    fi
    
    # Check if already logged in to the correct subscription
    local current_subscription
    current_subscription=$(az account show --query "id" --output tsv 2>>$LOG_FILE)
    
    if [[ "$current_subscription" != "$ARM_SUBSCRIPTION_ID" ]]; then
      echo "Current subscription ($current_subscription) differs from ARM_SUBSCRIPTION_ID ($ARM_SUBSCRIPTION_ID)" >> $LOG_FILE
      echo "Performing Azure login..." >> $LOG_FILE
      
      # Check if MSI authentication is requested
      if [[ "$ARM_USE_MSI" == "true" ]]; then
        if [[ -n "$ARM_CLIENT_ID" ]]; then
          echo "Using Managed Service Identity login with client ID: $ARM_CLIENT_ID" >> $LOG_FILE
          if az login --identity --username "$ARM_CLIENT_ID" --only-show-errors >>$LOG_FILE 2>&1; then
            echo "Azure MSI login successful" >> $LOG_FILE
          else
            echo "Azure MSI login failed" >> $LOG_FILE
            return 1
          fi
        else
          echo "ARM_USE_MSI is true but ARM_CLIENT_ID is not set" >> $LOG_FILE
          return 1
        fi
      else
        # Perform standard Azure login (will use device code flow or managed identity if available)
        if az login --only-show-errors >>$LOG_FILE 2>&1; then
          echo "Azure login successful" >> $LOG_FILE
        else
          echo "Azure login failed" >> $LOG_FILE
          return 1
        fi
      fi
      
      # Set the subscription
      echo "Setting subscription to: $ARM_SUBSCRIPTION_ID" >> $LOG_FILE
      if az account set --subscription "$ARM_SUBSCRIPTION_ID" --only-show-errors >>$LOG_FILE 2>&1; then
        echo "Successfully switched to subscription: $ARM_SUBSCRIPTION_ID" >> $LOG_FILE
      else
        echo "Failed to switch to subscription: $ARM_SUBSCRIPTION_ID" >> $LOG_FILE
        return 1
      fi
    else
      echo "Already logged in to correct subscription: $ARM_SUBSCRIPTION_ID" >> $LOG_FILE
    fi
    
    # Verify the subscription is set correctly
    local verified_subscription
    verified_subscription=$(az account show --query "id" --output tsv 2>>$LOG_FILE)
    if [[ "$verified_subscription" == "$ARM_SUBSCRIPTION_ID" ]]; then
      echo "Verified current subscription: $verified_subscription" >> $LOG_FILE
    else
      echo "Subscription verification failed. Expected: $ARM_SUBSCRIPTION_ID, Got: $verified_subscription" >> $LOG_FILE
      return 1
    fi
  else
    echo "ARM_SUBSCRIPTION_ID not set, skipping Azure login setup" >> $LOG_FILE
  fi
}

# We are not using function from helper.sh cause this function needs to be quiet. i.e. no output.
function enableSharedKeyAccess() {
  # Enable shared key access to storage account if not already enabled
  sharedKeyAccess=$(az storage account show --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --query "allowSharedKeyAccess" --output tsv 2>>$LOG_FILE)
  if [[ ${sharedKeyAccess} == "false" ]]; then
    az storage account update --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --allow-shared-key-access true >>$LOG_FILE 2>&1
  fi
}

function enablePublicNetworkAccess() {
  # Fetch public network access and default network rule in a single command
  networkSettings=$(az storage account show --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --query "{publicNetworkAccess:publicNetworkAccess, defaultAction:networkRuleSet.defaultAction}" --output json 2>>$LOG_FILE)

  publicNetworkAccess=$(echo "$networkSettings" | jq -r '.publicNetworkAccess')
  defaultAction=$(echo "$networkSettings" | jq -r '.defaultAction')

  # Enable public network access if not already enabled
  if [[ "$publicNetworkAccess" != "Enabled" || "$defaultAction" != "Allow" ]]; then
    az storage account update --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --public-network-access Enabled --default-action Allow >>$LOG_FILE 2>&1
  fi
}

# We are not using function from helper.sh cause this function needs to be quiet. i.e. no output.
function init() {
  # Initialize terraform only if not.
  if [[ ! -f .terraform/terraform.tfstate ]] || [[ ! -f .terraform.lock.hcl ]]; then
    local attempt=1
    local max_attempts=3
    local success=false
    
    while [[ $attempt -le $max_attempts ]]; do
      echo "terraform init attempt $attempt of $max_attempts $(date)" >> $LOG_FILE
      
      # Check if using local Azurite storage emulator
      if [[ "$storage_account_name" == "devstoreaccount1" ]]; then
        # Set terraform backend to local for development
        sed -i 's/backend "azurerm" {/backend "local" {/' providers.tf
        if terraform init >>$LOG_FILE 2>&1; then
          success=true
          echo "terraform init succeeded on attempt $attempt $(date)" >> $LOG_FILE
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
          echo "terraform init succeeded on attempt $attempt $(date)" >> $LOG_FILE
          break
        fi
      fi
      
      echo "terraform init failed on attempt $attempt $(date)" >> $LOG_FILE
      
      # If not the last attempt, wait before retrying with backoff
      if [[ $attempt -lt $max_attempts ]]; then
        local wait_time=$((5 * attempt))
        echo "waiting $wait_time seconds before retry $(date)" >> $LOG_FILE
        sleep $wait_time
      fi
      
      ((attempt++))
    done
    
    if [[ $success != true ]]; then
      echo "terraform init failed after $max_attempts attempts $(date)" >> $LOG_FILE
      exit 1
    fi
  fi
}

function listWorkspaces() {

  workspaces=$(terraform workspace list)
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

function selectWorkspace() {
  terraform workspace select $WORKSPACE >>$LOG_FILE 2>&1
}

function createWorkspace() {
  terraform workspace create $WORKSPACE >>$LOG_FILE 2>&1
}

# Script starts here.
# cd ${ROOT_DIR}/tf

if [[ "$ARM_SUBSCRIPTION_ID" == "" ]]; then
  export ARM_SUBSCRIPTION_ID=$(az account show --output json --only-show-error | jq -r .id)
fi

# enableSharedKeyAccess
# enablePublicNetworkAccess
setupAzureLogin
init

if [[ "$OPTION" == "list" ]]; then
  listWorkspaces
  # Cleanup: Restore providers.tf if we're in development mode
  if [[ "$storage_account_name" == "devstoreaccount1" ]]; then
    restore_providers
  fi
  exit 0
fi

terraform workspace $OPTION $WORKSPACE

# Cleanup: Restore providers.tf if we're in development mode
if [[ "$storage_account_name" == "devstoreaccount1" ]]; then
  restore_providers
fi

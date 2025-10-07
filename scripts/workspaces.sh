#!/bin/bash

# This scripts output is fed as is to the go code.
# This must not print anything but the output its intended to print.

OPTION=$1
WORKSPACE=$2

LOG_FILE="workspaces.log"

echo "script executed $(date)" >> $LOG_FILE

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

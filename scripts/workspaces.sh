#!/bin/bash

# This scripts output is fed as is to the go code.
# This must not print anything but the output its intended to print.

OPTION=$1
WORKSPACE=$2

LOG_FILE="workspaces.log"

# We are not using function from helper.sh cause this function needs to be quiet. i.e. no output.
function enableSharedKeyAccess() {
  # Enable shared key access to storage account if not already enabled
  sharedKeyAccess=$(az storage account show --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --query "allowSharedKeyAccess" --output tsv 2>>$LOG_FILE)
  if [[ ${sharedKeyAccess} == "false" ]]; then
    az storage account update --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --allow-shared-key-access true >>$LOG_FILE 2>&1
  fi
}

function enablePublicNetworkAccess() {
  # Enable public network access to storage account if not already enabled
  publicNetworkAccess=$(az storage account show --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --query "publicNetworkAccess" --output tsv 2>>$LOG_FILE)
  if [[ ${publicNetworkAccess} != "Enabled" ]]; then
    az storage account update --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --public-network-access Enabled >>$LOG_FILE 2>&1
  fi
}

# We are not using function from helper.sh cause this function needs to be quiet. i.e. no output.
function init() {
  # Initialize terraform only if not.
  if [[ ! -f .terraform/terraform.tfstate ]] || [[ ! -f .terraform.lock.hcl ]]; then
    terraform init \
      -migrate-state \
      -backend-config="subscription_id=$subscription_id" \
      -backend-config="resource_group_name=$resource_group_name" \
      -backend-config="storage_account_name=$storage_account_name" \
      -backend-config="container_name=$container_name" \
      -backend-config="key=$tf_state_file_name" >>$LOG_FILE 2>&1
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
  terraform workspace select $WORKSPACE
}

function createWorkspace() {
  terraform workspace create $WORKSPACE
}

# Script starts here.
cd ${ROOT_DIR}/tf

if [[ "$ARM_SUBSCRIPTION_ID" == "" ]]; then
  export ARM_SUBSCRIPTION_ID=$(az account show --output json --only-show-error | jq -r .id)
fi

enableSharedKeyAccess
enablePublicNetworkAccess
init

if [[ "$OPTION" == "list" ]]; then
  listWorkspaces
  exit 0
fi

terraform workspace $OPTION $WORKSPACE

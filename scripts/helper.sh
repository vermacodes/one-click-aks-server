#!/bin/bash

# This script breaks cluster.
cd $ROOT_DIR

# Add some color
RED='\033[0;91m'
GREEN='\033[0;92m'
YELLOW='\033[0;93m'
PURPLE='\033[0;95m'
NC='\033[0m' # No Color

err() {
  echo -e "${RED}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: ERROR - $* ${NC}" >&1
}

log() {
  echo -e "[$(date +'%Y-%m-%dT%H:%M:%S%z')]: INFO - $*" >&1
}

warn() {
  echo -e "${YELLOW}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: WARN - $* ${NC}" >&1
}

ok() {
  echo -e "${GREEN}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: OKAY - $* ${NC}" >&1
}

chat() {
  echo -e "${PURPLE}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: CHAT - $* ${NC}" >&1
}

gap() {
  echo -e ""
  echo -e ""
  echo -e "******************************************************************"
  echo -e ""
  echo -e ""
}

# function that seeps for n seconds and prints remaining time in minutes and seconds every 10 seconds
function sleep_and_print() {
  local total_seconds=$1
  while ((total_seconds > 0)); do
    sleep 10
    ((total_seconds -= 10))
    local minutes=$((total_seconds / 60))
    local seconds=$((total_seconds % 60))
    echo "Remaining time: $minutes minutes $seconds seconds"
  done
}

function change_to_root_dir() {
  log "Changing to root directory"
  cd $ROOT_DIR
}

function changeToTerraformDirectory() {
  log "Changing to terraform directory"
  cd $ROOT_DIR/tf
}

function get_aks_credentials() {
  log "Pulling AKS credentials"

  if [[ ${AKS_LOGIN} != "" ]]; then
    ok "AKS Login Command -> ${AKS_LOGIN}"
    echo ${AKS_LOGIN} --only-show-errors | bash
  elif [[ ${AKS_LOGIN} == "" ]]; then
    log "AKS Login command not available"
  else
    err "Expected either AKS login command or an empty string. Found this -> ${AKS_LOGIN}"
  fi

  change_to_root_dir
}

function get_kubectl() {
  log "Checking if kubectl exists"
  which kubectl >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    log "kubectl not found. installing."
    az aks install-cli --only-show-errors
  fi
}

function tf_init() {
  log "Initializing"

  # Change to TF Directory
  changeToTerraformDirectory
  enableSharedKeyAccess

  # Initialize terraform only if not.
  if [[ ! -f .terraform/terraform.tfstate ]] || [[ ! -f .terraform.lock.hcl ]]; then
    terraform init \
      -migrate-state \
      -backend-config="subscription_id=$subscription_id" \
      -backend-config="resource_group_name=$resource_group_name" \
      -backend-config="storage_account_name=$storage_account_name" \
      -backend-config="container_name=$container_name" \
      -backend-config="key=$tf_state_file_name"
    ok "Initialization Complted"
  else
    ok "Already Initialized - Skipped"
  fi

  # Change to root directory
  # change_to_root_dir
}

function get_variables_from_tf_output() {
  log "Pulling variables from TF output"
  changeToTerraformDirectory

  output=$(terraform output -json)
  log "output -> ${output}"

  # Iterate through each output variable and set as an environment variable
  if [[ ${output} != "{}" ]]; then
    while read -r key value; do
      export "$(echo "$key" | tr '[:lower:]' '[:upper:]')"="$value"
    done <<<"$(echo "$output" | jq -r 'to_entries[] | "\(.key) \(.value.value)"')"

  elif [[ ${output} == "{}" ]]; then
    log "terraform output not found."
  else
    err "Expected terraform outputs or an empty object {}. But found -> ${output}"
  fi

  change_to_root_dir
}

function init() {
  if [[ ${SCRIPT_MODE} == "apply" ]]; then
    gap
  fi
  log "Initializing Environment"
  change_to_root_dir
  tf_init
  get_variables_from_tf_output
  get_aks_credentials
  # changeToTerraformDirectory
  get_kubectl
}

function enableSharedKeyAccess() {
  # Enable shared key access to storage account if not already enabled
  sharedKeyAccess=$(az storage account show --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --query "allowSharedKeyAccess" --output tsv)
  if [[ ${sharedKeyAccess} == "false" ]]; then
    az storage account update --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --allow-shared-key-access true
  fi
}

# Adding sources
source ${ROOT_DIR}/scripts/aro_shared_functions.sh
source ${ROOT_DIR}/scripts/shared_functions.sh

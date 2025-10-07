#!/bin/bash

# This script breaks cluster.
# cd $ROOT_DIR

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

  # change_to_root_dir
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
  # changeToTerraformDirectory
  # enableSharedKeyAccess

  log "Present directory $(pwd)"

  # Setting MSI variables to blank to allow authentication using MSI
  export MSI_ENDPOINT=""
  export MSI_SECRET=""

  # Initialize terraform only if not.
  if [[ ! -f .terraform/terraform.tfstate ]] || [[ ! -f .terraform.lock.hcl ]]; then
    # Check if using local Azurite storage emulator
    if [[ "$storage_account_name" == "devstoreaccount1" ]]; then
      
      # Set terraform backend to local for development
      sed -i 's/backend "azurerm" {/backend "local" {/' providers.tf
      terraform init
    else
      # Set terraform backend to azurerm for production
      sed -i 's/backend "local" {/backend "azurerm" {/' providers.tf
      terraform init \
        -migrate-state \
        -backend-config="subscription_id=$subscription_id" \
        -backend-config="resource_group_name=$resource_group_name" \
        -backend-config="storage_account_name=$storage_account_name" \
        -backend-config="container_name=$container_name" \
        -backend-config="key=$tf_state_file_name"
    fi
    ok "Initialization Completed"
  else
    ok "Already Initialized - Skipped"
  fi

  # Setting MSI variables to back to its original value
  export MSI_ENDPOINT=${IDENTITY_ENDPOINT}
  export MSI_SECRET=${IDENTITY_HEADER}

  # Change to root directory
  # change_to_root_dir
}

# function get_variables_from_tf_output() {
#   log "Pulling variables from TF output"
#   changeToTerraformDirectory

#   output=$(terraform output -json)
#   log "output -> ${output}"

#   # Iterate through each output variable and set as an environment variable
#   if [[ ${output} != "{}" ]]; then
#     while read -r key value; do
#       export "$(echo "$key" | tr '[:lower:]' '[:upper:]')"="$value"
#     done <<<"$(echo "$output" | jq -r 'to_entries[] | "\(.key) \(.value.value)"')"

#   elif [[ ${output} == "{}" ]]; then
#     log "terraform output not found."
#   else
#     err "Expected terraform outputs or an empty object {}. But found -> ${output}"
#   fi

#   change_to_root_dir
# }


# get_variables_from_tf_output
# ---------------------------
# This function pulls all outputs from Terraform in JSON format and recursively flattens any nested objects.
# It then exports each key-value pair as an environment variable, using uppercase and underscores for nested keys.
# Example: a nested output like {"foo": {"bar": "baz"}} will result in FOO_BAR=baz in the environment.
#
# Usage: Call this function after running 'terraform output -json' in the correct directory.
# It is robust for any level of nesting in the Terraform outputs.
#
# Dependencies: jq (for JSON parsing and flattening)
#
# This is useful for scripting and CI/CD pipelines where you want to consume all Terraform outputs as environment variables.

# get_variables_from_tf_output
# ---------------------------
# Exports all Terraform outputs as environment variables, flattening nested objects.
# Handles the standard Terraform output -json structure, including .value fields.
# Example: cluster_identity_ids.0_aro-operator will become CLUSTER_IDENTITY_IDS_0_ARO_OPERATOR
# Requires: jq

# function get_variables_from_tf_output() {
#   output=$(terraform output -json)
#   if [[ ${output} != "{}" ]]; then
#     jq -r '
#       to_entries[] |
#       if (.value.value | type == "object") then
#         .value.value | to_entries[] | "\(.key | ascii_upcase)=\(.value)" | gsub("[\.-]"; "_")
#       else
#         "\(.key | ascii_upcase)=\(.value.value)" | gsub("[\.-]"; "_")
#       end
#     ' <<< "$output" | while IFS= read -r line; do
#       if [[ "$line" =~ ^[A-Z0-9_]+=.*$ ]]; then
#         echo "next export would be $line"
#         export "$line"
#       fi
#     done
#   else
#     echo "terraform output not found."
#   fi
# }

# Recursive function to flatten nested objects and export as environment variables
_flatten_object() {
    local json_data="$1"
    local path_prefix="$2"
    
    # Get all keys at current level
    while IFS= read -r key; do
        if [[ -z "$key" ]]; then
            continue
        fi
        
        # Build the environment variable name
        local env_var_name
        if [[ -z "$path_prefix" ]]; then
            env_var_name="$(echo "$key" | tr '[:lower:]' '[:upper:]' | tr '-' '_')"
        else
            env_var_name="${path_prefix}_$(echo "$key" | tr '[:lower:]' '[:upper:]' | tr '-' '_')"
        fi
        
        # Get the value type
        local value_type=$(echo "$json_data" | jq -r ".[\"$key\"] | type")
        
        if [[ "$value_type" == "object" ]]; then
            # Recursively process nested objects
            local nested_json=$(echo "$json_data" | jq -c ".[\"$key\"]")
            _flatten_object "$nested_json" "$env_var_name"
        elif [[ "$value_type" == "array" ]]; then
            # Handle arrays by creating indexed variables
            local array_length=$(echo "$json_data" | jq -r ".[\"$key\"] | length")
            
            for ((i=0; i<array_length; i++)); do
                local item_value=$(echo "$json_data" | jq -r ".[\"$key\"][$i]")
                local item_type=$(echo "$json_data" | jq -r ".[\"$key\"][$i] | type")
                
                if [[ "$item_type" == "object" ]]; then
                    # Recursive call for object array items
                    local nested_json=$(echo "$json_data" | jq -c ".[\"$key\"][$i]")
                    _flatten_object "$nested_json" "${env_var_name}_${i}"
                else
                    # Simple array item
                    export "${env_var_name}_${i}"="$item_value"
                    echo "Exported: ${env_var_name}_${i}=\"$item_value\""
                fi
            done
        else
            # Simple value (string, number, boolean, null)
            local value=$(echo "$json_data" | jq -r ".[\"$key\"]")
            export "$env_var_name"="$value"
            echo "Exported: $env_var_name=\"$value\""
        fi
        
    done < <(echo "$json_data" | jq -r 'keys[]')
}

# Main function to flatten terraform output and export as environment variables
get_variables_from_tf_output() {
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        echo "Error: jq is required but not installed"
        return 1
    fi
    
    # Check if terraform is available
    if ! command -v terraform &> /dev/null; then
        echo "Error: terraform is required but not installed"
        return 1
    fi
    
    echo "Getting Terraform output..."
    
    # Get terraform output as JSON
    local tf_output
    if ! tf_output=$(terraform output -json 2>/dev/null); then
        echo "Error: Failed to get terraform output. Make sure you're in a terraform directory with state."
        return 1
    fi
    
    if [[ "$tf_output" == "{}" ]] || [[ -z "$tf_output" ]]; then
        echo "Warning: No terraform outputs found"
        return 0
    fi
    
    echo "Processing Terraform outputs..."
    echo "----------------------------------------"
    
    # Process each terraform output
    while IFS= read -r output_key; do
        if [[ -z "$output_key" ]]; then
            continue
        fi
        
        # Get the value from the terraform output
        local output_value=$(echo "$tf_output" | jq -c ".[\"$output_key\"].value")
        local value_type=$(echo "$tf_output" | jq -r ".[\"$output_key\"].value | type")
        
        # Create base environment variable name
        local base_env_name="$(echo "$output_key" | tr '[:lower:]' '[:upper:]' | tr '-' '_')"
        
        if [[ "$value_type" == "object" ]]; then
            # Recursively flatten object
            _flatten_object "$output_value" "$base_env_name"
        elif [[ "$value_type" == "array" ]]; then
            # Handle arrays
            local array_length=$(echo "$output_value" | jq -r 'length')
            
            for ((i=0; i<array_length; i++)); do
                local item_value=$(echo "$output_value" | jq -r ".[$i]")
                local item_type=$(echo "$output_value" | jq -r ".[$i] | type")
                
                if [[ "$item_type" == "object" ]]; then
                    local nested_json=$(echo "$output_value" | jq -c ".[$i]")
                    _flatten_object "$nested_json" "${base_env_name}_${i}"
                else
                    export "${base_env_name}_${i}"="$item_value"
                    echo "Exported: ${base_env_name}_${i}=\"$item_value\""
                fi
            done
        else
            # Simple value
            local simple_value=$(echo "$tf_output" | jq -r ".[\"$output_key\"].value")
            export "$base_env_name"="$simple_value"
            echo "Exported: $base_env_name=\"$simple_value\""
        fi
        
    done < <(echo "$tf_output" | jq -r 'keys[]')
    
    echo "----------------------------------------"
    echo "Terraform output variables exported successfully!"
}

function init() {
  if [[ ${SCRIPT_MODE} == "apply" ]]; then
    gap
  fi
  log "Initializing Environment"
  # change_to_root_dir
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

function enablePublicNetworkAccess() {
  # Enable public network access to storage account if not already enabled
  publicNetworkAccess=$(az storage account show --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --query "networkRuleSet.defaultAction" --output tsv)
  if [[ ${publicNetworkAccess} == "Deny" ]]; then
    az storage account update --name "$storage_account_name" -g "$resource_group_name" --subscription "$subscription_id" --default-action Allow
  fi
}

# Adding sources
source ${ROOT_DIR}/scripts/aro_shared_functions.sh
source ${ROOT_DIR}/scripts/shared_functions.sh


#!/bin/bash

# Usage:
#   terraform_debug.sh <action> [options]
#
# Actions:
#   plan        Run terraform plan
#   apply       Run terraform apply
#   destroy     Run terraform destroy
#   init        Run terraform init
#   list        List terraform state resources
#
# Options:
#   -l <level>, --log <level>      Set TF_LOG level (debug, trace)
#   debug, trace                   Shortcut for log level
#   -b local, --backend local      Remove azurerm backend for local state
#   -m, --use-msi                  Enable MSI authentication for Terraform
#
# Examples:
#   ./terraform_debug.sh plan -l debug
#   ./terraform_debug.sh apply --log trace
#   ./terraform_debug.sh destroy -b local -m
#   ./terraform_debug.sh init


action=""
tf_log=""
backend_local=""
use_msi=""

# Parse arguments for action, log level, backend, and MSI usage
while [[ $# -gt 0 ]]; do
  case "$1" in
    plan|apply|destroy|init|list)
      action="$1"
      shift
      ;;
    -l|--log)
      if [[ -n "$2" ]]; then
        tf_log="$2"
        shift 2
      else
        echo "Error: Missing log level after $1"
        exit 1
      fi
      ;;
    debug|trace)
      tf_log="$1"
      shift
      ;;
    -b|--backend)
      if [[ -n "$2" && "$2" == "local" ]]; then
        backend_local="true"
        shift 2
      else
        shift
      fi
      ;;
    --use-msi|-m)
      use_msi="true"
      shift
      ;;
    *)
      shift
      ;;
  esac
done

# Read the instructions very carefully.

# no need to update these variable.
export ROOT_DIR=$(pwd)
source $ROOT_DIR/scripts/helper.sh

# Set TF_LOG if requested
if [[ -n "$tf_log" ]]; then
  export TF_LOG="$tf_log"
  echo "Terraform logging enabled: TF_LOG=$TF_LOG"
fi

# Following variables are used by terraform. Change them as you need.
# You will get these from the output on the UI.
# just run terraform plan and copy the environment variables which start with TF_VAR
# modify them to be on liners and paste them here.
export TF_VAR_resource_group='{"location": "East US"}'
export TF_VAR_network_security_groups='[]'
export TF_VAR_container_registries='[]'
export TF_VAR_subnets='[]'
export TF_VAR_firewalls='[]'
export TF_VAR_app_gateways='[]'
export TF_VAR_jumpservers='[]'
export TF_VAR_virtual_networks='[]'
export TF_VAR_kubernetes_clusters='[]'
export TF_VAR_aro_clusters='[]'
export TF_VAR_aro_rp_first_party_service_principal_id=$AZURE_RED_HAT_OPENSHIFT_RP_FIRST_PARTY_SP_ID

# Following variables are used by the helper script. No need to change them.
export terraform_directory="tf"
export root_directory=$ROOT_DIR
export subscription_id=$ACTLABS_HUB_SUBSCRIPTION_ID
export resource_group_name=$ACTLABS_HUB_RESOURCE_GROUP_NAME
export storage_account_name=$ACTLABS_HUB_STORAGE_ACCOUNT_NAME
export container_name="repro-project-tf-state-files"
export tf_state_file_name="${USER_ALIAS}-terraform.tfstate"

# Remove backend config only if requested
if [[ "$backend_local" == "true" ]]; then
  sed -i '/backend "azurerm" {}/d' $root_directory/$terraform_directory/providers.tf
  echo "Removed azurerm backend configuration for local backend."
fi

# Set MSI environment variables only if requested
if [[ "$use_msi" == "true" ]]; then
  export ARM_USE_MSI=true
  export ARM_USE_AZUREAD=true
  export ARM_CLIENT_ID=589f5c83-f27d-4a89-9dd2-75a11a0c7d6a # only necessary for user assigned identity
  export ARM_MSI_ENDPOINT="http://localhost:${ARM_MSI_API_PROXY_PORT}/msi/token"
  export ARM_MSI_API_VERSION="2019-08-01"
  export MSI_ENDPOINT=""
  export MSI_SECRET=""
  echo "MSI environment variables set."
fi

function init() {
  log "Initializing"

  # Initialize terraform only if not.
  if [[ ! -f .terraform/terraform.tfstate ]] || [[ ! -f .terraform.lock.hcl ]]; then
    terraform init \
      -migrate-state \
      -backend-config="subscription_id=$subscription_id" \
      -backend-config="resource_group_name=$resource_group_name" \
      -backend-config="storage_account_name=$storage_account_name" \
      -backend-config="container_name=$container_name" \
      -backend-config="key=$tf_state_file_name"
    ok "Initialization Completed"
  else
    ok "Already Initialized - Skipped"
  fi
}

function plan() {
  log "Planning"
  terraform plan
}

function apply() {
  log "Applying"
  terraform apply -auto-approve
  if [ $? -ne 0 ]; then
    err "Terraform Apply Failed"
    exit 1
  fi
}

function destroy() {
  log "Destroying"
  terraform destroy -auto-approve
  if [ $? -ne 0 ]; then
    err "Terraform Destroy Failed"
    exit 1
  fi
}

function list() {
  log "Listing"
  terraform state list
}

##
## Script starts here.
##


if [[ "$ARM_SUBSCRIPTION_ID" == "" ]]; then
  export ARM_SUBSCRIPTION_ID=$(az account show --output json | jq -r .id)
fi

cd $root_directory/$terraform_directory
log "Terraform Environment Variables"
env | grep "TF_VAR" | awk -F"=" '{printf "%s=", $1; print $2 | "jq ."; close("jq ."); }'
echo ""

# Delete existing if init
if [[ "$action" == "init" ]]; then
  rm -rf .terraform*
  init
fi

# Terraform Init - Sourced from helper script.
# tf_init

if [[ "$action" == "plan" ]]; then
  plan
elif [[ "$action" == "apply" ]]; then
  apply
elif [[ "$action" == "destroy" ]]; then
  destroy
fi

ok "Terraform Action End"

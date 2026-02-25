#!/bin/bash

action=$1

source $ROOT_DIR/scripts/helper.sh || { echo "Failed to source helper.sh"; exit 1; }

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
  export ARM_SUBSCRIPTION_ID=$(az account show --output json --only-show-error | jq -r .id)
fi

# cd $root_directory/$terraform_directory
log "Terraform Environment Variables"
env | grep "TF_VAR" | while IFS='=' read -r key val; do
  # Try to parse as JSON, fallback to printing as a quoted string if jq fails
  if echo "$val" | jq . >/dev/null 2>&1; then
    printf "%s=" "$key"
    echo "$val" | jq .
  else
    printf "%s=" "$key"
    printf '"%s"\n' "$val"
  fi
done
echo ""

if [[ -n "$ARM_USER_PRINCIPAL_NAME" ]]; then
  export TF_VAR_created_by=$ARM_USER_PRINCIPAL_NAME
fi

# Delete existing if init
if [[ "$action" == "init" ]]; then
  rm -rf .terraform*
fi

# Terraform Init - Sourced from helper script.
tf_init

if [[ "$action" == "plan" ]]; then
  plan
elif [[ "$action" == "apply" ]]; then
  apply
elif [[ "$action" == "destroy" ]]; then
  destroy
fi

ok "Terraform Action End"

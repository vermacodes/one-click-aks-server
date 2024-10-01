#!/bin/bash

# This scripts output is fed as is to the go code.
# This must not print anything but the output its intended to print.

OPTION=$1
WORKSPACE=$2

source $ROOT_DIR/scripts/helper.sh

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

tf_init

if [[ "$OPTION" == "list" ]]; then
  listWorkspaces
  exit 0
fi

terraform workspace $OPTION $WORKSPACE

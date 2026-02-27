
# Loop to create all required identities for each cluster
locals {
  aro_identity_role = {
    "aro-cluster"              = "Azure Red Hat OpenShift Federated Credential"
    "cloud-controller-manager" = "Azure Red Hat OpenShift Cloud Controller Manager"
    "ingress"                  = "Azure Red Hat OpenShift Cluster Ingress Operator"
    "machine-api"              = "Azure Red Hat OpenShift Machine API Operator"
    "file-csi-driver"          = "Azure Red Hat OpenShift File Storage Operator"
    "disk-csi-driver"          = "Azure Red Hat OpenShift Disk Storage Operator"
    "cloud-network-config"     = "Azure Red Hat OpenShift Network Operator"
    "image-registry"           = "Azure Red Hat OpenShift Image Registry Operator"
    "aro-operator"             = "Azure Red Hat OpenShift Service Operator"
  }

  identity_types = keys(local.aro_identity_role)

  # Build a map of all assignments needed: key is "identitytype_clusteridx", value is a map with identity_type, cluster_idx, and role
  identity_assignments = var.aro_clusters == null ? {} : {
    for pair in setproduct(local.identity_types, range(length(var.aro_clusters))) :
    "${pair[0]}_${pair[1]}" => {
      identity_type = pair[0]
      cluster_idx   = pair[1]
      role          = local.aro_identity_role[pair[0]]
    }
  }
}

resource "azurerm_user_assigned_identity" "aro_cluster_identity" {
  for_each            = local.identity_assignments
  name                = "${module.naming.user_assigned_identity.name}-${each.value.identity_type}-${each.value.cluster_idx}"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
}


# Single role assignment resource for all identities
resource "azurerm_role_assignment" "aro_identity_role" {
  for_each             = local.identity_assignments
  principal_id         = azurerm_user_assigned_identity.aro_cluster_identity[each.key].principal_id
  scope                = azurerm_resource_group.this.id
  role_definition_name = each.value.role
  principal_type       = "ServicePrincipal"
}


# Role assignment for ARO RP first-party service principal on vnet
resource "azurerm_role_assignment" "aro_rp_first_party_id_network_contributor" {
  count                = length(var.aro_clusters) >= 1 ? 1 : 0
  principal_id         = var.aro_rp_first_party_service_principal_id
  scope                = azurerm_resource_group.this.id
  role_definition_name = "Network Contributor"
  principal_type       = "ServicePrincipal"
}

# ARO Cluster

# This feature is still in preview. So the APIs are not yet available in Terraform.
# we will complete this as soon as the apis are available.

# Create ARO cluster using null_resource and local-exec
resource "null_resource" "aro_cluster" {
  count = var.aro_clusters == null ? 0 : length(var.aro_clusters)

  # Store all critical values in triggers to ensure proper dependency tracking
  triggers = {
    resource_group  = azurerm_resource_group.this.name
    cluster_name    = "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${count.index}"
    subscription_id = data.azurerm_client_config.current.subscription_id
    vnet_name       = azurerm_virtual_network.this[count.index].name
    # Explicitly capture subnet IDs to create dependencies
    master_subnet = [for s in azurerm_subnet.this : s.id if s.name == "AROMasterSubnet"][count.index]
    worker_subnet = [for s in azurerm_subnet.this : s.id if s.name == "AROWorkerSubnet"][count.index]
    # Capture identity IDs
    cluster_identity = azurerm_user_assigned_identity.aro_cluster_identity["aro-cluster_${count.index}"].id
  }

  provisioner "local-exec" {
    command     = <<EOT

      # Ensure Microsoft.RedHatOpenShift provider is registered (not tracked in TF state)
      echo "Checking Microsoft.RedHatOpenShift provider registration status..."
      provider_state=$(az provider show -n Microsoft.RedHatOpenShift --subscription ${self.triggers.subscription_id} --query "registrationState" -o tsv 2>/dev/null || echo "NotRegistered")
      
      if [[ "$provider_state" != "Registered" ]]; then
        echo "Provider not registered. Registering now (this may take a few minutes)..."
        az provider register -n Microsoft.RedHatOpenShift --subscription ${self.triggers.subscription_id} --wait
        echo "Provider registered successfully"
      else
        echo "Provider already registered, proceeding..."
      fi

      az aro create \
        --resource-group ${self.triggers.resource_group} \
        --subscription ${self.triggers.subscription_id} \
        --name ${self.triggers.cluster_name} \
        --vnet ${self.triggers.vnet_name} \
        --master-subnet ${self.triggers.master_subnet} \
        --worker-subnet ${self.triggers.worker_subnet} \
        --version ${var.aro_clusters[count.index].version} \
        --enable-managed-identity \
        --assign-cluster-identity ${self.triggers.cluster_identity} \
        --assign-platform-workload-identity file-csi-driver ${azurerm_user_assigned_identity.aro_cluster_identity["file-csi-driver_${count.index}"].id} \
        --assign-platform-workload-identity cloud-controller-manager ${azurerm_user_assigned_identity.aro_cluster_identity["cloud-controller-manager_${count.index}"].id} \
        --assign-platform-workload-identity ingress ${azurerm_user_assigned_identity.aro_cluster_identity["ingress_${count.index}"].id} \
        --assign-platform-workload-identity image-registry ${azurerm_user_assigned_identity.aro_cluster_identity["image-registry_${count.index}"].id} \
        --assign-platform-workload-identity machine-api ${azurerm_user_assigned_identity.aro_cluster_identity["machine-api_${count.index}"].id} \
        --assign-platform-workload-identity cloud-network-config ${azurerm_user_assigned_identity.aro_cluster_identity["cloud-network-config_${count.index}"].id} \
        --assign-platform-workload-identity aro-operator ${azurerm_user_assigned_identity.aro_cluster_identity["aro-operator_${count.index}"].id} \
        --assign-platform-workload-identity disk-csi-driver ${azurerm_user_assigned_identity.aro_cluster_identity["disk-csi-driver_${count.index}"].id}
    EOT
    interpreter = ["bash", "-c"]
  }

  provisioner "local-exec" {
    when        = destroy
    command     = <<EOT
      echo "Starting ARO cluster destruction for ${self.triggers.cluster_name}"
      
      # Check if cluster exists before attempting deletion
      if az aro show --resource-group ${self.triggers.resource_group} --name ${self.triggers.cluster_name} \
        --subscription ${self.triggers.subscription_id} >/dev/null 2>&1; then
        echo "Found ARO cluster ${self.triggers.cluster_name}, proceeding with deletion"
        
        az aro delete \
          --resource-group ${self.triggers.resource_group} \
          --subscription ${self.triggers.subscription_id} \
          --name ${self.triggers.cluster_name} \
          --yes
        
        # Wait for deletion to complete to avoid race conditions
        echo "Waiting for ARO cluster deletion to complete..."
        while az aro show --resource-group ${self.triggers.resource_group} --name ${self.triggers.cluster_name} \
          --subscription ${self.triggers.subscription_id} >/dev/null 2>&1; do
          echo "Still deleting ARO cluster..."
          sleep 30
        done
        echo "ARO cluster ${self.triggers.cluster_name} successfully deleted"
      else
        echo "ARO cluster ${self.triggers.cluster_name} not found - may already be deleted"
      fi
    EOT
    interpreter = ["bash", "-c"]
  }

  # Explicit dependencies to ensure proper destroy order
  depends_on = [
    azurerm_user_assigned_identity.aro_cluster_identity,
    azurerm_role_assignment.aro_identity_role,
    azurerm_role_assignment.aro_rp_first_party_id_network_contributor,
    azurerm_subnet.this, # This is crucial!
    azurerm_virtual_network.this
  ]
}

# Use external data source to get cluster details via Azure CLI
data "external" "aro_cluster_details" {
  count = var.aro_clusters == null ? 0 : length(var.aro_clusters)

  program = ["bash", "-c", <<EOT
    cluster_name="${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${count.index}"
    rg_name="${azurerm_resource_group.this.name}"
    subscription_id="${data.azurerm_client_config.current.subscription_id}"
    
    # Get cluster details
    cluster_info=$(az aro show --name "$cluster_name" --resource-group "$rg_name" --subscription "$subscription_id" --output json 2>/dev/null)
    
    if [ $? -eq 0 ] && [ "$cluster_info" != "null" ]; then
      # Extract key information
      cluster_id=$(echo "$cluster_info" | jq -r '.id // empty')
      api_server_url=$(echo "$cluster_info" | jq -r '.apiserverProfile.url // empty')
      console_url=$(echo "$cluster_info" | jq -r '.consoleProfile.url // empty')
      ingress_ip=$(echo "$cluster_info" | jq -r '.ingressProfiles[0].ip // empty')
      domain=$(echo "$cluster_info" | jq -r '.clusterProfile.domain // empty')
      version=$(echo "$cluster_info" | jq -r '.clusterProfile.version // empty')
      
      # Return as JSON
      jq -n \
        --arg cluster_id "$cluster_id" \
        --arg api_server_url "$api_server_url" \
        --arg console_url "$console_url" \
        --arg ingress_ip "$ingress_ip" \
        --arg domain "$domain" \
        --arg version "$version" \
        '{
          cluster_id: $cluster_id,
          api_server_url: $api_server_url,
          console_url: $console_url,
          ingress_ip: $ingress_ip,
          domain: $domain,
          version: $version
        }'
    else
      echo '{"cluster_id":"","api_server_url":"","console_url":"","ingress_ip":"","domain":"","version":""}'
    fi
  EOT
  ]

  depends_on = [null_resource.aro_cluster]
}


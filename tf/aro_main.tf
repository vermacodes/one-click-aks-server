
# Loop to create all required identities for each cluster
locals {
  identity_roles = {
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

  identity_types = keys(local.identity_roles)

  # Build a map of all assignments needed: key is "identitytype_clusteridx", value is a map with identity_type, cluster_idx, and role
  identity_assignments = var.aro_clusters == null ? {} : {
    for pair in setproduct(local.identity_types, range(length(var.aro_clusters))) :
    "${pair[0]}_${pair[1]}" => {
      identity_type = pair[0]
      cluster_idx   = pair[1]
      role          = local.identity_roles[pair[0]]
    }
  }
}

resource "azurerm_user_assigned_identity" "cluster_identities" {
  for_each            = local.identity_assignments
  name                = "${module.naming.user_assigned_identity.name}-${each.value.identity_type}-${each.value.cluster_idx}"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
}


# Single role assignment resource for all identities
resource "azurerm_role_assignment" "identity_roles" {
  for_each             = local.identity_assignments
  principal_id         = azurerm_user_assigned_identity.cluster_identities[each.key].principal_id
  scope                = azurerm_resource_group.this.id
  role_definition_name = each.value.role
  principal_type       = "ServicePrincipal"
}


# Role assignment for ARO RP first-party service principal on vnet
resource "azurerm_role_assignment" "aro_rp_first_party_vnet" {
  principal_id         = var.aro_rp_first_party_service_principal_id
  scope                = azurerm_resource_group.this.id
  role_definition_name = "Network Contributor"
  principal_type       = "ServicePrincipal"
}


# ARO Cluster

# This feature is still in preview. So the APIs are not yet available in Terraform.
# we will complete this as soon as the apis are available.

# Create ARO cluster using null_resource and local-exec
resource "null_resource" "create_aro_clusters" {
  count = var.aro_clusters == null ? 0 : length(var.aro_clusters)

  provisioner "local-exec" {
    command     = <<EOT
      az aro create \
        --resource-group ${azurerm_resource_group.this.name} \
        --subscription ${data.azurerm_client_config.current.subscription_id} \
        --name ${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${count.index} \
        --vnet ${azurerm_virtual_network.this[count.index].name} \
        --master-subnet ${[for s in azurerm_subnet.this : s.id if s.name == "AROMasterSubnet"][count.index]} \
        --worker-subnet ${[for s in azurerm_subnet.this : s.id if s.name == "AROWorkerSubnet"][count.index]} \
        --version ${var.aro_clusters[count.index].version} \
        --enable-managed-identity \
        --assign-cluster-identity ${azurerm_user_assigned_identity.cluster_identities["aro-cluster_${count.index}"].id} \
        --assign-platform-workload-identity file-csi-driver ${azurerm_user_assigned_identity.cluster_identities["file-csi-driver_${count.index}"].id} \
        --assign-platform-workload-identity cloud-controller-manager ${azurerm_user_assigned_identity.cluster_identities["cloud-controller-manager_${count.index}"].id} \
        --assign-platform-workload-identity ingress ${azurerm_user_assigned_identity.cluster_identities["ingress_${count.index}"].id} \
        --assign-platform-workload-identity image-registry ${azurerm_user_assigned_identity.cluster_identities["image-registry_${count.index}"].id} \
        --assign-platform-workload-identity machine-api ${azurerm_user_assigned_identity.cluster_identities["machine-api_${count.index}"].id} \
        --assign-platform-workload-identity cloud-network-config ${azurerm_user_assigned_identity.cluster_identities["cloud-network-config_${count.index}"].id} \
        --assign-platform-workload-identity aro-operator ${azurerm_user_assigned_identity.cluster_identities["aro-operator_${count.index}"].id} \
        --assign-platform-workload-identity disk-csi-driver ${azurerm_user_assigned_identity.cluster_identities["disk-csi-driver_${count.index}"].id}
    EOT
    interpreter = ["bash", "-c"]
  }

  triggers = {
    resource_group = azurerm_resource_group.this.name
    cluster_name   = "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${count.index}"
  }
  provisioner "local-exec" {
    when        = destroy
    command     = <<EOT
    az aro delete \
      --resource-group ${self.triggers.resource_group} \
      --name ${self.triggers.cluster_name} \
      --yes
  EOT
    interpreter = ["bash", "-c"]
  }

  depends_on = [azurerm_user_assigned_identity.cluster_identities, azurerm_subnet.this, azurerm_virtual_network.this]
}

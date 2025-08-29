# Loop to create all required identities for each cluster
locals {
  identity_types = [
    "aro-cluster",
    "cloud-controller-manager",
    "ingress",
    "machine-api",
    "disk-csi-driver",
    "cloud-network-config",
    "image-registry",
    "aro-operator"
  ]
}

resource "azurerm_user_assigned_identity" "cluster_identities" {
  for_each = var.aro_clusters == null ? {} : {
    for pair in setproduct(range(length(var.aro_clusters)), local.identity_types) :
    "${pair[0]}_${pair[1]}" => {
      cluster_idx   = pair[0]
      identity_type = pair[1]
    }
  }
  name                = "${module.naming.user_assigned_identity.name}-${each.value.identity_type}-${each.value.cluster_idx}"
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
}

resource "azurerm_role_assignment" "aro_operator_azure_red_hat_openshift_federated_credential" {
  count                = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id         = azurerm_user_assigned_identity.aro_operator_identity[count.index].principal_id
  scope                = azurerm_resource_group.this.id
  role_definition_name = "Azure Red Hat OpenShift Federated Credential"
}

# Repeat role assignment for each required identity
resource "azurerm_role_assignment" "aro_cluster_identity_azure_red_hat_openshift_federated_credential" {
  count                = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id         = azurerm_user_assigned_identity.aro_cluster_identity[count.index].principal_id
  scope                = azurerm_resource_group.this.id
  role_definition_name = "Azure Red Hat OpenShift Federated Credential"
  principal_type       = "ServicePrincipal"
}

# Role assignment for cloud-controller-manager on master subnet
resource "azurerm_role_assignment" "cloud_controller_manager_master_subnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.cloud_controller_manager_identity[count.index].principal_id
  scope              = azurerm_subnet.master.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4"
  principal_type     = "ServicePrincipal"
}

# Role assignment for cloud-controller-manager on worker subnet
resource "azurerm_role_assignment" "cloud_controller_manager_worker_subnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.cloud_controller_manager_identity[count.index].principal_id
  scope              = azurerm_subnet.worker.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4"
  principal_type     = "ServicePrincipal"
}

# Role assignments for ingress identity
resource "azurerm_role_assignment" "ingress_master_subnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.ingress_identity[count.index].principal_id
  scope              = azurerm_subnet.master.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c"
  principal_type     = "ServicePrincipal"
}

resource "azurerm_role_assignment" "ingress_worker_subnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.ingress_identity[count.index].principal_id
  scope              = azurerm_subnet.worker.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/0336e1d3-7a87-462b-b6db-342b63f7802c"
  principal_type     = "ServicePrincipal"
}

# Role assignments for machine-api identity
resource "azurerm_role_assignment" "machine_api_master_subnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.machine_api_identity[count.index].principal_id
  scope              = azurerm_subnet.master.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/0358943c-7e01-48ba-8889-02cc51d78637"
  principal_type     = "ServicePrincipal"
}

resource "azurerm_role_assignment" "machine_api_worker_subnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.machine_api_identity[count.index].principal_id
  scope              = azurerm_subnet.worker.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/0358943c-7e01-48ba-8889-02cc51d78637"
  principal_type     = "ServicePrincipal"
}

# Role assignment for cloud-network-config identity on vnet
resource "azurerm_role_assignment" "cloud_network_config_vnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.cloud_network_config_identity[count.index].principal_id
  scope              = azurerm_virtual_network.vnet.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/be7a6435-15ae-4171-8f30-4a343eff9e8f"
  principal_type     = "ServicePrincipal"
}

# Role assignment for file-csi-driver identity on vnet
resource "azurerm_role_assignment" "file_csi_driver_vnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.file_csi_driver_identity[count.index].principal_id
  scope              = azurerm_virtual_network.vnet.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/0d7aedc0-15fd-4a67-a412-efad370c947e"
  principal_type     = "ServicePrincipal"
}

# Role assignment for image-registry identity on vnet
resource "azurerm_role_assignment" "image_registry_vnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.image_registry_identity[count.index].principal_id
  scope              = azurerm_virtual_network.vnet.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/8b32b316-c2f5-4ddf-b05b-83dacd2d08b5"
  principal_type     = "ServicePrincipal"
}

# Role assignments for aro-operator identity
resource "azurerm_role_assignment" "aro_operator_master_subnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.aro_operator_identity[count.index].principal_id
  scope              = azurerm_subnet.master.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/4436bae4-7702-4c84-919b-c4069ff25ee2"
  principal_type     = "ServicePrincipal"
}

resource "azurerm_role_assignment" "aro_operator_worker_subnet" {
  count              = var.aro_clusters == null ? 0 : length(var.aro_clusters)
  principal_id       = azurerm_user_assigned_identity.aro_operator_identity[count.index].principal_id
  scope              = azurerm_subnet.worker.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/4436bae4-7702-4c84-919b-c4069ff25ee2"
  principal_type     = "ServicePrincipal"
}

# Role assignment for ARO RP first-party service principal on vnet
resource "azurerm_role_assignment" "aro_rp_first_party_vnet" {
  principal_id       = var.aro_rp_first_party_service_principal_id
  scope              = azurerm_virtual_network.vnet.id
  role_definition_id = "/subscriptions/${data.azurerm_client_config.current.subscription_id}/providers/Microsoft.Authorization/roleDefinitions/4d97b98b-1d4f-4787-a291-c67834d212e7"
  principal_type     = "ServicePrincipal"
}


# ARO Cluster

# This feature is still in preview. So the APIs are not yet available in Terraform.
# we will complete this as soon as the apis are available.

# Output all cluster identity IDs
output "aro_cluster_identity_ids" {
  description = "Map of cluster/identity keys to their Azure resource IDs."
  value       = { for k, v in azurerm_user_assigned_identity.aro_cluster_identity : k => v.id }
}

output "aro_cluster_ids" {
  description = "Map of cluster indices to their Azure resource IDs."
  value = var.aro_clusters == null ? {} : {
    for i, details in data.external.aro_cluster_details :
    "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}" => details.result.cluster_id
  }
}

output "aro_cluster_api_server_urls" {
  description = "Map of cluster names to their API server URLs."
  value = var.aro_clusters == null ? {} : {
    for i, details in data.external.aro_cluster_details :
    "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}" => details.result.api_server_url
  }
}

output "aro_cluster_console_urls" {
  description = "Map of cluster names to their console URLs."
  value = var.aro_clusters == null ? {} : {
    for i, details in data.external.aro_cluster_details :
    "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}" => details.result.console_url
  }
}

output "aro_cluster_domains" {
  description = "Map of cluster names to their domains."
  value = var.aro_clusters == null ? {} : {
    for i, details in data.external.aro_cluster_details :
    "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}" => details.result.domain
  }
}

output "aro_cluster_versions" {
  description = "Map of cluster names to their OpenShift versions."
  value = var.aro_clusters == null ? {} : {
    for i, details in data.external.aro_cluster_details :
    "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}" => details.result.version
  }
}

output "aro_ingress_ips" {
  description = "Map of cluster names to their ingress IPs."
  value = var.aro_clusters == null ? {} : {
    for i, details in data.external.aro_cluster_details :
    "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}" => details.result.ingress_ip
  }
}

output "aro_cluster_details_external" {
  description = "Map of cluster names to their detailed information."
  value = var.aro_clusters == null ? {} : {
    for i, details in data.external.aro_cluster_details :
    "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}" => {
      cluster_id     = details.result.cluster_id
      api_server_url = details.result.api_server_url
      console_url    = details.result.console_url
      ingress_ip     = details.result.ingress_ip
      domain         = details.result.domain
      version        = details.result.version
    }
  }
}

output "aro_clusters_summary" {
  description = "Map of cluster names to their summary information."
  value = var.aro_clusters == null ? {} : {
    for i, details in data.external.aro_cluster_details :
    "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}" => {
      index          = i
      name           = "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}"
      id             = details.result.cluster_id
      resource_group = azurerm_resource_group.this.name
      api_server_url = details.result.api_server_url
      console_url    = details.result.console_url
      domain         = details.result.domain
      version        = details.result.version
      ingress_ip     = details.result.ingress_ip
    }
  }
}

output "aro_cluster_names" {
  description = "Map of indices to the created ARO cluster names."
  value = var.aro_clusters == null ? {} : {
    for i in range(length(var.aro_clusters)) :
    i => "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}"
  }
}

output "aro_cluster_credentials_commands" {
  description = "Map of cluster names to commands for getting ARO cluster credentials."
  value = var.aro_clusters == null ? {} : {
    for i in range(length(var.aro_clusters)) :
    "${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i}" => "az aro list-credentials --name ${replace(module.naming.kubernetes_cluster.name, "aks", "aro")}-${i} --resource-group ${azurerm_resource_group.this.name} --subscription ${data.azurerm_client_config.current.subscription_id}"
  }
  sensitive = false
}

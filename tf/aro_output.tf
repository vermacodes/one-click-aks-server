# Output all cluster identity IDs
output "aro_cluster_identity_ids" {
  description = "Map of cluster/identity keys to their Azure resource IDs."
  value       = { for k, v in azurerm_user_assigned_identity.cluster_identities : k => v.id }
}

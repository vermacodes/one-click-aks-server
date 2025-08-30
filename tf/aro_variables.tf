variable "aro_clusters" {
  description = "AKS Cluster Object"
  type = list(object({
    version = string
  }))
}

variable "aro_rp_first_party_service_principal_id" {
  description = "The ID of the first-party service principal for the ARO RP."
  type        = string
}

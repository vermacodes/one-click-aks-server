variable "resource_group" {
  description = "Resource Group"
  type = object({
    location = string
  })
  default = {
    location = "eastus"
  }
}

variable "created_by" {
  description = "Created By"
  type        = string
  default     = "ACTLabs"
}

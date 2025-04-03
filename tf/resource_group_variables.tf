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
variable "security_control" {
  description = "Security Control"
  type        = string
  default     = "Ignore"
}
variable "cost_control" {
  description = "Cost Control"
  type        = string
  default     = "Ignore"
}

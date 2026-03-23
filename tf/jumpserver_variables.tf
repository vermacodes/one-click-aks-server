variable "jumpservers" {
  description = "Jump Server"
  type = list(object({
    admin_username = string
    admin_password = string
    vm_size        = string
  }))
  default = []
}

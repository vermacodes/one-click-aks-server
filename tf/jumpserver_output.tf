output "jump_server_name" {
  value       = var.jumpservers == null ? "" : length(var.jumpservers) == 0 ? "" : azurerm_virtual_machine.this[0].name
  description = "Jump Server Name"
}

output "jump_server_resource_name" {
  value       = var.jumpservers == null ? "" : length(var.jumpservers) == 0 ? "" : azurerm_virtual_machine.this[0].id
  description = "Jump Server Resource ID"
}

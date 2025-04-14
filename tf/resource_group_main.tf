resource "azurerm_resource_group" "this" {
  name     = module.naming.resource_group.name
  location = var.resource_group.location
  tags = {
    CreatedBy       = var.created_by
    SecurityControl = var.security_control
    CostControl     = var.cost_control
  }
}

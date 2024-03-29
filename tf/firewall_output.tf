output "firewall_private_ip" {
  value = length(var.firewalls) == 0 ? "" : azurerm_firewall.this[0].ip_configuration[0].private_ip_address
}

output "firewall_name" {
  value = length(var.firewalls) == 0 ? "" : azurerm_firewall.this[0].name
}

output "firewall_id" {
  value = length(var.firewalls) == 0 ? "" : azurerm_firewall.this[0].id
}

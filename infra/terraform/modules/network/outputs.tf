output "names" {
  description = "Імена мереж за призначенням (public, app, private, egress)."
  value       = { for k, n in docker_network.this : k => n.name }
}

output "ids" {
  description = "Ідентифікатори мереж."
  value       = { for k, n in docker_network.this : k => n.id }
}

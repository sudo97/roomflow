output "url" {
  description = "Адреса застосунку через балансувальник."
  value       = "http://localhost:${var.port}"
}

output "active_color" {
  description = "Середовище, що зараз обслуговує трафік."
  value       = var.active_color
}

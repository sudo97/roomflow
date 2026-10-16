output "url" {
  description = "Адреса застосунку."
  value       = module.lb.url
}

output "active_color" {
  description = "Активне середовище."
  value       = module.lb.active_color
}

output "networks" {
  description = "Мережі середовища."
  value       = module.network.names
}

output "prometheus_url" {
  description = "Prometheus."
  value       = module.monitoring.prometheus_url
}

output "db_backups_volume" {
  description = "Том резервних копій БД."
  value       = module.database.backups_volume
}

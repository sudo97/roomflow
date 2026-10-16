output "prometheus_url" {
  description = "Веб-інтерфейс Prometheus."
  value       = "http://localhost:9090"
}

output "alertmanager_url" {
  description = "Веб-інтерфейс Alertmanager."
  value       = "http://localhost:9093"
}

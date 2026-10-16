output "replicas" {
  description = "Імена контейнерів-реплік."
  value       = docker_container.replica[*].name
}

output "service_name" {
  description = "DNS-ім'я групи реплік у мережі застосунку."
  value       = "app-${var.color}"
}

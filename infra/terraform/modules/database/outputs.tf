output "host" {
  description = "DNS-ім'я бази даних у приватній мережі."
  value       = "db"
}

output "password" {
  description = "Пароль адміністратора БД (зберігається у стані; у ЛР8 винесено у Vault)."
  value       = random_password.db.result
  sensitive   = true
}

output "backups_volume" {
  description = "Том із резервними копіями."
  value       = docker_volume.backups.name
}

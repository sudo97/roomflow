output "color" {
  description = "Колір середовища (workspace)."
  value       = terraform.workspace
}

output "replicas" {
  description = "Репліки групи."
  value       = module.app.replicas
}

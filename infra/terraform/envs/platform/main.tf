# Кореневий модуль спільної платформи середовища: мережі, балансувальник (перемикач blue/green),
# база даних і моніторинг. Групи реплік застосунку описані окремо (envs/app) і розгортаються
# у workspace blue та green.

terraform {
  required_version = ">= 1.9.0"
  # Віддалений стан у PostgreSQL з блокуванням (advisory locks); рядок підключення — PG_CONN_STR.
  backend "pg" {}

  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

provider "docker" {}

locals {
  prefix = "rf-${var.environment}"
  # Єдина схема тегування: середовище, власник, центр витрат, призначення.
  labels = {
    project     = "roomflow"
    environment = var.environment
    owner       = var.owner
    cost_center = var.cost_center
    managed_by  = "terraform"
  }
}

module "network" {
  source = "../../modules/network"
  prefix = local.prefix
  labels = local.labels
}

module "lb" {
  source       = "../../modules/lb"
  prefix       = local.prefix
  active_color = var.active_color
  port         = var.lb_port
  replicas     = var.app_replicas
  networks     = [module.network.names.public, module.network.names.app]
  config_dir   = "${path.root}/.generated/lb"
  labels       = local.labels
}

module "database" {
  source          = "../../modules/database"
  prefix          = local.prefix
  network         = module.network.names.private
  retention_count = var.backup_retention
  labels          = local.labels
}

module "monitoring" {
  source     = "../../modules/monitoring"
  prefix     = local.prefix
  networks   = [module.network.names.app, module.network.names.public]
  config_dir = "${path.root}/.generated/monitoring"
  labels     = local.labels
}

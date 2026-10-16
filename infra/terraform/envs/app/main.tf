# Група реплік застосунку. Один і той самий код розгортається у двох workspace — blue і green;
# workspace визначає колір, імена ресурсів і окремий файл стану (сині-зелене розгортання).

terraform {
  required_version = ">= 1.9.0"
  backend "pg" {}

  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

provider "docker" {}

# Колір визначає workspace; у workspace default (напр. terraform validate) — blue, а check нижче попереджає.
locals {
  color  = contains(["blue", "green"], terraform.workspace) ? terraform.workspace : "blue"
  prefix = "rf-${var.environment}"
  labels = {
    project     = "roomflow"
    environment = var.environment
    owner       = var.owner
    cost_center = var.cost_center
    managed_by  = "terraform"
  }
}

# Мережі створює кореневий модуль platform; тут вони лише зчитуються.
data "docker_network" "app" {
  name = "${local.prefix}-app"
}

data "docker_network" "private" {
  name = "${local.prefix}-private"
}

module "app" {
  source       = "../../modules/app"
  name         = "${local.prefix}-${local.color}"
  color        = local.color
  image        = var.image
  desired_size = var.replicas
  networks     = [data.docker_network.app.name, data.docker_network.private.name]
  labels       = local.labels
}

check "workspace_is_a_color" {
  assert {
    condition     = contains(["blue", "green"], terraform.workspace)
    error_message = "Використовуйте workspace blue або green."
  }
}

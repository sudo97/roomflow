# Модуль групи екземплярів застосунку (аналог Auto Scaling Group + launch template).
# Кількість реплік задається desired_size у межах [min_size, max_size]; усі репліки отримують
# спільне DNS-ім'я app-<color>, через яке балансувальник і Prometheus знаходять їх.

locals {
  labels = merge(var.labels, { component = "app", color = var.color })
  env    = merge({ COLOR = var.color }, var.env)
}

resource "docker_image" "app" {
  name         = var.image
  keep_locally = true
}

resource "docker_container" "replica" {
  count = var.desired_size

  name    = "${var.name}-${count.index}"
  image   = docker_image.app.image_id
  restart = "unless-stopped"
  memory  = var.memory_mb
  env     = [for k, v in local.env : "${k}=${v}"]

  # Docker за замовчуванням встановлює swap = 2 × memory; без явного значення провайдер
  # бачить різницю (256 -> null) після кожного apply, і перевірка дрейфу ніколи не сходиться
  memory_swap = var.memory_mb * 2

  dynamic "networks_advanced" {
    for_each = var.networks
    content {
      name    = networks_advanced.value
      aliases = ["app-${var.color}"]
    }
  }

  healthcheck {
    test     = ["CMD", "/roomflow", "-healthcheck"]
    interval = "5s"
    timeout  = "2s"
    retries  = 3
  }

  dynamic "labels" {
    for_each = local.labels
    content {
      label = labels.key
      value = labels.value
    }
  }
}

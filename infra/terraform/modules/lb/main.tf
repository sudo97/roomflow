# Балансувальник навантаження (nginx) — точка входу і перемикач blue/green.
# Активне середовище визначається змінною active_color: зміна генерує нову конфігурацію,
# після чого nginx перечитує її (nginx -s reload) без розриву з'єднань — аналог зміни DNS-запису.

locals {
  labels = merge(var.labels, { component = "lb" })
}

resource "local_file" "config" {
  filename        = "${var.config_dir}/nginx.conf"
  file_permission = "0644"
  content         = templatefile("${path.module}/nginx.conf.tftpl", { color = var.active_color })
}

resource "docker_image" "nginx" {
  name         = var.image
  keep_locally = true
}

resource "docker_container" "lb" {
  name    = "${var.prefix}-lb"
  image   = docker_image.nginx.image_id
  restart = "unless-stopped"
  memory  = 64
  command = ["nginx", "-g", "daemon off;", "-c", "/etc/nginx/rf/nginx.conf"]

  ports {
    internal = 80
    external = var.port
  }

  mounts {
    type      = "bind"
    source    = abspath(var.config_dir)
    target    = "/etc/nginx/rf"
    read_only = true
  }

  dynamic "networks_advanced" {
    for_each = var.networks
    content {
      name = networks_advanced.value
    }
  }

  dynamic "labels" {
    for_each = local.labels
    content {
      label = labels.key
      value = labels.value
    }
  }

  depends_on = [local_file.config]
}

# Застосування нової конфігурації без перезапуску контейнера.
resource "terraform_data" "reload" {
  triggers_replace = [local_file.config.content]

  provisioner "local-exec" {
    command = "docker exec ${docker_container.lb.name} nginx -s reload"
  }
}

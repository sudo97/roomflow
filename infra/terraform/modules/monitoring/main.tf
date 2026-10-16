# Модуль моніторингу інфраструктури: Prometheus (збір метрик реплік через DNS-виявлення),
# правила тривог, Alertmanager і приймач сповіщень (webhook, аналог інтеграції з системою інцидентів).

locals {
  labels = merge(var.labels, { component = "monitoring" })
  dir    = abspath(var.config_dir)
  files = {
    "prometheus.yml"   = templatefile("${path.module}/prometheus.yml.tftpl", { colors = var.colors })
    "alerts.yml"       = templatefile("${path.module}/alerts.yml.tftpl", { latency_p95_seconds = var.latency_p95_seconds, error_ratio = var.error_ratio })
    "alertmanager.yml" = file("${path.module}/alertmanager.yml")
  }
  containers = {
    prometheus = {
      image   = "prom/prometheus:v2.54.1"
      port    = 9090
      command = ["--config.file=/etc/rf/prometheus.yml", "--storage.tsdb.retention.time=2h", "--web.enable-lifecycle"]
    }
    alertmanager = {
      image   = "prom/alertmanager:v0.27.0"
      port    = 9093
      command = ["--config.file=/etc/rf/alertmanager.yml"]
    }
    alert-sink = {
      image   = "python:3.12-alpine"
      port    = 0
      command = ["python", "-u", "-c", "import http.server as h\nclass S(h.BaseHTTPRequestHandler):\n def do_POST(self):\n  n=int(self.headers['Content-Length']); print('ALERT', self.rfile.read(n).decode(), flush=True); self.send_response(200); self.end_headers()\nh.HTTPServer(('',8080),S).serve_forever()"]
    }
  }
}

resource "local_file" "config" {
  for_each        = local.files
  filename        = "${var.config_dir}/${each.key}"
  file_permission = "0644"
  content         = each.value
}

resource "docker_image" "this" {
  for_each     = local.containers
  name         = each.value.image
  keep_locally = true
}

resource "docker_container" "this" {
  for_each = local.containers

  name    = "${var.prefix}-${each.key}"
  image   = docker_image.this[each.key].image_id
  restart = "unless-stopped"
  memory  = 256
  command = each.value.command

  dynamic "ports" {
    for_each = each.value.port > 0 ? [each.value.port] : []
    content {
      internal = ports.value
      external = ports.value
    }
  }

  mounts {
    type      = "bind"
    source    = local.dir
    target    = "/etc/rf"
    read_only = true
  }

  dynamic "networks_advanced" {
    for_each = var.networks
    content {
      name    = networks_advanced.value
      aliases = [each.key]
    }
  }

  dynamic "labels" {
    for_each = merge(local.labels, { service = each.key })
    content {
      label = labels.key
      value = labels.value
    }
  }

  depends_on = [local_file.config]
}

# Prometheus перечитує правила після зміни конфігурації (POST /-/reload).
resource "terraform_data" "reload" {
  triggers_replace = [for f in local_file.config : f.content]

  provisioner "local-exec" {
    command = "for i in $(seq 1 20); do curl -sf -X POST http://localhost:9090/-/reload && exit 0; sleep 1; done; exit 1"
  }

  depends_on = [docker_container.this]
}

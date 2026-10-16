# Модуль бази даних: PostgreSQL лише у приватній мережі (аналог RDS у приватних підмережах),
# без опублікованих портів; доступ мають тільки ресурси, підключені до приватної мережі (застосунок).
# Резервне копіювання: супутній контейнер виконує pg_dump з інтервалом і зберігає retention_count копій.

locals {
  labels = merge(var.labels, { component = "database" })
}

resource "random_password" "db" {
  length  = 24
  special = false
}

resource "docker_image" "postgres" {
  name         = var.image
  keep_locally = true
}

resource "docker_volume" "data" {
  name = "${var.prefix}-db-data"
  dynamic "labels" {
    for_each = local.labels
    content {
      label = labels.key
      value = labels.value
    }
  }
}

resource "docker_volume" "backups" {
  name = "${var.prefix}-db-backups"
  dynamic "labels" {
    for_each = local.labels
    content {
      label = labels.key
      value = labels.value
    }
  }
}

resource "docker_container" "db" {
  name    = "${var.prefix}-db"
  image   = docker_image.postgres.image_id
  restart = "unless-stopped"
  memory  = var.memory_mb
  env     = ["POSTGRES_PASSWORD=${random_password.db.result}", "POSTGRES_DB=${var.database}"]

  mounts {
    type   = "volume"
    source = docker_volume.data.name
    target = "/var/lib/postgresql/data"
  }

  networks_advanced {
    name    = var.network
    aliases = ["db"]
  }

  healthcheck {
    test     = ["CMD", "pg_isready", "-U", "postgres"]
    interval = "5s"
    timeout  = "3s"
    retries  = 5
  }

  dynamic "labels" {
    for_each = local.labels
    content {
      label = labels.key
      value = labels.value
    }
  }
}

resource "docker_container" "backup" {
  name    = "${var.prefix}-db-backup"
  image   = docker_image.postgres.image_id
  restart = "unless-stopped"
  memory  = 64
  env     = ["PGPASSWORD=${random_password.db.result}", "RETENTION=${var.retention_count}", "INTERVAL=${var.backup_interval_seconds}", "DB=${var.database}"]
  command = ["sh", "-c", "sleep 5; while true; do pg_dump -h db -U postgres \"$DB\" > /backups/$DB-$(date +%Y%m%d%H%M%S).sql && ls -1t /backups/*.sql | tail -n +$((RETENTION + 1)) | xargs -r rm; sleep \"$INTERVAL\"; done"]

  mounts {
    type   = "volume"
    source = docker_volume.backups.name
    target = "/backups"
  }

  networks_advanced {
    name = var.network
  }

  dynamic "labels" {
    for_each = merge(local.labels, { component = "database-backup" })
    content {
      label = labels.key
      value = labels.value
    }
  }

  depends_on = [docker_container.db]
}

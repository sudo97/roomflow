# Мережевий модуль: аналог VPC з публічною, прикладною та приватною підмережами.
# public  – вхідний трафік (лише балансувальник має опубліковані порти);
# app     – зв'язок балансувальника, моніторингу та екземплярів застосунку;
# private – внутрішня мережа без доступу до Інтернету (база даних);
# egress  – вихідний доступ без вхідних з'єднань (аналог NAT gateway для приватних ресурсів).

locals {
  networks = {
    public  = { internal = false, subnet = var.cidrs.public }
    app     = { internal = false, subnet = var.cidrs.app }
    private = { internal = true, subnet = var.cidrs.private }
    egress  = { internal = false, subnet = var.cidrs.egress }
  }
}

resource "docker_network" "this" {
  for_each = local.networks

  name     = "${var.prefix}-${each.key}"
  driver   = "bridge"
  internal = each.value.internal

  ipam_config {
    subnet = each.value.subnet
  }

  dynamic "labels" {
    for_each = merge(var.labels, { tier = each.key })
    content {
      label = labels.key
      value = labels.value
    }
  }
}

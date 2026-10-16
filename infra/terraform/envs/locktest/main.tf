# Допоміжна конфігурація для демонстрації блокування віддаленого стану:
# поки один процес виконує apply (30 с), другий не може змінювати той самий стан.
terraform {
  required_version = ">= 1.9.0"
  backend "pg" {}
  required_providers {
    time = {
      source  = "hashicorp/time"
      version = "~> 0.12"
    }
  }
}

variable "run_id" {
  description = "Унікальний ідентифікатор запуску (змушує створити ресурс заново)."
  type        = string
}

resource "time_sleep" "hold_lock" {
  create_duration = "30s"
  triggers        = { run = var.run_id }
}

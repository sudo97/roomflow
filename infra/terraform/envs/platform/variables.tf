variable "environment" {
  description = "Назва середовища (ci, staging, production)."
  type        = string
  default     = "ci"
}

variable "owner" {
  description = "Команда-власник ресурсів."
  type        = string
  default     = "platform-team"
}

variable "cost_center" {
  description = "Центр витрат для обліку вартості."
  type        = string
  default     = "roomflow-edu"
}

variable "active_color" {
  description = "Середовище, що обслуговує користувачів: blue або green."
  type        = string
  default     = "blue"
}

variable "lb_port" {
  description = "Порт балансувальника на хості."
  type        = number
  default     = 8088
}

variable "backup_retention" {
  description = "Кількість резервних копій БД."
  type        = number
  default     = 3
}

variable "prefix" {
  description = "Префікс імен ресурсів середовища."
  type        = string
}

variable "image" {
  description = "Рушій і версія бази даних."
  type        = string
  default     = "postgres:16-alpine"
}

variable "database" {
  description = "Ім'я бази даних."
  type        = string
  default     = "roomflow"
}

variable "network" {
  description = "Приватна мережа, у якій розміщується база даних."
  type        = string
}

variable "memory_mb" {
  description = "Ліміт пам'яті (аналог класу інстанса БД), МБ."
  type        = number
  default     = 256
}

variable "retention_count" {
  description = "Кількість резервних копій, що зберігаються."
  type        = number
  default     = 3
}

variable "backup_interval_seconds" {
  description = "Інтервал резервного копіювання, с."
  type        = number
  default     = 30
}

variable "labels" {
  description = "Обов'язкові мітки (теги)."
  type        = map(string)
}

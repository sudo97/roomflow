variable "name" {
  description = "Базове ім'я реплік, наприклад rf-ci-blue."
  type        = string
}

variable "color" {
  description = "Колір середовища blue/green."
  type        = string
  validation {
    condition     = contains(["blue", "green"], var.color)
    error_message = "color має бути blue або green."
  }
}

variable "image" {
  description = "Образ застосунку з конкретною версією (тег latest заборонено політикою)."
  type        = string
}

variable "min_size" {
  description = "Мінімальна кількість реплік."
  type        = number
  default     = 1
}

variable "max_size" {
  description = "Максимальна кількість реплік."
  type        = number
  default     = 4
}

variable "desired_size" {
  description = "Бажана кількість реплік."
  type        = number
  default     = 2
  validation {
    condition     = var.desired_size >= var.min_size && var.desired_size <= var.max_size
    error_message = "desired_size має бути в межах [min_size, max_size]."
  }
}

variable "networks" {
  description = "Мережі, до яких підключаються репліки."
  type        = list(string)
}

variable "env" {
  description = "Додаткові змінні середовища застосунку."
  type        = map(string)
  default     = {}
}

variable "memory_mb" {
  description = "Ліміт пам'яті репліки, МБ (аналог типу інстанса)."
  type        = number
  default     = 128
}

variable "labels" {
  description = "Обов'язкові мітки (теги)."
  type        = map(string)
}

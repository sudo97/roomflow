variable "prefix" {
  description = "Префікс імен ресурсів середовища."
  type        = string
}

variable "image" {
  description = "Образ nginx з конкретною версією."
  type        = string
  default     = "nginx:1.27-alpine"
}

variable "active_color" {
  description = "Середовище, що обслуговує трафік: blue або green."
  type        = string
  validation {
    condition     = contains(["blue", "green"], var.active_color)
    error_message = "active_color має бути blue або green."
  }
}

variable "port" {
  description = "Порт на хості, через який доступний застосунок."
  type        = number
  default     = 8088
}

variable "networks" {
  description = "Мережі балансувальника (публічна і прикладна)."
  type        = list(string)
}

variable "config_dir" {
  description = "Каталог для згенерованої конфігурації nginx."
  type        = string
}

variable "labels" {
  description = "Обов'язкові мітки (теги)."
  type        = map(string)
}

variable "replicas" {
  description = "Кількість реплік у кожній групі (для прив'язки орендарів до реплік)."
  type        = number
  default     = 2
  validation {
    condition     = var.replicas >= 1
    error_message = "replicas має бути не менше 1."
  }
}

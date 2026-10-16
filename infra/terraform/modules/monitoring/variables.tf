variable "prefix" {
  description = "Префікс імен ресурсів середовища."
  type        = string
}

variable "networks" {
  description = "Мережі моніторингу (прикладна для збору метрик і публічна)."
  type        = list(string)
}

variable "colors" {
  description = "Групи реплік, що підлягають моніторингу."
  type        = list(string)
  default     = ["blue", "green"]
}

variable "latency_p95_seconds" {
  description = "Поріг тривоги для p95 затримки, с."
  type        = number
  default     = 0.5
}

variable "error_ratio" {
  description = "Поріг тривоги для частки відповідей 5xx."
  type        = number
  default     = 0.05
}

variable "config_dir" {
  description = "Каталог для згенерованих конфігурацій."
  type        = string
}

variable "labels" {
  description = "Обов'язкові мітки (теги)."
  type        = map(string)
}

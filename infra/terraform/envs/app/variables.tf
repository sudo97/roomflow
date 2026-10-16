variable "environment" {
  description = "Назва середовища."
  type        = string
  default     = "ci"
}

variable "owner" {
  description = "Команда-власник."
  type        = string
  default     = "platform-team"
}

variable "cost_center" {
  description = "Центр витрат."
  type        = string
  default     = "roomflow-edu"
}

variable "image" {
  description = "Версія образу застосунку для цього середовища."
  type        = string
}

variable "replicas" {
  description = "Кількість реплік."
  type        = number
  default     = 2
}

variable "prefix" {
  description = "Префікс імен ресурсів середовища, наприклад rf-ci."
  type        = string
}

variable "cidrs" {
  description = "Адресні простори підмереж."
  type = object({
    public  = string
    app     = string
    private = string
    egress  = string
  })
  default = {
    public  = "10.60.1.0/24"
    app     = "10.60.2.0/24"
    private = "10.60.3.0/24"
    egress  = "10.60.4.0/24"
  }
}

variable "labels" {
  description = "Обов'язкові мітки (теги) для обліку витрат і власності."
  type        = map(string)
}

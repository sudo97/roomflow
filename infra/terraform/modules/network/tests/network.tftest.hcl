# Інтеграційний тест модуля: реально створює мережі в Docker, перевіряє ізоляцію і знищує їх.
variables {
  prefix = "rf-tftest"
  cidrs = {
    public  = "10.70.1.0/24"
    app     = "10.70.2.0/24"
    private = "10.70.3.0/24"
    egress  = "10.70.4.0/24"
  }
  labels = { project = "roomflow", environment = "test", owner = "tftest", cost_center = "edu" }
}

run "creates_isolated_private_network" {
  command = apply

  assert {
    condition     = docker_network.this["private"].internal
    error_message = "Приватна мережа має бути internal (без виходу в Інтернет)."
  }

  assert {
    condition     = !docker_network.this["egress"].internal && !docker_network.this["public"].internal
    error_message = "Публічна та egress мережі мають мати вихід назовні."
  }

  assert {
    condition     = length(output.names) == 4
    error_message = "Очікуються чотири мережі."
  }
}

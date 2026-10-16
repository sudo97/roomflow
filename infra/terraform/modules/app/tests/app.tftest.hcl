# Модульні тести модуля app на mock-провайдері: перевіряють логіку без створення контейнерів.
mock_provider "docker" {}

variables {
  name     = "rf-test-blue"
  color    = "blue"
  image    = "localhost:5000/roomflow:v1"
  networks = ["rf-test-app", "rf-test-private"]
  labels   = { project = "roomflow", environment = "test", owner = "tftest", cost_center = "edu" }
}

run "default_group_has_two_replicas" {
  command = plan

  assert {
    condition     = length(docker_container.replica) == 2
    error_message = "За замовчуванням очікуються дві репліки."
  }

  assert {
    condition     = output.service_name == "app-blue"
    error_message = "Репліки мають бути доступні під іменем app-blue."
  }
}

run "scales_up_to_max" {
  command = plan
  variables {
    desired_size = 4
  }

  assert {
    condition     = length(docker_container.replica) == 4
    error_message = "Група має масштабуватися до max_size."
  }
}

run "rejects_desired_above_max" {
  command = plan
  variables {
    desired_size = 5
  }
  expect_failures = [var.desired_size]
}

run "rejects_unknown_color" {
  command = plan
  variables {
    color = "red"
  }
  expect_failures = [var.color]
}

run "mandatory_labels_and_color_env" {
  command = plan

  assert {
    condition     = alltrue([for k in ["project", "environment", "owner", "cost_center", "component", "color"] : contains([for l in docker_container.replica[0].labels : l.label], k)])
    error_message = "Кожна репліка має містити обов'язкові мітки."
  }

  assert {
    condition     = contains(docker_container.replica[0].env, "COLOR=blue")
    error_message = "Репліка має знати свій колір (COLOR)."
  }
}

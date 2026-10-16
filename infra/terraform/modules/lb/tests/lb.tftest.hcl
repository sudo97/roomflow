mock_provider "docker" {}
mock_provider "local" {}

variables {
  prefix       = "rf-test"
  active_color = "green"
  networks     = ["rf-test-public", "rf-test-app"]
  config_dir   = "/tmp/rf-test-lb"
  labels       = { project = "roomflow", environment = "test", owner = "tftest", cost_center = "edu" }
}

run "routes_to_active_color" {
  command = plan

  assert {
    condition     = strcontains(local_file.config.content, "http://app-green:8080")
    error_message = "Трафік має йти на активне середовище green."
  }

  assert {
    condition     = docker_container.lb.ports[0].external == 8088
    error_message = "Балансувальник публікує порт 8088."
  }
}

run "rejects_unknown_color" {
  command = plan
  variables {
    active_color = "purple"
  }
  expect_failures = [var.active_color]
}

service {
  id      = "my-service1"
  name    = "my-service"
  tags    = ["v1"]
  address = "127.0.0.1"
  port    = 8000

  enable_tag_override = true

  weights {
    passing = 10
    warning = 1
  }
}

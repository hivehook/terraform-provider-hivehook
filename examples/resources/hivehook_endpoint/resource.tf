resource "hivehook_endpoint" "acme_primary" {
  application_id = hivehook_application.acme.id
  url            = "https://hooks.acme.example.com/events"
  rate_limit_rps = 100
  timeout_ms     = 5000
  retry_policy = jsonencode({
    maxAttempts   = 5
    initialDelay  = "1s"
    maxDelay      = "30s"
    backoffFactor = 2
  })
  filter_config = jsonencode({
    eventTypes = ["order.created", "order.shipped"]
  })
}

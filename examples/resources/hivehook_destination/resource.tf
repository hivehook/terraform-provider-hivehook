resource "hivehook_destination" "billing_api" {
  name          = "Billing API"
  url           = "https://billing.internal.example.com/webhooks"
  timeout_ms    = 5000
  rate_limit_rps = 50
  retry_policy = jsonencode({
    maxAttempts   = 5
    initialDelay  = "1s"
    maxDelay      = "30s"
    backoffFactor = 2
  })
  headers = jsonencode({
    "X-Internal-Source" = "hivehook"
  })
}

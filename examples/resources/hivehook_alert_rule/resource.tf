resource "hivehook_alert_rule" "dlq_growth" {
  name           = "DLQ growth"
  condition_type = "dlq_size_exceeded"
  threshold      = 100
  webhook_url    = "https://alerts.example.com/hivehook"
  cooldown       = "1h"
  enabled        = true
}

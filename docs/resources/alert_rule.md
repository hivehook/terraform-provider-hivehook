---
page_title: "hivehook_alert_rule Resource"
description: |-
  Manages a Hivehook alert rule.
---

# hivehook_alert_rule (Resource)

Manages a Hivehook alert rule. Alert rules monitor system conditions (e.g. DLQ
size) and emit webhook notifications when a threshold is exceeded.

## Example Usage

```hcl
resource "hivehook_alert_rule" "dlq_growth" {
  name           = "DLQ growth"
  condition_type = "dlq_size_exceeded"
  threshold      = 100
  webhook_url    = "https://alerts.example.com/hivehook"
  cooldown       = "1h"
}
```

## Schema

### Required

- `name` (String) Display name.
- `condition_type` (String) Alert condition (e.g. `dlq_size_exceeded`).
- `threshold` (Number) Numeric threshold for the condition.
- `webhook_url` (String) Webhook URL to notify on trigger.

### Optional

- `cooldown` (String) Duration between repeat alerts (e.g. `1h`).
- `enabled` (Boolean) Whether the rule is active (default `true`).

### Read-Only

- `id` (String) Alert rule UUID.

## Import

```bash
terraform import hivehook_alert_rule.dlq_growth <alert-rule-uuid>
```

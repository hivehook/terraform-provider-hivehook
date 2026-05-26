---
page_title: "hivehook_destination Resource"
description: |-
  Manages a Hivehook delivery destination.
---

# hivehook_destination (Resource)

Manages a Hivehook delivery destination: the HTTP endpoint that receives
forwarded webhooks.

## Example Usage

```hcl
resource "hivehook_destination" "billing_api" {
  name           = "Billing API"
  url            = "https://billing.internal.example.com/webhooks"
  timeout_ms     = 5000
  rate_limit_rps = 50
}
```

## Schema

### Required

- `name` (String) Display name.

### Optional

- `url` (String) Destination URL.
- `status` (String) Destination status.
- `timeout_ms` (Number) HTTP request timeout in milliseconds.
- `rate_limit_rps` (Number) Rate limit in requests per second.
- `retry_policy` (String, JSON) JSON-encoded retry policy.
- `headers` (String, JSON) JSON-encoded custom headers.
- `auth_type` (String) Authentication type (default `NONE`).
- `delivery_mode` (String) Delivery mode (`PUSH` or `PULL`).
- `ordered` (Boolean) Whether deliveries are ordered.

### Read-Only

- `id` (String) Destination UUID.
- `signing_secret` (String, Sensitive) HMAC signing secret.

## Import

```bash
terraform import hivehook_destination.billing_api <destination-uuid>
```

---
page_title: "hivehook_endpoint Resource"
description: |-
  Manages a Hivehook outbound endpoint.
---

# hivehook_endpoint (Resource)

Manages a Hivehook outbound endpoint: the URL that a Hivehook application
delivers signed webhooks to.

## Example Usage

```hcl
resource "hivehook_endpoint" "acme_primary" {
  application_id = hivehook_application.acme.id
  url            = "https://hooks.acme.example.com/events"
  rate_limit_rps = 100
  timeout_ms     = 5000
}
```

## Schema

### Required

- `application_id` (String) Owning application UUID.

### Optional

- `url` (String) Endpoint URL.
- `status` (String) Endpoint status.
- `rate_limit_rps` (Number) Rate limit in requests per second.
- `timeout_ms` (Number) HTTP timeout in milliseconds.
- `retry_policy` (String, JSON) JSON-encoded retry policy.
- `headers` (String, JSON) JSON-encoded custom headers.
- `filter_config` (String, JSON) JSON-encoded filter configuration.
- `transformation_id` (String) Optional transformation UUID.
- `auth_type` (String) Authentication type (default `NONE`).
- `delivery_mode` (String) Delivery mode (`PUSH` or `PULL`).
- `ordered` (Boolean) Whether deliveries are ordered.

### Read-Only

- `id` (String) Endpoint UUID.
- `signing_secret` (String, Sensitive) HMAC signing secret.

## Import

```bash
terraform import hivehook_endpoint.acme_primary <endpoint-uuid>
```

---
page_title: "hivehook_source Resource"
description: |-
  Manages a Hivehook inbound webhook source.
---

# hivehook_source (Resource)

Manages a Hivehook inbound webhook source. Each source has a slug used as the
ingest endpoint (`/ingest/{slug}`) and a provider type that determines how the
HMAC verification is performed.

## Example Usage

```hcl
resource "hivehook_source" "stripe_prod" {
  name          = "Stripe production"
  slug          = "stripe-prod"
  provider_type = "stripe"
  verify_config = jsonencode({
    secret = var.stripe_signing_secret
  })
}
```

## Schema

### Required

- `name` (String) Display name.
- `slug` (String) URL slug for the ingest endpoint (`/ingest/{slug}`).
- `provider_type` (String) Webhook provider (e.g. `generic`, `stripe`, `github`).

### Optional

- `verify_config` (String, JSON) JSON-encoded provider verification config.
- `status` (String) Source status (`ACTIVE` or `INACTIVE`).
- `rate_limit_rps` (Number) Rate limit in requests per second (0 = unlimited).

### Read-Only

- `id` (String) Source UUID.

## Import

```bash
terraform import hivehook_source.stripe_prod <source-uuid>
```

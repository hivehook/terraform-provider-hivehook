---
page_title: "hivehook_subscription Resource"
description: |-
  Manages a Hivehook subscription linking a source to a destination.
---

# hivehook_subscription (Resource)

Manages a Hivehook subscription. A subscription links a source to a destination
and optionally applies a filter and transformation.

## Example Usage

```hcl
resource "hivehook_subscription" "stripe_to_billing" {
  name           = "Stripe to billing API"
  source_id      = hivehook_source.stripe_prod.id
  destination_id = hivehook_destination.billing_api.id
  enabled        = true
}
```

## Schema

### Required

- `name` (String) Display name.
- `source_id` (String) Source UUID to subscribe to.
- `destination_id` (String) Destination UUID to deliver to.

### Optional

- `filter_config` (String, JSON) JSON-encoded filter configuration.
- `transformation_id` (String) Optional transformation UUID.
- `enabled` (Boolean) Whether the subscription is enabled (default `true`).

### Read-Only

- `id` (String) Subscription UUID.

## Import

```bash
terraform import hivehook_subscription.stripe_to_billing <subscription-uuid>
```

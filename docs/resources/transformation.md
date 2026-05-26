---
page_title: "hivehook_transformation Resource"
description: |-
  Manages a Hivehook JavaScript transformation.
---

# hivehook_transformation (Resource)

Manages a Hivehook JavaScript transformation. Transformations are user-supplied
JS functions invoked during delivery to reshape the payload.

## Example Usage

```hcl
resource "hivehook_transformation" "normalize" {
  name        = "Normalize Stripe events"
  description = "Strip Stripe-specific fields and add a tenant tag."
  code        = <<-EOT
    export default function (event) {
      return { ...event, tenant: 'acme' };
    }
  EOT
}
```

## Schema

### Required

- `name` (String) Display name.
- `code` (String) JavaScript transformation code.

### Optional

- `description` (String) Optional description.
- `enabled` (Boolean) Whether the transformation is active (default `true`).
- `fail_open` (Boolean) Pass through on errors instead of dropping the event (default `false`).
- `timeout_ms` (Number) Per-invocation timeout in milliseconds (default `1000`).

### Read-Only

- `id` (String) Transformation UUID.

## Import

```bash
terraform import hivehook_transformation.normalize <transformation-uuid>
```

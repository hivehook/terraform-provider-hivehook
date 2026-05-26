---
page_title: "hivehook_destination Data Source"
description: |-
  Retrieves a Hivehook destination by ID.
---

# hivehook_destination (Data Source)

Retrieves a Hivehook destination by ID.

## Example Usage

```hcl
data "hivehook_destination" "billing" {
  id = "8d5f6c80-1f0d-4b97-8aa3-6dc7a0e6f4f0"
}
```

## Schema

### Required

- `id` (String) Destination UUID.

### Read-Only

- `name` (String)
- `url` (String)
- `status` (String)
- `auth_type` (String)
- `delivery_mode` (String)
- `ordered` (Boolean)

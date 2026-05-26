---
page_title: "hivehook_source Data Source"
description: |-
  Retrieves a Hivehook source by ID.
---

# hivehook_source (Data Source)

Retrieves a Hivehook source by ID.

## Example Usage

```hcl
data "hivehook_source" "stripe" {
  id = "8d5f6c80-1f0d-4b97-8aa3-6dc7a0e6f4f0"
}
```

## Schema

### Required

- `id` (String) Source UUID.

### Read-Only

- `name` (String)
- `slug` (String)
- `provider_type` (String)
- `status` (String)
- `rate_limit_rps` (Number)

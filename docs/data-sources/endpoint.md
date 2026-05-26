---
page_title: "hivehook_endpoint Data Source"
description: |-
  Retrieves a Hivehook endpoint by ID.
---

# hivehook_endpoint (Data Source)

Retrieves a Hivehook endpoint by ID.

## Example Usage

```hcl
data "hivehook_endpoint" "acme_primary" {
  id = "8d5f6c80-1f0d-4b97-8aa3-6dc7a0e6f4f0"
}
```

## Schema

### Required

- `id` (String) Endpoint UUID.

### Read-Only

- `application_id` (String)
- `url` (String)
- `status` (String)
- `auth_type` (String)
- `delivery_mode` (String)
- `ordered` (Boolean)
- `filter_config` (String, JSON)

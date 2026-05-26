---
page_title: "hivehook_application Data Source"
description: |-
  Retrieves a Hivehook application by ID.
---

# hivehook_application (Data Source)

Retrieves a Hivehook application by ID.

## Example Usage

```hcl
data "hivehook_application" "acme" {
  id = "8d5f6c80-1f0d-4b97-8aa3-6dc7a0e6f4f0"
}
```

## Schema

### Required

- `id` (String) Application UUID.

### Read-Only

- `name` (String)
- `uid` (String)

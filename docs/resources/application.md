---
page_title: "hivehook_application Resource"
description: |-
  Manages a Hivehook outbound application.
---

# hivehook_application (Resource)

Manages a Hivehook outbound application: the top-level container for outbound
endpoints and messages.

## Example Usage

```hcl
resource "hivehook_application" "acme" {
  name = "Acme Corp"
}
```

## Schema

### Required

- `name` (String) Display name.

### Read-Only

- `id` (String) Application UUID.
- `uid` (String) Stable user-facing identifier used in outbound URLs.

## Import

```bash
terraform import hivehook_application.acme <application-uuid>
```

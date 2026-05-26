---
page_title: "hivehook_api_key Resource"
description: |-
  Manages a Hivehook API key.
---

# hivehook_api_key (Resource)

Manages a Hivehook API key. API keys are immutable: any change to a configured
attribute forces Terraform to destroy and recreate the key (rotation). The raw
key value is only available at creation time and is stored sensitively in
state.

## Example Usage

```hcl
resource "hivehook_api_key" "ci" {
  name       = "ci-pipeline"
  scopes     = ["sources:read", "deliveries:read"]
  expires_at = "2027-01-01T00:00:00Z"
}

output "ci_api_key" {
  value     = hivehook_api_key.ci.raw_key
  sensitive = true
}
```

## Schema

### Required

- `name` (String, Forces replacement) Display name.

### Optional

- `scopes` (List of String, Forces replacement) Scopes granted to the key.
- `source_ids` (List of String, Forces replacement) Source UUIDs the key is scoped to.
- `expires_at` (String, Forces replacement) Optional RFC3339 expiration timestamp.

### Read-Only

- `id` (String) API key UUID.
- `key_prefix` (String) Public prefix of the API key (safe to display).
- `raw_key` (String, Sensitive) Full API key (only available at creation).

## Import

```bash
terraform import hivehook_api_key.ci <api-key-uuid>
```

After import, `raw_key` will be unknown. The server only returns it at creation time.

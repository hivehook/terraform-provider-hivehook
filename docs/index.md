---
page_title: "Hivehook Provider"
description: |-
  Manage Hivehook webhook gateway resources from Terraform.
---

# Hivehook Provider

The Hivehook provider lets you declaratively manage a
[Hivehook](https://hivehook.com) account: inbound sources, outbound
destinations, subscriptions, applications, endpoints, API keys, alert rules,
and JavaScript transformations.

## Example Usage

```hcl
terraform {
  required_providers {
    hivehook = {
      source  = "hivehook/hivehook"
      version = "~> 0.1"
    }
  }
}

provider "hivehook" {
  endpoint = "https://app.hivehook.com"
  api_key  = var.hivehook_api_key
}
```

## Schema

### Optional

- `endpoint` (String) Hivehook server endpoint (e.g. `https://app.hivehook.com`). May also be set via the `HIVEHOOK_URL` environment variable.
- `api_key` (String, Sensitive) API key used to authenticate against the Hivehook GraphQL admin API. May also be set via the `HIVEHOOK_API_KEY` environment variable.

## Environment Variables

| Variable            | Description                                        |
| ------------------- | -------------------------------------------------- |
| `HIVEHOOK_URL`      | Default endpoint when not set in provider config.  |
| `HIVEHOOK_API_KEY`  | Default API key when not set in provider config.   |

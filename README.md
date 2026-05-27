# Terraform Provider for Hivehook

Manage [Hivehook](https://hivehook.com) sources, destinations, subscriptions, applications, endpoints, transformations, API keys, and alert rules from Terraform. Works against the hosted service at `app.hivehook.com` by default, or against an enterprise self-hosted endpoint.

## Status

Not yet on the Terraform Registry. Build and run it from source with a `dev_overrides` block (below); registry distribution is coming.

## Build from source

```bash
git clone https://github.com/hivehook/terraform-provider-hivehook.git
cd terraform-provider-hivehook
go build -o terraform-provider-hivehook
```

Point Terraform at the directory holding the built binary, in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "hivehook/hivehook" = "/absolute/path/to/terraform-provider-hivehook"
  }
  direct {}
}
```

With `dev_overrides` set, skip `terraform init` and use the provider directly (no `required_providers` block needed):

```hcl
provider "hivehook" {
  endpoint = "https://app.hivehook.com"
  api_key  = var.hivehook_api_key
}

resource "hivehook_source" "stripe_prod" {
  name          = "Stripe production"
  slug          = "stripe-prod"
  provider_type = "stripe"
  verify_config = jsonencode({
    secret = var.stripe_signing_secret
  })
}

resource "hivehook_destination" "billing_api" {
  name = "Billing API"
  url  = "https://billing.internal.example.com/webhooks"
}

resource "hivehook_subscription" "stripe_to_billing" {
  name           = "Stripe to billing"
  source_id      = hivehook_source.stripe_prod.id
  destination_id = hivehook_destination.billing_api.id
  enabled        = true
}
```

## Configuration

| Attribute  | Env var             | Default                  | Description                                                |
| ---------- | ------------------- | ------------------------ | ---------------------------------------------------------- |
| `endpoint` | `HIVEHOOK_URL`      | `https://app.hivehook.com`  | Hivehook server endpoint.                                  |
| `api_key`  | `HIVEHOOK_API_KEY`  |                          | API key for the GraphQL admin API. Sensitive.              |

## Resources

- `hivehook_source`
- `hivehook_destination`
- `hivehook_subscription`
- `hivehook_application`
- `hivehook_endpoint`
- `hivehook_api_key`
- `hivehook_alert_rule`
- `hivehook_transformation`

## Data sources

- `hivehook_source`
- `hivehook_destination`
- `hivehook_application`
- `hivehook_endpoint`

## Importing existing resources

Every managed resource supports `terraform import` by UUID:

```bash
terraform import hivehook_source.stripe_prod <uuid>
```

## Generating docs

`docs/` is generated from schemas and the `examples/` folder via
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs):

```bash
go generate ./...
# or
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate
```

## Acceptance tests

```bash
TF_ACC=1 \
HIVEHOOK_URL=https://app.hivehook.com \
HIVEHOOK_API_KEY=... \
go test ./internal/provider -run TestAcc -count=1 -v
```

## Documentation

See the full reference at [hivehook.com/docs](https://hivehook.com/docs).

## License

MIT. See [LICENSE](LICENSE).

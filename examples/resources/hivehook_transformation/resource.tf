resource "hivehook_transformation" "normalize" {
  name        = "Normalize Stripe events"
  description = "Strip Stripe-specific fields and add a tenant tag."
  code        = <<-EOT
    export default function (event) {
      return { ...event, tenant: 'acme' };
    }
  EOT
  enabled    = true
  fail_open  = false
  timeout_ms = 1000
}

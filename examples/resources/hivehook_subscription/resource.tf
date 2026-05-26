resource "hivehook_subscription" "stripe_to_billing" {
  name           = "Stripe to billing API"
  source_id      = hivehook_source.stripe_prod.id
  destination_id = hivehook_destination.billing_api.id
  enabled        = true
  filter_config = jsonencode({
    eventTypes = ["invoice.paid", "invoice.payment_failed"]
  })
}

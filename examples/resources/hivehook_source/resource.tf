resource "hivehook_source" "stripe_prod" {
  name          = "Stripe production"
  slug          = "stripe-prod"
  provider_type = "stripe"
  verify_config = jsonencode({
    secret = var.stripe_signing_secret
  })
}

variable "stripe_signing_secret" {
  type      = string
  sensitive = true
}

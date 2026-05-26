data "hivehook_source" "stripe" {
  id = "8d5f6c80-1f0d-4b97-8aa3-6dc7a0e6f4f0"
}

output "stripe_slug" {
  value = data.hivehook_source.stripe.slug
}

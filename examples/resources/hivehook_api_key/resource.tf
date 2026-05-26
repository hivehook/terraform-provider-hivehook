resource "hivehook_api_key" "ci" {
  name       = "ci-pipeline"
  scopes     = ["sources:read", "deliveries:read"]
  expires_at = "2027-01-01T00:00:00Z"
}

output "ci_api_key" {
  value     = hivehook_api_key.ci.raw_key
  sensitive = true
}

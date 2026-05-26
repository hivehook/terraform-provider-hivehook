terraform {
  required_providers {
    hivehook = {
      source  = "hivehook/hivehook"
      version = "~> 0.1"
    }
  }
}

provider "hivehook" {
  endpoint = "http://localhost:8080"
  api_key  = var.hivehook_api_key
}

variable "hivehook_api_key" {
  type      = string
  sensitive = true
}

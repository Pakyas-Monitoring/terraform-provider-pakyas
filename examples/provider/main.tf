terraform {
  required_providers {
    pakyas = {
      source = "pakyas/pakyas"
    }
  }
}

provider "pakyas" {
  # API key can be set via PAKYAS_API_KEY environment variable
  # api_key = "pk_live_..."

  # Optional: Override API URL (defaults to https://api.pakyas.com)
  # api_url = "https://api.pakyas.com"
}

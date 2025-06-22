terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "6.39.0"
    }
  }
}

provider "google" {
  project     = var.project_id
  region      = var.region
  credentials = file("~/terraform-key.json")
}
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "6.29.0"
    }
  }

  required_version = ">= 1.5.0"
}

provider "google" {
  credentials = file("./secrets/svc_acc_key.json")
  project     = var.project_id
}

module "gcp_apis" {
  source = "./modules/gcp-apis"
  project_id = var.project_id
}

module "gcp_networking" {
  source = "./modules/gcp-networking"
  project_id = var.project_id
  depends_on = [ module.gcp_apis ]
}

module "gcp_gke_cluster" {
  source = "./modules/gcp-gke-cluster"
  project_id = var.project_id
  depends_on = [ module.gcp_networking ]
}

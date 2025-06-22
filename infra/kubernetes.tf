resource "google_container_cluster" "lime_primary" {
  name     = "${var.project_id}-cluster"
  location = var.region
  deletion_protection = false

  remove_default_node_pool = true
  initial_node_count       = 1
}

resource "google_container_node_pool" "lime_primary_nodes" {
  name       = "${var.project_id}-node-pool"
  location   = var.region
  cluster    = google_container_cluster.lime_primary.name
  node_count = 1

  node_config {
    preemptible  = true
    machine_type = "e2-micro"

    service_account = google_service_account.lime_account.email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}
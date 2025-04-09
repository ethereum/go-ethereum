locals {
  # Defining a service account to grant necessary GKE cluster permisisons
  service_account_email = "terraformer@${var.project_id}.iam.gserviceaccount.com"
}

resource "random_string" "random_node_pool_suffix" {
  lower   = true
  length  = 5
  special = false
  upper   = false
  keepers = {
    node_pool_machine_type = var.node_pool_machine_type
    node_pool_disk_size_gb = var.node_pool_disk_size_gb
  }
}


resource "google_container_cluster" "app_cluster" {
  project                  = var.project_id
  name                     = var.cluster_name
  location                 = var.cluster_location
  # node_locations           = slice(data.google_compute_zones.region_zones.names, 0, 3)
  remove_default_node_pool = true
  initial_node_count       = 1
  deletion_protection      = false # Change this to true when going to production :)

  # Automation section from the UI
  maintenance_policy {
    recurring_window {
      end_time   = "2023-10-02T15:00:00Z"
      recurrence = "FREQ=WEEKLY;BYDAY=WE,TH"
      start_time = "2023-10-02T07:00:00Z"
    }
  }

  notification_config {
    pubsub {
      enabled = false
    }
  }

  vertical_pod_autoscaling {
    enabled = false
  }

  cluster_autoscaling {
    enabled = false
    # autoscaling_profile = "BALANCED" # Defaults to BALANCED
  }

  # Networking
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }
  network         = "projects/${var.project_id}/global/networks/default" # "default"
  subnetwork      = "default"                                            # "default"
  networking_mode = "VPC_NATIVE"
  ip_allocation_policy {
    cluster_ipv4_cidr_block  = "10.16.0.0/14" # The IP address range for the cluster pod IPs
    services_ipv4_cidr_block = "10.20.0.0/20" # The IP address range of the services IPs in this cluster
  }

  default_max_pods_per_node   = 110
  enable_intranode_visibility = false

  addons_config {
    dns_cache_config {
      enabled = true
    }
    http_load_balancing {
      disabled = false
    }
    gce_persistent_disk_csi_driver_config {
      enabled = true
    }
  }

  # SECURITY TAB FROM THE UI
  binary_authorization {
    evaluation_mode = "DISABLED"
  }

  enable_shielded_nodes = true

  confidential_nodes {
    enabled = false
  }

  logging_config {
    enable_components = ["SYSTEM_COMPONENTS", "WORKLOADS"]
  }

  monitoring_config {
    enable_components = ["SYSTEM_COMPONENTS"]
  }

  release_channel {
    channel = "STABLE"
  }

  lifecycle {
    ignore_changes = [
      maintenance_policy
    ]
  }
}

resource "google_container_node_pool" "node_pool" {
  name       = "pool-${random_string.random_node_pool_suffix.result}"
  cluster    = google_container_cluster.app_cluster.id
  node_count = var.node_pool_node_count
  location = var.cluster_location
  management {
    auto_repair  = true
    auto_upgrade = true
  }

  upgrade_settings {
    max_surge       = 1
    max_unavailable = 0
  }

  node_config {
    machine_type    = var.node_pool_machine_type
    disk_size_gb    = var.node_pool_disk_size_gb
    disk_type       = var.node_pool_disk_type
    service_account = local.service_account_email
    oauth_scopes = [
      # The set of Google API scopes to be made available on all of the node VMs under the "default" service account.
      # Use the "https://www.googleapis.com/auth/cloud-platform" scope to grant access to all APIs.
      # It is recommended that you set service_account to a non-default service account and grant IAM roles to that service account for only the resources that it needs.
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/trace.append"
      # "https://www.googleapis.com/auth/cloud-platform"
    ]
    metadata = {
      "disable-legacy-endpoints" = "true"
    }
    # kubelet_config - enabled or disabled
    # Render this block only if variable app_cluster_kubelet_config is true
    dynamic "kubelet_config" {
      for_each = var.cluster_kubelet_config == true ? [1] : []
      content {
        cpu_cfs_quota                          = false
        insecure_kubelet_readonly_port_enabled = "FALSE"
        pod_pids_limit                         = 0
      }
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

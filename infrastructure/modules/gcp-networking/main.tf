data "google_compute_network" "default" {
  project = var.project_id
  name    = "default"
}


# =============================================================================
# === NAT Gateway

# Router
resource "google_compute_router" "default_nat_router" {
  name    = "router"
  network = "default"
  region  = var.region
}

# NAT addresses
resource "google_compute_address" "nat_1" {
  name         = "nat-1-address"
  address_type = "EXTERNAL"
  region       = var.region
  network_tier = "PREMIUM"
}

# Gateway
resource "google_compute_router_nat" "nat_gateway" {
  name                                = "nat-1"
  router                              = google_compute_router.default_nat_router.name
  region                              = var.region
  source_subnetwork_ip_ranges_to_nat  = "ALL_SUBNETWORKS_ALL_IP_RANGES"
  nat_ip_allocate_option              = "MANUAL_ONLY"
  nat_ips                             = google_compute_address.nat_1.*.self_link
  enable_dynamic_port_allocation      = true
  min_ports_per_vm                    = 1024
  max_ports_per_vm                    = 4096
  enable_endpoint_independent_mapping = false
}
